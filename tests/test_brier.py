from __future__ import annotations

import pytest

from orca.math.brier import (
    BinStats,
    MurphyResult,
    brier_score,
    murphy_decomposition,
)


class TestBrierScore:
    def test_perfect_predictions(self):
        score = brier_score([1.0, 1.0, 0.0, 0.0], [1, 1, 0, 0])
        assert score == pytest.approx(0.0)

    def test_worst_predictions(self):
        score = brier_score([0.0, 0.0, 1.0, 1.0], [1, 1, 0, 0])
        assert score == pytest.approx(1.0)

    def test_mid_predictions(self):
        score = brier_score([0.5, 0.5, 0.5, 0.5], [1, 1, 0, 0])
        assert score == pytest.approx(0.25)

    def test_random_better_than_worst(self):
        perfect = brier_score([1.0, 1.0, 0.0, 0.0], [1, 1, 0, 0])
        mid = brier_score([0.5, 0.5, 0.5, 0.5], [1, 1, 0, 0])
        worst = brier_score([0.0, 0.0, 1.0, 1.0], [1, 1, 0, 0])
        assert perfect < mid < worst

    def test_length_mismatch_raises(self):
        with pytest.raises(ValueError, match="same length"):
            brier_score([0.5, 0.5], [1])

    def test_empty_input_raises(self):
        with pytest.raises(ValueError, match="Empty"):
            brier_score([], [])

    def test_single_element(self):
        score = brier_score([0.8], [1])
        assert score == pytest.approx(0.04)

    def test_single_element_wrong(self):
        score = brier_score([0.2], [1])
        assert score == pytest.approx(0.64)


class TestMurphyDecomposition:
    def test_returns_murphy_result(self):
        predictions = [0.1, 0.3, 0.5, 0.7, 0.9]
        outcomes = [0, 0, 1, 1, 1]
        result = murphy_decomposition(predictions, outcomes, n_bins=5)
        assert isinstance(result, MurphyResult)

    def test_perfect_predictions_decomposition(self):
        predictions = [0.99] * 100 + [0.01] * 100
        outcomes = [1] * 100 + [0] * 100
        result = murphy_decomposition(predictions, outcomes, n_bins=5)
        assert result.brier < 0.01
        assert result.reliability < 0.01
        assert result.resolution > 0.2
        assert result.uncertainty == pytest.approx(0.25)

    def test_brier_equals_decomposition_sum(self):
        predictions = [0.05] * 3 + [0.25] * 3 + [0.45] * 3 + [0.65] * 3 + [0.85] * 3
        outcomes = [0, 0, 0] + [0, 1, 1] + [1, 1, 0] + [1, 1, 1] + [1, 1, 1]
        result = murphy_decomposition(predictions, outcomes, n_bins=5)
        computed = result.reliability - result.resolution + result.uncertainty
        assert computed == pytest.approx(result.brier, abs=1e-10)

    def test_empty_input_raises(self):
        with pytest.raises(ValueError, match="Empty"):
            murphy_decomposition([], [], n_bins=5)

    def test_bin_stats_count(self):
        predictions = [0.1, 0.3, 0.5, 0.7, 0.9]
        outcomes = [0, 0, 1, 1, 1]
        result = murphy_decomposition(predictions, outcomes, n_bins=5)
        assert len(result.bin_stats) == 5
        total_count = sum(bs.count for bs in result.bin_stats)
        assert total_count == len(predictions)

    def test_bin_stats_values(self):
        predictions = [0.05, 0.95, 0.55, 0.15, 0.85]
        outcomes = [0, 1, 1, 0, 1]
        result = murphy_decomposition(predictions, outcomes, n_bins=10)
        non_empty = [bs for bs in result.bin_stats if bs.count > 0]
        assert len(non_empty) > 0

    def test_base_rate_computation(self):
        predictions = [0.6, 0.7, 0.8]
        outcomes = [1, 1, 1]
        result = murphy_decomposition(predictions, outcomes, n_bins=3)
        assert result.uncertainty == pytest.approx(0.0)

    def test_uniform_predictions(self):
        predictions = [0.5] * 100
        outcomes = [1] * 50 + [0] * 50
        result = murphy_decomposition(predictions, outcomes, n_bins=10)
        assert result.resolution == pytest.approx(0.0, abs=1e-10)
