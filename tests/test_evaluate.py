"""Tests for model evaluation functions."""

import numpy as np
import pytest

from orca.ml.train.evaluate import (
    brier_score,
    evaluate_model,
    murphy_decomposition,
    profit_curve,
)


class TestBrierScore:
    def test_perfect_prediction(self):
        y_true = np.array([1, 0, 1, 0])
        y_prob = np.array([1.0, 0.0, 1.0, 0.0])
        assert brier_score(y_true, y_prob) == pytest.approx(0.0)

    def test_bad_prediction(self):
        y_true = np.array([1, 0, 1, 0])
        y_prob = np.array([0.0, 1.0, 0.0, 1.0])
        assert brier_score(y_true, y_prob) == pytest.approx(1.0)

    def test_uniform_prediction(self):
        y_true = np.array([1, 0, 1, 0])
        y_prob = np.array([0.5, 0.5, 0.5, 0.5])
        assert brier_score(y_true, y_prob) == pytest.approx(0.25)


class TestMurphyDecomposition:
    def test_perfect_calibration(self):
        rng = np.random.default_rng(42)
        n = 1000
        y_prob = rng.uniform(0.3, 0.7, n)
        y_true = (rng.uniform(0, 1, n) < y_prob).astype(int)
        result = murphy_decomposition(y_true, y_prob)
        assert result.brier >= 0
        assert result.reliability >= 0
        assert result.resolution >= 0
        assert result.uncertainty >= 0

    def test_decomposition_sums_correctly(self):
        rng = np.random.default_rng(42)
        n = 500
        y_prob = rng.uniform(0, 1, n)
        y_true = rng.binomial(1, 0.5, n)
        result = murphy_decomposition(y_true, y_prob)
        assert result.brier >= 0
        assert result.uncertainty <= 0.25


class TestEvaluateModel:
    def test_perfect_model(self):
        y_true = np.array([1, 0, 1, 0, 1, 0, 1, 0])
        y_prob = np.array([0.9, 0.1, 0.8, 0.2, 0.7, 0.3, 0.6, 0.4])
        result = evaluate_model(y_true, y_prob, threshold=0.55)
        assert result.accuracy >= 0.5
        assert result.precision >= 0.5

    def test_random_model_fails_gates(self):
        rng = np.random.default_rng(42)
        n = 200
        y_true = rng.binomial(1, 0.5, n)
        y_prob = rng.uniform(0.4, 0.6, n)
        result = evaluate_model(y_true, y_prob)
        assert not result.passed_gates

    def test_all_metrics_returned(self):
        rng = np.random.default_rng(42)
        n = 200
        y_true = rng.binomial(1, 0.5, n)
        y_prob = rng.uniform(0, 1, n)
        result = evaluate_model(y_true, y_prob)
        assert result.brier_score >= 0
        assert result.roc_auc >= 0
        assert result.accuracy >= 0
        assert result.precision >= 0
        assert result.recall >= 0
        assert result.f1_score >= 0


class TestProfitCurve:
    def test_returns_expected_keys(self):
        rng = np.random.default_rng(42)
        n = 200
        y_true = rng.binomial(1, 0.5, n)
        y_prob = rng.uniform(0, 1, n)
        curve = profit_curve(y_true, y_prob)
        assert "thresholds" in curve
        assert "n_accepted" in curve
        assert "win_rate" in curve
        assert "rejection_rate" in curve

    def test_higher_threshold_fewer_accepted(self):
        rng = np.random.default_rng(42)
        n = 500
        y_true = rng.binomial(1, 0.5, n)
        y_prob = rng.uniform(0, 1, n)
        curve = profit_curve(y_true, y_prob, n_thresholds=20)
        for i in range(len(curve["thresholds"]) - 1):
            assert curve["n_accepted"][i] >= curve["n_accepted"][i + 1]
