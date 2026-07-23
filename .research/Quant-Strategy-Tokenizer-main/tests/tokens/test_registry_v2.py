from __future__ import annotations

from qst.hash import token_pack_hash_for_pack_v2
from qst.token_evolution import TokenLifecycleStatus
from qst.tokens import (
    TokenPackDependency,
    TokenPackManifestV2,
    TokenRegistryV2,
    validate_token_pack_dependencies,
)
from tests.tokens.test_token_pack_v2 import make_pack
from tests.tokens.test_token_spec_v2 import make_spec


def diagnostic_codes(registry: TokenRegistryV2) -> list[str]:
    return [diagnostic.code for diagnostic in registry.result.diagnostics]


def test_registry_deduplicates_same_non_core_hash() -> None:
    left = make_pack("left", namespace="demo")
    right = make_pack("right", namespace="demo")

    registry = TokenRegistryV2.from_packs((right, left))

    assert registry.result.ok
    assert len(registry.records) == 1
    assert any(entry.startswith("dedupe demo.identity") for entry in registry.resolution_log)


def test_registry_rejects_duplicate_non_core_different_hash() -> None:
    left = make_pack("left", namespace="demo")
    right = make_pack(
        "right",
        namespace="demo",
        tokens=(
            make_spec(
                namespace="demo",
                name="identity",
                lifecycle=TokenLifecycleStatus(lifecycle="deprecated"),
            ),
        ),
    )

    registry = TokenRegistryV2.from_packs((left, right))

    assert not registry.result.ok
    assert "QST_V2_TOKEN_REF_CONFLICT" in diagnostic_codes(registry)


def test_project_local_override_is_allowed_for_owned_namespace() -> None:
    installed = make_pack("installed", namespace="demo")
    local = make_pack(
        "local",
        namespace="demo",
        tokens=(
            make_spec(
                namespace="demo",
                name="identity",
                lifecycle=TokenLifecycleStatus(lifecycle="deprecated"),
            ),
        ),
        origin_tier="user_local",
    )

    registry = TokenRegistryV2.from_packs((installed, local))

    assert registry.result.ok
    assert registry.get("demo.identity").pack_id == "local"
    assert "QST_V2_PROJECT_LOCAL_OVERRIDE" in diagnostic_codes(registry)


def test_core_namespace_collision_rejected() -> None:
    bad_core = make_pack(
        "bad-core",
        namespace="core",
        tokens=(make_spec(namespace="core", name="identity"),),
        origin_tier="community_pack",
    )

    registry = TokenRegistryV2.from_packs((bad_core,))

    assert not registry.result.ok
    assert "QST_V2_CORE_NAMESPACE_SHADOWED" in diagnostic_codes(registry)


def test_attestation_is_not_self_trusted() -> None:
    pack = TokenPackManifestV2(
        pack_id="verified-claim",
        version="1.0.0",
        namespaces=("demo",),
        tokens=(make_spec(attestation_kind="signed_pack"),),
        origin_tier="community_pack",
        attestation_kind="qst_verified",
    )

    registry = TokenRegistryV2.from_packs((pack,))

    assert registry.result.ok
    assert diagnostic_codes(registry).count("QST_V2_ATTESTATION_NOT_SELF_TRUSTED") == 2


def test_resolution_log_stable_across_input_order() -> None:
    a = make_pack("a", namespace="a", tokens=(make_spec(namespace="a"),))
    b = make_pack("b", namespace="b", tokens=(make_spec(namespace="b"),))

    assert TokenRegistryV2.from_packs((a, b)).resolution_log == TokenRegistryV2.from_packs((b, a)).resolution_log


def test_dependency_resolution_transitive_order() -> None:
    c = make_pack("c", namespace="c", tokens=(make_spec(namespace="c"),))
    b = make_pack(
        "b",
        namespace="b",
        tokens=(make_spec(namespace="b"),),
        dependencies=(TokenPackDependency(pack_id="c", version_constraint=">=1"),),
    )
    a = make_pack(
        "a",
        namespace="a",
        tokens=(make_spec(namespace="a"),),
        dependencies=(TokenPackDependency(pack_id="b", version_constraint=">=1"),),
    )

    resolution = validate_token_pack_dependencies((a, c, b))

    assert resolution.result.ok
    assert [pack.pack_id for pack in resolution.ordered_packs] == ["c", "b", "a"]


def test_dependency_resolution_reports_missing_version_hash_and_cycle() -> None:
    base = make_pack("base", namespace="base", tokens=(make_spec(namespace="base"),))
    missing = make_pack(
        "missing",
        namespace="missing",
        tokens=(make_spec(namespace="missing"),),
        dependencies=(TokenPackDependency(pack_id="absent", version_constraint=">=1"),),
    )
    mismatch = make_pack(
        "mismatch",
        namespace="mismatch",
        tokens=(make_spec(namespace="mismatch"),),
        dependencies=(TokenPackDependency(pack_id="base", version_constraint=">=2"),),
    )
    hash_mismatch = make_pack(
        "hash-mismatch",
        namespace="hashmismatch",
        tokens=(make_spec(namespace="hashmismatch"),),
        dependencies=(
            TokenPackDependency(
                pack_id="base",
                version_constraint=">=1",
                token_pack_hash="sha256:" + "0" * 64,
            ),
        ),
    )
    cycle_a = make_pack(
        "cycle-a",
        namespace="cyclea",
        tokens=(make_spec(namespace="cyclea"),),
        dependencies=(TokenPackDependency(pack_id="cycle-b", version_constraint=">=1"),),
    )
    cycle_b = make_pack(
        "cycle-b",
        namespace="cycleb",
        tokens=(make_spec(namespace="cycleb"),),
        dependencies=(TokenPackDependency(pack_id="cycle-a", version_constraint=">=1"),),
    )

    resolution = validate_token_pack_dependencies(
        (missing, mismatch, hash_mismatch, base, cycle_a, cycle_b)
    )
    codes = [diagnostic.code for diagnostic in resolution.result.diagnostics]

    assert "QST_V2_TOKEN_PACK_DEP_MISSING" in codes
    assert "QST_V2_TOKEN_PACK_DEP_VERSION_MISMATCH" in codes
    assert "QST_V2_TOKEN_PACK_DEP_HASH_MISMATCH" in codes
    assert "QST_V2_TOKEN_PACK_DEP_CYCLE" in codes


def test_dependency_resolution_accepts_hash_pin() -> None:
    base = make_pack("base", namespace="base", tokens=(make_spec(namespace="base"),))
    actual_hash = token_pack_hash_for_pack_v2(base)
    dependent = make_pack(
        "dependent",
        namespace="dependent",
        tokens=(make_spec(namespace="dependent"),),
        dependencies=(
            TokenPackDependency(
                pack_id="base",
                version_constraint=">=1",
                token_pack_hash=actual_hash,
            ),
        ),
    )

    resolution = validate_token_pack_dependencies((dependent, base))

    assert resolution.result.ok


def test_registry_get_missing_raises_key_error() -> None:
    registry = TokenRegistryV2.from_packs((make_pack(),))

    try:
        registry.get("missing.token")
    except KeyError as exc:
        assert "missing.token" in str(exc)
    else:
        raise AssertionError("expected missing token lookup to raise")
