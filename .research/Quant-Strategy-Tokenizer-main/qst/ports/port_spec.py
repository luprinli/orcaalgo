"""Port contracts for Token System v2."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.types import AvailableAt, TypeSpec, parse_type_spec

PORT_SPEC_SCHEMA_VERSION: Literal["qst-portspec/0.4"] = "qst-portspec/0.4"
PORT_TEMPORAL_SCHEMA_VERSION: Literal["qst-port-temporal/0.4"] = "qst-port-temporal/0.4"
TemporalRuleKind = Literal[
    "constant",
    "inherit_from_input",
    "param_value",
    "param_max_floor",
    "param_plus_constant",
    "max_inputs",
    "param_predicate",
    "window_min_history",
    "centered_window_unsafe",
]
TemporalField = Literal["available_at", "latency_bars", "min_history_bars", "unsafe_future"]


class TemporalRequirement(BaseModel):
    """Input-side requirement for upstream data availability."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-port-temporal/0.4"] = PORT_TEMPORAL_SCHEMA_VERSION
    max_available_at: AvailableAt = "bar_close"
    allow_unsafe_future: bool = False


class PortTemporalSpec(BaseModel):
    """Output-side temporal promise for a port.

    The port layer stores temporal contracts; temporal rule resolution and validation are separate passes.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-port-temporal/0.4"] = PORT_TEMPORAL_SCHEMA_VERSION
    available_at: AvailableAt = "bar_close"
    latency_bars: int = Field(default=0, ge=0)
    min_history_bars: int = Field(default=0, ge=0)
    unsafe_future: bool = False


class TemporalRule(BaseModel):
    """Declarative temporal rule shell."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-port-temporal/0.4"] = PORT_TEMPORAL_SCHEMA_VERSION
    kind: TemporalRuleKind
    value: PortTemporalSpec | None = None
    input: str | None = None
    inputs: list[str] | None = None
    param: str | None = None
    field: TemporalField | None = None
    floor: int | None = None
    constant: int | None = None
    equals: Any | None = None
    when_true: PortTemporalSpec | None = None
    when_false: PortTemporalSpec | None = None


class InputSpec(BaseModel):
    """Input port contract."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-portspec/0.4"] = PORT_SPEC_SCHEMA_VERSION
    type: TypeSpec
    temporal_requirement: TemporalRequirement | None = None

    @field_validator("type", mode="before")
    @classmethod
    def _parse_type(cls, value: Any) -> TypeSpec:
        return parse_type_spec(value)


class OutputSpec(BaseModel):
    """Output port contract."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-portspec/0.4"] = PORT_SPEC_SCHEMA_VERSION
    type: TypeSpec
    port_temporal: PortTemporalSpec | None = None
    temporal_rule: TemporalRule | None = None

    @field_validator("type", mode="before")
    @classmethod
    def _parse_type(cls, value: Any) -> TypeSpec:
        return parse_type_spec(value)


class PortSignature(BaseModel):
    """Token node input/output signature."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-portspec/0.4"] = PORT_SPEC_SCHEMA_VERSION
    inputs: dict[str, InputSpec] = Field(default_factory=dict)
    outputs: dict[str, OutputSpec] = Field(default_factory=dict)
