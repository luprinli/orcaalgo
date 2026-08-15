"""Self-documenting feature/indicator metadata registry (R9).

A documentation layer for the indicators Orca computes (Go ``cinar/indicator`` +
Python features). It maps a canonical name to its category, formula, parameters,
and the engine that computes it, so ``orca attribute`` / calibration audits can
render human-readable feature provenance without re-deriving it from code.

This is metadata only — it never computes a value (HP #1: no reimplemented math).
"""

from __future__ import annotations

from typing import Any

FEATURE_REGISTRY: dict[str, dict[str, Any]] = {
    "sma": {
        "name": "Simple Moving Average",
        "category": "trend",
        "formula": "mean(close, window)",
        "params": ["window"],
        "engine": "go:cinar/indicator",
    },
    "ema": {
        "name": "Exponential Moving Average",
        "category": "trend",
        "formula": "alpha*close + (1-alpha)*ema_prev, alpha=2/(span+1)",
        "params": ["span"],
        "engine": "go:cinar/indicator",
    },
    "rsi": {
        "name": "Relative Strength Index",
        "category": "momentum",
        "formula": "100 - 100/(1 + avg_gain/avg_loss)",
        "params": ["window"],
        "engine": "go:cinar/indicator",
    },
    "atr": {
        "name": "Average True Range",
        "category": "volatility",
        "formula": "mean(max(h-l, |h-c_prev|, |l-c_prev|), window)",
        "params": ["window"],
        "engine": "go:cinar/indicator",
    },
    "macd": {
        "name": "Moving Average Convergence Divergence",
        "category": "momentum",
        "formula": "ema(close, fast) - ema(close, slow)",
        "params": ["fast", "slow", "signal"],
        "engine": "go:cinar/indicator",
    },
    "bollinger": {
        "name": "Bollinger Bands",
        "category": "volatility",
        "formula": "sma(close, window) +/- k*std(close, window)",
        "params": ["window", "k"],
        "engine": "go:cinar/indicator",
    },
    "vwap": {
        "name": "Volume-Weighted Average Price",
        "category": "volume",
        "formula": "sum(price*volume)/sum(volume)",
        "params": ["session"],
        "engine": "go:internal",
    },
    "ret_log": {
        "name": "Log Return",
        "category": "return",
        "formula": "log(close_t / close_{t-1})",
        "params": ["horizon"],
        "engine": "python:orca/ml/features.py",
    },
    "volatility_ewma": {
        "name": "EWMA Volatility",
        "category": "volatility",
        "formula": "sqrt(ewma_var), alpha=2/(span+1)",
        "params": ["span"],
        "engine": "python:orca/math/ewma.py",
    },
}


def get_feature_metadata(name: str) -> dict[str, Any] | None:
    """Return metadata for a canonical feature name, or None if unknown."""
    return FEATURE_REGISTRY.get(name.lower())


def list_features(category: str | None = None) -> list[dict[str, Any]]:
    """List feature metadata, optionally filtered by category."""
    out = []
    for key, meta in FEATURE_REGISTRY.items():
        if category and meta.get("category") != category:
            continue
        out.append({"key": key, **meta})
    return out


__all__ = ["FEATURE_REGISTRY", "get_feature_metadata", "list_features"]
