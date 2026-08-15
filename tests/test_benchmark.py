"""Tests for the market-based benchmark filter (Phase 0)."""

from __future__ import annotations

import numpy as np
import pytest
from pydantic import ValidationError

from orca.benchmark import (
    BenchmarkSpec,
    BenchmarkThresholds,
    apply_benchmark_filter,
    compute_benchmark_metrics,
)


def _series(n: int, mean: float, std: float, seed: int = 0) -> np.ndarray:
    return np.random.default_rng(seed).normal(mean, std, n)


# ─── Spec ───────────────────────────────────────────────────────────────


def test_spec_is_frozen():
    spec = BenchmarkSpec(kind="equity_index")
    with pytest.raises(ValidationError):
        spec.kind = "custom"  # type: ignore[misc]
    with pytest.raises(ValidationError):
        BenchmarkSpec(kind="unknown")  # type: ignore[arg-type]


def test_spec_default_ticker():
    assert BenchmarkSpec(kind="equity_index").ticker == "SPY"
    assert BenchmarkSpec(kind="growth_index").ticker == "QQQ"


def test_spec_custom_requires_tickers():
    with pytest.raises(ValidationError):
        BenchmarkSpec(kind="custom")


def test_spec_custom_weights_must_sum_to_one():
    with pytest.raises(ValidationError):
        BenchmarkSpec(kind="custom", tickers=["A", "B"], weights=[0.5, 0.4])


def test_spec_custom_weights_must_match_tickers():
    with pytest.raises(ValidationError):
        BenchmarkSpec(kind="custom", tickers=["A", "B"], weights=[1.0])


def test_spec_sector_requires_ticker():
    with pytest.raises(ValidationError):
        BenchmarkSpec(kind="sector_etf")


def test_spec_resolved_tickers_and_weights():
    spec = BenchmarkSpec(kind="custom", tickers=["A", "B", "C"])
    assert spec.resolved_tickers() == ["A", "B", "C"]
    assert spec.resolved_weights() == pytest.approx([1 / 3] * 3)


def test_thresholds_frozen_defaults():
    t = BenchmarkThresholds()
    assert t.information_ratio == 0.4
    assert t.alpha == 0.0
    assert t.active_sharpe == 0.0


# ─── Metrics ─────────────────────────────────────────────────────────────


def test_metrics_beta_replicator_identity():
    b = _series(1000, 0.0005, 0.01, seed=1)
    m = compute_benchmark_metrics(b, b)
    assert m["beta"] == pytest.approx(1.0, abs=1e-6)
    assert m["alpha_annualized"] == pytest.approx(0.0, abs=1e-6)
    assert m["information_ratio"] == pytest.approx(0.0, abs=1e-6)
    assert m["correlation"] == pytest.approx(1.0, abs=1e-6)


def test_metrics_beta_scales_linearly():
    b = _series(1000, 0.0005, 0.01, seed=2)
    m = compute_benchmark_metrics(2.0 * b, b)
    assert m["beta"] == pytest.approx(2.0, abs=1e-6)
    assert m["alpha_annualized"] == pytest.approx(0.0, abs=1e-4)


def test_metrics_constant_edge_yields_alpha():
    b = _series(1000, 0.0005, 0.01, seed=3)
    edge = 0.001
    m = compute_benchmark_metrics(b + edge, b)
    assert m["alpha_annualized"] == pytest.approx(edge * 252.0, abs=1e-4)


def test_metrics_mismatched_lengths_raise():
    with pytest.raises(ValueError):
        compute_benchmark_metrics(_series(100, 0, 0.01), _series(90, 0, 0.01))


def test_metrics_win_rate_and_capture_bounded():
    s = _series(1000, 0.001, 0.01, seed=4)
    b = _series(1000, 0.0005, 0.01, seed=5)
    m = compute_benchmark_metrics(s, b)
    assert 0.0 <= m["win_rate_vs_benchmark"] <= 1.0
    assert m["relative_max_drawdown"] <= 0.0


# ─── Filter ─────────────────────────────────────────────────────────────


def _beating_series(n: int = 1000, seed: int = 7) -> tuple[np.ndarray, np.ndarray]:
    b = _series(n, 0.0005, 0.01, seed=seed)
    s = b + _series(n, 0.001, 0.005, seed=seed + 1)
    return s, b


def test_filter_passes_benchmark_beating_strategy():
    s, b = _beating_series()
    verdict = apply_benchmark_filter(s, b, BenchmarkSpec(kind="equity_index"), n_trials=1)
    assert verdict.passed
    assert verdict.metrics["information_ratio"] >= 0.4
    assert verdict.metrics["alpha_annualized"] >= 0.0


def test_filter_rejects_pure_beta_replicator():
    b = _series(1000, 0.0005, 0.01, seed=9)
    verdict = apply_benchmark_filter(b, b, BenchmarkSpec(kind="equity_index"), n_trials=1)
    assert not verdict.passed
    names = [c[0] for c in verdict.checks if not c[1]]
    assert "information_ratio" in names


def test_filter_deflates_with_more_trials():
    s, b = _beating_series(seed=10)
    one = apply_benchmark_filter(s, b, BenchmarkSpec(kind="equity_index"), n_trials=1)
    many = apply_benchmark_filter(s, b, BenchmarkSpec(kind="equity_index"), n_trials=1000)
    assert many.deflated_active_sharpe < one.deflated_active_sharpe


def test_filter_risk_free_hurdle():
    rng = np.random.default_rng(11)
    risk_free = np.full(500, 0.00008)
    strategy = risk_free + rng.normal(0.00005, 0.003, 500)
    spec = BenchmarkSpec(kind="risk_free", risk_free_hurdle=0.0)
    verdict = apply_benchmark_filter(strategy, risk_free, spec, n_trials=1)
    assert "excess_return" in {c[0] for c in verdict.checks}


def test_filter_mismatched_lengths_raise():
    with pytest.raises(ValueError):
        apply_benchmark_filter(_series(100, 0, 0.01), _series(90, 0, 0.01), BenchmarkSpec())


def test_filter_verdict_as_dict_shape():
    s, b = _beating_series()
    d = apply_benchmark_filter(s, b, BenchmarkSpec()).as_dict()
    assert set(d) >= {"passed", "kind", "checks", "metrics", "deflated_active_sharpe", "n_trials"}


# ─── Promotion-gate benchmark-column veto ───────────────────────────────


def _run_gate(rows: list[dict]) -> dict:
    import csv
    import io
    import os
    import tempfile

    from orca.sizing.promotion_gate import apply_promotion_gate

    buf = io.StringIO()
    w = csv.DictWriter(buf, fieldnames=list(rows[0].keys()))
    w.writeheader()
    for r in rows:
        w.writerow(r)
    fd, path = tempfile.mkstemp(suffix=".csv")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(buf.getvalue())
        return apply_promotion_gate(path).as_dict()
    finally:
        os.unlink(path)


def _strong_row(**overrides: str) -> dict[str, str]:
    row = {
        "Strategy": "s1",
        "Symbol": "X",
        "Tf": "1d",
        "Trades": "100",
        "Sharpe": "2.0",
        "WfISSharpe": "",
        "WfOOSSharpe": "",
    }
    row.update(overrides)
    return row


def test_gate_benchmark_fail_excluded():
    result = _run_gate([_strong_row(BenchmarkPass="false")])
    assert result["n_benchmark_failed"] == 1
    assert result["passed"] is False
    assert result["survivors"] == []


def test_gate_benchmark_pass_included():
    result = _run_gate([_strong_row(BenchmarkPass="true")])
    assert result["n_benchmark_failed"] == 0
    assert len(result["survivors"]) == 1
    assert result["survivors"][0]["benchmark"] == "pass"


def test_gate_benchmark_absent_is_na():
    result = _run_gate([_strong_row()])
    assert result["n_benchmark_failed"] == 0
    assert len(result["survivors"]) == 1
    assert result["survivors"][0]["benchmark"] == "n/a"
