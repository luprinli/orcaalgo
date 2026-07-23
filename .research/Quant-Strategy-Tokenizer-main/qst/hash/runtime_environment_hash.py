"""Runtime environment hash framework for Token System v2."""

from __future__ import annotations

from typing import Any

from qst.hash.common import hash_v2_payload


def runtime_environment_hash_v2(payload: Any | None = None) -> str:
    """Hash runtime environment material."""

    return hash_v2_payload("runtime_environment", {} if payload is None else payload)
