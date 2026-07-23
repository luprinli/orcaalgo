from __future__ import annotations

from qst.hash import (
    HASH_V2_PATTERN,
    compute_hashes_v2,
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.ir import load_ir_v04_file
from qst.tokens import TokenPackManifestV2, TokenRegistryV2, builtin_token_packs
from tests.token_acceptance._helpers import DEMO_ROOT, all_specs


def test_stage3b_registry_hashes_are_stable_and_well_formed() -> None:
    packs = builtin_token_packs()
    first = [(pack.pack_id, token_pack_hash_for_pack_v2(pack)) for pack in packs]
    second = [(pack.pack_id, token_pack_hash_for_pack_v2(pack)) for pack in builtin_token_packs()]

    assert first == second
    for _, pack_hash in first:
        assert HASH_V2_PATTERN.match(pack_hash)
    for spec in all_specs():
        assert HASH_V2_PATTERN.match(token_spec_hash_for_spec_v2(spec))


def test_stage3b_surface_contract_changes_token_spec_hash_not_strategy_identity() -> None:
    spec = next(spec for spec in all_specs() if spec.token_ref.name == "math.add")
    changed = spec.model_copy(
        update={
            "surface": spec.surface.model_copy(
                update={
                    "contract": spec.surface.contract.model_copy(
                        update={"numeric": "changed by acceptance test"}
                    )
                }
            )
        }
    )
    strategy = load_ir_v04_file(DEMO_ROOT / "01_ema_cross" / "strategy.gkr.yaml")

    assert token_spec_hash_for_spec_v2(spec) != token_spec_hash_for_spec_v2(changed)
    assert compute_hashes_v2(strategy) == compute_hashes_v2(strategy)


def test_stage3b_duplicate_token_ref_conflict_is_deterministic() -> None:
    spec = next(spec for spec in all_specs() if spec.token_ref.name == "math.add")
    changed = spec.model_copy(update={"state": {"stage3b": "changed"}})
    pack_a = TokenPackManifestV2(
        pack_id="stage3b-a",
        version="0.1.0",
        namespaces=("core",),
        tokens=(spec,),
        origin_tier="core",
    )
    pack_b = TokenPackManifestV2(
        pack_id="stage3b-b",
        version="0.1.0",
        namespaces=("core",),
        tokens=(changed,),
        origin_tier="core",
    )

    first = TokenRegistryV2.from_packs((pack_b, pack_a))
    second = TokenRegistryV2.from_packs((pack_a, pack_b))

    assert [diagnostic.code for diagnostic in first.result.diagnostics] == [
        "QST_V2_TOKEN_REF_CONFLICT"
    ]
    assert [diagnostic.model_dump(mode="json") for diagnostic in first.result.diagnostics] == [
        diagnostic.model_dump(mode="json") for diagnostic in second.result.diagnostics
    ]
