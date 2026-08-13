"""Template-level scoring — rank whole strategy families, not just parameter sets.

Where ``param_score`` picks the best cached parameter row for a single template,
this module answers a different question: *how good is this template as a
strategy family?* It consumes stored default-strategy backtests across multiple
periods, with separate training and validation results, and folds in a
verification multiplier from the template's best parameter row.

The final score (0-100) is::

    baseScore = weighted mean of per-period scores
    final     = baseScore * verificationMultiplier   (0.8x .. 1.2x)

Period scores reward validation return, training/validation consistency, low
validation drawdown, and adequate liquidity, and penalise negative validation
CAGR. Longer and fresher windows carry more weight.
"""

from __future__ import annotations

import math
from dataclasses import dataclass
from typing import Any

LOG2 = math.log(2.0)


@dataclass(frozen=True)
class TemplateScoreSettings:
    """Tuning knobs for template scoring (defaults mirror the reference)."""

    return_scale: float = 0.2
    validation_negative_penalty_strength: float = 2.0
    drawdown_lambda: float = 2.5
    trade_target: float = 200.0
    trade_weight: float = 0.25
    recency_half_life_days: float = 365.0
    verify_sharpe_scale: float = 2.0
    verify_calmar_scale: float = 2.0
    verify_cagr_scale: float = 0.25
    verify_cagr_negative_scale: float = 0.1
    verify_drawdown_lambda: float = 2.5
    verify_min_multiplier: float = 0.8
    verify_max_multiplier: float = 1.2


def _clamp01(x: float) -> float:
    return max(0.0, min(1.0, x))


def _period_score(period: dict[str, Any], s: TemplateScoreSettings) -> float | None:
    """Per-period score, or None when required inputs are missing."""
    t_cagr = period.get("training_cagr")
    v_cagr = period.get("validation_cagr")
    v_dd = period.get("validation_max_drawdown_pct")
    trades = period.get("trades")
    months = period.get("period_months")
    if t_cagr is None or v_cagr is None or v_dd is None or trades is None or months is None:
        return None

    if float(v_cagr) >= 0:
        return_score = 1.0 - math.exp(-max(0.0, float(v_cagr)) / s.return_scale)
    else:
        return_score = 0.0

    denom = abs(float(t_cagr)) + abs(float(v_cagr))
    shortfall = max(0.0, float(t_cagr) - float(v_cagr))
    consistency = _clamp01(1.0 - (shortfall / denom if denom > 0 else 0.0))

    risk = math.exp(-s.drawdown_lambda * max(0.0, float(v_dd) / 100.0))

    years = max(float(months) / 12.0, 1e-9)
    trades_per_year = float(trades) / years
    liquidity = (1.0 - s.trade_weight) + s.trade_weight * (
        1.0 - math.exp(-trades_per_year / s.trade_target)
    )

    negative_penalty = (
        math.exp(-s.validation_negative_penalty_strength * abs(float(v_cagr)))
        if float(v_cagr) < 0
        else 1.0
    )

    return _clamp01(return_score * consistency * risk * liquidity * negative_penalty)


def _weight(period: dict[str, Any], s: TemplateScoreSettings) -> float:
    months = max(1.0, float(period.get("period_months", 1) or 1))
    length = math.sqrt(months)
    age = float(period.get("age_days", 0) or 0)
    recency = 0.6 + 0.4 * math.exp(-LOG2 * age / s.recency_half_life_days)
    return length * recency


def _verification_multiplier(verify: dict[str, Any] | None, s: TemplateScoreSettings) -> float:
    if not verify:
        return 1.0
    sharpe = max(0.0, float(verify.get("sharpe_ratio", 0.0) or 0.0))
    calmar = max(0.0, float(verify.get("calmar_ratio", 0.0) or 0.0))
    cagr = float(verify.get("cagr", 0.0) or 0.0)
    dd = max(0.0, float(verify.get("max_drawdown_ratio", 0.0) or 0.0))

    sharpe_score = 1.0 - math.exp(-sharpe / s.verify_sharpe_scale)
    calmar_score = 1.0 - math.exp(-calmar / s.verify_calmar_scale)
    if cagr >= 0:
        cagr_score = 0.5 + 0.5 * (1.0 - math.exp(-cagr / s.verify_cagr_scale))
    else:
        cagr_score = 0.5 - 0.5 * (1.0 - math.exp(-abs(cagr) / s.verify_cagr_negative_scale))
    dd_score = math.exp(-s.verify_drawdown_lambda * dd)

    verification: float = (sharpe_score * calmar_score * cagr_score * dd_score) ** (1.0 / 4.0)
    band = s.verify_max_multiplier - s.verify_min_multiplier
    return s.verify_min_multiplier + band * verification


def compute_template_scores(
    periods: list[dict[str, Any]],
    template_verify: dict[str, dict[str, Any]] | None = None,
    settings: TemplateScoreSettings | None = None,
) -> list[dict[str, Any]]:
    """Score strategy templates from per-period training/validation results.

    Each period dict must contain::

        {
            "template": str,              # strategy-family identifier
            "period_months": int,
            "age_days": int,              # recency for weighting
            "training_cagr": float,
            "validation_cagr": float,
            "validation_max_drawdown_pct": float,
            "trades": int,
        }

    ``template_verify`` optionally maps a template id to its best parameter
    row's verification metrics::

        {"sharpe_ratio", "calmar_ratio", "cagr", "max_drawdown_ratio"}

    Returns a list of template score dicts sorted by ``final_score_100``
    descending.
    """
    settings = settings or TemplateScoreSettings()
    verify_map = template_verify or {}

    groups: dict[str, list[dict[str, Any]]] = {}
    for p in periods:
        groups.setdefault(p["template"], []).append(p)

    results: list[dict[str, Any]] = []
    for template, ps in groups.items():
        weighted_sum = 0.0
        weight_total = 0.0
        components: list[dict[str, Any]] = []
        for p in ps:
            score = _period_score(p, settings)
            if score is None:
                continue
            w = _weight(p, settings)
            weighted_sum += score * w
            weight_total += w
            components.append({
                "period_months": p.get("period_months"),
                "age_days": p.get("age_days", 0),
                "score": score,
                "weight": w,
            })

        base_score = _clamp01(weighted_sum / weight_total) if weight_total > 0 else 0.0
        multiplier = _verification_multiplier(verify_map.get(template), settings)
        final = base_score * multiplier
        results.append({
            "template": template,
            "base_score": base_score,
            "verification_multiplier": multiplier,
            "final_score": final,
            "final_score_100": round(_clamp01(final) * 100),
            "n_periods": len(components),
            "components": components,
        })

    results.sort(key=lambda r: r["final_score"], reverse=True)
    return results


__all__ = ["TemplateScoreSettings", "compute_template_scores"]
