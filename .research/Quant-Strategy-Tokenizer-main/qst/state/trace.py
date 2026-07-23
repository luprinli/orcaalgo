"""State trace models for Token System v2 WP6a."""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes


class StateTraceEvent(BaseModel):
    """One deterministic state-transition trace event."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    index: int = Field(ge=0)
    input: Any
    output: Any
    state_before: Any
    state_after: Any
    reset: bool = False
    policy_decision: str = Field(min_length=1)

    @field_validator("input", "output", "state_before", "state_after")
    @classmethod
    def _payload_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="state trace payload")
        return value


class StateExecutionTrace(BaseModel):
    """Trace for one state helper invocation."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    token_id: str = Field(min_length=1)
    events: list[StateTraceEvent] = Field(default_factory=list)


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
