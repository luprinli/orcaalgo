"""Closed-set FSM reference semantics for Token System v2 WP6b."""

from __future__ import annotations

from collections.abc import Sequence
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.canonical_json import stable_json_bytes
from qst.state.policy import StatePolicy, default_state_policy
from qst.validation import Diagnostic, Severity, ValidationResult

STATE_FSM_SCHEMA_VERSION: Literal["qst-state-fsm/0.4"] = "qst-state-fsm/0.4"
FSMFailurePolicy = Literal["stay", "transition_to_unknown", "raise"]

UNKNOWN_EVENT_CODE = "QST_V2_FSM_UNKNOWN_EVENT"
MISSING_TRANSITION_CODE = "QST_V2_FSM_TRANSITION_MISSING"
REPLAY_MISMATCH_CODE = "QST_V2_FSM_REPLAY_MISMATCH"


class FSMTransition(BaseModel):
    """One closed-set FSM transition."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    from_state: str = Field(min_length=1)
    event: str = Field(min_length=1)
    to_state: str = Field(min_length=1)


class FSMDefinition(BaseModel):
    """Closed-set FSM definition for the v0.4 state.fsm helper."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-state-fsm/0.4"] = STATE_FSM_SCHEMA_VERSION
    states: tuple[str, ...]
    events: tuple[str, ...]
    initial_state: str = Field(min_length=1)
    transitions: tuple[FSMTransition, ...] = Field(default_factory=tuple)
    failure_policy: FSMFailurePolicy = "raise"
    unknown_state: str = "unknown"
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("states", "events", mode="before")
    @classmethod
    def _closed_set_is_sorted_unique(cls, value: Any) -> tuple[str, ...]:
        if not isinstance(value, Sequence) or isinstance(value, str):
            raise ValueError("FSM closed sets must be sequences of strings")
        values = tuple(str(item) for item in value)
        if any(not item for item in values):
            raise ValueError("FSM closed sets cannot contain empty strings")
        if len(set(values)) != len(values):
            raise ValueError("FSM closed sets must not contain duplicates")
        return tuple(sorted(values))

    @field_validator("transitions", mode="before")
    @classmethod
    def _transitions_are_sorted(cls, value: Any) -> tuple[Any, ...]:
        transitions = tuple(value or ())
        return tuple(sorted(transitions, key=_transition_sort_key))

    @field_validator("metadata")
    @classmethod
    def _metadata_is_json(cls, value: dict[str, Any]) -> dict[str, Any]:
        _ensure_json(value, field_name="FSMDefinition.metadata")
        return value

    @model_validator(mode="after")
    def _validate_closed_sets(self) -> FSMDefinition:
        states = set(self.states)
        events = set(self.events)
        if self.initial_state not in states:
            raise ValueError("FSM initial_state must be in states")
        if self.failure_policy == "transition_to_unknown" and self.unknown_state not in states:
            raise ValueError("transition_to_unknown requires unknown_state in states")

        transition_keys: set[tuple[str, str]] = set()
        for transition in self.transitions:
            if transition.from_state not in states:
                raise ValueError(f"Transition from_state {transition.from_state!r} is not closed")
            if transition.to_state not in states:
                raise ValueError(f"Transition to_state {transition.to_state!r} is not closed")
            if transition.event not in events:
                raise ValueError(f"Transition event {transition.event!r} is not closed")
            key = (transition.from_state, transition.event)
            if key in transition_keys:
                raise ValueError(f"Duplicate transition for state/event {key!r}")
            transition_keys.add(key)
        return self

    @property
    def transition_map(self) -> dict[tuple[str, str], str]:
        """Deterministic transition lookup map."""

        return {
            (transition.from_state, transition.event): transition.to_state
            for transition in self.transitions
        }


class FSMTraceEvent(BaseModel):
    """One FSM transition trace event."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    index: int = Field(ge=0)
    event: Any
    state_before: str
    state_after: str
    reset: bool = False
    transition_found: bool = False
    failure_policy_applied: FSMFailurePolicy | None = None
    diagnostic_code: str | None = None

    @field_validator("event")
    @classmethod
    def _event_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="FSM trace event")
        return value


class FSMExecutionTrace(BaseModel):
    """Trace for one state.fsm reference helper invocation."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    token_id: str = "core.state.fsm"
    events: list[FSMTraceEvent] = Field(default_factory=list)


class FSMExecutionResult(BaseModel):
    """FSM output states, trace, and diagnostics."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    outputs: list[str] = Field(default_factory=list)
    trace: FSMExecutionTrace
    result: ValidationResult = Field(default_factory=ValidationResult)


def state_fsm(
    events: Sequence[Any],
    definition: FSMDefinition,
    *,
    reset_events: Sequence[bool] | None = None,
    policy: StatePolicy | None = None,
) -> FSMExecutionResult:
    """Execute a closed-set FSM deterministically and emit a transition trace."""

    active_policy = policy or default_state_policy()
    resets = _normalize_resets(events, reset_events)
    transition_map = definition.transition_map
    closed_events = set(definition.events)
    diagnostics: list[Diagnostic] = []
    outputs: list[str] = []
    trace_events: list[FSMTraceEvent] = []
    state = definition.initial_state

    for index, event in enumerate(events):
        state_before = state
        reset = active_policy.reset_policy == "on_event" and resets[index]
        if reset:
            state = definition.initial_state

        transition_found = False
        failure_policy_applied: FSMFailurePolicy | None = None
        diagnostic_code: str | None = None
        if not isinstance(event, str) or event not in closed_events:
            state, failure_policy_applied, diagnostic_code = _apply_failure_policy(
                state=state,
                definition=definition,
                diagnostics=diagnostics,
                code=UNKNOWN_EVENT_CODE,
                message=f"Unknown FSM event at index {index}: {event!r}.",
            )
        else:
            next_state = transition_map.get((state, event))
            if next_state is None:
                state, failure_policy_applied, diagnostic_code = _apply_failure_policy(
                    state=state,
                    definition=definition,
                    diagnostics=diagnostics,
                    code=MISSING_TRANSITION_CODE,
                    message=f"Missing FSM transition from {state!r} on {event!r}.",
                )
            else:
                state = next_state
                transition_found = True

        outputs.append(state)
        trace_events.append(
            FSMTraceEvent(
                index=index,
                event=event,
                state_before=state_before,
                state_after=state,
                reset=reset,
                transition_found=transition_found,
                failure_policy_applied=failure_policy_applied,
                diagnostic_code=diagnostic_code,
            )
        )

    return FSMExecutionResult(
        outputs=outputs,
        trace=FSMExecutionTrace(events=trace_events),
        result=ValidationResult(diagnostics=diagnostics),
    )


def replay_fsm_trace(
    events: Sequence[Any],
    definition: FSMDefinition,
    trace: FSMExecutionTrace,
    *,
    reset_events: Sequence[bool] | None = None,
    policy: StatePolicy | None = None,
) -> ValidationResult:
    """Replay an FSM trace and confirm the state sequence is deterministic."""

    replayed = state_fsm(events, definition, reset_events=reset_events, policy=policy)
    expected = [event.state_after for event in trace.events]
    observed = [event.state_after for event in replayed.trace.events]
    if observed == expected:
        return ValidationResult()
    return ValidationResult(
        diagnostics=[
            Diagnostic(
                code=REPLAY_MISMATCH_CODE,
                severity="error",
                phase="runtime",
                message="FSM replay state_after sequence did not match the supplied trace.",
                remediation="Rebuild the trace from the same events and FSM definition.",
            )
        ]
    )


def _apply_failure_policy(
    *,
    state: str,
    definition: FSMDefinition,
    diagnostics: list[Diagnostic],
    code: str,
    message: str,
) -> tuple[str, FSMFailurePolicy, str]:
    severity: Severity = "error" if definition.failure_policy == "raise" else "warning"
    diagnostics.append(
        Diagnostic(
            code=code,
            severity=severity,
            phase="runtime",
            message=message,
            remediation="Declare the event/transition or choose an explicit FSM failure policy.",
        )
    )
    if definition.failure_policy == "transition_to_unknown":
        return definition.unknown_state, definition.failure_policy, code
    return state, definition.failure_policy, code


def _normalize_resets(events: Sequence[Any], reset_events: Sequence[bool] | None) -> list[bool]:
    if reset_events is None:
        return [False] * len(events)
    if len(reset_events) != len(events):
        raise ValueError("reset_events length must match events length")
    return [bool(value) for value in reset_events]


def _transition_sort_key(value: Any) -> tuple[str, str, str]:
    if isinstance(value, FSMTransition):
        return (value.from_state, value.event, value.to_state)
    if isinstance(value, dict):
        return (
            str(value.get("from_state", "")),
            str(value.get("event", "")),
            str(value.get("to_state", "")),
        )
    return ("", "", "")


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
