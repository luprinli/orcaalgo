from __future__ import annotations

from collections import Counter

import pytest

from qst.hash import compute_hashes_v2, token_spec_hash_for_spec_v2
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.tokens import (
    TokenRegistryV2,
    TokenSpecV2,
    builtin_token_packs,
    validate_token_maturity_for_profile,
)


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def test_builtin_token_packs_are_deterministic_and_conflict_free() -> None:
    first = builtin_token_packs()
    second = builtin_token_packs()
    first_ids = [(pack.pack_id, pack.version) for pack in first]
    refs = [spec.ref_key for spec in _all_specs()]
    counts = Counter(refs)

    assert first_ids == [(pack.pack_id, pack.version) for pack in second]
    assert [pack.pack_id for pack in first] == sorted(pack.pack_id for pack in first)
    assert all(count == 1 for count in counts.values())
    assert TokenRegistryV2.from_packs(first).result.ok


def test_all_builtin_tokens_have_surface_contracts() -> None:
    for spec in _all_specs():
        surface = spec.surface
        assert surface.category
        assert surface.family
        assert surface.layer
        assert surface.maturity
        assert surface.execution_support
        assert surface.contract.temporal
        assert surface.contract.numeric
        assert surface.contract.missing_data
        assert surface.contract.failure_mode
        if surface.maturity == "accepted":
            assert surface.execution_support in {
                "metadata_only",
                "reference_helper",
                "runtime_executor",
                "external_only",
            }


@pytest.mark.parametrize(
    ("token_name", "profile", "expected_severity"),
    [
        ("optimizer.mean_variance", "research", "warning"),
        ("optimizer.mean_variance", "pretrade", "error"),
        ("event.join_asof", "research", "error"),
        ("distribution.normal_fit", "production_guarded", "error"),
    ],
)
def test_maturity_profile_gate_matrix(
    token_name: str,
    profile: str,
    expected_severity: str,
) -> None:
    spec = next(spec for spec in _all_specs() if spec.token_ref.name == token_name)
    diagnostics = validate_token_maturity_for_profile(spec, profile=profile)  # type: ignore[arg-type]

    assert [diagnostic.severity for diagnostic in diagnostics] == [expected_severity]


def test_deprecated_token_profile_gate() -> None:
    base = next(spec for spec in _all_specs() if spec.token_ref.name == "math.add")
    deprecated = base.model_copy(
        update={"surface": base.surface.model_copy(update={"maturity": "deprecated"})}
    )

    assert validate_token_maturity_for_profile(deprecated, profile="pretrade")[0].severity == "warning"
    assert (
        validate_token_maturity_for_profile(deprecated, profile="production_guarded")[0].severity
        == "error"
    )


def test_reserved_design_token_is_rejected_by_ir_validator() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="reserved_token",
            nodes=[
                NodeV04(
                    id="event_join",
                    token_ref={
                        "namespace": "core",
                        "name": "event.join_asof",
                        "version": 1,
                        "behavior_version": 1,
                    },
                )
            ],
        )
    )

    result = validate_ir_v04(ir)

    assert not result.ok
    assert result.errors[0].code == "QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE"


def test_surface_contract_changes_token_spec_hash_but_not_strategy_hash_algorithm() -> None:
    spec = next(spec for spec in _all_specs() if spec.token_ref.name == "math.add")
    changed = spec.model_copy(
        update={
            "surface": spec.surface.model_copy(
                update={
                    "contract": spec.surface.contract.model_copy(
                        update={"numeric": "changed numeric contract"}
                    )
                }
            )
        }
    )
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="hash_surface",
            nodes=[
                NodeV04(
                    id="add",
                    token_ref={
                        "namespace": "core",
                        "name": "math.add",
                        "version": 1,
                        "behavior_version": 1,
                    },
                    inputs={"a": "$externals.a", "b": "$externals.b"},
                    params={},
                )
            ],
            outputs={"value": "add.value"},
        )
    )

    assert token_spec_hash_for_spec_v2(spec) != token_spec_hash_for_spec_v2(changed)
    assert compute_hashes_v2(ir) == compute_hashes_v2(ir)
