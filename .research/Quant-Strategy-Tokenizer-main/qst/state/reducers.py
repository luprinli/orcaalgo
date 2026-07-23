"""Reducer registry for Token System v2 WP6a state accumulation."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any, TypeAlias

from qst.validation import Diagnostic, ValidationResult

Reducer: TypeAlias = Callable[[Any, Any], Any]


class ReducerRegistry:
    """Deterministic registry for state.accumulate reducers."""

    def __init__(self) -> None:
        self._reducers: dict[str, Reducer] = {}

    def register(self, name: str, reducer: Reducer) -> None:
        """Register a reducer under a unique name."""

        if name in self._reducers:
            raise ValueError(f"Duplicate state reducer: {name}")
        self._reducers[name] = reducer

    def names(self) -> list[str]:
        """Return registered reducer names in deterministic order."""

        return sorted(self._reducers)

    def get(self, name: str) -> Reducer:
        """Resolve a reducer by name."""

        return self._reducers[name]

    def validate_name(self, name: str) -> ValidationResult:
        """Return a diagnostic when a reducer is not registered."""

        if name in self._reducers:
            return ValidationResult()
        return ValidationResult(
            diagnostics=[
                Diagnostic(
                    code="QST_V2_STATE_REDUCER_UNKNOWN",
                    severity="error",
                    phase="runtime",
                    message=f"State reducer {name!r} is not registered.",
                    remediation="Use a reducer from ReducerRegistry.names().",
                )
            ]
        )


def default_reducer_registry() -> ReducerRegistry:
    """Return the WP6a built-in reducer registry."""

    registry = ReducerRegistry()
    registry.register("count", _count)
    registry.register("last", _last)
    registry.register("max", _max)
    registry.register("min", _min)
    registry.register("sum", _sum)
    return registry


def _count(state: Any, _value: Any) -> int:
    if state is None:
        return 1
    if not isinstance(state, int):
        raise TypeError("count reducer state must be int or None")
    return state + 1


def _last(_state: Any, value: Any) -> Any:
    return value


def _max(state: Any, value: Any) -> Any:
    if state is None:
        return value
    return max(state, value)


def _min(state: Any, value: Any) -> Any:
    if state is None:
        return value
    return min(state, value)


def _sum(state: Any, value: Any) -> Any:
    if state is None:
        return value
    return state + value
