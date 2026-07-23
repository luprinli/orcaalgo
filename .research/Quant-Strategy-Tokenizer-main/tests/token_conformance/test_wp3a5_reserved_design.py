from __future__ import annotations

import json

from typer.testing import CliRunner

from qst.cli import app
from qst.hash import token_spec_hash_for_spec_v2
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.tokens import (
    TokenSpecV2,
    builtin_token_packs,
    validate_token_maturity_for_profile,
)

RESERVED_TOKENS = {
    "event.join_asof",
    "event.filter",
    "event.window_count",
    "distribution.normal_fit",
    "distribution.quantile",
    "distribution.tail_probability",
    "execution.submit_order",
    "execution.cancel_order",
    "execution.fill_report",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _strategy_for(token_name: str) -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBodyV04(
            id=f"reserved_{token_name.replace('.', '_')}",
            nodes=[
                NodeV04(
                    id="reserved_node",
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


def test_reserved_design_tokens_have_non_executable_surface() -> None:
    specs = _spec_by_name()

    assert RESERVED_TOKENS <= set(specs)
    for name in sorted(RESERVED_TOKENS):
        surface = specs[name].surface
        assert surface.maturity == "reserved_design"
        assert surface.execution_support == "metadata_only"
        assert surface.contract.scope == "validation_only"
        assert surface.capabilities.reserved_only is True
        assert surface.capabilities.deterministic_level == "reserved"
        assert any("Reserved design token" in note for note in surface.agent_metadata.usage_notes)
        assert "implementation instruction" in " ".join(surface.agent_metadata.usage_notes)


def test_reserved_design_strategy_usage_is_rejected_in_every_profile() -> None:
    for name in sorted(RESERVED_TOKENS):
        for profile in ("research", "paper", "pretrade", "production_guarded"):
            result = validate_ir_v04(_strategy_for(name), profile=profile)  # type: ignore[arg-type]

            assert not result.ok
            assert result.errors[0].code == "QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE"
            assert result.errors[0].node_id == "reserved_node"
            assert result.errors[0].profile == profile


def test_execution_plan_shells_remain_metadata_only_without_facade() -> None:
    specs = _spec_by_name()

    for name in {"plan.noop", "plan.order_intent"}:
        surface = specs[name].surface
        assert surface.maturity == "accepted"
        assert surface.family == "execution"
        assert surface.execution_support == "metadata_only"
        assert surface.capabilities.reserved_only is False
        assert surface.capabilities.deterministic_level == "annotation_only"
        assert specs[name].tests == [{"kind": "metadata_only", "deterministic": False}]

    import qst.tokens as tokens

    assert not hasattr(tokens, "evaluate_execution_token")


def test_continuous_score_surface_profile_gates() -> None:
    specs = _spec_by_name()

    zscore = specs["score.zscore"]
    calibrate = specs["score.calibrate"]

    assert zscore.surface.family == "continuous_score"
    assert zscore.surface.maturity == "accepted"
    assert zscore.surface.execution_support == "reference_helper"

    assert calibrate.surface.family == "continuous_score"
    assert calibrate.surface.maturity == "experimental"
    assert calibrate.surface.execution_support == "metadata_only"
    assert calibrate.surface.capabilities.deterministic_level == "annotation_only"
    assert "DecisionKind" in " ".join(calibrate.surface.agent_metadata.usage_notes)
    assert validate_token_maturity_for_profile(calibrate, profile="research")[0].severity == "warning"
    assert validate_token_maturity_for_profile(calibrate, profile="paper")[0].severity == "warning"
    assert validate_token_maturity_for_profile(calibrate, profile="pretrade")[0].severity == "error"
    assert (
        validate_token_maturity_for_profile(calibrate, profile="production_guarded")[0].severity
        == "error"
    )


def test_reserved_surface_metadata_changes_token_spec_hash() -> None:
    spec = _spec_by_name()["event.filter"]
    changed = spec.model_copy(
        update={
            "surface": spec.surface.model_copy(
                update={
                    "contract": spec.surface.contract.model_copy(
                        update={"failure_mode": "changed reserved boundary"}
                    )
                }
            )
        }
    )

    assert token_spec_hash_for_spec_v2(spec) != token_spec_hash_for_spec_v2(changed)


def test_vocabulary_check_reports_reserved_surface_gate() -> None:
    result = CliRunner().invoke(app, ["vocabulary", "--check"])

    assert result.exit_code == 0
    payload = json.loads(result.output)
    assert payload["ok"] is True
    assert payload["diagnostics"] == []
    assert payload["token_count"] == len(_all_specs())
