"""Current QST strategy IR models."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.canonical_json import stable_json_bytes
from qst.ports import PortSignature

IR_VERSION_V04: Literal["qst-ir/0.4"] = "qst-ir/0.4"
CANONICAL_VERSION_V04: Literal["qst-canonical/0.4"] = "qst-canonical/0.4"
IR_SCHEMA_VERSION_V04: Literal["qst-ir-schema/0.4"] = "qst-ir-schema/0.4"
CapabilityV04 = Literal[
    "core",
    "panel_type",
    "panel_ops",
    "panel_weights",
    "panel_recipes",
    "custom_token_runtime",
]
def _default_capabilities() -> list[CapabilityV04]:
    return ["core"]


def _ensure_canonical_json(value: Any, *, field_name: str) -> Any:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
    return value


class TokenRefV04(BaseModel):
    """Canonical token reference for qst-ir/0.4 nodes."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    namespace: str = Field(min_length=1)
    name: str = Field(min_length=1)
    version: int = Field(ge=1)
    behavior_version: int = Field(ge=1)


class NodeV04(BaseModel):
    """Current QST graph node."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    id: str
    token: str | None = None
    version: int | None = Field(default=None, ge=1)
    token_ref: TokenRefV04 | None = None
    inputs: dict[str, Any] = Field(default_factory=dict)
    params: dict[str, Any] = Field(default_factory=dict)
    signature: PortSignature = Field(default_factory=PortSignature)
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("inputs", "params", "metadata")
    @classmethod
    def _validate_json_fields(cls, value: dict[str, Any]) -> dict[str, Any]:
        _ensure_canonical_json(value, field_name="node JSON field")
        return value


class StrategyBodyV04(BaseModel):
    """Strategy content envelope."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    id: str
    version: int = Field(default=1, ge=1)
    nodes: list[NodeV04] = Field(default_factory=list)
    outputs: dict[str, str] = Field(default_factory=dict)

    @field_validator("outputs")
    @classmethod
    def _validate_outputs(cls, value: dict[str, str]) -> dict[str, str]:
        _ensure_canonical_json(value, field_name="strategy outputs")
        return value

    @model_validator(mode="after")
    def _unique_node_ids(self) -> StrategyBodyV04:
        node_ids = [node.id for node in self.nodes]
        if len(node_ids) != len(set(node_ids)):
            raise ValueError("qst-ir/0.4 strategy node ids must be unique")
        return self


class StrategyIRV04(BaseModel):
    """Top-level current QST strategy document."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-ir-schema/0.4"] = IR_SCHEMA_VERSION_V04
    ir_version: Literal["qst-ir/0.4"] = IR_VERSION_V04
    canonical_version: Literal["qst-canonical/0.4"] = CANONICAL_VERSION_V04
    capabilities: list[CapabilityV04] = Field(default_factory=_default_capabilities)
    strategy: StrategyBodyV04
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("metadata")
    @classmethod
    def _validate_metadata(cls, value: dict[str, Any]) -> dict[str, Any]:
        _ensure_canonical_json(value, field_name="metadata")
        return value

    @field_validator("capabilities")
    @classmethod
    def _validate_capabilities(cls, value: list[CapabilityV04]) -> list[CapabilityV04]:
        if len(value) != len(set(value)):
            raise ValueError("qst-ir/0.4 capabilities must be unique")
        if "core" not in value:
            raise ValueError('qst-ir/0.4 capabilities must include "core"')
        return sorted(value)
