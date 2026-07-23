"""Temporal rule resolution for Token System v2."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any, Literal

from qst.ports.port_spec import (
    PortTemporalSpec,
    TemporalField,
    TemporalRule,
)
from qst.types import AvailableAt

_AVAILABLE_AT_ORDER = {
    "bar_open": 0,
    "bar_close": 1,
    "next_bar_open": 2,
    "unknown": 3,
}


class TemporalRuleResolutionError(ValueError):
    """Raised when a temporal rule cannot be resolved deterministically."""


def resolve_temporal_rule(
    rule: TemporalRule | Mapping[str, Any],
    *,
    inputs: Mapping[str, PortTemporalSpec],
    params: Mapping[str, Any],
) -> PortTemporalSpec:
    """Resolve a TemporalRule into a PortTemporalSpec."""

    parsed = rule if isinstance(rule, TemporalRule) else TemporalRule.model_validate(rule)
    match parsed.kind:
        case "constant":
            if parsed.value is None:
                raise TemporalRuleResolutionError("constant temporal rule requires value")
            return parsed.value
        case "inherit_from_input":
            return _get_input(inputs, parsed.input)
        case "param_value":
            value = _get_param(params, parsed.param)
            if parsed.field is None:
                if not isinstance(value, Mapping):
                    raise TemporalRuleResolutionError("param_value without field requires mapping value")
                return PortTemporalSpec.model_validate(value)
            return _with_field(PortTemporalSpec(), parsed.field, value)
        case "param_max_floor":
            field = _require_int_field(parsed.field, parsed.kind)
            value = _coerce_int(_get_param(params, parsed.param), field_name=parsed.param, allow_negative=True)
            if parsed.floor is None:
                raise TemporalRuleResolutionError("param_max_floor requires floor")
            return _with_field(PortTemporalSpec(), field, max(value, parsed.floor))
        case "param_plus_constant":
            field = _require_int_field(parsed.field, parsed.kind)
            value = _coerce_int(_get_param(params, parsed.param), field_name=parsed.param)
            if parsed.constant is None:
                raise TemporalRuleResolutionError("param_plus_constant requires constant")
            return _with_field(PortTemporalSpec(), field, value + parsed.constant)
        case "max_inputs":
            names = parsed.inputs or sorted(inputs)
            if not names:
                raise TemporalRuleResolutionError("max_inputs requires at least one input")
            return _join_temporal([_get_input(inputs, name) for name in names])
        case "param_predicate":
            value = _get_param(params, parsed.param)
            matches = bool(value) if parsed.equals is None else value == parsed.equals
            result = parsed.when_true if matches else parsed.when_false
            if result is None:
                raise TemporalRuleResolutionError("param_predicate requires matching result temporal")
            return result
        case "window_min_history":
            value = _coerce_int(_get_param(params, parsed.param), field_name=parsed.param)
            return PortTemporalSpec(min_history_bars=value)
        case "centered_window_unsafe":
            value = _coerce_int(_get_param(params, parsed.param), field_name=parsed.param)
            return PortTemporalSpec(min_history_bars=value, unsafe_future=True)


def temporal_is_later(left: PortTemporalSpec, max_available_at: str) -> bool:
    """Return true when ``left`` is later than the allowed availability."""

    if left.available_at == "event_time":
        return max_available_at not in {"event_time", "unknown"}
    if max_available_at == "event_time":
        return True
    left_order = _AVAILABLE_AT_ORDER[left.available_at]
    max_order = _AVAILABLE_AT_ORDER[max_available_at]
    return left_order > max_order


def _get_input(inputs: Mapping[str, PortTemporalSpec], name: str | None) -> PortTemporalSpec:
    if name is None:
        raise TemporalRuleResolutionError("temporal rule requires input")
    try:
        return inputs[name]
    except KeyError as exc:
        raise TemporalRuleResolutionError(f"Unknown temporal input: {name}") from exc


def _get_param(params: Mapping[str, Any], name: str | None) -> Any:
    if name is None:
        raise TemporalRuleResolutionError("temporal rule requires param")
    try:
        return params[name]
    except KeyError as exc:
        raise TemporalRuleResolutionError(f"Unknown temporal param: {name}") from exc


def _coerce_int(value: Any, *, field_name: str | None, allow_negative: bool = False) -> int:
    if isinstance(value, bool) or not isinstance(value, int):
        raise TemporalRuleResolutionError(f"Temporal param {field_name!r} must be an integer")
    if value < 0 and not allow_negative:
        raise TemporalRuleResolutionError(f"Temporal param {field_name!r} must be non-negative")
    return value


LiteralIntTemporalField = Literal["latency_bars", "min_history_bars"]


def _require_int_field(field: TemporalField | None, kind: str) -> LiteralIntTemporalField:
    if field == "latency_bars" or field == "min_history_bars":
        return field
    else:
        raise TemporalRuleResolutionError(f"{kind} requires latency_bars or min_history_bars field")


def _with_field(base: PortTemporalSpec, field: TemporalField, value: Any) -> PortTemporalSpec:
    payload = base.model_dump(mode="json")
    payload[field] = value
    return PortTemporalSpec.model_validate(payload)


def _join_temporal(values: list[PortTemporalSpec]) -> PortTemporalSpec:
    available = _max_available_at([value.available_at for value in values])
    return PortTemporalSpec(
        available_at=available,
        latency_bars=max(value.latency_bars for value in values),
        min_history_bars=max(value.min_history_bars for value in values),
        unsafe_future=any(value.unsafe_future for value in values),
    )


def _max_available_at(values: list[AvailableAt]) -> AvailableAt:
    if any(value == "event_time" for value in values):
        raise TemporalRuleResolutionError("event_time cannot be ordered in temporal max_inputs")
    return max(values, key=lambda value: _AVAILABLE_AT_ORDER[value])
