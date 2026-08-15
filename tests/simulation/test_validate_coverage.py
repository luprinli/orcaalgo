"""Tests for validation functions in orca.simulation.validate."""

import numpy as np

from orca.simulation.validate import (
    _compute_daily_returns,
    autocorrelation_check,
    drawdown_check,
    fat_tail_check,
    ks_test_synthetic_vs_real,
    validate_strategy_coverage,
)


def test_compute_daily_returns_1d() -> None:
    prices = np.array([100.0, 101.0, 99.5, 102.0, 103.0])
    rets = _compute_daily_returns(prices)
    assert len(rets) == 4
    assert np.all(np.isfinite(rets))


def test_ks_test_insufficient_data() -> None:
    result = ks_test_synthetic_vs_real(np.array([0.01]), np.array([0.02]))
    assert result["passed"] is False


def test_ks_test_passes_on_similar_distributions() -> None:
    rng = np.random.default_rng(42)
    syn = rng.normal(0, 0.01, 1000)
    real = rng.normal(0, 0.01, 1000)
    result = ks_test_synthetic_vs_real(syn, real)
    assert result["p_value"] > 0.01


def test_autocorrelation_insufficient_data() -> None:
    result = autocorrelation_check(np.array([0.01]), np.array([0.01]))
    assert result["passed"] is False


def test_autocorrelation_computes_rmse() -> None:
    rng = np.random.default_rng(42)
    syn = rng.normal(0, 0.01, 500)
    real = rng.normal(0, 0.01, 500)
    result = autocorrelation_check(syn, real, max_lag=5)
    assert "rmse" in result


def test_fat_tail_insufficient_data() -> None:
    result = fat_tail_check(np.array([0.01]), np.array([0.02]))
    assert result["passed"] is False


def test_fat_tail_reports_kurtosis() -> None:
    rng = np.random.default_rng(42)
    syn = rng.normal(0, 0.01, 200)
    real = rng.normal(0, 0.01, 200)
    result = fat_tail_check(syn, real)
    assert "synthetic_kurtosis" in result
    assert "real_kurtosis" in result


def test_drawdown_insufficient_data() -> None:
    result = drawdown_check(np.array([0.01]), np.array([0.02]))
    assert result["passed"] is False


def test_drawdown_reports_values() -> None:
    rng = np.random.default_rng(42)
    syn = rng.normal(0, 0.01, 200)
    real = rng.normal(0, 0.01, 200)
    result = drawdown_check(syn, real)
    assert "synthetic_max_drawdown" in result
    assert "real_max_drawdown" in result


def test_validate_strategy_coverage_requires_symbol() -> None:
    result = validate_strategy_coverage("gen-1", symbol="")
    assert result["passed"] is False
    assert "error" in result


def test_validate_strategy_coverage_no_orca_cli() -> None:
    result = validate_strategy_coverage(
        "gen-1",
        symbol="SPY",
        orca_cli_path="/nonexistent/orca-cli",
    )
    assert result["passed"] is False
    # Each strategy should have an error
    for strat in ["intraday_mr", "trend_following", "opening_range_breakout", "grid_trading"]:
        assert "error" in result["strategies"].get(strat, {})
