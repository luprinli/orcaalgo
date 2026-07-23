"""Deterministic reference semantics for Token System v2 WP6a state tokens."""

from __future__ import annotations

from collections.abc import Sequence
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes
from qst.state.policy import StatePolicy, default_state_policy
from qst.state.reducers import ReducerRegistry, default_reducer_registry
from qst.state.trace import StateExecutionTrace, StateTraceEvent
from qst.validation import Diagnostic, ValidationResult

EdgeMode = Literal["rising", "falling", "change"]


class StateExecutionResult(BaseModel):
    """Output, trace, and diagnostics for one state helper invocation."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    outputs: list[Any] = Field(default_factory=list)
    trace: StateExecutionTrace
    result: ValidationResult = Field(default_factory=ValidationResult)

    @field_validator("outputs")
    @classmethod
    def _outputs_are_json(cls, value: list[Any]) -> list[Any]:
        _ensure_json(value, field_name="state outputs")
        return value


def state_delay(
    values: Sequence[Any],
    *,
    steps: int = 1,
    initial: Any | None = None,
    policy: StatePolicy | None = None,
    reset_events: Sequence[bool] | None = None,
) -> StateExecutionResult:
    """Delay a series by ``steps`` valid inputs with deterministic tracing."""

    if steps < 1:
        raise ValueError("state.delay steps must be >= 1")
    active_policy = policy or default_state_policy()
    resets = _normalize_resets(values, reset_events)
    diagnostics: list[Diagnostic] = []
    outputs: list[Any] = []
    events: list[StateTraceEvent] = []
    buffer: list[Any] = []

    for index, value in enumerate(values):
        state_before = list(buffer)
        reset = _should_reset(active_policy, resets[index])
        if reset:
            buffer.clear()
        output, warmup_decision = _delay_output(buffer, steps, initial, active_policy, index)
        missing_decision = _handle_missing_input(value, active_policy, index, diagnostics)
        if missing_decision.update_state:
            buffer.append(value)
        policy_decision = _join_decisions(reset, warmup_decision, missing_decision.decision)
        outputs.append(output if missing_decision.output_override is _NO_OVERRIDE else missing_decision.output_override)
        events.append(
            StateTraceEvent(
                index=index,
                input=value,
                output=outputs[-1],
                state_before=state_before,
                state_after=list(buffer),
                reset=reset,
                policy_decision=policy_decision,
            )
        )

    return StateExecutionResult(
        outputs=outputs,
        trace=_trace("core.state.delay", events),
        result=ValidationResult(diagnostics=diagnostics),
    )


def state_accumulate(
    values: Sequence[Any],
    *,
    reducer: str,
    initial: Any | None = None,
    policy: StatePolicy | None = None,
    reset_events: Sequence[bool] | None = None,
    registry: ReducerRegistry | None = None,
) -> StateExecutionResult:
    """Accumulate values using a registered reducer only."""

    active_registry = registry or default_reducer_registry()
    reducer_check = active_registry.validate_name(reducer)
    if not reducer_check.ok:
        return StateExecutionResult(
            outputs=[],
            trace=_trace("core.state.accumulate", []),
            result=reducer_check,
        )

    active_policy = policy or default_state_policy()
    resets = _normalize_resets(values, reset_events)
    reducer_fn = active_registry.get(reducer)
    diagnostics: list[Diagnostic] = []
    outputs: list[Any] = []
    events: list[StateTraceEvent] = []
    state = initial

    for index, value in enumerate(values):
        state_before = state
        reset = _should_reset(active_policy, resets[index])
        if reset:
            state = initial
        missing_decision = _handle_missing_input(value, active_policy, index, diagnostics)
        if missing_decision.update_state:
            state = reducer_fn(state, value)
        output = state if missing_decision.output_override is _NO_OVERRIDE else missing_decision.output_override
        outputs.append(output)
        events.append(
            StateTraceEvent(
                index=index,
                input=value,
                output=output,
                state_before=state_before,
                state_after=state,
                reset=reset,
                policy_decision=_join_decisions(reset, "accumulate", missing_decision.decision),
            )
        )

    return StateExecutionResult(
        outputs=outputs,
        trace=_trace("core.state.accumulate", events),
        result=ValidationResult(diagnostics=diagnostics),
    )


def state_edge_detect(
    values: Sequence[Any],
    *,
    mode: EdgeMode = "rising",
    policy: StatePolicy | None = None,
) -> StateExecutionResult:
    """Detect boolean edges deterministically."""

    active_policy = policy or default_state_policy()
    diagnostics: list[Diagnostic] = []
    outputs: list[Any] = []
    events: list[StateTraceEvent] = []
    previous: bool | None = None

    for index, value in enumerate(values):
        state_before = previous
        missing_decision = _handle_missing_input(value, active_policy, index, diagnostics)
        output: bool | None
        if missing_decision.update_state:
            current = bool(value)
            output = _edge(previous, current, mode)
            previous = current
        else:
            output = False if missing_decision.output_override is _NO_OVERRIDE else missing_decision.output_override
        outputs.append(output)
        events.append(
            StateTraceEvent(
                index=index,
                input=value,
                output=output,
                state_before=state_before,
                state_after=previous,
                reset=False,
                policy_decision=_join_decisions(False, f"edge_{mode}", missing_decision.decision),
            )
        )

    return StateExecutionResult(
        outputs=outputs,
        trace=_trace("core.state.edge_detect", events),
        result=ValidationResult(diagnostics=diagnostics),
    )


class _NoOverride:
    pass


_NO_OVERRIDE = _NoOverride()


class _MissingDecision(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True, arbitrary_types_allowed=True)

    decision: str
    update_state: bool
    output_override: Any = _NO_OVERRIDE


def _delay_output(
    buffer: list[Any],
    steps: int,
    initial: Any | None,
    policy: StatePolicy,
    index: int,
) -> tuple[Any, str]:
    if len(buffer) >= steps:
        return buffer[-steps], "ready"
    if policy.warmup_policy == "emit_initial":
        return initial, "warmup_emit_initial"
    if policy.warmup_policy == "error":
        return None, f"warmup_error_{index}"
    return None, "warmup_emit_null"


def _handle_missing_input(
    value: Any,
    policy: StatePolicy,
    index: int,
    diagnostics: list[Diagnostic],
) -> _MissingDecision:
    if value is not None:
        return _MissingDecision(decision="input_valid", update_state=True)
    if policy.missing_event_policy == "skip":
        return _MissingDecision(decision="missing_skip", update_state=False)
    if policy.missing_event_policy == "emit_null":
        return _MissingDecision(decision="missing_emit_null", update_state=False, output_override=None)
    diagnostics.append(
        Diagnostic(
            code="QST_V2_STATE_MISSING_EVENT",
            severity="error",
            phase="runtime",
            message=f"Missing state input at index {index}.",
            remediation="Set missing_event_policy to skip or emit_null if missing events are expected.",
        )
    )
    return _MissingDecision(decision="missing_error", update_state=False, output_override=None)


def _normalize_resets(values: Sequence[Any], reset_events: Sequence[bool] | None) -> list[bool]:
    if reset_events is None:
        return [False] * len(values)
    if len(reset_events) != len(values):
        raise ValueError("reset_events length must match values length")
    return [bool(value) for value in reset_events]


def _should_reset(policy: StatePolicy, reset_event: bool) -> bool:
    return policy.reset_policy == "on_event" and reset_event


def _join_decisions(reset: bool, primary: str, missing: str) -> str:
    parts = []
    if reset:
        parts.append("reset")
    parts.extend([primary, missing])
    return "|".join(parts)


def _edge(previous: bool | None, current: bool, mode: EdgeMode) -> bool:
    if previous is None:
        return False
    if mode == "rising":
        return not previous and current
    if mode == "falling":
        return previous and not current
    return previous != current


def _trace(token_id: str, events: list[StateTraceEvent]) -> StateExecutionTrace:
    return StateExecutionTrace.model_validate({"token_id": token_id, "events": events})


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
