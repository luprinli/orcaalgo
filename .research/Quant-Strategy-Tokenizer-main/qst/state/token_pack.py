"""Core TokenPack metadata for Token System v2 WP6a state tokens."""

from __future__ import annotations

from typing import Any

from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy
from qst.ports import InputSpec, OutputSpec
from qst.state.policy import default_state_policy
from qst.tokens import (
    TokenPackDependency,
    TokenPackManifestV2,
    TokenRiskSpec,
    TokenSpecV2,
    token_surface,
)
from qst.types import parse_type_spec

STATE_BASIC_PACK_ID = "qst-tokenpack-state-basic"
STATE_BASIC_PACK_VERSION = "0.1.0"
STATE_FSM_PACK_ID = "qst-tokenpack-state-fsm"
STATE_FSM_PACK_VERSION = "0.1.0"


def state_basic_token_pack_v2() -> TokenPackManifestV2:
    """Return the accepted WP6a core TokenPack metadata."""

    return TokenPackManifestV2(
        pack_id=STATE_BASIC_PACK_ID,
        version=STATE_BASIC_PACK_VERSION,
        namespaces=("core",),
        tokens=(
            _state_token_spec(
                name="state.delay",
                params_schema={
                    "type": "object",
                    "additionalProperties": False,
                    "properties": {
                        "steps": {"type": "integer", "minimum": 1, "default": 1},
                        "initial": {},
                    },
                },
                outputs={"y": _output("TimeSeries[object]")},
            ),
            _state_token_spec(
                name="state.accumulate",
                params_schema={
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["reducer"],
                    "properties": {
                        "reducer": {"enum": ["count", "last", "max", "min", "sum"]},
                        "initial": {},
                    },
                },
                outputs={"y": _output("TimeSeries[object]")},
            ),
            _state_token_spec(
                name="state.edge_detect",
                params_schema={
                    "type": "object",
                    "additionalProperties": False,
                    "properties": {
                        "mode": {"enum": ["rising", "falling", "change"], "default": "rising"},
                    },
                },
                inputs={"x": _input("TimeSeries[bool]")},
                outputs={"edge": _output("TimeSeries[bool]")},
            ),
        ),
        origin_tier="core",
    )


def state_fsm_token_pack_v2() -> TokenPackManifestV2:
    """Return the accepted WP6b core TokenPack metadata for state.fsm."""

    return TokenPackManifestV2(
        pack_id=STATE_FSM_PACK_ID,
        version=STATE_FSM_PACK_VERSION,
        namespaces=("core",),
        tokens=(
            _state_token_spec(
                name="state.fsm",
                params_schema={
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["states", "events", "initial_state", "transitions"],
                    "properties": {
                        "states": {
                            "type": "array",
                            "uniqueItems": True,
                        },
                        "events": {
                            "type": "array",
                            "uniqueItems": True,
                        },
                        "initial_state": {"type": "string", "minLength": 1},
                        "transitions": {"type": "array"},
                        "failure_policy": {
                            "enum": ["stay", "transition_to_unknown", "raise"],
                            "default": "raise",
                        },
                        "unknown_state": {"type": "string", "default": "unknown"},
                    },
                },
                inputs={"event": _input("EventStream[string]")},
                outputs={"state": _output("State[string]")},
            ),
        ),
        dependencies=(
            TokenPackDependency(
                pack_id=STATE_BASIC_PACK_ID,
                version_constraint=f">={STATE_BASIC_PACK_VERSION}",
            ),
        ),
        origin_tier="core",
    )


def _state_token_spec(
    *,
    name: str,
    params_schema: dict[str, object],
    outputs: dict[str, OutputSpec],
    inputs: dict[str, InputSpec] | None = None,
) -> TokenSpecV2:
    state_metadata: dict[str, Any] = {
        "stateful": True,
        "state_policy": default_state_policy().model_dump(mode="json"),
        "wp": "WP6a",
    }
    if name == "state.fsm":
        state_metadata = {
            "stateful": True,
            "closed_state_set": True,
            "closed_event_set": True,
            "failure_policy": ["stay", "transition_to_unknown", "raise"],
            "state_policy": default_state_policy().model_dump(mode="json"),
            "wp": "WP6b",
        }
    return TokenSpecV2(
        token_id=f"core.{name}",
        token_ref=TokenRefV04(
            namespace="core",
            name=name,
            version=1,
            behavior_version=1,
        ),
        version=1,
        behavior_version=1,
        origin_tier="core",
        inputs=inputs or {"x": _input("TimeSeries[object]")},
        outputs=outputs,
        params_schema=params_schema,
        purity="contextual_read",
        state=state_metadata,
        numeric_policy=NumericPolicy(
            representation="object",
            deterministic_level="semantic",
            reduction_order="fixed_input_order",
            nan_policy="propagate",
            inf_policy="reject",
        ),
        risk=TokenRiskSpec(risk_level="medium"),
        surface=token_surface(
            family="state",
            category="fsm" if name == "state.fsm" else "basic_state",
            layer="primitive",
            maturity="accepted",
            execution_support="reference_helper",
            temporal="state evolves in input order; reset before current event",
            numeric="declared_by_numeric_policy",
            missing_data="state policy controls missing input behavior",
            failure_mode="validation_result_diagnostics",
            state="deterministic reference state trace",
            stateful=True,
            examples=(f"core.{name}/v1/bv1",),
            common_mistakes=("state.fsm failure_policy='raise' emits diagnostics, not Python exceptions.",),
        ),
        tests=[
            {
                "kind": "reference_helper",
                "deterministic": True,
            }
        ],
    )


def _input(type_spec: str) -> InputSpec:
    return InputSpec(type=parse_type_spec(type_spec))


def _output(type_spec: str) -> OutputSpec:
    return OutputSpec(type=parse_type_spec(type_spec))
