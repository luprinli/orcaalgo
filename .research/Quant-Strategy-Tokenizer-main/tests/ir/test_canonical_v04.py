from __future__ import annotations

from collections.abc import Callable

from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04
from qst.ir.canonical import canonical_bytes_v04, canonicalize_v04


def _strategy() -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBodyV04(
            id="canonical",
            nodes=[
                NodeV04(id="z", params={"b": 2, "a": 1}),
                NodeV04(id="a", params={"nested": {"y": 2, "x": 1}}),
            ],
            outputs={"z": "z.value", "a": "a.value"},
        ),
        metadata={"z": True, "a": False},
    )


def test_canonical_bytes_are_deterministic() -> None:
    left = canonical_bytes_v04(_strategy())
    right = canonical_bytes_v04(_strategy().model_dump(mode="json"))

    assert left == right


def test_canonicalize_sorts_nodes_and_mapping_values() -> None:
    canonical = canonicalize_v04(_strategy())

    assert [node.id for node in canonical.strategy.nodes] == ["a", "z"]
    assert list(canonical.strategy.outputs) == ["a", "z"]
    assert list(canonical.metadata) == ["a", "z"]
    assert list(canonical.strategy.nodes[0].params["nested"]) == ["x", "y"]


def test_default_values_are_stable() -> None:
    canonical = canonicalize_v04({"strategy": {"id": "defaults"}})

    assert canonical.ir_version == "qst-ir/0.4"
    assert canonical.canonical_version == "qst-canonical/0.4"
    assert canonical.strategy.version == 1
    assert canonical.strategy.nodes == []
    assert canonical.strategy.outputs == {}
    assert canonical.metadata == {}


def test_canonicalize_signature_accepts_mapping() -> None:
    fn: Callable[[dict[str, object]], StrategyIRV04] = canonicalize_v04

    assert fn({"strategy": {"id": "mapping"}}).strategy.id == "mapping"
