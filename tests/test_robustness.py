"""Tests for the backtest statistical-robustness summary (R1/R5 wiring)."""

from __future__ import annotations

import numpy as np

from orca.sizing.robustness import backtest_robustness_stats


def _series(n: int, mean: float, std: float, seed: int = 0) -> np.ndarray:
    return np.random.default_rng(seed).normal(mean, std, n)


def test_insufficient_returns_errors():
    out = backtest_robustness_stats(np.array([0.01, 0.02]))
    assert "error" in out
    assert out["n_returns"] == 2


def test_strong_series_has_positive_sharpe_and_tight_ci():
    out = backtest_robustness_stats(_series(1000, 0.002, 0.01))
    assert out["sharpe"] > 1.5
    assert out["sharpe_se"] > 0
    assert out["sharpe_ci_low"] > 0
    assert out["sharpe_ci_high"] > out["sharpe_ci_low"]
    assert out["deflated_sharpe_ratio"] > 0.99
    assert out["min_trl"] < out["n_returns"]


def test_weak_series_deflates_below_psr():
    rng = np.random.default_rng(1)
    returns = rng.normal(0.0001, 0.01, 200)
    out = backtest_robustness_stats(returns, n_trials=50)
    # A barely-positive Sharpe under 50 trials is not credible.
    assert out["deflated_sharpe_ratio"] < 0.95


def test_more_trials_lowers_dsr():
    returns = _series(400, 0.0005, 0.01, seed=2)
    one = backtest_robustness_stats(returns, n_trials=1)
    many = backtest_robustness_stats(returns, n_trials=100)
    assert many["deflated_sharpe_ratio"] < one["deflated_sharpe_ratio"]


def test_min_trl_is_finite_for_positive_edge():
    out = backtest_robustness_stats(_series(500, 0.001, 0.01, seed=3))
    assert out["min_trl"] is not None
    assert out["min_trl"] > 0
