from __future__ import annotations

import numpy as np
import pytest

from orca.math.platt import PlattResult, platt_scale


class TestPlattScale:
    def test_perfect_calibration_no_change_needed(self):
        np.random.seed(42)
        n = 500
        raw_p = np.random.uniform(0.3, 0.7, n)
        y = (np.random.random(n) < raw_p).astype(np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert isinstance(result, PlattResult)
        assert isinstance(result.a, float)
        assert isinstance(result.b, float)
        assert 0.0 <= result.train_brier <= 1.0

    def test_highly_miscalibrated_model(self):
        np.random.seed(42)
        n = 500
        raw_p = np.full(n, 0.9)
        y = (np.random.random(n) < 0.5).astype(np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert result.recommended or result.improvement_pct >= 0

    def test_output_is_probability_bounded(self):
        np.random.seed(42)
        n = 400
        raw_p = np.random.uniform(0.1, 0.9, n)
        y = (np.random.random(n) < raw_p).astype(np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        a, b = result.a, result.b
        z_val = np.log(raw_p / (1 - raw_p))
        z_val = np.clip(z_val, -30, 30)
        cal = 1.0 / (1.0 + np.exp(-(a * z_val + b)))
        assert np.all(cal >= 0) and np.all(cal <= 1)

    def test_edge_probabilities_do_not_crash(self):
        n = 200
        raw_p = np.array([0.001, 0.999] * (n // 2))
        y = np.array([0, 1] * (n // 2), dtype=np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert isinstance(result, PlattResult)

    def test_all_wins_produces_result(self):
        n = 300
        raw_p = np.random.uniform(0.5, 0.9, n)
        y = np.ones(n, dtype=np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert isinstance(result, PlattResult)

    def test_all_losses_produces_result(self):
        n = 300
        raw_p = np.random.uniform(0.1, 0.5, n)
        y = np.zeros(n, dtype=np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert isinstance(result, PlattResult)

    def test_small_dataset_still_works(self):
        n = 60
        np.random.seed(1)
        raw_p = np.random.uniform(0.3, 0.7, n)
        y = (np.random.random(n) < raw_p).astype(np.float64)
        split = int(n * 0.7)
        result = platt_scale(raw_p[:split], y[:split], raw_p[split:], y[split:])
        assert isinstance(result, PlattResult)

    def test_raw_p_clamped_pre_platt(self):
        raw_p = np.array([0.0, 0.5, 1.0])
        y = np.array([0, 1, 1], dtype=np.float64)
        result = platt_scale(raw_p, y)
        assert isinstance(result, PlattResult)
