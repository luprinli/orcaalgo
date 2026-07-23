"""Audit-chain hash framework for Token System v2."""

from __future__ import annotations

from typing import Any

from qst.hash.common import hash_v2_payload


def audit_chain_hash_v2(payload: Any | None = None) -> str:
    """Hash audit chain material for future Token System v2 locks."""

    return hash_v2_payload("audit_chain", {} if payload is None else payload)
