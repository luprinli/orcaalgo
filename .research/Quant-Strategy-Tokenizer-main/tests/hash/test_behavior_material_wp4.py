from __future__ import annotations

import pytest

from qst.hash import (
    BehaviorMaterialV2,
    behavior_hash_for_material_v2,
)
from qst.numeric import NumericPolicy, semantic_float64_policy
from qst.token_evolution import TokenLifecycleStatus


def test_behavior_material_requires_numeric_policy() -> None:
    with pytest.raises(ValueError):
        BehaviorMaterialV2.model_validate({"behavior_version": 1})


def test_behavior_hash_changes_when_numeric_policy_changes() -> None:
    base = {
        "behavior_version": 1,
        "numeric_policy": semantic_float64_policy().model_dump(mode="json"),
    }
    changed_policy = NumericPolicy(
        representation="float64",
        deterministic_level="bit_exact",
        reduction_order="fixed_input_order",
        nan_policy="propagate",
        inf_policy="reject",
    )
    changed = {**base, "numeric_policy": changed_policy.model_dump(mode="json")}

    assert behavior_hash_for_material_v2(base) != behavior_hash_for_material_v2(changed)


def test_lifecycle_status_affects_behavior_hash() -> None:
    active = BehaviorMaterialV2(
        behavior_version=1,
        numeric_policy=semantic_float64_policy(),
        lifecycle=TokenLifecycleStatus(lifecycle="active"),
    )
    deprecated = BehaviorMaterialV2(
        behavior_version=1,
        numeric_policy=semantic_float64_policy(),
        lifecycle=TokenLifecycleStatus(lifecycle="deprecated"),
    )

    assert behavior_hash_for_material_v2(active) != behavior_hash_for_material_v2(deprecated)


def test_behavior_material_rejects_non_json_contracts() -> None:
    with pytest.raises(ValueError):
        BehaviorMaterialV2(
            behavior_version=1,
            numeric_policy=semantic_float64_policy(),
            contracts=[{"bad": float("nan")}],
        )
