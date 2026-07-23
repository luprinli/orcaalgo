"""Instance hash for qst-ir/0.4."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from qst.hash.common import IRHashesV2, hash_v2_payload
from qst.hash.graph_hash import graph_hash_v2
from qst.hash.param_hash import param_hash_v2
from qst.ir.canonical import canonicalize_v04
from qst.ir.schema import StrategyIRV04


def instance_hash_v2(ir: StrategyIRV04 | Mapping[str, Any]) -> str:
    """Hash the v0.4 strategy instance from graph and param identities."""

    canonical = canonicalize_v04(ir)
    payload = {
        "ir_version": canonical.ir_version,
        "canonical_version": canonical.canonical_version,
        "graph_hash": graph_hash_v2(canonical),
        "param_hash": param_hash_v2(canonical),
    }
    return hash_v2_payload("instance", payload)


def compute_hashes_v2(ir: StrategyIRV04 | Mapping[str, Any]) -> IRHashesV2:
    """Compute graph, param, and instance hashes for qst-ir/0.4."""

    canonical = canonicalize_v04(ir)
    graph_hash = graph_hash_v2(canonical)
    param_hash = param_hash_v2(canonical)
    return IRHashesV2(
        graph_hash=graph_hash,
        param_hash=param_hash,
        instance_hash=hash_v2_payload(
            "instance",
            {
                "ir_version": canonical.ir_version,
                "canonical_version": canonical.canonical_version,
                "graph_hash": graph_hash,
                "param_hash": param_hash,
            },
        ),
    )
