from __future__ import annotations

from qst.hash import behavior_hash_v2


def test_behavior_hash_supports_empty_payload() -> None:
    assert behavior_hash_v2() == behavior_hash_v2({})


def test_behavior_hash_is_deterministic_for_minimal_material() -> None:
    behavior_material = {
        "token": "opaque.compute",
        "behavior_version": 1,
        "contracts": [{"name": "identity", "expected": "pass"}],
    }

    assert behavior_hash_v2(behavior_material) == behavior_hash_v2(behavior_material)
