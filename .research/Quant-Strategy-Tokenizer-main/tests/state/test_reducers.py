from __future__ import annotations

import pytest

from qst.state import ReducerRegistry, default_reducer_registry


def test_default_reducer_registry_order_is_deterministic() -> None:
    registry = default_reducer_registry()

    assert registry.names() == ["count", "last", "max", "min", "sum"]


def test_unknown_reducer_returns_diagnostic() -> None:
    result = default_reducer_registry().validate_name("median")

    assert not result.ok
    assert [diagnostic.code for diagnostic in result.diagnostics] == [
        "QST_V2_STATE_REDUCER_UNKNOWN"
    ]


def test_reducer_registry_rejects_duplicate_names() -> None:
    registry = ReducerRegistry()
    registry.register("x", lambda state, value: value)

    with pytest.raises(ValueError):
        registry.register("x", lambda state, value: state)
