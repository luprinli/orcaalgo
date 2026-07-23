from __future__ import annotations

import json
from hashlib import sha256
from typing import Any

MAX_CANONICAL_DEPTH = 12
HASH_NAMESPACE = "orca-algo-v1/0.1"


def _canonicalize_value(value: Any, depth: int = 0) -> Any:
    if depth > MAX_CANONICAL_DEPTH:
        raise ValueError(f"Exceeded max canonicalization depth of {MAX_CANONICAL_DEPTH}")

    if isinstance(value, dict):
        return {k: _canonicalize_value(v, depth + 1) for k, v in sorted(value.items())}

    if isinstance(value, (list, tuple)):
        return [_canonicalize_value(v, depth + 1) for v in value]

    if isinstance(value, float):
        if value != value or value in (float("inf"), float("-inf")):
            raise ValueError(f"NaN/Inf not permitted in canonical JSON: {value}")
        return value

    if isinstance(value, (bytes, bytearray)):
        raise ValueError("bytes/bytearray not permitted in canonical JSON")

    return value


def stable_json_bytes(value: Any) -> bytes:
    canonical = _canonicalize_value(value)
    return json.dumps(
        canonical,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
        allow_nan=False,
    ).encode("utf-8")


def hash_v2(kind: str, payload: dict[str, Any]) -> str:
    wrapper = {
        "hash_namespace": HASH_NAMESPACE,
        "kind": kind,
        "payload": payload,
    }
    digest = sha256(stable_json_bytes(wrapper)).hexdigest()
    return f"sha256:{digest}"
