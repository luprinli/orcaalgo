from __future__ import annotations

from qst.hash import (
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.state import (
    STATE_BASIC_PACK_ID,
    STATE_BASIC_PACK_VERSION,
    state_basic_token_pack_v2,
)
from qst.tokens import (
    TokenLockSnapshotV04,
    TokenRegistryV2,
    token_lock_entry_from_spec,
    token_pack_lock_dependency_from_pack,
    token_pack_package_section_from_packs,
    verify_token_lock_snapshot,
    verify_token_pack_package_section,
)


def test_state_basic_token_pack_validates_and_resolves() -> None:
    pack = state_basic_token_pack_v2()
    registry = TokenRegistryV2.from_packs((pack,))

    assert pack.pack_id == STATE_BASIC_PACK_ID
    assert pack.version == STATE_BASIC_PACK_VERSION
    assert pack.origin_tier == "core"
    assert registry.result.ok
    assert [record.spec.token_id for record in registry.records] == [
        "core.state.accumulate",
        "core.state.delay",
        "core.state.edge_detect",
    ]


def test_state_basic_token_refs_are_canonical_and_hash_stable() -> None:
    pack = state_basic_token_pack_v2()
    hashes = [token_spec_hash_for_spec_v2(spec) for spec in pack.tokens]

    assert [spec.token_ref.namespace for spec in pack.tokens] == ["core", "core", "core"]
    assert [spec.token_ref.version for spec in pack.tokens] == [1, 1, 1]
    assert [spec.token_ref.behavior_version for spec in pack.tokens] == [1, 1, 1]
    assert hashes == [token_spec_hash_for_spec_v2(spec) for spec in state_basic_token_pack_v2().tokens]
    assert token_pack_hash_for_pack_v2(pack) == token_pack_hash_for_pack_v2(state_basic_token_pack_v2())


def test_state_basic_pack_records_policy_metadata() -> None:
    pack = state_basic_token_pack_v2()

    for spec in pack.tokens:
        assert spec.state["stateful"] is True
        assert spec.state["state_policy"]["warmup_policy"] == "emit_null"
        assert spec.state["state_policy"]["reset_policy"] == "never"
        assert spec.state["state_policy"]["missing_event_policy"] == "error"


def test_state_basic_pack_works_with_wp5b_lock_and_package_helpers() -> None:
    pack = state_basic_token_pack_v2()
    snapshot = TokenLockSnapshotV04(
        tokens=tuple(token_lock_entry_from_spec(spec, pack) for spec in pack.tokens),
        token_pack_dependencies=(token_pack_lock_dependency_from_pack(pack),),
    )
    package_section = token_pack_package_section_from_packs(
        (pack,),
        embedded_policy="spec_only",
    )

    assert verify_token_lock_snapshot(snapshot, (pack,)).ok
    assert verify_token_pack_package_section(package_section, (pack,)).ok
