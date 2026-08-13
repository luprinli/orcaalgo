"""Deterministic train/validation ticker split (cross-sectional out-of-sample).

Uses a deterministic hash-based split by *ticker*, not a random per-run shuffle,
so the same ticker never bounces between buckets across runs.
This gives a stable cross-sectional holdout that complements the time-based
walk-forward window.

Algorithm
---------
1. ``always_validation_tickers`` are forced into validation.
2. Every other ticker is hashed with SHA-256 (uppercase symbol); the first four
   bytes are read as a big-endian uint32 and scaled to ``[0, 1)``.
3. If the value is below ``training_allocation_ratio`` the ticker is training,
   otherwise validation.

The hash is stable and independent of import order, so the split is reproducible
across processes and languages that use the same byte ordering.
"""

from __future__ import annotations

import hashlib

DEFAULT_ALWAYS_VALIDATION: tuple[str, ...] = ("SPY", "QQQ")
DEFAULT_TRAINING_ALLOCATION_RATIO = 0.65


def _scaled_hash(symbol: str) -> float:
    """Return a uniform float in [0, 1) derived from the symbol's SHA-256."""
    digest = hashlib.sha256(symbol.encode("utf-8")).digest()
    value = int.from_bytes(digest[:4], byteorder="big")
    return value / 0xFFFFFFFF


def is_training_ticker(
    symbol: str,
    always_validation_tickers: tuple[str, ...] = DEFAULT_ALWAYS_VALIDATION,
    training_allocation_ratio: float = DEFAULT_TRAINING_ALLOCATION_RATIO,
) -> bool:
    """Return True if `symbol` belongs to the training bucket.

    Args:
        symbol: Ticker symbol (case-insensitive).
        always_validation_tickers: Symbols forced into validation.
        training_allocation_ratio: Fraction of tickers assigned to training.
    """
    normalized = symbol.upper()
    forced = {t.upper() for t in always_validation_tickers}
    if normalized in forced:
        return False
    return _scaled_hash(normalized) < training_allocation_ratio


def split_tickers(
    tickers: list[str],
    always_validation_tickers: tuple[str, ...] = DEFAULT_ALWAYS_VALIDATION,
    training_allocation_ratio: float = DEFAULT_TRAINING_ALLOCATION_RATIO,
) -> tuple[list[str], list[str]]:
    """Split a ticker list into (training, validation) buckets.

    Returns:
        A ``(training, validation)`` tuple of lists, preserving input order.
    """
    training: list[str] = []
    validation: list[str] = []
    for ticker in tickers:
        if is_training_ticker(ticker, always_validation_tickers, training_allocation_ratio):
            training.append(ticker)
        else:
            validation.append(ticker)
    return training, validation


__all__ = [
    "DEFAULT_ALWAYS_VALIDATION",
    "DEFAULT_TRAINING_ALLOCATION_RATIO",
    "is_training_ticker",
    "split_tickers",
]
