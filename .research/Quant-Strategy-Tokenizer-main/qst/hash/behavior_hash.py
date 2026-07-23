"""Behavior hash framework for Token System v2."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes
from qst.hash.common import hash_v2_payload
from qst.numeric import NumericPolicy
from qst.token_evolution import TokenLifecycleStatus

BEHAVIOR_MATERIAL_SCHEMA_VERSION: Literal["qst-behavior-material/0.4"] = (
    "qst-behavior-material/0.4"
)


class BehaviorMaterialV2(BaseModel):
    """Hash material for v0.4 behavior hashes before TokenSpec v2 lands."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-behavior-material/0.4"] = BEHAVIOR_MATERIAL_SCHEMA_VERSION
    behavior_version: int = Field(ge=1)
    numeric_policy: NumericPolicy
    lifecycle: TokenLifecycleStatus = Field(default_factory=TokenLifecycleStatus)
    token_ref: dict[str, Any] | None = None
    contracts: list[dict[str, Any]] = Field(default_factory=list)
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("token_ref")
    @classmethod
    def _validate_token_ref(cls, value: dict[str, Any] | None) -> dict[str, Any] | None:
        if value is None:
            return value
        _ensure_json(value, field_name="token_ref")
        return value

    @field_validator("contracts")
    @classmethod
    def _validate_contracts(cls, value: list[dict[str, Any]]) -> list[dict[str, Any]]:
        _ensure_json(value, field_name="contracts")
        return value

    @field_validator("metadata")
    @classmethod
    def _validate_metadata(cls, value: dict[str, Any]) -> dict[str, Any]:
        _ensure_json(value, field_name="metadata")
        return value


def behavior_hash_v2(payload: Any | None = None) -> str:
    """Hash behavior material or behavior contract summaries."""

    return hash_v2_payload("behavior", {} if payload is None else payload)


def behavior_hash_for_material_v2(material: BehaviorMaterialV2 | dict[str, Any]) -> str:
    """Hash typed Wbehavior material with required numeric policy and lifecycle."""

    if not isinstance(material, BehaviorMaterialV2):
        material = BehaviorMaterialV2.model_validate(material)
    return behavior_hash_v2(material.model_dump(mode="json"))


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
