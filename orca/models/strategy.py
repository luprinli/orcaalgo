from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class TokenRef(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    token_id: str = Field(min_length=1)
    version: str = ">=1.0"


class TypeSpec(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    kind: Literal["Scalar", "TimeSeries", "Panel", "Decision", "State", "Event", "EventStream"]
    value_type: Literal["float", "int", "bool", "string", "decimal", "DecisionKind"] = "float"
    intrinsic_temporal: bool = False


class InputSpec(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    type: TypeSpec = Field(default_factory=TypeSpec)
    required: bool = True
    default: Any = None


class PortTemporalSpec(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    available_at: Literal["bar_open", "bar_close", "next_bar_open", "unknown"] = "bar_close"
    latency_bars: int = Field(default=0, ge=0)
    min_history_bars: int = Field(default=0, ge=0)
    unsafe_future: bool = False


class TemporalRule(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    kind: Literal[
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
    value: Any = None


class OutputSpec(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    type: TypeSpec = Field(default_factory=TypeSpec)
    port_temporal: PortTemporalSpec | None = None
    temporal_rule: TemporalRule | None = None


class PortSignature(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    inputs: dict[str, InputSpec] = Field(default_factory=dict)
    outputs: dict[str, OutputSpec] = Field(default_factory=dict)
    external_read: bool = False
    external_write: bool = False


class Node(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    id: str = Field(min_length=1)
    token_ref: TokenRef
    inputs: dict[str, str] = Field(default_factory=dict)
    outputs: dict[str, str] = Field(default_factory=dict)
    params: dict[str, Any] = Field(default_factory=dict)
    signature: PortSignature | None = None


class StrategyBody(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    id: str = Field(min_length=1)
    version: str = Field(min_length=1)
    nodes: list[Node] = Field(min_length=1)
    outputs: dict[str, str] = Field(default_factory=dict)


class Capability(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    name: str

    @model_validator(mode="before")
    @classmethod
    def _coerce_string(cls, data: object) -> dict[str, str]:
        if isinstance(data, str):
            return {"name": data}
        return data  # type: ignore[return-value]


class StrategyIRV04(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    ir_version: Literal["qst-ir/0.4"] = "qst-ir/0.4"
    canonical_version: Literal["qst-canonical/0.4"] = "qst-canonical/0.4"
    strategy: StrategyBody
    capabilities: list[Capability] = Field(default_factory=list)

    risk_profile: RiskProfile | None = Field(default=None)


class RiskProfile(BaseModel):
    """Per-strategy risk sizing profile for dynamic position sizing.

    Translated into BacktestConfig.StrategyParams by the GKR compiler.
    """
    model_config = ConfigDict(frozen=True, extra="forbid")

    risk_per_trade_pct: float = Field(default=0.02, ge=0.001, le=0.10)
    kelly_multiplier: float = Field(default=0.25, ge=0.05, le=1.0)
    regime_multipliers: tuple[float, float, float, float] = (1.0, 0.85, 0.75, 0.0)

    @field_validator("regime_multipliers")
    @classmethod
    def validate_regime_multipliers(cls, v):
        if any(m < 0.0 or m > 2.0 for m in v):
            raise ValueError("regime_multipliers must be in range [0.0, 2.0]")
        return v

    vix_scaling_enabled: bool = Field(default=False)


StrategyIRV04.model_rebuild()
