"""Structured TypeSpec models for Token System v2."""

from __future__ import annotations

import re
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, model_validator

from qst.types.value_type import ValueType

TypeKind = Literal[
    "Scalar",
    "TimeSeries",
    "Panel",
    "Decision",
    "Plan",
    "State",
    "Event",
    "EventStream",
]
TypedKind = Literal["Scalar", "TimeSeries", "Panel", "State", "Event", "EventStream"]
AvailableAt = Literal["bar_open", "bar_close", "next_bar_open", "event_time", "unknown"]
Clock = Literal["bar", "event", "none", "unknown"]
MissingPolicy = Literal["unknown", "dense", "sparse"]
SelectionKind = Literal["none", "static", "dynamic", "weighted"]
TYPE_SPEC_SCHEMA_VERSION: Literal["qst-typespec/0.4"] = "qst-typespec/0.4"

_SHORTHAND_RE = re.compile(
    r"^(Scalar|TimeSeries|Panel|State|Event|EventStream)\[([A-Za-z_][A-Za-z0-9_]*)\]$"
)
_BARE_KINDS: set[str] = {"Decision", "Plan"}
_TYPED_KINDS: set[str] = {"Scalar", "TimeSeries", "Panel", "State", "Event", "EventStream"}


class IntrinsicTemporalSpec(BaseModel):
    """Default temporal nature of a type or data source."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-typespec/0.4"] = TYPE_SPEC_SCHEMA_VERSION
    default_available_at: AvailableAt = "bar_close"
    default_clock: Clock = "bar"


class TypeSpec(BaseModel):
    """Structured v2 type specification."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-typespec/0.4"] = TYPE_SPEC_SCHEMA_VERSION
    kind: TypeKind
    value_type: ValueType | None = None
    intrinsic_temporal: IntrinsicTemporalSpec | None = None
    axes: list[str] | None = None
    universe: dict[str, Any] | None = None
    missing_policy: MissingPolicy | None = None
    group_spec_ref: str | None = None
    selection_kind: SelectionKind | None = None
    weight_constraints: dict[str, Any] | None = None
    panel_capability_required: bool | None = None

    @model_validator(mode="before")
    @classmethod
    def _parse_shorthand(cls, value: Any) -> Any:
        if isinstance(value, str):
            if value in _BARE_KINDS:
                return {"kind": value}
            match = _SHORTHAND_RE.fullmatch(value)
            if match is None:
                raise ValueError(f"Unsupported TypeSpec shorthand: {value!r}")
            kind, value_type = match.groups()
            value = {"kind": kind, "value_type": value_type}
        if isinstance(value, dict) and value.get("kind") == "Panel":
            return {
                "axes": [],
                "universe": {"kind": "unspecified"},
                "missing_policy": "unknown",
                "group_spec_ref": "",
                "selection_kind": "none",
                "weight_constraints": {},
                "panel_capability_required": True,
                **value,
            }
        return value

    @model_validator(mode="after")
    def _check_value_type(self) -> TypeSpec:
        if self.kind in _TYPED_KINDS and self.value_type is None:
            raise ValueError(f"{self.kind} TypeSpec requires value_type")
        if self.kind in _BARE_KINDS and self.value_type is not None:
            raise ValueError(f"{self.kind} TypeSpec does not accept value_type")
        panel_fields = (
            self.axes,
            self.universe,
            self.missing_policy,
            self.group_spec_ref,
            self.selection_kind,
            self.weight_constraints,
            self.panel_capability_required,
        )
        if self.kind != "Panel" and any(field is not None for field in panel_fields):
            raise ValueError("Panel shell fields are only valid for Panel TypeSpec")
        if self.kind == "Panel":
            _ensure_canonical_json(self.universe, field_name="panel universe")
            _ensure_canonical_json(self.weight_constraints, field_name="panel weight_constraints")
        return self


def parse_type_spec(value: TypeSpec | dict[str, Any] | str) -> TypeSpec:
    """Parse a TypeSpec from structured input or shorthand."""

    if isinstance(value, TypeSpec):
        return value
    return TypeSpec.model_validate(value)


def _ensure_canonical_json(value: Any, *, field_name: str) -> None:
    from qst.canonical_json import stable_json_bytes

    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
