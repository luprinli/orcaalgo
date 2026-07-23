"""Tests for regime classifier training pipeline."""

import numpy as np
import pytest

from orca.ml.train.regime_classifier import (
    build_regime_features,
    continuous_regime_score,
    score_to_kelly_multiplier,
)

try:
    import xgboost  # noqa: F401
    HAS_XGBOOST = True
except ImportError:
    HAS_XGBOOST = False


class TestContinuousRegimeScore:
    def test_calm_gives_high_score(self):
        probs = np.array([1.0, 0.0, 0.0, 0.0, 0.0, 0.0])
        score = continuous_regime_score(probs)
        assert score == pytest.approx(1.0)

    def test_crisis_gives_zero_score(self):
        probs = np.array([0.0, 0.0, 0.0, 0.0, 0.0, 1.0])
        score = continuous_regime_score(probs)
        assert score == pytest.approx(0.0)

    def test_uniform_gives_mid_score(self):
        probs = np.ones(6) / 6.0
        score = continuous_regime_score(probs)
        assert 0.3 < score < 0.9

    def test_high_vol_gives_low_score(self):
        probs = np.array([0.0, 0.0, 0.0, 0.0, 1.0, 0.0])
        score = continuous_regime_score(probs)
        assert score == pytest.approx(0.4)

    def test_custom_weights(self):
        probs = np.array([0.5, 0.5, 0.0, 0.0, 0.0, 0.0])
        score = continuous_regime_score(probs, [1.0, 0.5])
        assert score == pytest.approx(0.75)


class TestScoreToKellyMultiplier:
    def test_max_score_gives_max_mult(self):
        assert score_to_kelly_multiplier(1.0) == pytest.approx(1.5)

    def test_zero_score_gives_zero_mult(self):
        assert score_to_kelly_multiplier(0.0) == pytest.approx(0.0)

    def test_mid_score_gives_proportional(self):
        mult = score_to_kelly_multiplier(0.5)
        assert 0.5 < mult < 1.0

    def test_no_negative(self):
        assert score_to_kelly_multiplier(-1.0) >= 0

    def test_no_above_cap(self):
        assert score_to_kelly_multiplier(2.0) <= 1.5


class TestBuildRegimeFeatures:
    def test_returns_14_features(self):
        alpha = np.array([0.5, 0.3, 0.15, 0.05])
        fv = build_regime_features(alpha)
        assert len(fv) == 14

    def test_alpha_values_preserved(self):
        alpha = np.array([0.6, 0.3, 0.07, 0.03])
        fv = build_regime_features(alpha)
        np.testing.assert_array_equal(fv[:4], alpha)

    def test_vix_normalized(self):
        fv = build_regime_features(np.ones(4), vix=30.0)
        assert fv[4] == pytest.approx(0.3)

    def test_vix_high_flag(self):
        fv = build_regime_features(np.ones(4), vix=30.0)
        assert fv[12] == 1.0
        assert fv[13] == 0.0

    def test_vix_crisis_flag(self):
        fv = build_regime_features(np.ones(4), vix=40.0)
        assert fv[12] == 1.0
        assert fv[13] == 1.0


@pytest.mark.skipif(not HAS_XGBOOST, reason="xgboost not installed")
class TestRegimeClassifier:
    def test_train_returns_result(self):
        from orca.ml.train.regime_classifier import RegimeClassifier
        rng = np.random.default_rng(42)
        n = 300
        X = rng.normal(0, 1, (n, 14))
        y = rng.integers(0, 6, n)

        trainer = RegimeClassifier(n_estimators=10, max_depth=2)
        result = trainer.train(X, y)
        assert result.model is not None
        assert 0 <= result.accuracy <= 1

    def test_save_and_predict(self, tmp_path):
        from orca.ml.train.regime_classifier import RegimeClassifier
        rng = np.random.default_rng(42)
        n = 300
        X = rng.normal(0, 1, (n, 14))
        y = rng.integers(0, 6, n)

        trainer = RegimeClassifier(n_estimators=10, max_depth=2)
        result = trainer.train(X, y)

        path = tmp_path / "regime_model.json"
        trainer.save_model(result, path)
        assert path.exists()
