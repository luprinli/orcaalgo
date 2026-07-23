"""Base frame helpers for frame models."""

from __future__ import annotations

import hashlib
import re
from datetime import UTC, datetime
from typing import Annotated

from pydantic import AfterValidator, BaseModel, ConfigDict

from qst.canonical_json import stable_json_bytes
from qst.hash.common import HashString

_SYMBOL_PATTERN = re.compile(r"^[A-Z0-9]+/[A-Z0-9]+$")


def validate_symbol(value: str) -> str:
    if not isinstance(value, str):
        raise TypeError(f"QSTSymbol must be str, got {type(value).__name__}")
    if not _SYMBOL_PATTERN.match(value):
        raise ValueError(f"Invalid QSTSymbol {value!r}; expected BASE/QUOTE uppercase ASCII")
    return value


QSTSymbol = Annotated[str, AfterValidator(validate_symbol)]


def normalize_timestamp(value: datetime) -> datetime:
    if value.tzinfo is None or value.utcoffset() is None:
        raise ValueError("Frame timestamps must be timezone-aware")
    return value.astimezone(UTC)


def timestamp_to_string(value: datetime) -> str:
    normalized = normalize_timestamp(value)
    return normalized.isoformat().replace("+00:00", "Z")


class FrameBase(BaseModel):
    """Common fields shared by QST frame records."""

    model_config = ConfigDict(extra="forbid")

    frame_version: str
    frame_hash: HashString | None = None


def compute_frame_hash(frame: FrameBase) -> str:
    payload = frame.model_dump(mode="json")
    payload.pop("frame_hash", None)
    return f"sha256:{hashlib.sha256(stable_json_bytes(payload)).hexdigest()}"
