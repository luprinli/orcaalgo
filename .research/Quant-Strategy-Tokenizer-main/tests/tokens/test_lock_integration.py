from __future__ import annotations

from qst.hash import (
    implementation_ref_hash_v2,
    runtime_environment_hash_v2,
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.tokens import (
    TokenLockEntryV04,
    TokenLockSnapshotV04,
    TokenPackLockDependencyV04,
    token_lock_entry_from_spec,
    token_pack_lock_dependency_from_pack,
    verify_token_lock_snapshot,
)
from tests.tokens.test_token_pack_v2 import make_pack
from tests.tokens.test_token_spec_v2 import make_spec


def diagnostic_codes(snapshot: TokenLockSnapshotV04, packs: object) -> list[str]:
    result = verify_token_lock_snapshot(snapshot, packs)  # type: ignore[arg-type]
    return [diagnostic.code for diagnostic in result.diagnostics]


def test_lock_entry_records_token_hashes() -> None:
    spec = make_spec(risk={"risk_level": "medium"})
    pack = make_pack(tokens=(spec,))

    entry = token_lock_entry_from_spec(spec, pack)
    dependency = token_pack_lock_dependency_from_pack(pack)

    assert entry.token_ref == spec.token_ref
    assert entry.token_spec_hash == token_spec_hash_for_spec_v2(spec)
    assert entry.token_pack_hash == token_pack_hash_for_pack_v2(pack)
    assert entry.implementation_ref_hash == implementation_ref_hash_v2(spec.implementation_ref)
    assert entry.runtime_environment_hash == runtime_environment_hash_v2(spec.runtime_environment_ref)
    assert entry.origin_tier == spec.origin_tier
    assert entry.attestation_kind == spec.attestation_kind
    assert entry.risk_level == "medium"
    assert dependency.pack_id == pack.pack_id
    assert dependency.token_pack_hash == token_pack_hash_for_pack_v2(pack)


def test_lock_snapshot_verifies_available_pack() -> None:
    spec = make_spec()
    pack = make_pack(tokens=(spec,))
    snapshot = TokenLockSnapshotV04(
        tokens=(token_lock_entry_from_spec(spec, pack),),
        token_pack_dependencies=(token_pack_lock_dependency_from_pack(pack),),
    )

    result = verify_token_lock_snapshot(snapshot, (pack,))

    assert result.ok
    assert result.diagnostics == []


def test_lock_snapshot_reports_missing_pack() -> None:
    spec = make_spec()
    pack = make_pack(tokens=(spec,))
    snapshot = TokenLockSnapshotV04(
        tokens=(token_lock_entry_from_spec(spec, pack),),
        token_pack_dependencies=(token_pack_lock_dependency_from_pack(pack),),
    )

    codes = diagnostic_codes(snapshot, ())

    assert codes.count("QST_V2_LOCK_TOKEN_PACK_MISSING") == 2


def test_lock_snapshot_reports_pack_hash_mismatch() -> None:
    pack = make_pack()
    bad_dependency = TokenPackLockDependencyV04(
        pack_id=pack.pack_id,
        version=pack.version,
        token_pack_hash="sha256:" + "0" * 64,
    )
    entry_data = token_lock_entry_from_spec(pack.tokens[0], pack).model_dump(mode="json")
    entry_data["token_pack_hash"] = "sha256:" + "0" * 64
    bad_entry = TokenLockEntryV04.model_validate(entry_data)
    snapshot = TokenLockSnapshotV04(
        tokens=(bad_entry,),
        token_pack_dependencies=(bad_dependency,),
    )

    codes = diagnostic_codes(snapshot, (pack,))

    assert codes.count("QST_V2_LOCK_TOKEN_PACK_HASH_MISMATCH") == 2


def test_lock_snapshot_reports_token_spec_hash_mismatch() -> None:
    pack = make_pack()
    entry_data = token_lock_entry_from_spec(pack.tokens[0], pack).model_dump(mode="json")
    entry_data["token_spec_hash"] = "sha256:" + "1" * 64
    bad_entry = TokenLockEntryV04.model_validate(entry_data)
    snapshot = TokenLockSnapshotV04(tokens=(bad_entry,))

    codes = diagnostic_codes(snapshot, (pack,))

    assert "QST_V2_LOCK_TOKEN_SPEC_HASH_MISMATCH" in codes


def test_lock_snapshot_reports_implementation_and_runtime_hash_mismatch() -> None:
    pack = make_pack()
    entry_data = token_lock_entry_from_spec(pack.tokens[0], pack).model_dump(mode="json")
    entry_data["implementation_ref_hash"] = "sha256:" + "2" * 64
    entry_data["runtime_environment_hash"] = "sha256:" + "3" * 64
    bad_entry = TokenLockEntryV04.model_validate(entry_data)
    snapshot = TokenLockSnapshotV04(tokens=(bad_entry,))

    codes = diagnostic_codes(snapshot, (pack,))

    assert "QST_V2_LOCK_IMPLEMENTATION_REF_HASH_MISMATCH" in codes
    assert "QST_V2_LOCK_RUNTIME_ENVIRONMENT_HASH_MISMATCH" in codes
