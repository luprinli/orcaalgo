"""Small result container used by agent-facing APIs."""

from __future__ import annotations

from typing import Generic, TypeVar

from pydantic import BaseModel, ConfigDict

T = TypeVar("T")


class Failure(BaseModel):
    """Structured failure with a stable kind and human-readable message."""

    kind: str
    message: str
    details: dict[str, object] = {}


class Result(BaseModel, Generic[T]):
    """Generic ok/value/error result."""

    model_config = ConfigDict(arbitrary_types_allowed=True)

    ok: bool
    value: T | None = None
    error: Failure | None = None

    @classmethod
    def success(cls, value: T) -> Result[T]:
        return cls(ok=True, value=value)

    @classmethod
    def failure(cls, kind: str, message: str, details: dict[str, object] | None = None) -> Result[T]:
        return cls(ok=False, error=Failure(kind=kind, message=message, details=details or {}))
