"""Parameter hash for qst-ir/0.4."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from qst.hash.common import hash_v2_payload
from qst.ir.canonical import canonicalize_v04
from qst.ir.schema import StrategyIRV04


def param_hash_v2(ir: StrategyIRV04 | Mapping[str, Any]) -> str:
    """Hash v0.4 parameter material without graph topology metadata."""

    canonical = canonicalize_v04(ir)
    payload = {
        "ir_version": canonical.ir_version,
        "canonical_version": canonical.canonical_version,
        "strategy": {
            "id": canonical.strategy.id,
            "version": canonical.strategy.version,
            "nodes": [
                {
                    "id": node.id,
                    "params": node.params,
                }
                for node in canonical.strategy.nodes
            ],
        },
    }
    return hash_v2_payload("param", payload)
