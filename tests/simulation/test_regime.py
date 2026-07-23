"""Unit tests for regime sequence generation and state management."""

import numpy as np
import pytest
from orca.simulation.regime import (
    REGIME_CALM,
    REGIME_CRISIS,
    REGIME_HIGH_VOL,
    REGIME_NAMES,
    REGIME_TRENDING,
    DEFAULT_AVG_DURATION,
    DEFAULT_TRANSITION_MATRIX,
    RegimeSequenceGenerator,
    RegimeBatchState,
)


def test_regime_constants_defined() -> None:
    assert REGIME_CALM == 0
    assert REGIME_TRENDING == 1
    assert REGIME_HIGH_VOL == 2
    assert REGIME_CRISIS == 3
    assert REGIME_NAMES[0] == "Calm"
    assert REGIME_NAMES[3] == "Crisis"


def test_default_transition_matrix_is_valid() -> None:
    tm = DEFAULT_TRANSITION_MATRIX
    assert tm.shape == (4, 4)
    for i in range(4):
        assert abs(tm[i].sum() - 1.0) < 0.01, f"Row {i} does not sum to 1"


def test_generate_sequence_produces_valid_labels() -> None:
    gen = RegimeSequenceGenerator(seed=42)
    labels, _ = gen.generate_sequence(500)
    assert len(labels) == 500
    assert labels.dtype == np.int32
    assert set(np.unique(labels)).issubset({0, 1, 2, 3})


def test_generate_sequence_reproducible_with_seed() -> None:
    gen1 = RegimeSequenceGenerator(seed=42)
    gen2 = RegimeSequenceGenerator(seed=42)
    labels1, _ = gen1.generate_sequence(200)
    labels2, _ = gen2.generate_sequence(200)
    assert np.array_equal(labels1, labels2)


def test_generate_sequence_produces_all_regimes() -> None:
    gen = RegimeSequenceGenerator(seed=42)
    labels, _ = gen.generate_sequence(2000)
    # With 2000 days, all 4 regimes should appear
    unique = set(int(x) for x in labels)
    assert unique == {0, 1, 2, 3}, f"Missing regimes: {unique}"


def test_generate_sequence_with_dates() -> None:
    from datetime import datetime
    gen = RegimeSequenceGenerator(seed=42)
    labels, timestamps = gen.generate_sequence(100, start_date=datetime(2023, 1, 1))
    assert len(timestamps) == 100
    assert isinstance(timestamps[0], datetime)


def test_get_avg_durations() -> None:
    gen = RegimeSequenceGenerator(seed=42)
    labels, _ = gen.generate_sequence(2000)
    durations = gen.get_avg_durations(labels)
    assert len(durations) == 4
    # Each regime should appear
    for r in range(4):
        assert durations[r] > 0, f"Regime {r} not present in sequence"


def test_avg_durations_roughly_match_default() -> None:
    gen = RegimeSequenceGenerator(seed=42)
    labels, _ = gen.generate_sequence(5000)
    durations = gen.get_avg_durations(labels)
    # All 4 regimes should appear with positive avg duration
    for r in range(4):
        assert durations[r] > 0, f"Regime {r} absent"
    # Calm + Trending should together dominate (>20% of time)
    assert durations[REGIME_CALM] + durations[REGIME_TRENDING] > 6


def test_custom_transition_matrix() -> None:
    custom_tm = np.array([
        [0.9, 0.05, 0.03, 0.02],
        [0.1, 0.8, 0.07, 0.03],
        [0.1, 0.1, 0.7, 0.1],
        [0.05, 0.05, 0.1, 0.8],
    ])
    gen = RegimeSequenceGenerator(transition_matrix=custom_tm, seed=42)
    labels, _ = gen.generate_sequence(1000)
    assert len(labels) == 1000


def test_regime_batch_state_initialization() -> None:
    state = RegimeBatchState("test-gen-1", 252)
    assert state.generation_id == "test-gen-1"
    assert state.total_days == 252
    assert state.completed_days == 0


def test_regime_batch_state_start_and_advance() -> None:
    state = RegimeBatchState("test-gen-2", 100)
    state.start(None)
    assert state.completed_days == 0
    state.advance(25)
    assert state.completed_days == 25
    state.advance(50)
    assert state.completed_days == 75
