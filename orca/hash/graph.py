from __future__ import annotations

from typing import Any

from orca.hash.common import hash_v2
from orca.models.strategy import StrategyIRV04


def _build_graph_payload(ir: StrategyIRV04) -> dict[str, Any]:
    nodes = [
        {
            "id": n.id,
            "token_id": n.token_ref.token_id,
            "token_version": n.token_ref.version,
            "inputs": dict(sorted(n.inputs.items())),
        }
        for n in ir.strategy.nodes
    ]
    return {
        "ir_version": ir.ir_version,
        "canonical_version": ir.canonical_version,
        "strategy": {
            "id": ir.strategy.id,
            "version": ir.strategy.version,
            "nodes": sorted(nodes, key=lambda x: x["id"]),
            "outputs": dict(sorted(ir.strategy.outputs.items())),
        },
    }


def _build_param_payload(ir: StrategyIRV04) -> dict[str, Any]:
    nodes = [{"id": n.id, "params": dict(sorted(n.params.items()))} for n in ir.strategy.nodes]
    return {
        "ir_version": ir.ir_version,
        "canonical_version": ir.canonical_version,
        "strategy": {
            "id": ir.strategy.id,
            "version": ir.strategy.version,
            "nodes": sorted(nodes, key=lambda x: x["id"]),
        },
    }


def graph_hash_v2(ir: StrategyIRV04) -> str:
    return hash_v2("graph", _build_graph_payload(ir))


def param_hash_v2(ir: StrategyIRV04) -> str:
    return hash_v2("param", _build_param_payload(ir))


def instance_hash_v2(ir: StrategyIRV04) -> str:
    payload = {"graph_hash": graph_hash_v2(ir), "param_hash": param_hash_v2(ir)}
    return hash_v2("instance", payload)
