"""Tests for feature computation."""

from datetime import UTC, datetime

import numpy as np
import pytest

from orca.ml.config import FEATURE_NAMES
from orca.ml.feature_selection import validate_feature_vector
from orca.ml.features import (
    compute_full_feature_vector,
    compute_indicator_features,
    compute_price_features,
    compute_regime_features,
    compute_signal_features,
    compute_time_features,
)


def make_test_data(
    n_bars: int = 60, seed: int = 42,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    """Generate synthetic OHLCV data."""
    rng = np.random.default_rng(seed)
    closes = 100.0 + np.cumsum(rng.normal(0, 0.2, n_bars))
    highs = closes + rng.uniform(0.1, 0.5, n_bars)
    lows = closes - rng.uniform(0.1, 0.5, n_bars)
    volumes = rng.uniform(500, 2000, n_bars)
    return closes, highs, lows, volumes


class TestPriceFeatures:
    def test_computes_all_price_features(self):
        closes, highs, lows, _volumes = make_test_data(60)
        features = compute_price_features(closes, highs, lows)
        assert 0 in features  # ret1
        assert 1 in features  # ret5
        assert 2 in features  # ret20
        assert 3 in features  # volatility20
        assert 4 in features  # atr_ratio

    def test_ret1_valid(self):
        closes, highs, lows, _ = make_test_data(60)
        features = compute_price_features(closes, highs, lows)
        # ret1 should be roughly the difference of the last two closes
        assert abs(features[0]) < 0.1  # small moves

    def test_insufficient_bars_raises(self):
        closes = np.array([100.0, 100.5, 100.3])
        with pytest.raises(ValueError):
            compute_price_features(closes, closes, closes)

    def test_volatility_positive(self):
        closes, highs, lows, _ = make_test_data(60)
        features = compute_price_features(closes, highs, lows)
        assert features[3] > 0

    def test_atr_ratio_positive(self):
        closes, highs, lows, _ = make_test_data(60)
        features = compute_price_features(closes, highs, lows)
        assert features[4] > 0


class TestIndicatorFeatures:
    def test_computes_all_indicator_features(self):
        closes, highs, lows, volumes = make_test_data(60)
        features = compute_indicator_features(closes, volumes, highs, lows)
        assert 5 in features  # rsi14
        assert 6 in features  # macd_hist
        assert 7 in features  # adx14
        assert 8 in features  # bb_percent_b
        assert 9 in features  # volume_ratio

    def test_rsi_in_range(self):
        closes, highs, lows, volumes = make_test_data(60)
        features = compute_indicator_features(closes, volumes, highs, lows)
        assert 0 <= features[5] <= 100

    def test_bb_percent_b_in_range(self):
        """%B should typically be in [0, 1] for data within bands."""
        closes, highs, lows, volumes = make_test_data(60)
        features = compute_indicator_features(closes, volumes, highs, lows)
        # Can be outside [0,1] for extreme moves, but usually in range
        assert -1.0 <= features[8] <= 2.0

    def test_volume_ratio_around_one(self):
        closes, highs, lows, volumes = make_test_data(60)
        features = compute_indicator_features(closes, volumes, highs, lows)
        # Volume ratio should be near 1.0 for random data
        assert 0.1 <= features[9] <= 5.0


class TestRegimeFeatures:
    def test_computes_all_regime_features(self):
        alpha = (0.7, 0.2, 0.08, 0.02)
        features = compute_regime_features(alpha, 0.85)
        assert features[12] == 0.7
        assert features[13] == 0.2
        assert features[14] == 0.08
        assert features[15] == 0.02
        assert features[16] == 0.85

    def test_none_alpha_zeros(self):
        features = compute_regime_features(None)
        for i in range(12, 16):
            assert features[i] == 0.0


class TestSignalFeatures:
    def test_computes_signal_features(self):
        features = compute_signal_features(3, 0.75)
        assert features[17] == 3.0
        assert features[18] == 0.75


class TestTimeFeatures:
    def test_midday_features(self):
        ts = datetime(2026, 7, 5, 12, 0, 0, tzinfo=UTC)
        features = compute_time_features(ts)
        # 12:00 → sin(π) ≈ 0, cos(π) ≈ -1
        assert features[19] == pytest.approx(0.0, abs=1e-6)
        assert features[20] == pytest.approx(-1.0, abs=1e-6)

    def test_midnight_features(self):
        ts = datetime(2026, 7, 5, 0, 0, 0, tzinfo=UTC)
        features = compute_time_features(ts)
        # 00:00 → sin(0) = 0, cos(0) = 1
        assert features[19] == pytest.approx(0.0, abs=1e-6)
        assert features[20] == pytest.approx(1.0, abs=1e-6)


class TestFullFeatureVector:
    def test_computes_all_21_features(self):
        closes, highs, lows, volumes = make_test_data(60)
        ts = datetime(2026, 7, 5, 14, 30, 0, tzinfo=UTC)
        fv = compute_full_feature_vector(closes, highs, lows, volumes, ts)
        assert len(fv) == len(FEATURE_NAMES)
        assert fv.shape == (21,)

    def test_no_nan_inf(self):
        closes, highs, lows, volumes = make_test_data(60)
        ts = datetime(2026, 7, 5, 14, 30, 0, tzinfo=UTC)
        fv = compute_full_feature_vector(closes, highs, lows, volumes, ts)
        assert validate_feature_vector(fv)
        assert not np.any(np.isnan(fv))
        assert not np.any(np.isinf(fv))

    def test_with_regime_and_signal(self):
        closes, highs, lows, volumes = make_test_data(60)
        ts = datetime(2026, 7, 5, 14, 30, 0, tzinfo=UTC)
        alpha = (0.6, 0.3, 0.07, 0.03)
        fv = compute_full_feature_vector(
            closes, highs, lows, volumes, ts,
            hmm_alpha=alpha, hmm_confidence=0.9,
            signal_type=2, signal_strength=0.8,
            cvd_divergence=0.5, spread_pct=0.001,
        )
        assert fv[12] == 0.6
        assert fv[16] == 0.9
        assert fv[17] == 2.0
        assert fv[10] == 0.5


class TestValidateFeatureVector:
    def test_valid_vector(self):
        fv = np.zeros(21)
        assert validate_feature_vector(fv)

    def test_nan_invalidates(self):
        fv = np.zeros(21)
        fv[0] = np.nan
        assert not validate_feature_vector(fv)

    def test_inf_invalidates(self):
        fv = np.zeros(21)
        fv[0] = np.inf
        assert not validate_feature_vector(fv)

    def test_wrong_dimension(self):
        fv = np.zeros(10)
        assert not validate_feature_vector(fv)
