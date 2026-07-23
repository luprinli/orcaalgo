"""Expected artifact hash framework for Token System v2."""

from __future__ import annotations

from typing import Any

from qst.hash.common import hash_v2_payload


def expected_artifact_hash_v2(payload: Any | None = None) -> str:
    """Hash expected artifact descriptors."""

    return hash_v2_payload("expected_artifact", {} if payload is None else payload)
