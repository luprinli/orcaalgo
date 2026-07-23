from __future__ import annotations

from orca.hash.graph import graph_hash_v2, instance_hash_v2
from orca.models.strategy import StrategyIRV04


def verify_graph_hash(ir: StrategyIRV04, expected: str) -> bool:
    return graph_hash_v2(ir) == expected


def verify_instance_hash(ir: StrategyIRV04, expected: str) -> bool:
    return instance_hash_v2(ir) == expected
