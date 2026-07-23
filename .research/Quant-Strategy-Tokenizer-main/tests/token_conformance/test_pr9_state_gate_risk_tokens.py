from __future__ import annotations

# ruff: noqa: I001

import pytest

from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_gate_token,
    evaluate_risk_token,
)
from qst.decision import DecisionV2
from qst.panel import PanelPoint, PanelValue, WeightPanelValue, WeightPoint

PR9_TOKENS = {
    "gate.volatility_regime",
    "gate.drawdown",
    "gate.time_window",
    "gate.rebalance",
    "gate.min_hold",
    "gate.max_hold",
    "risk.stop_loss_record",
    "risk.take_profit_record",
    "risk.trailing_stop_record",
    "risk.max_drawdown_record",
    "risk.volatility_target_record",
    "risk.exposure_cap_record",
    "risk.turnover_limit_record",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _kinds(decisions: tuple[object, ...] | list[object]) -> list[str]:
    return [DecisionV2.model_validate(decision).kind for decision in decisions]


def _weights() -> WeightPanelValue:
    return WeightPanelValue(
        rows=(
            WeightPoint(timestamp="t1", symbol="AAA", weight="0.5"),
            WeightPoint(timestamp="t1", symbol="BBB", weight="-0.25"),
        )
    )


def _previous_weights() -> WeightPanelValue:
    return WeightPanelValue(
        rows=(
            WeightPoint(timestamp="t1", symbol="AAA", weight="0.1"),
            WeightPoint(timestamp="t1", symbol="BBB", weight="-0.1"),
        )
    )


def test_pr9_tokens_have_accepted_reference_surface() -> None:
    specs = _spec_by_name()

    assert PR9_TOKENS <= set(specs)
    for name in sorted(PR9_TOKENS):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.scope == "reference_semantics"
        assert spec.surface.contract.failure_mode
        assert spec.surface.contract.numeric
        assert spec.surface.contract.temporal

    assert specs["gate.time_window"].surface.family == "gate"
    assert specs["risk.stop_loss_record"].risk.risk_level == "medium"
    assert specs["risk.turnover_limit_record"].surface.capabilities.panel_aware is True
    assert specs["risk.volatility_target_record"].surface.capabilities.solver_backed is False


def test_gate_records_return_decisions_without_runtime_execution() -> None:
    volatility = evaluate_gate_token("gate.volatility_regime", [0.2, 0.6], max_volatility=0.5)
    drawdown = evaluate_gate_token("gate.drawdown", [0.02, -0.2], max_drawdown=0.1)
    time_window = evaluate_gate_token(
        "gate.time_window",
        ["2026-05-16T09:30:00Z", "2026-05-16T16:00:00Z"],
        start_hhmm="09:30",
        end_hhmm="15:55",
    )
    rebalance = evaluate_gate_token("gate.rebalance", ["0.01", "0.08"], band="0.05")
    min_hold = evaluate_gate_token("gate.min_hold", [1, 3], bars=3)
    max_hold = evaluate_gate_token("gate.max_hold", [5, 20], bars=20)

    assert _kinds(volatility) == ["accept", "block"]
    assert _kinds(drawdown) == ["accept", "block"]
    assert _kinds(time_window) == ["accept", "block"]
    assert _kinds(rebalance) == ["block", "accept"]
    assert _kinds(min_hold) == ["block", "accept"]
    assert _kinds(max_hold) == ["block", "accept"]

    for decision in [*volatility, *drawdown, *time_window, *rebalance, *min_hold, *max_hold]:
        parsed = DecisionV2.model_validate(decision)
        assert parsed.kind in {"accept", "block"}
        assert "order" not in " ".join(parsed.reasons).lower()


def test_gate_numeric_records_reject_bool_and_nonfinite_material() -> None:
    with pytest.raises(TokenReferenceError) as bool_input:
        evaluate_gate_token("gate.volatility_regime", [True], max_volatility=0.5)
    assert bool_input.value.code == "QST_TOKEN_NUMERIC_TYPE_INVALID"

    with pytest.raises(TokenReferenceError) as nonfinite_input:
        evaluate_gate_token("gate.drawdown", [float("inf")], max_drawdown=0.1)
    assert nonfinite_input.value.code == "QST_TOKEN_NUMERIC_NONFINITE"


def test_stop_take_profit_trailing_and_drawdown_risk_records() -> None:
    stop = evaluate_risk_token(
        "risk.stop_loss_record",
        [100, 100, 100],
        [95, 91, 110],
        stop_loss_pct=0.08,
    )
    take = evaluate_risk_token(
        "risk.take_profit_record",
        [100, 100, 100],
        [105, 115, 99],
        take_profit_pct=0.1,
    )
    trailing = evaluate_risk_token("risk.trailing_stop_record", [100, 120, 108], trail_pct=0.1)
    drawdown = evaluate_risk_token("risk.max_drawdown_record", [100, 120, 90], max_drawdown=0.2)

    assert _kinds(stop.decisions) == ["accept", "block", "accept"]
    assert _kinds(take.decisions) == ["reject", "accept", "reject"]
    assert _kinds(trailing.decisions) == ["accept", "accept", "block"]
    assert _kinds(drawdown.decisions) == ["accept", "accept", "block"]
    assert trailing.trace["order_execution"] == "none"


def test_risk_record_aliases_delegate_to_existing_weight_and_cap_helpers() -> None:
    volatility = PanelValue(
        rows=(
            PanelPoint(timestamp="t1", symbol="AAA", value="0.5"),
            PanelPoint(timestamp="t1", symbol="BBB", value="0.25"),
        )
    )
    vol_target = evaluate_risk_token(
        "risk.volatility_target_record",
        _weights(),
        volatility,
        target_volatility="0.1",
    )
    exposure = evaluate_risk_token(
        "risk.exposure_cap_record",
        [DecisionV2(kind="accept"), DecisionV2(kind="accept")],
        ["0.5", "2"],
        max_abs_exposure="1",
    )
    turnover = evaluate_risk_token(
        "risk.turnover_limit_record",
        _weights(),
        _previous_weights(),
        max_turnover="0.2",
    )

    assert vol_target.weights is not None
    assert [(row.symbol, row.weight) for row in vol_target.weights.rows] == [
        ("AAA", "0.1"),
        ("BBB", "-0.1"),
    ]
    assert _kinds(exposure.decisions) == ["accept", "block"]
    assert turnover.weights is not None
    assert [(row.symbol, row.weight) for row in turnover.weights.rows] == [
        ("AAA", "0.3"),
        ("BBB", "-0.25"),
    ]


def test_risk_records_reject_bool_and_nonfinite_numeric_material() -> None:
    with pytest.raises(TokenReferenceError) as bool_input:
        evaluate_risk_token("risk.stop_loss_record", [100], [True], stop_loss_pct=0.1)
    assert bool_input.value.code == "QST_TOKEN_NUMERIC_TYPE_INVALID"

    with pytest.raises(TokenReferenceError) as nonfinite_input:
        evaluate_risk_token("risk.trailing_stop_record", [100, float("nan")], trail_pct=0.1)
    assert nonfinite_input.value.code == "QST_TOKEN_NUMERIC_NONFINITE"
