from __future__ import annotations

import pytest

from qst.canonical_json import stable_json_bytes
from qst.state import StateExecutionTrace, StatePolicy, StateTraceEvent


def test_state_policy_defaults_are_stable_and_canonical() -> None:
    policy = StatePolicy()

    assert policy.warmup_policy == "emit_null"
    assert policy.reset_policy == "never"
    assert policy.missing_event_policy == "error"
    assert stable_json_bytes(policy.model_dump(mode="json"))


def test_state_policy_metadata_must_be_canonical_json() -> None:
    with pytest.raises(ValueError):
        StatePolicy(metadata={"bad": float("nan")})


def test_state_trace_payloads_are_canonical_json() -> None:
    event = StateTraceEvent(
        index=0,
        input=1,
        output=None,
        state_before=[],
        state_after=[1],
        policy_decision="ready|input_valid",
    )
    trace = StateExecutionTrace(token_id="core.state.delay", events=[event])

    assert stable_json_bytes(trace.model_dump(mode="json"))

    with pytest.raises(ValueError):
        StateTraceEvent(
            index=0,
            input=float("nan"),
            output=None,
            state_before=[],
            state_after=[],
            policy_decision="bad",
        )
