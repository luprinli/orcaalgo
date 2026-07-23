from __future__ import annotations

from qst.decision import (
    DECISION_ALGEBRA_PACK_ID,
    DECISION_ALGEBRA_PACK_VERSION,
    decision_algebra_token_pack_v2,
)
from qst.hash import (
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.tokens import TokenRegistryV2


def test_decision_algebra_token_pack_validates_and_resolves() -> None:
    pack = decision_algebra_token_pack_v2()
    registry = TokenRegistryV2.from_packs((pack,))

    assert pack.pack_id == DECISION_ALGEBRA_PACK_ID
    assert pack.version == DECISION_ALGEBRA_PACK_VERSION
    assert pack.origin_tier == "core"
    assert registry.result.ok
    assert [record.spec.token_id for record in registry.records] == [
        "core.decision.any_accept",
        "core.decision.majority",
        "core.decision.permissive_and",
        "core.decision.quorum",
        "core.decision.strict_and",
        "core.decision.unknown_propagating_and",
        "core.decision.weighted_vote",
    ]


def test_decision_algebra_token_categories_are_recorded() -> None:
    categories = {
        spec.token_id: spec.state["category"]
        for spec in decision_algebra_token_pack_v2().tokens
    }

    assert categories["core.decision.strict_and"] == "fold_policy"
    assert categories["core.decision.permissive_and"] == "fold_policy"
    assert categories["core.decision.unknown_propagating_and"] == "monoid"
    assert categories["core.decision.any_accept"] == "monoid"
    assert categories["core.decision.majority"] == "aggregator"
    assert categories["core.decision.weighted_vote"] == "aggregator"
    assert categories["core.decision.quorum"] == "aggregator"


def test_decision_algebra_token_refs_are_canonical_and_hash_stable() -> None:
    pack = decision_algebra_token_pack_v2()
    hashes = [token_spec_hash_for_spec_v2(spec) for spec in pack.tokens]

    assert [spec.token_ref.namespace for spec in pack.tokens] == ["core"] * 7
    assert [spec.token_ref.version for spec in pack.tokens] == [1] * 7
    assert [spec.token_ref.behavior_version for spec in pack.tokens] == [1] * 7
    assert hashes == [
        token_spec_hash_for_spec_v2(spec)
        for spec in decision_algebra_token_pack_v2().tokens
    ]
    assert token_pack_hash_for_pack_v2(pack) == token_pack_hash_for_pack_v2(
        decision_algebra_token_pack_v2()
    )


def test_decision_algebra_token_hash_changes_on_schema_material() -> None:
    pack = decision_algebra_token_pack_v2()
    original = pack.tokens[0]
    changed = original.model_copy(
        update={
            "params_schema": {
                **original.params_schema,
                "description": "changed",
            }
        }
    )

    assert token_spec_hash_for_spec_v2(original) != token_spec_hash_for_spec_v2(changed)
