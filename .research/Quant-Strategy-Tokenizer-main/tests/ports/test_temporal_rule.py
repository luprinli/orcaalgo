from __future__ import annotations

import pytest

from qst.ports import (
    PortTemporalSpec,
    TemporalRuleResolutionError,
    resolve_temporal_rule,
    temporal_is_later,
)


def test_constant_temporal_rule_resolves_fixed_output() -> None:
    resolved = resolve_temporal_rule(
        {"kind": "constant", "value": {"available_at": "next_bar_open", "latency_bars": 1}},
        inputs={},
        params={},
    )

    assert resolved.available_at == "next_bar_open"
    assert resolved.latency_bars == 1


def test_inherit_from_input_rule_resolves_upstream_temporal() -> None:
    upstream = PortTemporalSpec(min_history_bars=7)

    resolved = resolve_temporal_rule(
        {"kind": "inherit_from_input", "input": "series"},
        inputs={"series": upstream},
        params={},
    )

    assert resolved == upstream


def test_window_min_history_uses_param() -> None:
    resolved = resolve_temporal_rule(
        {"kind": "window_min_history", "param": "lookback"},
        inputs={},
        params={"lookback": 20},
    )

    assert resolved.min_history_bars == 20
    assert not resolved.unsafe_future


def test_param_plus_constant_rule_uses_param_and_offset() -> None:
    resolved = resolve_temporal_rule(
        {
            "kind": "param_plus_constant",
            "param": "lookback",
            "field": "min_history_bars",
            "constant": 1,
        },
        inputs={},
        params={"lookback": 9},
    )

    assert resolved.min_history_bars == 10


def test_param_max_floor_rule_applies_floor() -> None:
    resolved = resolve_temporal_rule(
        {
            "kind": "param_max_floor",
            "param": "lookback",
            "field": "min_history_bars",
            "floor": 5,
        },
        inputs={},
        params={"lookback": 3},
    )

    assert resolved.min_history_bars == 5


def test_param_max_floor_accepts_signed_param_before_floor() -> None:
    resolved = resolve_temporal_rule(
        {
            "kind": "param_max_floor",
            "param": "lag",
            "field": "latency_bars",
            "floor": 0,
        },
        inputs={},
        params={"lag": -1},
    )

    assert resolved.latency_bars == 0


def test_max_inputs_joins_input_temporal() -> None:
    resolved = resolve_temporal_rule(
        {"kind": "max_inputs", "inputs": ["a", "b"]},
        inputs={
            "a": PortTemporalSpec(available_at="bar_close", min_history_bars=3),
            "b": PortTemporalSpec(available_at="next_bar_open", min_history_bars=10),
        },
        params={},
    )

    assert resolved.available_at == "next_bar_open"
    assert resolved.min_history_bars == 10


def test_centered_window_marks_unsafe_future() -> None:
    resolved = resolve_temporal_rule(
        {"kind": "centered_window_unsafe", "param": "window"},
        inputs={},
        params={"window": 5},
    )

    assert resolved.min_history_bars == 5
    assert resolved.unsafe_future


def test_unresolved_rule_fails_deterministically() -> None:
    with pytest.raises(TemporalRuleResolutionError, match="Unknown temporal param"):
        resolve_temporal_rule(
            {"kind": "window_min_history", "param": "missing"},
            inputs={},
            params={},
        )


def test_event_time_does_not_satisfy_bar_clock_requirements() -> None:
    event_time = PortTemporalSpec(available_at="event_time")

    assert temporal_is_later(event_time, "bar_close")
    assert temporal_is_later(PortTemporalSpec(available_at="bar_close"), "event_time")
    assert not temporal_is_later(event_time, "event_time")
    assert not temporal_is_later(event_time, "unknown")
