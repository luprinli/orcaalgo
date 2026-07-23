from __future__ import annotations

import math
import re

import pytest

from qst.hash import (
    HASH_V2_PATTERN,
    audit_chain_hash_v2,
    behavior_hash_v2,
    compute_hashes_v2,
    expected_artifact_hash_v2,
    graph_hash_v2,
    implementation_ref_hash_v2,
    instance_hash_v2,
    param_hash_v2,
    runtime_environment_hash_v2,
    signature_hash_v2,
    token_pack_hash_v2,
    token_spec_hash_v2,
)
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04


def _ir() -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBodyV04(
            id="hashable",
            nodes=[
                NodeV04(id="n1", token="opaque.input", version=1),
                NodeV04(
                    id="n2",
                    token="opaque.compute",
                    version=1,
                    inputs={"x": "n1.value"},
                    params={"alpha": "0.5"},
                ),
            ],
            outputs={"result": "n2.value"},
        ),
    )


def _assert_hash(value: str) -> None:
    assert re.match(HASH_V2_PATTERN, value), value


def test_three_layer_hashes_return_sha256_values() -> None:
    hashes = compute_hashes_v2(_ir())

    _assert_hash(hashes.graph_hash)
    _assert_hash(hashes.param_hash)
    _assert_hash(hashes.instance_hash)
    assert hashes.graph_hash == graph_hash_v2(_ir())
    assert hashes.param_hash == param_hash_v2(_ir())
    assert hashes.instance_hash == instance_hash_v2(_ir())


def test_hashes_are_sensitive_to_structure_and_params() -> None:
    base = _ir()
    changed_param = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="hashable",
            nodes=[
                NodeV04(id="n1", token="opaque.input", version=1),
                NodeV04(
                    id="n2",
                    token="opaque.compute",
                    version=1,
                    inputs={"x": "n1.value"},
                    params={"alpha": "0.7"},
                ),
            ],
            outputs={"result": "n2.value"},
        )
    )

    assert graph_hash_v2(base) == graph_hash_v2(changed_param)
    assert param_hash_v2(base) != param_hash_v2(changed_param)
    assert instance_hash_v2(base) != instance_hash_v2(changed_param)


def test_all_hash_kinds_return_sha256_values() -> None:
    payload = {"id": "thing", "version": 1}

    for hash_fn in (
        signature_hash_v2,
        behavior_hash_v2,
        token_spec_hash_v2,
        token_pack_hash_v2,
        implementation_ref_hash_v2,
        audit_chain_hash_v2,
        runtime_environment_hash_v2,
        expected_artifact_hash_v2,
    ):
        _assert_hash(hash_fn(payload))


def test_hash_kinds_reject_non_canonical_json_values() -> None:
    with pytest.raises(ValueError):
        signature_hash_v2({"bad": math.nan})

    with pytest.raises(TypeError):
        behavior_hash_v2({"bad": (1, 2)})


def test_new_wp1_hash_kinds_are_deterministic_and_empty_capable() -> None:
    assert runtime_environment_hash_v2() == runtime_environment_hash_v2({})
    assert expected_artifact_hash_v2() == expected_artifact_hash_v2({})
    assert runtime_environment_hash_v2({"python": "3.11"}) == runtime_environment_hash_v2({"python": "3.11"})
