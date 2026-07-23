from __future__ import annotations

from qst.hash import (
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.state import (
    STATE_BASIC_PACK_ID,
    STATE_FSM_PACK_ID,
    STATE_FSM_PACK_VERSION,
    state_basic_token_pack_v2,
    state_fsm_token_pack_v2,
)
from qst.tokens import (
    TokenLockSnapshotV04,
    TokenRegistryV2,
    token_lock_entry_from_spec,
    token_pack_lock_dependency_from_pack,
    token_pack_package_section_from_packs,
    validate_token_pack_dependencies,
    verify_token_lock_snapshot,
    verify_token_pack_package_section,
)


def test_state_fsm_token_pack_validates_and_resolves_with_state_basic() -> None:
    basic = state_basic_token_pack_v2()
    fsm = state_fsm_token_pack_v2()
    registry = TokenRegistryV2.from_packs((fsm, basic))

    assert fsm.pack_id == STATE_FSM_PACK_ID
    assert fsm.version == STATE_FSM_PACK_VERSION
    assert registry.result.ok
    assert registry.get("core.state.fsm").spec.state["wp"] == "WP6b"
    assert "core.state.fsm" in [record.spec.token_id for record in registry.records]


def test_state_fsm_token_ref_and_hash_are_stable() -> None:
    fsm = state_fsm_token_pack_v2()
    spec = fsm.tokens[0]

    assert spec.token_id == "core.state.fsm"
    assert spec.token_ref.namespace == "core"
    assert spec.token_ref.name == "state.fsm"
    assert spec.token_ref.version == 1
    assert spec.token_ref.behavior_version == 1
    assert token_spec_hash_for_spec_v2(spec) == token_spec_hash_for_spec_v2(
        state_fsm_token_pack_v2().tokens[0]
    )
    assert token_pack_hash_for_pack_v2(fsm) == token_pack_hash_for_pack_v2(
        state_fsm_token_pack_v2()
    )


def test_state_fsm_pack_declares_state_basic_dependency() -> None:
    basic = state_basic_token_pack_v2()
    fsm = state_fsm_token_pack_v2()

    assert [(dependency.pack_id, dependency.version_constraint) for dependency in fsm.dependencies] == [
        (STATE_BASIC_PACK_ID, ">=0.1.0")
    ]
    dependency_result = validate_token_pack_dependencies((fsm, basic))

    assert dependency_result.result.ok
    assert [pack.pack_id for pack in dependency_result.ordered_packs] == [
        STATE_BASIC_PACK_ID,
        STATE_FSM_PACK_ID,
    ]


def test_state_fsm_pack_missing_dependency_is_diagnostic() -> None:
    dependency_result = validate_token_pack_dependencies((state_fsm_token_pack_v2(),))

    assert not dependency_result.result.ok
    assert [diagnostic.code for diagnostic in dependency_result.result.diagnostics] == [
        "QST_V2_TOKEN_PACK_DEP_MISSING"
    ]


def test_state_fsm_pack_works_with_wp5b_lock_and_package_helpers() -> None:
    packs = (state_basic_token_pack_v2(), state_fsm_token_pack_v2())
    fsm = packs[1]
    snapshot = TokenLockSnapshotV04(
        tokens=tuple(token_lock_entry_from_spec(spec, fsm) for spec in fsm.tokens),
        token_pack_dependencies=tuple(token_pack_lock_dependency_from_pack(pack) for pack in packs),
    )
    package_section = token_pack_package_section_from_packs(
        packs,
        embedded_policy="spec_only",
    )

    assert verify_token_lock_snapshot(snapshot, packs).ok
    assert verify_token_pack_package_section(package_section, packs).ok
