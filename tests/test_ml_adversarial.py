"""Adversarial testing for ML models.

Tests model behavior under adversarial conditions:
  - Corrupted feature vectors (NaN, Inf, extreme values)
  - Missing features (zero vectors, partial inputs)
  - Regime shift scenarios (rapid VIX changes, state transitions)
  - High-volume signal bursts
"""

from __future__ import annotations

import numpy as np

from orca.ml.drift_detection import compute_psi
from orca.ml.feature_selection import validate_feature_vector
from orca.ml.features import (
    compute_full_feature_vector,
)
from orca.ml.train.evaluate import evaluate_model
from orca.ml.train.regime_classifier import continuous_regime_score


def make_test_data(n_bars: int = 60, seed: int = 42, vol_scale: float = 0.2):
    rng = np.random.default_rng(seed)
    closes = 100.0 + np.cumsum(rng.normal(0, vol_scale, n_bars))
    highs = closes + rng.uniform(0.1, 0.5, n_bars)
    lows = closes - rng.uniform(0.1, 0.5, n_bars)
    volumes = rng.uniform(500, 2000, n_bars)
    return closes, highs, lows, volumes


class TestCorruptedFeatures:
    def test_nan_features_rejected(self):
        closes, highs, lows, volumes = make_test_data(60)
        from datetime import UTC, datetime
        fv = compute_full_feature_vector(
            closes, highs, lows, volumes,
            datetime(2026, 7, 6, 14, 0, tzinfo=UTC),
        )
        fv[0] = np.nan
        assert not validate_feature_vector(fv)

    def test_inf_features_rejected(self):
        closes, highs, lows, volumes = make_test_data(60)
        from datetime import UTC, datetime
        fv = compute_full_feature_vector(
            closes, highs, lows, volumes,
            datetime(2026, 7, 6, 14, 0, tzinfo=UTC),
        )
        fv[3] = np.inf
        assert not validate_feature_vector(fv)

    def test_zero_vector_validates(self):
        fv = np.zeros(21)
        assert validate_feature_vector(fv)

    def test_extreme_prices_dont_produce_nan(self):
        rng = np.random.default_rng(99)
        closes = np.exp(np.cumsum(rng.normal(0, 0.5, 80)))
        highs = closes * 1.01
        lows = closes * 0.99
        volumes = np.full(80, 1000.0)
        from datetime import UTC, datetime
        fv = compute_full_feature_vector(
            closes, highs, lows, volumes,
            datetime(2026, 7, 6, 14, 0, tzinfo=UTC),
        )
        assert validate_feature_vector(fv)

    def test_negative_prices_handled(self):
        closes = np.array([-50.0] * 40 + [100.0] * 40)
        from datetime import UTC, datetime
        fv = compute_full_feature_vector(
            closes, closes, closes, closes,
            datetime(2026, 7, 6, 14, 0, tzinfo=UTC),
        )
        assert validate_feature_vector(fv)


class TestRegimeShift:
    def test_psi_detects_vol_regime_change(self):
        ref_data = make_test_data(200, seed=1, vol_scale=0.2)[0].reshape(-1, 1)
        shift_data = make_test_data(200, seed=2, vol_scale=2.0)[0].reshape(-1, 1)
        psi = compute_psi(ref_data, shift_data)
        assert psi > 0

    def test_gradual_shift_lower_psi(self):
        ref_data = make_test_data(200, seed=1)[0].reshape(-1, 1)
        shift_data = ref_data * 1.001  # tiny change
        psi = compute_psi(ref_data, shift_data)
        assert psi < 0.5

    def test_regime_score_crisis_is_low(self):
        probs = np.array([0.0, 0.0, 0.0, 0.0, 0.0, 1.0])
        score = continuous_regime_score(probs)
        assert score < 0.1


class TestSignalBurst:
    def test_many_signals_dont_crash(self):
        closes, highs, lows, volumes = make_test_data(200)
        from datetime import UTC, datetime
        for i in range(100):
            fv = compute_full_feature_vector(
                closes[i:i + 60], highs[i:i + 60], lows[i:i + 60],
                volumes[i:i + 60], datetime(2026, 7, 6, 10, 0, tzinfo=UTC),
            )
            assert validate_feature_vector(fv)

    def test_model_predictions_stable(self):
        rng = np.random.default_rng(42)
        y_true = rng.binomial(1, 0.5, 200)
        y_prob = rng.uniform(0, 1, 200)
        result = evaluate_model(y_true, y_prob)
        assert 0 <= result.brier_score <= 1
        assert 0 <= result.accuracy <= 1
        assert 0 <= result.f1_score <= 1
