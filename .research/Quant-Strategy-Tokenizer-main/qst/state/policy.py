"""State policy models for Token System v2 WP6a."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes

STATE_POLICY_SCHEMA_VERSION: Literal["qst-state-policy/0.4"] = "qst-state-policy/0.4"

WarmupPolicy = Literal["emit_null", "emit_initial", "error"]
ResetPolicy = Literal["never", "on_event"]
MissingEventPolicy = Literal["error", "skip", "emit_null"]


class StatePolicy(BaseModel):
    """Deterministic policy shell for basic state tokens."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-state-policy/0.4"] = STATE_POLICY_SCHEMA_VERSION
    warmup_policy: WarmupPolicy = "emit_null"
    reset_policy: ResetPolicy = "never"
    missing_event_policy: MissingEventPolicy = "error"
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("metadata")
    @classmethod
    def _metadata_is_json(cls, value: dict[str, Any]) -> dict[str, Any]:
        _ensure_json(value, field_name="StatePolicy.metadata")
        return value


def default_state_policy() -> StatePolicy:
    """Return the accepted WP6a default policy."""

    return StatePolicy()


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
