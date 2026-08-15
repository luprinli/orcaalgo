"""Layered per-parameter-set scoring for backtest candidates.

Implements a layered parameter-scoring methodology: a candidate is ranked not
only by raw performance but by a composite that penalises overfitting in four
orthogonal ways:

1. **Core quality** — percentile ranks of Sharpe, Calmar and total return,
   blended with the (optional) verification-window metrics via a geometric mean.
2. **Drawdown penalty** — exponential penalty on max drawdown.
3. **Neighbourhood stability** — preference for plateaus over isolated spikes in
   parameter space (see ``score_backtest_parameters``).
4. **Balance penalty** — asymmetric penalty when training CAGR collapses to
   validation CAGR on a common later window.

The final score is::

    final = core * ddPenalty * stability^gamma * balancePenalty

Rows are plain dicts so the scorer can run over any serialised backtest result
(JSON, CSV, DB). Canonical keys are snake_case (see :func:`score_backtest_parameters`).
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any

import numpy as np

# Smallest non-zero percentile/quality floor to keep the geometric means defined.
EPS = 1e-9


@dataclass(frozen=True)
class ParamScoreSettings:
    """Tuning knobs for parameter scoring (defaults mirror the reference)."""

    min_trades: int = 20
    drawdown_lambda: float = 3.5
    neighbor_threshold: float = 0.15
    pairwise_neighbor_limit: int = 1500
    stability_gamma: float = 2.0

    # Plateau detection is meaningless below this many eligible candidates, so
    # stability is held neutral (1.0) for tiny pools. Without this guard a sparse
    # pool would collapse every "isolated" best point to a zero score.
    min_pool_for_stability: int = 8

    # Parameter names that are not optimisation dimensions and are ignored when
    # measuring neighbourhood distances (they would otherwise dominate the
    # distance metric and mask real plateaus).
    ignored_params: tuple[str, ...] = field(default=("initial_capital", "max_leverage", "ticker"))


def _cbrt(x: float) -> float:
    """Sign-preserving cube root (avoids numpy's domain restriction)."""
    return math.copysign(abs(x) ** (1.0 / 3.0), x) if x != 0 else 0.0


def _percentile_ranks(values: list[float]) -> list[float]:
    """Empirical percentile ranks in [0, 1] with ties averaged."""
    n = len(values)
    if n == 0:
        return []
    order = sorted(range(n), key=lambda i: values[i])
    ranks = [0.0] * n
    i = 0
    while i < n:
        j = i
        while j < n and values[order[j]] == values[order[i]]:
            j += 1
        avg_rank = (i + j - 1) / 2.0
        pct = avg_rank / (n - 1) if n > 1 else 0.5
        for k in range(i, j):
            ranks[order[k]] = pct
        i = j
    return ranks


def _clamp(x: float, lo: float = 0.0, hi: float = 1.0) -> float:
    return max(lo, min(hi, x))


def _param_scales(rows: list[dict[str, Any]], ignored: tuple[str, ...]) -> dict[str, float]:
    """Per-parameter spread (p90 - p10) used to normalise neighbourhood distance."""
    names: set[str] = set()
    for r in rows:
        params = r.get("parameters") or {}
        names.update(params.keys())
    scales: dict[str, float] = {}
    for name in sorted(names):
        if name in ignored:
            continue
        vals = [
            r["parameters"][name] for r in rows if (r.get("parameters") or {}).get(name) is not None
        ]
        if len(vals) < 2:
            scales[name] = 1.0
            continue
        spread = float(np.percentile(vals, 90) - np.percentile(vals, 10))
        scales[name] = spread if spread > 0 else 1.0
    return scales


def _distance(
    a: dict[str, Any],
    b: dict[str, Any],
    scales: dict[str, float],
) -> float:
    """Normalised root-mean-square parameter distance (missing value -> z^2 = 1)."""
    if not scales:
        return 0.0
    sum_sq = 0.0
    count = 0
    for name in scales:
        va = a.get(name)
        vb = b.get(name)
        if va is None or vb is None:
            sum_sq += 1.0
        else:
            z = (va - vb) / scales[name]
            sum_sq += z * z
        count += 1
    if count == 0:
        return 0.0
    return math.sqrt(sum_sq / count)


def _eligible(rows: list[dict[str, Any]], min_trades: int) -> list[dict[str, Any]]:
    """Filter rows missing key inputs or with too few trades."""
    out: list[dict[str, Any]] = []
    for r in rows:
        if r.get("sharpe_ratio") is None or r.get("calmar_ratio") is None:
            continue
        if r.get("total_return") is None or not r.get("parameters"):
            continue
        if int(r.get("trades", 0) or 0) < min_trades:
            continue
        out.append(r)
    return out


def score_backtest_parameters(
    rows: list[dict[str, Any]],
    settings: ParamScoreSettings | None = None,
) -> list[dict[str, Any]]:
    """Score cached backtest parameter rows and return them sorted descending.

    Each input row is a dict with, at minimum::

        {
            "parameters": {"param": value, ...},
            "sharpe_ratio": float,
            "calmar_ratio": float,
            "total_return": float,
            "trades": int,
            "max_drawdown_ratio": float,   # 0..1
        }

    Optional keys, each strengthening the anti-overfit signal::

        "verify_sharpe_ratio", "verify_calmar_ratio", "verify_cagr",
        "verify_max_drawdown_ratio", "balance_training_cagr",
        "balance_validation_cagr"

    Returns a list of the same dicts augmented with ``core_score``,
    ``drawdown_penalty``, ``stability_score``, ``balance_penalty`` and
    ``final_score``, sorted by ``final_score`` descending.
    """
    settings = settings or ParamScoreSettings()
    eligible = _eligible(rows, settings.min_trades)
    if not eligible:
        return []

    # 1. Percentile ranks for the training core.
    sharpe_pct = _percentile_ranks([r["sharpe_ratio"] for r in eligible])
    calmar_pct = _percentile_ranks([r["calmar_ratio"] for r in eligible])
    return_pct = _percentile_ranks([r["total_return"] for r in eligible])

    # Verify percentiles (only over rows that have verify metrics).
    has_verify = [r for r in eligible if r.get("verify_sharpe_ratio") is not None]
    verify_pct: dict[int, dict[str, float]] = {}
    if has_verify:
        v_sharpe = _percentile_ranks([r["verify_sharpe_ratio"] for r in has_verify])
        v_calmar = _percentile_ranks([r["verify_calmar_ratio"] for r in has_verify])
        v_return = _percentile_ranks(
            [r.get("verify_total_return", r.get("verify_cagr", 0.0)) or 0.0 for r in has_verify]
        )
        for idx, r in enumerate(has_verify):
            verify_pct[id(r)] = {
                "sharpe": v_sharpe[idx],
                "calmar": v_calmar[idx],
                "return": v_return[idx],
            }

    scored: list[dict[str, Any]] = []
    for i, r in enumerate(eligible):
        core_train = _cbrt((sharpe_pct[i] + EPS) * (calmar_pct[i] + EPS) * (return_pct[i] + EPS))
        core_score = core_train
        if id(r) in verify_pct:
            v = verify_pct[id(r)]
            core_verify = _cbrt((v["sharpe"] + EPS) * (v["calmar"] + EPS) * (v["return"] + EPS))
            core_score = math.sqrt(core_train * core_verify)

        dd_ratio = max(0.0, float(r.get("max_drawdown_ratio", 0.0) or 0.0))
        dd_penalty = math.exp(-settings.drawdown_lambda * dd_ratio)
        if r.get("verify_max_drawdown_ratio") is not None:
            vdd = max(0.0, float(r["verify_max_drawdown_ratio"]))
            dd_penalty = math.sqrt(dd_penalty * math.exp(-settings.drawdown_lambda * vdd))

        quality = core_score * dd_penalty
        balance_penalty = _balance_penalty(r)

        scored.append(
            {
                **r,
                "core_score": core_score,
                "drawdown_penalty": dd_penalty,
                "quality": quality,
                "balance_penalty": balance_penalty,
            }
        )

    # 3. Neighbourhood stability over the *quality* values (core * ddPenalty).
    # Held neutral for tiny pools where plateau detection is not meaningful.
    if len(scored) < settings.min_pool_for_stability:
        for s in scored:
            s["stability_score"] = 1.0
            s["neighbor_count"] = 0
    else:
        _apply_stability(scored, settings)

    for s in scored:
        s["final_score"] = (
            s["core_score"]
            * s["drawdown_penalty"]
            * (s["stability_score"] ** settings.stability_gamma)
            * s["balance_penalty"]
        )

    scored.sort(key=lambda s: s["final_score"], reverse=True)
    return scored


def _apply_stability(scored: list[dict[str, Any]], settings: ParamScoreSettings) -> None:
    """Compute the plateau-preference stability score in place."""
    scales = _param_scales(scored, settings.ignored_params)
    qualities = [s["quality"] for s in scored]
    q_min, q_max = min(qualities), max(qualities)
    q_span = (q_max - q_min) or 1.0

    for idx, s in enumerate(scored):
        neighbors = 0
        neigh_quality_sum = 0.0
        for j, other in enumerate(scored):
            if j == idx:
                continue
            d = _distance(s["parameters"], other["parameters"], scales)
            if d <= settings.neighbor_threshold:
                neighbors += 1
                neigh_quality_sum += other["quality"]
        if neighbors > 0:
            neigh_mean = neigh_quality_sum / neighbors
            normalized = (neigh_mean - q_min) / q_span
            density = _clamp(neighbors / 4.0)
            stability = _clamp(normalized * density)
        else:
            stability = 0.0
        s["stability_score"] = stability
        s["neighbor_count"] = neighbors


def _balance_penalty(r: dict[str, Any]) -> float:
    """Asymmetric train->validation CAGR degradation penalty (defaults 1)."""
    train = r.get("balance_training_cagr")
    val = r.get("balance_validation_cagr")
    if train is None or val is None:
        return 1.0
    shortfall = max(0.0, float(train) - float(val))
    denom = abs(float(train)) + abs(float(val))
    if denom == 0:
        return 1.0
    base = _clamp(1.0 - shortfall / denom)
    return base * base


__all__ = ["ParamScoreSettings", "score_backtest_parameters"]
