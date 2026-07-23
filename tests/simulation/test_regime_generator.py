"""Unit tests for Heston regime-aware synthetic data generation."""

import numpy as np
import pytest
from orca.simulation.regime_generator import generate_regime_aware


def test_generate_regime_aware_returns_complete_tuple() -> None:
    """generate_regime_aware should return (gen_id, candles_df, labels, state)."""
    gen_id, df, labels, state = generate_regime_aware(
        symbol="TEST",
        start_date="2023-01-03",
        end_date="2023-01-10",
        seed=42,
    )
    assert isinstance(gen_id, str) and len(gen_id) > 0
    assert len(df) > 0
    assert len(labels) > 0
    assert state.total_days > 0


def test_generate_regime_aware_produces_ohlcv_columns() -> None:
    """Output DataFrame must have OHLCV + regime columns."""
    _, df, _, _ = generate_regime_aware(
        symbol="TEST",
        start_date="2023-01-03",
        end_date="2023-01-10",
        seed=42,
    )
    for col in ["open", "high", "low", "close", "volume", "regime_label"]:
        assert col in df.columns, f"Column {col} missing"


def test_generate_regime_aware_regime_labels_valid() -> None:
    """Regime labels must be integers 0-3."""
    _, df, labels, _ = generate_regime_aware(
        symbol="TEST",
        start_date="2023-01-03",
        end_date="2023-01-17",
        seed=99,
    )
    assert set(int(l) for l in labels).issubset({0, 1, 2, 3})


def test_generate_regime_aware_reproducible() -> None:
    """Same seed produces same output."""
    gid1, df1, l1, _ = generate_regime_aware(symbol="T", start_date="2023-01-03", end_date="2023-01-07", seed=42)
    gid2, df2, l2, _ = generate_regime_aware(symbol="T", start_date="2023-01-03", end_date="2023-01-07", seed=42)
    assert gid1 == gid2
    assert len(df1) == len(df2)
    assert np.array_equal(l1, l2)


def test_generate_regime_aware_different_seeds_different() -> None:
    """Different seeds produce different output."""
    _, _, l1, _ = generate_regime_aware(symbol="U", start_date="2023-01-03", end_date="2023-01-07", seed=1)
    _, _, l2, _ = generate_regime_aware(symbol="U", start_date="2023-01-03", end_date="2023-01-07", seed=9999)
    # Regime labels should almost certainly differ for extreme seeds
    assert not np.array_equal(l1, l2)


def test_generate_regime_aware_prices_positive() -> None:
    """All OHLC values must be positive."""
    _, df, _, _ = generate_regime_aware(
        symbol="TEST",
        start_date="2023-01-03",
        end_date="2023-01-31",
        seed=42,
    )
    for col in ["open", "high", "low", "close"]:
        assert (df[col] > 0).all(), f"Column {col} has non-positive values"


def test_generate_regime_aware_fractional_spread() -> None:
    """High must be >= low for every candle."""
    _, df, _, _ = generate_regime_aware(
        symbol="TEST",
        start_date="2023-01-03",
        end_date="2023-01-31",
        seed=42,
    )
    assert (df["high"] >= df["low"]).all()
