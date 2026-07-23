"""state-reference state-heavy reference cases for Token System v2 WP6c."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes
from qst.hash import expected_artifact_hash_v2
from qst.ir import StrategyIRV04
from qst.state.fsm import (
    FSMDefinition,
    FSMTransition,
    replay_fsm_trace,
    state_fsm,
)
from qst.state.reference import StateExecutionResult, state_accumulate
from qst.validation import Diagnostic, ValidationResult

STATE_PV_A_FIXTURE_VERSION: Literal["qst-v04-state-fixture/0.1"] = (
    "qst-v04-state-fixture/0.1"
)
STATE_PV_A_TRACE_ARTIFACT_VERSION: Literal["qst-v04-state-validation-trace/0.1"] = (
    "qst-v04-state-validation-trace/0.1"
)
STATE_PV_A_DIAGNOSTICS_ARTIFACT_VERSION: Literal[
    "qst-v04-state-expected-diagnostics/0.1"
] = "qst-v04-state-expected-diagnostics/0.1"

StatePVACase = Literal[
    "state_cooldown",
    "state_market_freeze",
    "state_circuit_breaker",
    "state_observe_period",
    "state_slot_budget_minimal",
]


class StatePVAFixture(BaseModel):
    """Fixture payload for one state-reference state reference case."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-state-fixture/0.1"] = STATE_PV_A_FIXTURE_VERSION
    case: StatePVACase
    events: list[Any] = Field(default_factory=list)
    values: list[Any] = Field(default_factory=list)
    params: dict[str, Any] = Field(default_factory=dict)

    @field_validator("events", "values", "params")
    @classmethod
    def _payload_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="state-reference fixture payload")
        return value


class StatePVAResult(BaseModel):
    """Deterministic state-reference result before artifact wrapping."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    case: StatePVACase
    outputs: dict[str, Any] = Field(default_factory=dict)
    diagnostics: list[Diagnostic] = Field(default_factory=list)
    state_traces: dict[str, Any] = Field(default_factory=dict)
    replay_checks: dict[str, bool] = Field(default_factory=dict)

    @field_validator("outputs", "state_traces", "replay_checks")
    @classmethod
    def _payload_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="state-reference result payload")
        return value

    @property
    def validation_result(self) -> ValidationResult:
        """Validation result derived from diagnostics."""

        return ValidationResult(diagnostics=self.diagnostics)


class StatePVATraceArtifact(BaseModel):
    """Serializable trace artifact for one state-reference case."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-state-validation-trace/0.1"] = (
        STATE_PV_A_TRACE_ARTIFACT_VERSION
    )
    strategy: str
    case: StatePVACase
    outputs: dict[str, Any]
    diagnostics: list[dict[str, Any]]
    state_traces: dict[str, Any]
    replay_checks: dict[str, bool]
    expected_artifact_hash: str


def load_state_pv_a_fixture(path: str | Path) -> StatePVAFixture:
    """Load a state-reference fixture from JSON."""

    with Path(path).open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    return StatePVAFixture.model_validate(loaded)


def run_state_pv_a_case(ir: StrategyIRV04, fixture: StatePVAFixture) -> StatePVAResult:
    """Run one deterministic state-reference reference case."""

    metadata_case = ir.metadata.get("reference_case")
    if metadata_case != fixture.case:
        return _error_result(
            fixture.case,
            "QST_V2_STATE_PVA_CASE_MISMATCH",
            f"Strategy {ir.strategy.id!r} declares case {metadata_case!r}, fixture is {fixture.case!r}.",
        )

    if fixture.case == "state_cooldown":
        return _state_cooldown(fixture)
    if fixture.case == "state_market_freeze":
        return _state_market_freeze(fixture)
    if fixture.case == "state_circuit_breaker":
        return _state_circuit_breaker(fixture)
    if fixture.case == "state_observe_period":
        return _state_observe_period(fixture)
    if fixture.case == "state_slot_budget_minimal":
        return _state_slot_budget_minimal(fixture)

    return _error_result(
        fixture.case,
        "QST_V2_STATE_PVA_CASE_UNKNOWN",
        f"Unsupported state-reference state case {fixture.case!r}.",
    )


def trace_state_pv_a_v04(ir: StrategyIRV04, fixture: StatePVAFixture) -> StatePVATraceArtifact:
    """Return a hash-bearing state-reference trace artifact."""

    result = run_state_pv_a_case(ir, fixture)
    material = {
        "artifact_version": STATE_PV_A_TRACE_ARTIFACT_VERSION,
        "strategy": ir.strategy.id,
        "case": result.case,
        "outputs": result.outputs,
        "diagnostics": [
            diagnostic.model_dump(mode="json", exclude_none=True)
            for diagnostic in result.diagnostics
        ],
        "state_traces": result.state_traces,
        "replay_checks": result.replay_checks,
    }
    return StatePVATraceArtifact.model_validate(
        {
            **material,
            "expected_artifact_hash": expected_artifact_hash_v2(material),
        }
    )


def diagnostics_state_pv_a_v04(ir: StrategyIRV04, fixture: StatePVAFixture) -> dict[str, Any]:
    """Return a hash-bearing state-reference diagnostics artifact."""

    result = run_state_pv_a_case(ir, fixture)
    material = {
        "artifact_version": STATE_PV_A_DIAGNOSTICS_ARTIFACT_VERSION,
        "strategy": ir.strategy.id,
        "case": result.case,
        "diagnostics": [
            diagnostic.model_dump(mode="json", exclude_none=True)
            for diagnostic in result.diagnostics
        ],
    }
    return {
        **material,
        "expected_artifact_hash": expected_artifact_hash_v2(material),
    }


def _state_cooldown(fixture: StatePVAFixture) -> StatePVAResult:
    definition = FSMDefinition(
        states=("cooldown", "ready"),
        events=("cooldown_expired", "fill", "signal"),
        initial_state="ready",
        failure_policy="raise",
        transitions=(
            _transition("ready", "signal", "ready"),
            _transition("ready", "fill", "cooldown"),
            _transition("ready", "cooldown_expired", "ready"),
            _transition("cooldown", "signal", "cooldown"),
            _transition("cooldown", "fill", "cooldown"),
            _transition("cooldown", "cooldown_expired", "ready"),
        ),
    )
    fsm = state_fsm(fixture.events, definition)
    decisions = [
        "blocked" if event == "signal" and state == "cooldown" else "active"
        for event, state in zip(fixture.events, fsm.outputs, strict=True)
    ]
    return _fsm_result(fixture.case, fsm, definition, fixture.events, {"decisions": decisions})


def _state_market_freeze(fixture: StatePVAFixture) -> StatePVAResult:
    definition = FSMDefinition(
        states=("active", "frozen"),
        events=("freeze_off", "freeze_on", "signal"),
        initial_state="active",
        failure_policy="raise",
        transitions=(
            _transition("active", "signal", "active"),
            _transition("active", "freeze_on", "frozen"),
            _transition("active", "freeze_off", "active"),
            _transition("frozen", "signal", "frozen"),
            _transition("frozen", "freeze_on", "frozen"),
            _transition("frozen", "freeze_off", "active"),
        ),
    )
    fsm = state_fsm(fixture.events, definition)
    decisions = [
        "blocked" if event == "signal" and state == "frozen" else "active"
        for event, state in zip(fixture.events, fsm.outputs, strict=True)
    ]
    return _fsm_result(fixture.case, fsm, definition, fixture.events, {"decisions": decisions})


def _state_circuit_breaker(fixture: StatePVAFixture) -> StatePVAResult:
    threshold = _int_param(fixture, "threshold", default=2)
    breaches = [1 if bool(value) else 0 for value in fixture.values]
    accumulated = state_accumulate(breaches, reducer="sum", initial=0)
    fsm_events = ["trip" if int(value) >= threshold else "ok" for value in accumulated.outputs]
    definition = FSMDefinition(
        states=("active", "tripped"),
        events=("ok", "trip"),
        initial_state="active",
        failure_policy="raise",
        transitions=(
            _transition("active", "ok", "active"),
            _transition("active", "trip", "tripped"),
            _transition("tripped", "ok", "tripped"),
            _transition("tripped", "trip", "tripped"),
        ),
    )
    fsm = state_fsm(fsm_events, definition)
    decisions = ["blocked" if state == "tripped" else "active" for state in fsm.outputs]
    return _combined_result(
        fixture.case,
        fsm,
        definition,
        fsm_events,
        accumulated,
        {"breach_count": accumulated.outputs, "decisions": decisions},
    )


def _state_observe_period(fixture: StatePVAFixture) -> StatePVAResult:
    min_observations = _int_param(fixture, "min_observations", default=3)
    observed = state_accumulate(fixture.values, reducer="count", initial=0)
    fsm_events = ["ready" if int(value) >= min_observations else "observe" for value in observed.outputs]
    definition = FSMDefinition(
        states=("active", "observing"),
        events=("observe", "ready"),
        initial_state="observing",
        failure_policy="raise",
        transitions=(
            _transition("observing", "observe", "observing"),
            _transition("observing", "ready", "active"),
            _transition("active", "observe", "active"),
            _transition("active", "ready", "active"),
        ),
    )
    fsm = state_fsm(fsm_events, definition)
    decisions = ["active" if state == "active" else "blocked" for state in fsm.outputs]
    return _combined_result(
        fixture.case,
        fsm,
        definition,
        fsm_events,
        observed,
        {"observed_count": observed.outputs, "decisions": decisions},
    )


def _state_slot_budget_minimal(fixture: StatePVAFixture) -> StatePVAResult:
    budget = _int_param(fixture, "budget", default=2)
    used = state_accumulate(fixture.values, reducer="sum", initial=0)
    decisions = ["active" if int(value) <= budget else "blocked" for value in used.outputs]
    return _state_result(
        fixture.case,
        used,
        {"slots_used": used.outputs, "decisions": decisions},
    )


def _fsm_result(
    case: StatePVACase,
    fsm: Any,
    definition: FSMDefinition,
    events: list[Any],
    outputs: dict[str, Any],
) -> StatePVAResult:
    diagnostics = list(fsm.result.diagnostics)
    replay = replay_fsm_trace(events, definition, fsm.trace)
    diagnostics.extend(replay.diagnostics)
    return StatePVAResult(
        case=case,
        outputs={"states": fsm.outputs, **outputs},
        diagnostics=diagnostics,
        state_traces={"fsm": fsm.trace.model_dump(mode="json")},
        replay_checks={"fsm": replay.ok},
    )


def _combined_result(
    case: StatePVACase,
    fsm: Any,
    definition: FSMDefinition,
    events: list[Any],
    state_result: StateExecutionResult,
    outputs: dict[str, Any],
) -> StatePVAResult:
    fsm_result = _fsm_result(case, fsm, definition, events, outputs)
    return StatePVAResult(
        case=case,
        outputs=fsm_result.outputs,
        diagnostics=[*state_result.result.diagnostics, *fsm_result.diagnostics],
        state_traces={
            "accumulate": state_result.trace.model_dump(mode="json"),
            **fsm_result.state_traces,
        },
        replay_checks=fsm_result.replay_checks,
    )


def _state_result(
    case: StatePVACase,
    state_result: StateExecutionResult,
    outputs: dict[str, Any],
) -> StatePVAResult:
    return StatePVAResult(
        case=case,
        outputs=outputs,
        diagnostics=state_result.result.diagnostics,
        state_traces={"accumulate": state_result.trace.model_dump(mode="json")},
        replay_checks={},
    )


def _error_result(case: StatePVACase, code: str, message: str) -> StatePVAResult:
    return StatePVAResult(
        case=case,
        diagnostics=[
            Diagnostic(
                code=code,
                severity="error",
                phase="runtime",
                message=message,
                remediation="Check the state-reference strategy metadata and fixture.",
            )
        ],
    )


def _int_param(fixture: StatePVAFixture, name: str, *, default: int) -> int:
    value = fixture.params.get(name, default)
    if not isinstance(value, int):
        raise ValueError(f"state-reference fixture param {name!r} must be an integer")
    return value


def _transition(from_state: str, event: str, to_state: str) -> FSMTransition:
    return FSMTransition(from_state=from_state, event=event, to_state=to_state)


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
