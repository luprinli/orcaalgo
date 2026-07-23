"""Common helpers for Token System v2 hashes."""

from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass
from typing import Annotated, Any

from pydantic import Field

from qst.canonical_json import stable_json_bytes

HASH_V2_PREFIX = "sha256:"
HASH_V2_PATTERN_TEXT = r"^sha256:[0-9a-f]{64}$"
HASH_V2_PATTERN = re.compile(HASH_V2_PATTERN_TEXT)
HashString = Annotated[str, Field(pattern=HASH_V2_PATTERN_TEXT)]


def sha256_bytes(payload: bytes) -> str:
    """Return a plain SHA-256 hex digest for artifact/file integrity checks."""

    return hashlib.sha256(payload).hexdigest()


def hash_v2_payload(kind: str, payload: Any) -> str:
    """Hash a canonical JSON-compatible v2 payload."""

    material = {
        "hash_namespace": "qst-hash-v2/0.1",
        "kind": kind,
        "payload": payload,
    }
    digest = hashlib.sha256(stable_json_bytes(material)).hexdigest()
    return f"{HASH_V2_PREFIX}{digest}"


@dataclass(frozen=True)
class IRHashesV2:
    """Three-layer qst-ir/0.4 strategy hash snapshot."""

    graph_hash: str
    param_hash: str
    instance_hash: str
