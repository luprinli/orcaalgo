"""Tests for the deflated Sharpe ratio and CSCV PBO (R1)."""

from __future__ import annotations

import numpy as np

from orca.sizing.deflated_sharpe import (
    cscv_pbo,
    deflated_sharpe_ratio,
    expected_max_sharpe,
    minimum_track_record_length,
    probabilistic_sharpe_ratio,
)


def test_expected_max_sharpe_single_trial_is_zero():
    assert expected_max_sharpe(1, 100) == 0.0


def test_expected_max_sharpe_scales_with_observations():
    # Same trial count, different sample sizes: scales as 1/sqrt(n-1).
    a = expected_max_sharpe(100, 100)
    b = expected_max_sharpe(100, 400)
    assert a > 0
    assert abs(a * np.sqrt(99) - b * np.sqrt(399)) < 1e-12


def test_expected_max_sharpe_monotonic_in_trials():
    lo = expected_max_sharpe(10, 100)
    mid = expected_max_sharpe(100, 100)
    assert mid > lo > expected_max_sharpe(1, 100)


def test_psr_zero_benchmark_normal():
    # PSR(SR=0, bench=0) == 0.5 under normality (the null sits at the median).
    assert abs(probabilistic_sharpe_ratio(0.0, 0.0, 100) - 0.5) < 1e-12


def test_psr_high_sharpe_near_one():
    p = probabilistic_sharpe_ratio(1.0, 0.0, 100)
    assert p > 0.99


def test_deflated_sharpe_ratio_deflates():
    # A modest Sharpe under many trials deflates below its undeflated PSR.
    result = deflated_sharpe_ratio(0.05, 100, n_trials=100)
    assert result["deflated_sharpe_ratio"] < result["probabilistic_sharpe_ratio"]
    assert result["expected_max_sharpe"] > 0


def test_deflated_sharpe_ratio_single_trial_equals_psr():
    # With one trial there is no deflation: DSR == PSR against zero.
    result = deflated_sharpe_ratio(0.5, 100, n_trials=1)
    assert abs(result["expected_max_sharpe"]) < 1e-15
    assert abs(result["deflated_sharpe_ratio"] - result["probabilistic_sharpe_ratio"]) < 1e-12


def test_minimum_track_record_length_non_positive_edge():
    assert minimum_track_record_length(0.0, benchmark=0.1) == float("inf")


def test_minimum_track_record_length_finite():
    n = minimum_track_record_length(1.0, benchmark=0.0)
    assert n > 0
    assert np.isfinite(n)


def test_cscv_pbo_rejects_1d_input():
    result = cscv_pbo(np.random.default_rng(0).normal(0, 1, 100))
    assert "error" in result
    assert np.isnan(result["pbo"])


def test_cscv_pbo_low_for_genuine_strategy():
    rng = np.random.default_rng(7)
    t = 400
    # Strategy 0 has strong genuine drift; strategies 1..3 are pure noise.
    returns = np.column_stack(
        [
            rng.normal(0.004, 0.01, t),
            rng.normal(0.0, 0.01, t),
            rng.normal(0.0, 0.01, t),
            rng.normal(0.0, 0.01, t),
        ]
    )
    result = cscv_pbo(returns, n_splits=16, seed=1)
    assert result["n_combinations"] > 0
    assert result["pbo"] < 0.1


def test_cscv_pbo_deterministic_with_seed():
    rng = np.random.default_rng(7)
    t = 320
    returns = np.column_stack([rng.normal(0.001, 0.01, t), rng.normal(0, 0.01, t)])
    a = cscv_pbo(returns, n_splits=16, seed=42)
    b = cscv_pbo(returns, n_splits=16, seed=42)
    assert a["pbo"] == b["pbo"]
