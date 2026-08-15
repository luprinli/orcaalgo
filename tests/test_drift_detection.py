from __future__ import annotations

import numpy as np
import pytest

from orca.ml.drift_detection import (
    DriftStatus,
    classify_drift,
    compute_per_feature_psi,
    compute_psi,
    should_retrain,
)


class TestComputePSI:
    def test_identical_distributions_yield_zero(self):
        data = np.random.default_rng(42).normal(0, 1, (1000, 3))
        psi = compute_psi(data, data)
        assert psi < 0.05

    def test_different_distributions_yield_positive(self):
        ref = np.random.default_rng(42).normal(0, 1, (500, 3))
        rec = np.random.default_rng(99).normal(1.5, 2, (500, 3))
        psi = compute_psi(ref, rec)
        assert psi > 0.1

    def test_1d_input_is_accepted(self):
        ref = np.random.default_rng(42).normal(0, 1, 200)
        rec = np.random.default_rng(99).normal(0, 1, 200)
        psi = compute_psi(ref, rec)
        assert psi >= 0

    def test_constant_feature_is_skipped(self):
        ref = np.array([[5.0], [5.0], [5.0]], dtype=np.float64)
        rec = np.array([[5.0], [5.0], [5.0]], dtype=np.float64)
        psi = compute_psi(ref, rec, bins=10, epsilon=1e-6)
        assert psi == 0.0

    def test_custom_bins(self):
        ref = np.random.default_rng(42).normal(0, 1, (300, 2))
        rec = np.random.default_rng(99).normal(0.5, 1, (300, 2))
        psi10 = compute_psi(ref, rec, bins=10)
        psi20 = compute_psi(ref, rec, bins=20)
        assert psi10 > 0
        assert psi20 > 0

    def test_epsilon_prevents_division_by_zero(self):
        ref = np.array([[0.0], [1.0], [2.0], [3.0]])
        rec = np.array([[0.0], [1.0], [2.0], [3.0]])
        psi = compute_psi(ref, rec, epsilon=1e-12)
        assert psi < 0.001

    def test_mismatched_sample_counts(self):
        ref = np.random.default_rng(42).normal(0, 1, (1000, 2))
        rec = np.random.default_rng(99).normal(0, 1, (50, 2))
        psi = compute_psi(ref, rec)
        assert psi >= 0


class TestComputePerFeaturePSI:
    def test_returns_dict_with_correct_keys(self):
        ref = np.random.default_rng(42).normal(0, 1, (500, 3))
        rec = np.random.default_rng(99).normal(0.2, 1.2, (500, 3))
        names = ["momentum", "vol", "skew"]
        result = compute_per_feature_psi(ref, rec, feature_names=names)
        assert set(result.keys()) == set(names)
        assert all(v >= 0 for v in result.values())

    def test_default_feature_names(self):
        ref = np.random.default_rng(42).normal(0, 1, (200, 4))
        rec = np.random.default_rng(99).normal(0.1, 1, (200, 4))
        result = compute_per_feature_psi(ref, rec)
        assert set(result.keys()) == {"f0", "f1", "f2", "f3"}

    def test_fewer_names_than_features(self):
        ref = np.random.default_rng(42).normal(0, 1, (200, 4))
        rec = np.random.default_rng(99).normal(0.1, 1, (200, 4))
        result = compute_per_feature_psi(ref, rec, feature_names=["a", "b"])
        assert "a" in result
        assert "b" in result
        assert "f2" in result
        assert "f3" in result


class TestClassifyDrift:
    def test_no_drift_when_psi_low(self):
        data = np.random.default_rng(42).normal(0, 1, (500, 3))
        report = classify_drift(data, data)
        assert report.status == DriftStatus.NO_DRIFT
        assert report.psi_total < 0.05
        assert len(report.triggers) == 0

    def test_moderate_drift(self):
        ref = np.random.default_rng(42).normal(0, 1, (500, 3))
        rec = np.random.default_rng(99).normal(0.5, 1, (500, 3))
        report = classify_drift(ref, rec, moderate_threshold=0.01)
        assert report.status in (DriftStatus.MODERATE_DRIFT, DriftStatus.SIGNIFICANT_DRIFT)

    def test_significant_drift(self):
        ref = np.random.default_rng(42).normal(0, 1, (500, 3))
        rec = np.random.default_rng(99).normal(2.0, 2, (500, 3))
        report = classify_drift(ref, rec, significant_threshold=0.05)
        assert report.status == DriftStatus.SIGNIFICANT_DRIFT
        assert len(report.triggers) > 0

    def test_vix_ratio_escalates_status(self):
        data = np.random.default_rng(42).normal(0, 1, (500, 3))
        report = classify_drift(data, data, vix_ratio=2.5)
        assert report.status == DriftStatus.MODERATE_DRIFT
        assert "vix_ratio" in report.triggers[0]

    def test_vix_ratio_below_threshold_does_not_trigger(self):
        data = np.random.default_rng(42).normal(0, 1, (500, 3))
        report = classify_drift(data, data, vix_ratio=1.5)
        assert report.status == DriftStatus.NO_DRIFT
        assert len(report.triggers) == 0

    def test_custom_thresholds(self):
        data = np.random.default_rng(42).normal(0, 1, (500, 3))
        report = classify_drift(data, data, significant_threshold=0.5, moderate_threshold=0.5)
        assert report.status == DriftStatus.NO_DRIFT

    def test_report_is_immutable(self):
        data = np.random.default_rng(42).normal(0, 1, (100, 2))
        report = classify_drift(data, data)
        with pytest.raises(AttributeError):
            report.psi_total = 999.0


class TestShouldRetrain:
    def test_no_retrain_when_all_ok(self):
        data = np.random.default_rng(42).normal(0, 1, (200, 3))
        should, triggers, _drift = should_retrain(
            recent_features=data,
            training_features=data,
            current_win_rate=0.55,
            baseline_win_rate=0.53,
        )
        assert not should
        assert len(triggers) == 0

    def test_retrain_on_significant_drift(self):
        ref = np.random.default_rng(42).normal(0, 1, (200, 3))
        rec = np.random.default_rng(99).normal(3.0, 3, (200, 3))
        should, _triggers, _ = should_retrain(
            recent_features=rec,
            training_features=ref,
            current_win_rate=0.55,
            baseline_win_rate=0.53,
        )
        assert should

    def test_retrain_on_win_rate_degradation(self):
        data = np.random.default_rng(42).normal(0, 1, (200, 3))
        should, triggers, _ = should_retrain(
            recent_features=data,
            training_features=data,
            current_win_rate=0.40,
            baseline_win_rate=0.55,
            win_rate_degradation_pp=0.05,
        )
        assert should
        assert any("win_rate" in t for t in triggers)

    def test_retrain_on_brier_degradation(self):
        data = np.random.default_rng(42).normal(0, 1, (200, 3))
        should, triggers, _ = should_retrain(
            recent_features=data,
            training_features=data,
            current_win_rate=0.55,
            baseline_win_rate=0.53,
            current_brier=0.15,
            training_brier=0.10,
            brier_degradation_ratio=1.10,
        )
        assert should
        assert any("brier" in t for t in triggers)

    def test_no_retrain_when_brier_near_parity(self):
        data = np.random.default_rng(42).normal(0, 1, (200, 3))
        should, _triggers, _ = should_retrain(
            recent_features=data,
            training_features=data,
            current_win_rate=0.55,
            baseline_win_rate=0.53,
            current_brier=0.11,
            training_brier=0.10,
        )
        assert not should

    def test_custom_degradation_thresholds(self):
        data = np.random.default_rng(42).normal(0, 1, (200, 3))
        should, _, _ = should_retrain(
            recent_features=data,
            training_features=data,
            current_win_rate=0.40,
            baseline_win_rate=0.55,
            win_rate_degradation_pp=0.20,
        )
        assert not should
