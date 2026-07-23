"""Deterministic custom-token-reference custom token entrypoint."""

from __future__ import annotations

from typing import Any


def run(inputs: dict[str, Any]) -> dict[str, list[float]]:
    """Return a simple deterministic Kalman-like exponential filter."""

    series = inputs.get("series")
    if not isinstance(series, list):
        raise TypeError("series input must be a list")
    alpha = inputs.get("alpha", 0.5)
    if not isinstance(alpha, int | float):
        raise TypeError("alpha input must be numeric")
    filtered: list[float] = []
    previous: float | None = None
    for value in series:
        if not isinstance(value, int | float):
            raise TypeError("series values must be numeric")
        current = float(value) if previous is None else previous + float(alpha) * (float(value) - previous)
        filtered.append(round(current, 6))
        previous = current
    return {"filtered": filtered}
