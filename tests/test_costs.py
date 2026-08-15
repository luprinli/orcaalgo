"""Tests for transaction-cost calibration (R2)."""

from __future__ import annotations

import numpy as np

from orca.costs.calibrate import calibrate_symbol_costs
from orca.costs.impact import fit_sqrt_impact, kyle_lambda
from orca.costs.spread import corwin_schultz, roll_spread


def test_corwin_schultz_output_shape():
    rng = np.random.default_rng(0)
    close = 100 * np.exp(np.cumsum(rng.normal(0, 0.01, 200)))
    high = close * 1.001
    low = close * 0.999
    spread = corwin_schultz(high, low)
    assert spread.shape == (200,)
    assert np.isnan(spread[0])


def test_roll_spread_negative_autocov_positive():
    # Deterministic alternating price changes => negative first-order autocov.
    dp = np.tile([1.0, -1.0], 50)
    close = np.cumsum(dp)
    s = roll_spread(close)
    assert np.isfinite(s)
    assert s > 0


def test_roll_spread_non_negative_autocov_nan():
    # Constant price changes => zero autocov => estimator undefined.
    dp = np.ones(100)
    close = np.cumsum(dp)
    assert np.isnan(roll_spread(close))


def test_fit_sqrt_impact_recovers_eta():
    rng = np.random.default_rng(1)
    n = 1000
    volume = rng.uniform(1000, 100000, n)
    adv = float(np.mean(volume))
    sigma = 0.01
    eta_true = 1.5
    x = np.sqrt(volume / adv)
    returns = sigma * eta_true * x * np.sign(rng.normal(size=n))
    eta_hat = fit_sqrt_impact(returns, volume, adv=adv, sigma=sigma)
    assert abs(eta_hat - eta_true) < 1e-6


def test_fit_sqrt_impact_insufficient_data():
    assert fit_sqrt_impact(np.ones(3), np.ones(3)) == 0.0


def test_kyle_lambda_recovers_lambda():
    rng = np.random.default_rng(2)
    n = 200
    flow = rng.uniform(-100, 100, n)
    lam = 0.02
    dp = lam * flow
    lam_hat = kyle_lambda(dp, flow)
    assert abs(lam_hat - lam) < 1e-9


def test_kyle_lambda_zero_flow_undefined():
    assert kyle_lambda(np.ones(3), np.zeros(3)) == 0.0


def test_calibrate_symbol_costs_shape_and_keys():
    rng = np.random.default_rng(3)
    close = 100 * np.exp(np.cumsum(rng.normal(0, 0.01, 300)))
    high = close * 1.002
    low = close * 0.998
    volume = rng.uniform(1000, 100000, 300)
    result = calibrate_symbol_costs(high, low, close, volume)
    assert result["n_bars"] == 300
    assert set(result) >= {"spread_bps", "roll_spread_bps", "impact_eta", "n_bars"}
    assert result["impact_eta"] >= 0.0
    assert result["mean_close"] > 0
