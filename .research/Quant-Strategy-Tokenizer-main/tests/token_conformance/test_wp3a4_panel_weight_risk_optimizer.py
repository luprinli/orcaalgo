from __future__ import annotations

import pytest

from qst.decision import DecisionV2
from qst.panel import PanelPoint, PanelValue, WeightPanelValue, WeightPoint
from qst.tokens import (
    TokenReferenceError,
    TokenReferenceResult,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_panel_token,
    evaluate_risk_token,
    evaluate_weight_token,
    validate_token_maturity_for_profile,
)

WP3A4_TOKENS = {
    "panel.mask",
    "panel.rank",
    "panel.zscore",
    "panel.top_k",
    "panel.bottom_k",
    "panel.demean",
    "panel.group_demean",
    "panel.winsorize",
    "panel.residualize",
    "selection.to_weights",
    "weight.normalize_gross",
    "weight.cap_per_symbol",
    "weight.market_neutral",
    "risk.position_cap",
    "risk.volatility_target",
    "risk.turnover_cap",
    "optimizer.mean_variance",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _panel() -> PanelValue:
    return PanelValue(
        rows=(
            PanelPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", value="1"),
            PanelPoint(timestamp="2026-05-16T09:00:00Z", symbol="BBB", value="2"),
        )
    )


def _weights() -> WeightPanelValue:
    return WeightPanelValue(
        rows=(
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", weight="0.5"),
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="BBB", weight="-0.25"),
        )
    )


def test_wp3a4_tokens_have_surface_contracts() -> None:
    specs = _spec_by_name()

    assert WP3A4_TOKENS <= set(specs)
    for name in sorted(WP3A4_TOKENS):
        spec = specs[name]
        assert spec.surface.category
        assert spec.surface.contract.temporal
        assert spec.surface.contract.numeric
        assert spec.surface.contract.missing_data
        assert spec.surface.contract.failure_mode

    for name in {"risk.position_cap", "risk.volatility_target", "risk.turnover_cap"}:
        assert specs[name].surface.maturity == "accepted"
        assert specs[name].surface.execution_support == "reference_helper"

    optimizer = specs["optimizer.mean_variance"]
    assert optimizer.surface.maturity == "experimental"
    assert optimizer.surface.execution_support == "metadata_only"
    assert optimizer.surface.capabilities.solver_backed is True
    assert optimizer.surface.capabilities.deterministic_level == "annotation_only"
    assert optimizer.surface.contract.solver is not None
    assert optimizer.surface.contract.solver.solver_required is True
    assert optimizer.surface.contract.solver.bit_exact_claim is False


def test_panel_weight_determinism_contracts_are_classified() -> None:
    specs = _spec_by_name()

    for name in {
        "panel.rank",
        "panel.zscore",
        "panel.demean",
        "panel.group_demean",
        "panel.winsorize",
        "panel.residualize",
    }:
        assert specs[name].surface.capabilities.deterministic_level == "semantic_float64"
        assert specs[name].surface.contract.numeric == "semantic_float64"

    for name in {"panel.mask", "panel.top_k", "panel.bottom_k", "selection.to_weights"}:
        assert specs[name].surface.capabilities.deterministic_level == "reference_exact"

    for name in {"weight.normalize_gross", "weight.cap_per_symbol", "weight.market_neutral"}:
        assert specs[name].surface.capabilities.deterministic_level == "reference_exact"
        assert "decimal canonical" in specs[name].surface.contract.numeric


def test_panel_and_weight_facades_delegate_to_existing_helpers() -> None:
    ranked = evaluate_panel_token("panel.rank", _panel())
    weights = WeightPanelValue(
        rows=(
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", weight="0.5"),
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="BBB", weight="-0.5"),
        )
    )
    normalized = evaluate_weight_token("weight.normalize_gross", weights, target_gross="1")

    assert ranked.panel is not None
    assert [(row.symbol, row.value) for row in ranked.panel.rows] == [("AAA", "2"), ("BBB", "1")]
    assert normalized.weights is not None
    assert normalized.trace["operator_id"] == "weight.normalize_gross"
    assert normalized.trace["gross_after"]["2026-05-16T09:00:00Z"] == "1"


def test_risk_position_cap_blocks_breaches_and_preserves_existing_blocks() -> None:
    result = evaluate_risk_token(
        "risk.position_cap",
        [
            DecisionV2(kind="accept", reasons=("base_accept",)),
            DecisionV2(kind="reject", reasons=("base_reject",)),
            DecisionV2(kind="block", reasons=("already_blocked",)),
        ],
        ["0.1", "2", "5"],
        max_abs_position="1",
    )

    assert isinstance(result, TokenReferenceResult)
    assert [decision.kind for decision in result.decisions] == ["accept", "block", "block"]
    assert result.decisions[1].reasons == ("RISK_POSITION_CAP_EXCEEDED", "base_reject")
    assert result.decisions[2].reasons == ("already_blocked",)
    assert result.trace["blocked_count"] == 1

    with pytest.raises(TokenReferenceError) as invalid_position:
        evaluate_risk_token("risk.position_cap", [DecisionV2(kind="accept")], [True], max_abs_position="1")
    assert invalid_position.value.code == "QST_TOKEN_NUMERIC_TYPE_INVALID"


def test_risk_volatility_target_scales_without_normalizing() -> None:
    volatility = PanelValue(
        rows=(
            PanelPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", value="0.5"),
            PanelPoint(timestamp="2026-05-16T09:00:00Z", symbol="BBB", value="0.25"),
        )
    )

    result = evaluate_risk_token(
        "risk.volatility_target",
        _weights(),
        volatility,
        target_volatility="0.1",
    )

    assert result.weights is not None
    assert [(row.symbol, row.weight) for row in result.weights.rows] == [("AAA", "0.1"), ("BBB", "-0.1")]
    assert result.trace["normalization"] == "none"

    zero_volatility = PanelValue(
        rows=(PanelPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", value="0"),)
    )
    zero_result = evaluate_risk_token("risk.volatility_target", _weights(), zero_volatility)
    assert not zero_result.diagnostics.ok
    assert zero_result.diagnostics.errors[0].code in {
        "QST_TOKEN_RISK_VOLATILITY_MISSING",
        "QST_TOKEN_RISK_VOLATILITY_NONPOSITIVE",
    }


def test_risk_turnover_cap_clips_per_symbol_delta_without_redistribution() -> None:
    previous = WeightPanelValue(
        rows=(
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="AAA", weight="0.1"),
            WeightPoint(timestamp="2026-05-16T09:00:00Z", symbol="BBB", weight="-0.1"),
        )
    )

    result = evaluate_risk_token("risk.turnover_cap", _weights(), previous, max_turnover="0.2")

    assert result.weights is not None
    assert [(row.symbol, row.weight) for row in result.weights.rows] == [("AAA", "0.3"), ("BBB", "-0.25")]
    assert result.trace["clipped_count"] == 1
    assert result.trace["redistribution"] == "none"


def test_optimizer_profile_gate_remains_experimental_without_execution_path() -> None:
    optimizer = _spec_by_name()["optimizer.mean_variance"]

    assert validate_token_maturity_for_profile(optimizer, profile="research")[0].severity == "warning"
    assert validate_token_maturity_for_profile(optimizer, profile="paper")[0].severity == "warning"
    assert validate_token_maturity_for_profile(optimizer, profile="pretrade")[0].severity == "error"
    assert (
        validate_token_maturity_for_profile(optimizer, profile="production_guarded")[0].severity
        == "error"
    )
    assert optimizer.tests == [{"kind": "metadata_only", "deterministic": False}]
