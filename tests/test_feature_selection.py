"""Tests for feature selection and PSI drift detection."""

import numpy as np
import pytest

from orca.ml.drift_detection import (
    DriftStatus,
    classify_drift,
    compute_per_feature_psi,
    compute_psi,
    should_retrain,
)
from orca.ml.feature_selection import (
    correlation_filter,
    mutual_information_score,
    rank_features_by_mi,
    validate_features,
)


def make_test_features(n_samples: int = 1000, n_features: int = 10, seed: int = 42) -> np.ndarray:
    rng = np.random.default_rng(seed)
    return rng.normal(0, 1, (n_samples, n_features))


def make_test_labels(n_samples: int = 1000, seed: int = 42) -> np.ndarray:
    rng = np.random.default_rng(seed)
    return rng.binomial(1, 0.55, n_samples)


class TestCorrelationFilter:
    def test_no_redundancy_for_independent_features(self):
        X = make_test_features(1000, 10)
        keep, pairs = correlation_filter(X, threshold=0.9)
        # Independent features should have low correlation, so all should be kept
        assert len(keep) == 10
        assert len(pairs) == 0

    def test_detects_high_correlation(self):
        X = make_test_features(1000, 3)
        X[:, 2] = X[:, 0] * 0.999  # almost identical to feature 0
        keep, pairs = correlation_filter(X, threshold=0.9)
        assert len(keep) == 2  # feature 2 should be removed
        assert len(pairs) >= 1

    def test_identical_features_removed(self):
        X = make_test_features(1000, 3)
        X[:, 1] = X[:, 0]  # identical
        keep, pairs = correlation_filter(X, threshold=0.9)
        assert len(keep) == 2
        assert 1 in {p[1] for p in pairs}  # feature 1 was removed


class TestMutualInformation:
    def test_returns_array(self):
        X = make_test_features(500, 5)
        y = make_test_labels(500)
        scores = mutual_information_score(X, y)
        assert len(scores) == 5
        assert all(s >= 0 for s in scores)

    def test_ranks_features(self):
        rng = np.random.default_rng(42)
        # Feature 0 is strongly predictive
        y = rng.binomial(1, 0.55, 500)
        X = np.column_stack([
            y * 2.0 + rng.normal(0, 0.1, 500),  # strong signal
            rng.normal(0, 1, 500),                # noise
            rng.normal(0, 1, 500),                # noise
            rng.normal(0, 1, 500),                # noise
            rng.normal(0, 1, 500),                # noise
        ])
        ranked = rank_features_by_mi(X, y)
        # Feature 0 should have highest MI
        assert ranked[0][0] == 0


class TestValidateFeatures:
    def test_returns_complete_result(self):
        X = make_test_features(500, 8)
        y = make_test_labels(500)
        result = validate_features(X, y)
        assert "keep_indices" in result
        assert "redundant_pairs" in result
        assert "mi_ranked" in result
        assert "flagged_low_mi" in result
        assert result["passed"] is True

    def test_too_few_features_fails(self):
        X = make_test_features(500, 2)
        y = make_test_labels(500)
        result = validate_features(X, y)
        assert result["passed"] is False


class TestComputePSI:
    def test_identical_distributions_zero_psi(self):
        X = make_test_features(1000, 5)
        psi = compute_psi(X, X)
        assert psi == pytest.approx(0.0, abs=1e-6)

    def test_different_distributions_positive_psi(self):
        X_ref = make_test_features(1000, 5, seed=42)
        X_rec = make_test_features(1000, 5, seed=99)  # different seed = different distribution
        psi = compute_psi(X_ref, X_rec)
        # Should be positive but small for two normal distributions
        assert psi >= 0

    def test_completely_different_distributions_high_psi(self):
        X_ref = make_test_features(1000, 3)
        X_rec = X_ref * 5.0 + 10.0  # shifted and scaled
        psi = compute_psi(X_ref, X_rec)
        assert psi > 0.1  # significant drift

    def test_1d_arrays(self):
        X_ref = np.random.default_rng(42).normal(0, 1, 1000)
        X_rec = np.random.default_rng(99).normal(1, 2, 1000)
        psi = compute_psi(X_ref, X_rec)
        assert psi > 0


class TestPerFeaturePSI:
    def test_returns_per_feature_dict(self):
        X_ref = make_test_features(500, 3)
        X_rec = make_test_features(500, 3, seed=99)
        result = compute_per_feature_psi(X_ref, X_rec)
        assert len(result) == 3


class TestClassifyDrift:
    def test_no_drift(self):
        X = make_test_features(500, 5)
        report = classify_drift(X, X)
        assert report.status == DriftStatus.NO_DRIFT
        assert report.psi_total == pytest.approx(0.0, abs=1e-6)

    def test_significant_drift(self):
        X_ref = make_test_features(500, 5)
        X_rec = X_ref * 10.0 + 50.0
        report = classify_drift(X_ref, X_rec)
        assert report.status == DriftStatus.SIGNIFICANT_DRIFT
        assert len(report.triggers) > 0

    def test_vix_ratio_escalates(self):
        X = make_test_features(500, 5)
        report = classify_drift(X, X, vix_ratio=3.0)
        assert report.status == DriftStatus.MODERATE_DRIFT


class TestShouldRetrain:
    def test_no_triggers(self):
        X = make_test_features(500, 5)
        should, triggers, _drift = should_retrain(X, X, 0.65, 0.60)
        assert not should
        assert len(triggers) == 0

    def test_win_rate_degradation(self):
        X = make_test_features(500, 5)
        should, triggers, _ = should_retrain(X, X, 0.50, 0.60)
        assert should
        assert any("win_rate" in t for t in triggers)

    def test_brier_degradation(self):
        X = make_test_features(500, 5)
        should, triggers, _ = should_retrain(
            X, X, 0.65, 0.60,
            current_brier=0.25, training_brier=0.15,
        )
        assert should
        assert any("brier" in t for t in triggers)

    def test_significant_drift_triggers(self):
        X_ref = make_test_features(500, 5)
        X_rec = X_ref * 10.0 + 50.0
        should, _triggers, drift = should_retrain(X_rec, X_ref, 0.65, 0.60)
        assert should
        assert drift.status == DriftStatus.SIGNIFICANT_DRIFT
