"""TraceLog frame model."""

from __future__ import annotations

import json
from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.canonical_json import stable_json_bytes
from qst.frames.base import FrameBase, normalize_timestamp


def canonicalize_payload(value: Any) -> Any:
    """Validate and normalize a payload to canonical JSON-compatible values."""

    try:
        payload = stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(str(exc)) from exc
    return json.loads(payload.decode("utf-8"))


class TraceEvent(BaseModel):
    model_config = ConfigDict(extra="forbid")

    timestamp: datetime
    node_id: str
    event: str
    payload: Any = Field(default_factory=dict)

    @field_validator("timestamp")
    @classmethod
    def _normalize_timestamp(cls, value: datetime) -> datetime:
        return normalize_timestamp(value)

    @field_validator("payload")
    @classmethod
    def _validate_payload(cls, value: Any) -> Any:
        return canonicalize_payload(value)


class TraceLog(FrameBase):
    frame_version: Literal["qst-trace-log/1"] = "qst-trace-log/1"
    events: list[TraceEvent] = Field(default_factory=list)

    @model_validator(mode="after")
    def _canonicalize(self) -> TraceLog:
        self.events = sorted(
            self.events,
            key=lambda item: (
                item.timestamp,
                item.node_id,
                item.event,
                stable_json_bytes(item.payload),
            ),
        )
        return self
