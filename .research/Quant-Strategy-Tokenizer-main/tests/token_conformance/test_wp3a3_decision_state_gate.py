from __future__ import annotations

import pytest

from qst.decision import DecisionCombineResult, DecisionV2
from qst.state import StateExecutionResult
from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_decision_token,
    evaluate_gate_token,
    evaluate_state_token,
)

WP3A3_TOKENS = {
    "decision.lift_bool",
    "decision.any_accept",
    "decision.majority",
    "decision.permissive_and",
    "decision.quorum",
    "decision.strict_and",
    "decision.unknown_propagating_and",
    "decision.weighted_vote",
    "gate.cooldown",
    "gate.market_freeze",
    "gate.circuit_breaker",
    "gate.observe_period",
    "gate.slot_budget",
    "state.accumulate",
    "state.delay",
    "state.edge_detect",
    "state.fsm",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _kinds(decisions: list[DecisionV2]) -> list[str]:
    return [decision.kind for decision in decisions]


def _decisions(kinds: list[str]) -> list[DecisionV2]:
    return [DecisionV2(kind=kind) for kind in kinds]  # type: ignore[arg-type]


def _assert_code(exc: pytest.ExceptionInfo[TokenReferenceError], code: str) -> None:
    assert exc.value.code == code


def test_wp3a3_tokens_have_surface_contracts() -> None:
    specs = _spec_by_name()

    assert WP3A3_TOKENS <= set(specs)
    for name in sorted(WP3A3_TOKENS):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.temporal
        assert spec.surface.contract.numeric
        assert spec.surface.contract.missing_data
        assert spec.surface.contract.failure_mode

    for gate_name in {
        "gate.cooldown",
        "gate.market_freeze",
        "gate.circuit_breaker",
        "gate.observe_period",
        "gate.slot_budget",
    }:
        assert specs[gate_name].surface.family == "gate"
        assert specs[gate_name].surface.capabilities.stateful is True

    assert specs["decision.strict_and"].surface.category == "fold_policy"
    assert specs["decision.any_accept"].surface.category == "monoid"
    assert specs["decision.majority"].surface.category == "aggregator"
    assert specs["state.fsm"].surface.capabilities.stateful is True


def test_decision_lift_bool_facade_truth_table() -> None:
    decisions = evaluate_decision_token(
        "decision.lift_bool",
        [
            ("2026-05-16T09:00:00Z", True),
            ("2026-05-16T09:01:00Z", False),
        ],
        accept_reason="signal_true",
        reject_reason="signal_false",
    )

    assert isinstance(decisions, list)
    assert _kinds(decisions) == ["accept", "reject"]
    assert decisions[0].reasons == ("signal_true",)
    assert decisions[1].reasons == ("signal_false",)

    with pytest.raises(TokenReferenceError) as invalid_bool:
        evaluate_decision_token("decision.lift_bool", [True, 1])
    _assert_code(invalid_bool, "QST_TOKEN_BOOL_TYPE_INVALID")


def test_decision_algebra_facade_delegates_to_existing_helpers() -> None:
    strict = evaluate_decision_token(
        "decision.strict_and",
        _decisions(["accept", "unknown"]),
    )
    permissive = evaluate_decision_token(
        "decision.permissive_and",
        _decisions(["accept", "unknown"]),
    )
    monoid = evaluate_decision_token(
        "decision.any_accept",
        _decisions(["reject", "unknown"]),
    )
    majority = evaluate_decision_token(
        "decision.majority",
        _decisions(["accept", "reject", "accept"]),
    )
    quorum = evaluate_decision_token(
        "decision.quorum",
        _decisions(["accept", "reject", "unknown"]),
        accept_quorum=1,
        min_known=2,
    )
    weighted = evaluate_decision_token(
        "decision.weighted_vote",
        _decisions(["accept", "reject"]),
        weights={"0": "2", "1": "1"},
    )

    for result in [strict, permissive, monoid, majority, quorum, weighted]:
        assert isinstance(result, DecisionCombineResult)
        assert result.decision is not None

    assert strict.decision.kind == "reject"
    assert permissive.decision.kind == "accept"
    assert monoid.decision.kind == "unknown"
    assert majority.decision.kind == "accept"
    assert quorum.decision.kind == "accept"
    assert weighted.decision.kind == "accept"


def test_state_facade_preserves_existing_trace_and_diagnostics() -> None:
    delay = evaluate_state_token("state.delay", [1, 2, 3], steps=2, initial=0)
    accumulate = evaluate_state_token("state.accumulate", [1, 2, 3], reducer="sum", initial=0)
    edge = evaluate_state_token("state.edge_detect", [False, True, True, False], mode="change")
    fsm = evaluate_state_token(
        "state.fsm",
        ["signal", "fill", "signal"],
        definition={
            "states": ["ready", "cooldown"],
            "events": ["signal", "fill"],
            "initial_state": "ready",
            "transitions": [
                {"from_state": "ready", "event": "signal", "to_state": "ready"},
                {"from_state": "ready", "event": "fill", "to_state": "cooldown"},
                {"from_state": "cooldown", "event": "signal", "to_state": "cooldown"},
                {"from_state": "cooldown", "event": "fill", "to_state": "cooldown"},
            ],
        },
    )

    assert isinstance(delay, StateExecutionResult)
    assert delay.outputs == [None, None, 1]
    assert delay.trace.token_id == "core.state.delay"
    assert accumulate.outputs == [1, 3, 6]
    assert edge.outputs == [False, True, False, True]
    assert fsm.outputs == ["ready", "cooldown", "cooldown"]
    assert fsm.result.diagnostics == []

    missing = evaluate_state_token("state.fsm", ["unknown"], definition={
        "states": ["ready"],
        "events": ["signal"],
        "initial_state": "ready",
        "transitions": [],
    })
    assert missing.result.diagnostics[0].code == "QST_V2_FSM_UNKNOWN_EVENT"


def test_gate_facade_outputs_decisions_not_diagnostic_kinds() -> None:
    cooldown = evaluate_gate_token(
        "gate.cooldown",
        ["signal", "fill", "signal", "cooldown_expired", "signal"],
    )
    market_freeze = evaluate_gate_token(
        "gate.market_freeze",
        ["signal", "freeze_on", "signal", "freeze_off", "signal"],
    )
    circuit_breaker = evaluate_gate_token(
        "gate.circuit_breaker",
        [0, 1, 1, 0, 1],
        threshold=2,
    )
    observe_period = evaluate_gate_token(
        "gate.observe_period",
        [1, 1, 1, 1],
        window=3,
    )
    slot_budget = evaluate_gate_token(
        "gate.slot_budget",
        [1, 1, 1, 1],
        slot_budget=2,
    )

    assert _kinds(cooldown) == ["accept", "accept", "block", "accept", "accept"]
    assert _kinds(market_freeze) == ["accept", "accept", "block", "accept", "accept"]
    assert _kinds(circuit_breaker) == ["accept", "accept", "block", "block", "block"]
    assert _kinds(observe_period) == ["block", "block", "accept", "accept"]
    assert _kinds(slot_budget) == ["accept", "accept", "block", "block"]

    for decision in [*cooldown, *market_freeze, *circuit_breaker, *observe_period, *slot_budget]:
        assert isinstance(decision, DecisionV2)
        assert decision.kind in {"accept", "block"}

    with pytest.raises(TokenReferenceError) as invalid_gate_input:
        evaluate_gate_token("gate.slot_budget", [True], slot_budget=1)
    _assert_code(invalid_gate_input, "QST_TOKEN_GATE_INPUT_INVALID")
