from __future__ import annotations

import pytest

from qst.canonical_json import stable_json_bytes
from qst.state import (
    FSMDefinition,
    FSMExecutionTrace,
    FSMTraceEvent,
    FSMTransition,
    StatePolicy,
    replay_fsm_trace,
    state_fsm,
)


def definition(*, failure_policy: str = "raise") -> FSMDefinition:
    states = ["closed", "open"]
    if failure_policy == "transition_to_unknown":
        states.append("unknown")
    return FSMDefinition(
        states=states,
        events=["close", "open", "tick"],
        initial_state="closed",
        failure_policy=failure_policy,  # type: ignore[arg-type]
        transitions=[
            FSMTransition(from_state="closed", event="open", to_state="open"),
            FSMTransition(from_state="open", event="close", to_state="closed"),
            FSMTransition(from_state="open", event="tick", to_state="open"),
        ],
    )


def codes(result: object) -> list[str]:
    return [diagnostic.code for diagnostic in result.result.diagnostics]  # type: ignore[attr-defined]


def severities(result: object) -> list[str]:
    return [diagnostic.severity for diagnostic in result.result.diagnostics]  # type: ignore[attr-defined]


def test_fsm_definition_canonicalizes_closed_sets_and_transitions() -> None:
    fsm = FSMDefinition(
        states=["open", "closed"],
        events=["tick", "open"],
        initial_state="closed",
        transitions=[
            {"from_state": "open", "event": "tick", "to_state": "open"},
            {"from_state": "closed", "event": "open", "to_state": "open"},
        ],
    )

    assert fsm.states == ("closed", "open")
    assert fsm.events == ("open", "tick")
    assert [(item.from_state, item.event, item.to_state) for item in fsm.transitions] == [
        ("closed", "open", "open"),
        ("open", "tick", "open"),
    ]
    assert stable_json_bytes(fsm.model_dump(mode="json"))


def test_invalid_transition_rejected_deterministically() -> None:
    with pytest.raises(ValueError, match="from_state"):
        FSMDefinition(
            states=["closed", "open"],
            events=["open"],
            initial_state="closed",
            transitions=[
                FSMTransition(from_state="missing", event="open", to_state="open"),
            ],
        )

    with pytest.raises(ValueError, match="Duplicate transition"):
        FSMDefinition(
            states=["closed", "open"],
            events=["open"],
            initial_state="closed",
            transitions=[
                FSMTransition(from_state="closed", event="open", to_state="open"),
                FSMTransition(from_state="closed", event="open", to_state="closed"),
            ],
        )


def test_transition_to_unknown_requires_closed_unknown_state() -> None:
    with pytest.raises(ValueError, match="unknown_state"):
        FSMDefinition(
            states=["closed", "open"],
            events=["open"],
            initial_state="closed",
            failure_policy="transition_to_unknown",
            transitions=[FSMTransition(from_state="closed", event="open", to_state="open")],
        )


@pytest.mark.parametrize(
    ("failure_policy", "expected_state", "expected_ok", "expected_severity"),
    [
        ("stay", "open", True, "warning"),
        ("transition_to_unknown", "unknown", True, "warning"),
        ("raise", "open", False, "error"),
    ],
)
def test_unknown_event_follows_failure_policy(
    failure_policy: str,
    expected_state: str,
    expected_ok: bool,
    expected_severity: str,
) -> None:
    result = state_fsm(["open", "mystery"], definition(failure_policy=failure_policy))

    assert result.outputs == ["open", expected_state]
    assert result.result.ok is expected_ok
    assert codes(result) == ["QST_V2_FSM_UNKNOWN_EVENT"]
    assert severities(result) == [expected_severity]
    assert result.trace.events[1].failure_policy_applied == failure_policy
    assert result.trace.events[1].diagnostic_code == "QST_V2_FSM_UNKNOWN_EVENT"


def test_missing_transition_follows_failure_policy() -> None:
    result = state_fsm(["tick"], definition(failure_policy="stay"))

    assert result.outputs == ["closed"]
    assert result.result.ok
    assert codes(result) == ["QST_V2_FSM_TRANSITION_MISSING"]
    assert result.trace.events[0].transition_found is False
    assert result.trace.events[0].failure_policy_applied == "stay"


def test_reset_happens_before_current_event_and_is_traced() -> None:
    result = state_fsm(
        ["open", "close"],
        definition(),
        policy=StatePolicy(reset_policy="on_event"),
        reset_events=[False, True],
    )

    assert result.outputs == ["open", "closed"]
    assert result.trace.events[1].state_before == "open"
    assert result.trace.events[1].state_after == "closed"
    assert result.trace.events[1].reset is True
    assert result.trace.events[1].transition_found is False
    assert codes(result) == ["QST_V2_FSM_TRANSITION_MISSING"]


def test_trace_records_every_event_including_failures() -> None:
    result = state_fsm(["open", "close", "mystery"], definition(failure_policy="stay"))

    assert len(result.trace.events) == 3
    assert [event.state_after for event in result.trace.events] == ["open", "closed", "closed"]
    assert result.trace.events[2].diagnostic_code == "QST_V2_FSM_UNKNOWN_EVENT"
    assert stable_json_bytes(result.trace.model_dump(mode="json"))


def test_replay_helper_reproduces_trace_state_sequence() -> None:
    fsm = definition(failure_policy="stay")
    result = state_fsm(["open", "close", "mystery"], fsm)

    assert replay_fsm_trace(["open", "close", "mystery"], fsm, result.trace).ok

    tampered = FSMExecutionTrace(
        events=[
            *result.trace.events[:-1],
            FSMTraceEvent(
                index=2,
                event="mystery",
                state_before="closed",
                state_after="open",
                diagnostic_code="QST_V2_FSM_UNKNOWN_EVENT",
                failure_policy_applied="stay",
            ),
        ]
    )

    replay = replay_fsm_trace(["open", "close", "mystery"], fsm, tampered)
    assert not replay.ok
    assert [diagnostic.code for diagnostic in replay.diagnostics] == [
        "QST_V2_FSM_REPLAY_MISMATCH"
    ]
