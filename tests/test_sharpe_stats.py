"""Tests for the closed-form Sharpe standard error (R5)."""

from __future__ import annotations

import numpy as np

from orca.sizing.sharpe_stats import (
    newey_west_se,
    sharpe_se,
    sharpe_se_from_stats,
    sharpe_variance,
)


def test_sharpe_se_from_stats_normal_zero_sharpe():
    # For SR=0, normal: SE = sqrt(1/(n-1)).
    se = sharpe_se_from_stats(0.0, 100)
    assert abs(se - np.sqrt(1.0 / 99.0)) < 1e-12


def test_sharpe_se_from_stats_grows_with_sharpe():
    # Non-zero Sharpe inflates the variance: sqrt((1 + SR^2/2)/(n-1)).
    se = sharpe_se_from_stats(1.0, 100)
    assert abs(se - np.sqrt(1.5 / 99.0)) < 1e-12


def test_sharpe_se_from_stats_too_few_observations():
    assert np.isnan(sharpe_se_from_stats(1.0, 2))


def test_sharpe_se_annualizes():
    rng = np.random.default_rng(0)
    returns = rng.normal(0.0005, 0.01, 1000)
    se_ann = sharpe_se(returns, periods_per_year=252.0)
    assert se_ann > 0
    # Roughly equal to the IID normal approximation scaled by sqrt(252).
    per_period_sr = np.mean(returns) / np.std(returns, ddof=1)
    approx = np.sqrt((1 + 0.5 * per_period_sr**2) / (len(returns) - 1)) * np.sqrt(252.0)
    assert abs(se_ann - approx) < 0.05


def test_sharpe_variance_is_squared_se():
    rng = np.random.default_rng(1)
    returns = rng.normal(0, 0.01, 500)
    assert abs(sharpe_variance(returns) - sharpe_se(returns) ** 2) < 1e-12


def test_newey_west_se_constant_is_zero():
    returns = np.full(200, 0.01)
    assert newey_west_se(returns) < 1e-12


def test_newey_west_se_positive_for_noisy_series():
    rng = np.random.default_rng(2)
    returns = rng.normal(0, 0.01, 1000)
    se = newey_west_se(returns)
    assert se > 0
    assert se < 0.01


def test_newey_west_se_reduces_to_iid_for_white_noise():
    rng = np.random.default_rng(3)
    returns = rng.normal(0, 0.02, 5000)
    se = newey_west_se(returns, max_lag=1)
    approx = np.std(returns, ddof=1) / np.sqrt(len(returns))
    assert abs(se - approx) < 0.001
