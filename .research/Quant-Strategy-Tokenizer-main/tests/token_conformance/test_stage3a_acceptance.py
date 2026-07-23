from __future__ import annotations

import json

from typer.testing import CliRunner

from qst.cli import app
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.tokens import TokenSpecV2, builtin_token_packs, validate_token_maturity_for_profile

EXPECTED_STAGE3A_FAMILIES = {
    "align",
    "bool",
    "compare",
    "continuous_score",
    "data",
    "decision",
    "distribution",
    "event",
    "execution",
    "factor",
    "gate",
    "indicator",
    "math",
    "optimizer",
    "panel",
    "risk",
    "signal",
    "state",
    "time",
    "weight",
    "window",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _strategy_for(token_name: str) -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBodyV04(
            id=f"stage3a_acceptance_{token_name.replace('.', '_')}",
            nodes=[
                NodeV04(
                    id="node",
                    token_ref={
                        "namespace": "core",
                        "name": token_name,
                        "version": 1,
                        "behavior_version": 1,
                    },
                )
            ],
        )
    )


def test_stage3a_builtin_vocabulary_covers_all_families_with_complete_surface() -> None:
    specs = _all_specs()
    families = {spec.surface.family for spec in specs}

    assert EXPECTED_STAGE3A_FAMILIES <= families
    for spec in specs:
        surface = spec.surface
        assert spec.token_ref.namespace == "core"
        assert surface.family in EXPECTED_STAGE3A_FAMILIES
        assert surface.category
        assert surface.layer
        assert surface.maturity
        assert surface.execution_support
        assert surface.contract.temporal
        assert surface.contract.numeric
        assert surface.contract.missing_data
        assert surface.contract.failure_mode
        assert surface.capabilities.deterministic_level


def test_stage3a_reserved_and_experimental_profile_gates_are_hard_acceptance_rules() -> None:
    specs = {spec.token_ref.name: spec for spec in _all_specs()}

    for name, spec in specs.items():
        if spec.surface.maturity == "reserved_design":
            assert spec.surface.execution_support == "metadata_only"
            assert spec.surface.capabilities.reserved_only is True
            assert spec.surface.capabilities.deterministic_level == "reserved"
            result = validate_ir_v04(_strategy_for(name), profile="research")
            assert not result.ok
            assert result.errors[0].code == "QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE"

    for spec in specs.values():
        if spec.surface.maturity == "experimental" and spec.surface.execution_support == "metadata_only":
            assert validate_token_maturity_for_profile(spec, profile="research")[0].severity == "warning"
            assert validate_token_maturity_for_profile(spec, profile="paper")[0].severity == "warning"
            assert validate_token_maturity_for_profile(spec, profile="pretrade")[0].severity == "error"
            assert (
                validate_token_maturity_for_profile(spec, profile="production_guarded")[0].severity
                == "error"
            )


def test_stage3a_vocabulary_check_is_clean_acceptance_gate() -> None:
    result = CliRunner().invoke(app, ["vocabulary", "--check"])

    assert result.exit_code == 0, result.output
    payload = json.loads(result.output)
    assert payload["ok"] is True
    assert payload["diagnostics"] == []
    assert payload["token_count"] == len(_all_specs())
