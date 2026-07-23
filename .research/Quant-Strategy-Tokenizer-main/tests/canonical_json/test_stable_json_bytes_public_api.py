from __future__ import annotations

import math

import pytest

from qst.canonical_json import MAX_CANONICAL_DEPTH, stable_json_bytes


def test_stable_json_bytes_are_order_stable() -> None:
    left = {"b": [2, {"d": None, "c": True}], "a": "x"}
    right = {"a": "x", "b": [2, {"c": True, "d": None}]}

    assert stable_json_bytes(left) == stable_json_bytes(right)
    assert stable_json_bytes(left) == b'{"a":"x","b":[2,{"c":true,"d":null}]}'


@pytest.mark.parametrize(
    "value",
    [
        {"x": math.nan},
        {"x": math.inf},
        {"x": b"bytes"},
        {"x": (1, 2)},
        {1: "non-string-key"},
        {"x": object()},
    ],
)
def test_stable_json_bytes_reject_invalid_json_values(value: object) -> None:
    with pytest.raises((TypeError, ValueError)):
        stable_json_bytes(value)


def test_stable_json_bytes_reject_depth_over_limit() -> None:
    value: object = "leaf"
    for _ in range(MAX_CANONICAL_DEPTH + 2):
        value = {"x": value}

    with pytest.raises(ValueError, match="depth"):
        stable_json_bytes(value)
