"""Benchmark filter — the mandatory market-relative promotion gate.

Determines whether a strategy's return series is economically meaningful
*relative to* its declared benchmark. The verdict is itself selection-bias
corrected: the active return's Sharpe is deflated by ``n_trials`` (DSR), so a
benchmark-beating result that is merely selection noise does not pass.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import numpy as np

from orca.benchmark.metrics import compute_benchmark_metrics
from orca.benchmark.spec import BenchmarkSpec
from orca.sizing.deflated_sharpe import deflated_sharpe_ratio
from orca.sizing.sharpe_stats import excess_kurtosis, skewness

_EPS = 1e-12


@dataclass(frozen=True)
class BenchmarkVerdict:
    """Outcome of the benchmark filter for one strategy/benchmark pair."""

    passed: bool
    checks: tuple[tuple[str, bool, float], ...]
    metrics: dict[str, Any]
    deflated_active_sharpe: float
    expected_max_sharpe: float
    n_trials: int
    kind: str

    def as_dict(self) -> dict[str, Any]:
        return {
            "passed": self.passed,
            "kind": self.kind,
            "checks": [{"name": n, "passed": ok, "value": v} for n, ok, v in self.checks],
            "metrics": self.metrics,
            "deflated_active_sharpe": self.deflated_active_sharpe,
            "expected_max_sharpe": self.expected_max_sharpe,
            "n_trials": self.n_trials,
        }


def _active_dsr(active: np.ndarray, n_trials: int) -> tuple[float, float]:
    n = active.size
    mean = float(np.mean(active))
    std = float(np.std(active, ddof=1))
    per_period_sharpe = mean / std if std > _EPS else 0.0
    dsr = deflated_sharpe_ratio(
        per_period_sharpe, n, max(n_trials, 1), skewness(active), excess_kurtosis(active)
    )
    return float(dsr["deflated_sharpe_ratio"]), float(dsr["expected_max_sharpe"])


def apply_benchmark_filter(
    strategy: np.ndarray,
    benchmark: np.ndarray,
    spec: BenchmarkSpec,
    n_trials: int = 1,
    periods_per_year: float = 252.0,
) -> BenchmarkVerdict:
    """Apply the benchmark filter to aligned per-period return series.

    Args:
        strategy: Strategy per-period (decimal) returns.
        benchmark: Benchmark per-period (decimal) returns (risk-free returns
            when ``spec.kind == "risk_free"``).
        spec: Frozen ``BenchmarkSpec`` (thresholds + kind).
        n_trials: Number of strategies/combinations tried (DSR deflation).
        periods_per_year: Annualization factor.

    Returns:
        Frozen ``BenchmarkVerdict``.
    """
    s = np.asarray(strategy, dtype=np.float64)
    b = np.asarray(benchmark, dtype=np.float64)
    s = s[np.isfinite(s)]
    b = b[np.isfinite(b)]
    if s.size != b.size:
        raise ValueError("strategy and benchmark must be aligned (same length)")

    metrics = compute_benchmark_metrics(s, b, periods_per_year)
    active = s - b
    dsr_value, expected_max = _active_dsr(active, n_trials)

    t = spec.thresholds
    checks: list[tuple[str, bool, float]] = [
        (
            "information_ratio",
            metrics["information_ratio"] >= t.information_ratio,
            metrics["information_ratio"],
        ),
        ("alpha", metrics["alpha_annualized"] >= t.alpha, metrics["alpha_annualized"]),
        ("deflated_active_sharpe", dsr_value >= t.active_sharpe, dsr_value),
    ]
    if spec.kind == "risk_free":
        annualized_excess = float(np.mean(active) * periods_per_year)
        checks.append(
            ("excess_return", annualized_excess >= spec.risk_free_hurdle, annualized_excess)
        )

    passed = all(ok for _, ok, _ in checks)
    return BenchmarkVerdict(
        passed=passed,
        checks=tuple(checks),
        metrics=metrics,
        deflated_active_sharpe=dsr_value,
        expected_max_sharpe=expected_max,
        n_trials=int(n_trials),
        kind=spec.kind,
    )


__all__ = ["BenchmarkVerdict", "apply_benchmark_filter"]
