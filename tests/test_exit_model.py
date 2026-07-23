"""Tests for path-dependent exit labeling and exit model."""

import numpy as np
import pytest

from orca.ml.train.exit_labels import (
    batch_exit_labels,
    build_exit_features,
    path_dependent_label,
)
from orca.ml.train.exit_model import urgency_to_stop_multiplier


class TestPathDependentLabel:
    def test_favorable_forward_return_hold(self):
        prices = np.array([100.0, 101.0, 102.0, 103.0, 104.0, 105.0])
        label = path_dependent_label(prices, 1, 101.0, 100.0, look_forward=3)
        assert label.urgency == 0.0
        assert label.label_type == "favorable"

    def test_adverse_forward_return_exit(self):
        prices = np.array([100.0, 99.0, 98.0, 97.0, 96.0, 95.0])
        label = path_dependent_label(prices, 1, 99.0, 100.0, look_forward=3)
        assert label.urgency == 1.0
        assert label.label_type == "adverse"

    def test_neutral_returns_flat(self):
        prices = np.full(20, 100.0)
        label = path_dependent_label(prices, 5, 100.0, 100.0, look_forward=5)
        assert label.urgency == 0.5
        assert label.label_type == "neutral"

    def test_near_end_of_array(self):
        prices = np.array([100.0, 101.0, 101.0, 101.0])
        label = path_dependent_label(prices, 1, 101.0, 100.0, look_forward=10)
        assert label.urgency == 0.5  # flat price → neutral when forward data limited


class TestBuildExitFeatures:
    def test_returns_12_features(self):
        fv = build_exit_features(
            entry_price=100.0, current_price=102.0,
            current_stop=98.0, high_since_entry=103.0,
            low_since_entry=97.0, bars_since_entry=5,
            atr=1.0, vol_at_entry=0.01, vol_current=0.012,
            hmm_state=1, cvd_trend=0.0, volume_trend=0.0,
            adx=25.0, hour=12.0, signal_confidence=0.5,
        )
        assert len(fv) == 12

    def test_pnl_computed_correctly(self):
        fv = build_exit_features(
            entry_price=100.0, current_price=105.0,
            current_stop=97.0, high_since_entry=105.0,
            low_since_entry=97.0, bars_since_entry=10,
            atr=1.0, vol_at_entry=0.01, vol_current=0.01,
            hmm_state=1, cvd_trend=0.0, volume_trend=0.0,
            adx=25.0, hour=12.0, signal_confidence=0.5,
        )
        assert fv[0] > 0  # positive PnL

    def test_adx_normalized(self):
        fv = build_exit_features(
            entry_price=100.0, current_price=100.0,
            current_stop=98.0, high_since_entry=100.0,
            low_since_entry=98.0, bars_since_entry=1,
            atr=1.0, vol_at_entry=0.01, vol_current=0.01,
            hmm_state=0, cvd_trend=0.0, volume_trend=0.0,
            adx=50.0, hour=6.0, signal_confidence=0.5,
        )
        assert fv[7] == pytest.approx(1.0)


class TestUrgencyToStop:
    def test_hold_gives_wider_stop(self):
        mult = urgency_to_stop_multiplier(0.0, base_multiplier=2.0)
        assert mult == pytest.approx(2.0)

    def test_exit_now_gives_tighter_stop(self):
        mult = urgency_to_stop_multiplier(1.0, base_multiplier=2.0)
        assert mult == pytest.approx(3.0)

    def test_neutral_gives_moderate_stop(self):
        mult = urgency_to_stop_multiplier(0.5, base_multiplier=2.0)
        assert mult == pytest.approx(2.5)


class TestBatchExitLabels:
    def test_returns_empty_on_no_data(self):
        X, y = batch_exit_labels([], {})
        assert X.shape == (0, 12)
        assert y.shape == (0,)

    def test_skips_invalid_trades(self):
        trades = [{"symbol": "TEST", "entry_price": 0, "exit_price": 0}]
        X, _y = batch_exit_labels(trades, {"TEST": np.array([100.0])})
        assert len(X) == 0
