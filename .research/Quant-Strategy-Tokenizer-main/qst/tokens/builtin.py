"""Built-in public TokenPack vocabulary."""

from __future__ import annotations

from typing import Any, Literal

from qst.decision import decision_algebra_token_pack_v2
from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy
from qst.panel import panel_ops_token_pack_v2, panel_weights_token_pack_v2
from qst.ports import InputSpec, OutputSpec
from qst.state import state_basic_token_pack_v2, state_fsm_token_pack_v2
from qst.tokens.pack import TokenPackManifestV2
from qst.tokens.spec import TokenRiskSpec, TokenSpecV2
from qst.tokens.surface import (
    ExecutionSupport,
    SolverContractSpec,
    TokenFamily,
    TokenLayer,
    TokenMaturity,
    token_surface,
)
from qst.types import parse_type_spec

CORE_SURFACE_PACK_ID = "qst-tokenpack-core-surface"
CORE_SURFACE_PACK_VERSION = "0.1.0"


def builtin_token_packs() -> tuple[TokenPackManifestV2, ...]:
    """Return built-in TokenPacks in deterministic public vocabulary order."""

    packs = (
        core_surface_token_pack_v2(),
        decision_algebra_token_pack_v2(),
        state_basic_token_pack_v2(),
        state_fsm_token_pack_v2(),
        panel_ops_token_pack_v2(),
        panel_weights_token_pack_v2(),
    )
    return tuple(sorted(packs, key=lambda pack: pack.pack_id))


def core_surface_token_pack_v2() -> TokenPackManifestV2:
    """Return Stage 3A core surface TokenSpecs."""

    return TokenPackManifestV2(
        pack_id=CORE_SURFACE_PACK_ID,
        version=CORE_SURFACE_PACK_VERSION,
        namespaces=("core",),
        tokens=tuple(_spec(**item) for item in _TOKEN_DEFINITIONS),
        origin_tier="core",
    )


_Numeric = Literal["float64", "bool", "decimal", "object"]


def _spec(
    *,
    name: str,
    family: TokenFamily,
    category: str,
    inputs: dict[str, str],
    outputs: dict[str, str],
    params_schema: dict[str, object] | None = None,
    numeric: _Numeric = "float64",
    layer: TokenLayer = "derived",
    maturity: TokenMaturity = "accepted",
    execution_support: ExecutionSupport = "reference_helper",
    stateful: bool = False,
    panel_aware: bool = False,
    solver_backed: bool = False,
    external_code: bool = False,
    reserved_only: bool = False,
    risk_level: str = "low",
    temporal: str = "declared_by_ports",
    missing_data: str = "reject_or_declared_by_token",
    failure_mode: str = "diagnostic_error",
    usage_notes: tuple[str, ...] = (),
) -> TokenSpecV2:
    return TokenSpecV2(
        token_id=f"core.{name}",
        token_ref=TokenRefV04(namespace="core", name=name, version=1, behavior_version=1),
        version=1,
        behavior_version=1,
        origin_tier="core",
        inputs={key: InputSpec(type=parse_type_spec(value)) for key, value in inputs.items()},
        outputs={key: OutputSpec(type=parse_type_spec(value)) for key, value in outputs.items()},
        params_schema=params_schema or {"type": "object", "additionalProperties": True},
        purity="pure",
        state={"token_surface_stage": "3A", "category": category},
        numeric_policy=_numeric_policy(numeric),
        risk=TokenRiskSpec(risk_level=risk_level),  # type: ignore[arg-type]
        surface=token_surface(
            family=family,
            category=category,
            layer=layer,
            maturity=maturity,
            execution_support=execution_support,
            contract_scope=(
                "validation_only" if execution_support == "metadata_only" else "reference_semantics"
            ),
            temporal=temporal,
            numeric=(
                "semantic_float64"
                if numeric == "float64"
                else "declared_by_numeric_policy"
            ),
            missing_data=missing_data,
            failure_mode=failure_mode,
            solver=(
                SolverContractSpec(
                    solver_required=True,
                    deterministic_contract="missing_solver_determinism_contract",
                    bit_exact_claim=False,
                )
                if solver_backed
                else None
            ),
            state="stateful reference semantics" if stateful else None,
            panel="panel-aware token" if panel_aware else None,
            stateful=stateful,
            panel_aware=panel_aware,
            solver_backed=solver_backed,
            external_code=external_code,
            reserved_only=reserved_only,
            deterministic_level=(
                "reserved"
                if reserved_only
                else "annotation_only"
                if execution_support == "metadata_only"
                else "semantic_float64"
                if numeric == "float64"
                else "reference_exact"
            ),
            examples=(f"core.{name}/v1/bv1",),
            usage_notes=usage_notes,
        ),
        tests=[{"kind": execution_support, "deterministic": execution_support != "metadata_only"}],
    )


def _numeric_policy(kind: _Numeric) -> NumericPolicy:
    representation = {
        "float64": "float64",
        "bool": "bool",
        "decimal": "decimal",
        "object": "object",
    }[kind]
    return NumericPolicy(
        representation=representation,  # type: ignore[arg-type]
        deterministic_level="semantic",
        reduction_order="fixed_input_order",
        nan_policy="reject",
        inf_policy="reject",
    )


_TS_FLOAT = "TimeSeries[float]"
_TS_BOOL = "TimeSeries[bool]"
_TS_OBJECT = "TimeSeries[object]"
_EVENT_FLOAT = "EventStream[float]"
_EVENT_BOOL = "EventStream[bool]"
_EVENT_OBJECT = "EventStream[object]"
_EVENT_OBJECT_SINGLE = "Event[object]"
_DECISION = "Decision"
_PLAN = "Plan"
_PANEL_FLOAT = "Panel[float]"
_PANEL_BOOL = "Panel[bool]"
_PANEL_DECIMAL = "Panel[decimal]"

_EMPTY_INPUT_POLICY = (
    "Empty fold inputs are validation errors unless explicitly allowed by params.",
)

_MATH_PARAMS: dict[str, object] = {"type": "object", "additionalProperties": False}
_BINARY_PARAMS: dict[str, object] = {"type": "object", "additionalProperties": False}
_UNARY_PARAMS: dict[str, object] = {"type": "object", "additionalProperties": False}
_CLIP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["lower", "upper"],
    "properties": {"lower": {"type": "number"}, "upper": {"type": "number"}},
}
_ROUND_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"ndigits": {"type": "integer", "default": 0}},
}
_FILL_NAN_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["replacement"],
    "properties": {"replacement": {"type": "number"}},
}
_ALLOW_EMPTY_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"allow_empty": {"type": "boolean", "default": False}},
}
_BETWEEN_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"inclusive": {"type": "boolean", "default": True}},
}
_SHIFT_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "periods": {"type": "integer", "default": 1},
        "allow_unsafe_future": {"type": "boolean", "default": False},
    },
}
_SESSION_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["start_hhmm", "end_hhmm"],
    "properties": {
        "start_hhmm": {"type": "string"},
        "end_hhmm": {"type": "string"},
    },
}
_WINDOW_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["window"],
    "properties": {
        "window": {"type": "integer", "minimum": 1},
        "min_periods": {"type": "integer", "minimum": 1},
    },
}
_BOLLINGER_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["window"],
    "properties": {
        "window": {"type": "integer", "minimum": 1},
        "min_periods": {"type": "integer", "minimum": 1},
        "width": {"type": "number", "default": 2},
    },
}
_MACD_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "fast_window": {"type": "integer", "minimum": 1, "default": 12},
        "slow_window": {"type": "integer", "minimum": 1, "default": 26},
        "signal_window": {"type": "integer", "minimum": 1, "default": 9},
    },
}
_ZSCORE_REVERT_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"threshold": {"type": "number", "default": 2}},
}
_PANEL_RANK_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["k"],
    "properties": {"k": {"type": "integer", "minimum": 1}},
}
_GROUP_NEUTRAL_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["group_spec_ref"],
    "properties": {
        "group_spec_ref": {"type": "string", "minLength": 1},
        "missing_group_policy": {
            "enum": ["error", "drop", "assign_unknown"],
            "default": "error",
        },
    },
}
_RANK_WEIGHT_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "side": {"enum": ["long", "short"], "default": "long"},
        "target_gross": {"type": "string", "default": "1"},
    },
}
_INVERSE_VOL_WEIGHT_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"target_gross": {"type": "string", "default": "1"}},
}
_NET_NORMALIZE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"target_net": {"type": "string", "default": "0"}},
}
_MAX_WEIGHT_CLIP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_abs_weight"],
    "properties": {"max_abs_weight": {"type": "string"}},
}
_SIGNAL_TO_DECISION_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "threshold": {"type": "number", "default": 0},
        "accept_reason": {"type": "string", "default": "SIGNAL_ACCEPTED"},
        "reject_reason": {"type": "string", "default": "SIGNAL_REJECTED"},
    },
}
_THRESHOLD_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["threshold"],
    "properties": {
        "threshold": {"type": "number"},
        "inclusive": {"type": "boolean", "default": True},
    },
}
_COOLDOWN_GATE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"cooldown_events": {"type": "integer", "minimum": 1, "default": 1}},
}
_BREACH_THRESHOLD_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"threshold": {"type": "integer", "minimum": 1, "default": 2}},
}
_OBSERVE_PERIOD_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"window": {"type": "integer", "minimum": 1, "default": 3}},
}
_SLOT_BUDGET_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"slot_budget": {"type": "integer", "minimum": 0, "default": 2}},
}
_POSITION_CAP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_abs_position"],
    "properties": {"max_abs_position": {"type": "string"}},
}
_VOLATILITY_TARGET_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"target_volatility": {"type": "string", "default": "1"}},
}
_TURNOVER_CAP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_turnover"],
    "properties": {"max_turnover": {"type": "string"}},
}
_HOLD_GATE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["bars"],
    "properties": {"bars": {"type": "integer", "minimum": 1}},
}
_DRAWDOWN_GATE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_drawdown"],
    "properties": {"max_drawdown": {"type": "number", "minimum": 0}},
}
_VOLATILITY_REGIME_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_volatility"],
    "properties": {"max_volatility": {"type": "number", "minimum": 0}},
}
_TIME_WINDOW_GATE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["start_hhmm", "end_hhmm"],
    "properties": {
        "start_hhmm": {"type": "string"},
        "end_hhmm": {"type": "string"},
    },
}
_REBALANCE_GATE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["band"],
    "properties": {"band": {"type": "string"}},
}
_STOP_LOSS_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["stop_loss_pct"],
    "properties": {"stop_loss_pct": {"type": "number", "minimum": 0}},
}
_TAKE_PROFIT_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["take_profit_pct"],
    "properties": {"take_profit_pct": {"type": "number", "minimum": 0}},
}
_TRAILING_STOP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["trail_pct"],
    "properties": {"trail_pct": {"type": "number", "minimum": 0}},
}
_MAX_DRAWDOWN_RECORD_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_drawdown"],
    "properties": {"max_drawdown": {"type": "number", "minimum": 0}},
}
_EXPOSURE_CAP_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["max_abs_exposure"],
    "properties": {"max_abs_exposure": {"type": "string"}},
}
_RESERVED_DESIGN_USAGE = (
    "Reserved design token: visible in vocabulary for planning, but not executable in strategies.",
    "Do not treat this token as an implementation instruction.",
)
_QUANTILE_PARAMS: dict[str, object] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["quantile"],
    "properties": {"quantile": {"type": "number", "minimum": 0, "maximum": 1}},
}

_TOKEN_DEFINITIONS: tuple[dict[str, Any], ...] = (
    # Math and boolean primitives.
    {"name": "math.add", "family": "math", "category": "arithmetic", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "math.sub", "family": "math", "category": "arithmetic", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "math.mul", "family": "math", "category": "arithmetic", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "math.div", "family": "math", "category": "arithmetic", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "failure_mode": "division_by_zero emits diagnostic error"},
    {"name": "math.neg", "family": "math", "category": "arithmetic", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.abs", "family": "math", "category": "arithmetic", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.pow", "family": "math", "category": "arithmetic", "inputs": {"base": _TS_FLOAT, "exponent": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _BINARY_PARAMS},
    {"name": "math.sqrt", "family": "math", "category": "arithmetic", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS, "failure_mode": "negative input emits diagnostic error"},
    {"name": "math.log", "family": "math", "category": "arithmetic", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS, "failure_mode": "non-positive input emits diagnostic error"},
    {"name": "math.exp", "family": "math", "category": "arithmetic", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.min", "family": "math", "category": "reduction", "inputs": {"values": _EVENT_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _ALLOW_EMPTY_PARAMS, "failure_mode": "empty input emits diagnostic error unless allow_empty=true", "usage_notes": _EMPTY_INPUT_POLICY},
    {"name": "math.max", "family": "math", "category": "reduction", "inputs": {"values": _EVENT_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _ALLOW_EMPTY_PARAMS, "failure_mode": "empty input emits diagnostic error unless allow_empty=true", "usage_notes": _EMPTY_INPUT_POLICY},
    {"name": "math.clip", "family": "math", "category": "transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _CLIP_PARAMS, "failure_mode": "lower greater than upper emits diagnostic error"},
    {"name": "math.floor", "family": "math", "category": "transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.ceil", "family": "math", "category": "transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.round", "family": "math", "category": "transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _ROUND_PARAMS},
    {"name": "math.sign", "family": "math", "category": "transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _UNARY_PARAMS},
    {"name": "math.isnan", "family": "math", "category": "predicate", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _UNARY_PARAMS, "numeric": "bool", "missing_data": "declared NaN predicate handling"},
    {"name": "math.isfinite", "family": "math", "category": "predicate", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _UNARY_PARAMS, "numeric": "bool", "missing_data": "declared finite predicate handling"},
    {"name": "math.where", "family": "math", "category": "conditional", "inputs": {"condition": _TS_BOOL, "if_true": _TS_FLOAT, "if_false": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _MATH_PARAMS},
    {"name": "math.fill_nan", "family": "math", "category": "missing_transform", "inputs": {"x": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _FILL_NAN_PARAMS, "missing_data": "NaN values are replaced by explicit replacement parameter"},
    {"name": "bool.and", "family": "bool", "category": "boolean_logic", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "bool.or", "family": "bool", "category": "boolean_logic", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "bool.not", "family": "bool", "category": "boolean_logic", "inputs": {"x": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "bool.xor", "family": "bool", "category": "boolean_logic", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "bool.any", "family": "bool", "category": "boolean_reduction", "inputs": {"values": _EVENT_BOOL}, "outputs": {"value": _TS_BOOL}, "params_schema": _ALLOW_EMPTY_PARAMS, "numeric": "bool", "failure_mode": "empty input emits diagnostic error unless allow_empty=true", "usage_notes": _EMPTY_INPUT_POLICY},
    {"name": "bool.all", "family": "bool", "category": "boolean_reduction", "inputs": {"values": _EVENT_BOOL}, "outputs": {"value": _TS_BOOL}, "params_schema": _ALLOW_EMPTY_PARAMS, "numeric": "bool", "failure_mode": "empty input emits diagnostic error unless allow_empty=true", "usage_notes": _EMPTY_INPUT_POLICY},
    {"name": "bool.count_true", "family": "bool", "category": "boolean_reduction", "inputs": {"values": _EVENT_BOOL}, "outputs": {"value": "TimeSeries[int]"}, "params_schema": _ALLOW_EMPTY_PARAMS, "numeric": "object", "failure_mode": "empty input emits diagnostic error unless allow_empty=true", "usage_notes": _EMPTY_INPUT_POLICY},
    {"name": "logic.and", "family": "bool", "category": "boolean_logic", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "logic.or", "family": "bool", "category": "boolean_logic", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "logic.not", "family": "bool", "category": "boolean_logic", "inputs": {"x": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.eq", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.ne", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.gt", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.gte", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.lt", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.lte", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "cmp.between", "family": "compare", "category": "range_comparison", "inputs": {"x": _TS_FLOAT, "lower": _TS_FLOAT, "upper": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _BETWEEN_PARAMS, "numeric": "bool", "failure_mode": "lower greater than upper emits diagnostic error"},
    {"name": "cmp.outside", "family": "compare", "category": "range_comparison", "inputs": {"x": _TS_FLOAT, "lower": _TS_FLOAT, "upper": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _BETWEEN_PARAMS, "numeric": "bool", "failure_mode": "lower greater than upper emits diagnostic error"},
    {"name": "compare.gt", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "compare.ge", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "compare.lt", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "compare.le", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "compare.eq", "family": "compare", "category": "comparison", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    # Data, time, alignment, window, signal, and indicators.
    {"name": "data.shift", "family": "data", "category": "lag", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _SHIFT_PARAMS, "temporal": "positive periods lag; negative periods require unsafe-future diagnostics", "failure_mode": "negative periods emit unsafe-future diagnostic unless explicitly allowed"},
    {"name": "data.identity", "family": "data", "category": "identity", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "data.diff", "family": "data", "category": "delta", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "temporal": "uses previous timestamp only"},
    {"name": "data.pct_change", "family": "data", "category": "return_transform", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "temporal": "uses previous timestamp only", "failure_mode": "previous value zero emits diagnostic error"},
    {"name": "data.log_return", "family": "data", "category": "return_transform", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "temporal": "uses previous timestamp only", "failure_mode": "non-positive current or previous value emits diagnostic error"},
    {"name": "time.session_filter", "family": "time", "category": "calendar_filter", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _SESSION_PARAMS},
    {"name": "align.inner_join", "family": "align", "category": "alignment", "inputs": {"left": _TS_FLOAT, "right": _TS_FLOAT}, "outputs": {"left": _TS_FLOAT, "right": _TS_FLOAT}},
    {"name": "align.left_join", "family": "align", "category": "alignment", "inputs": {"left": _TS_FLOAT, "right": _TS_FLOAT}, "outputs": {"left": _TS_FLOAT, "right": _TS_FLOAT}, "missing_data": "right side may be None where timestamp is absent"},
    {"name": "align.forward_fill", "family": "align", "category": "alignment", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "align.drop_missing", "family": "align", "category": "alignment", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "missing_data": "drops None values from active timestamp set"},
    {"name": "window.max", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.min", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.mean", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.std", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.sum", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.count", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": "TimeSeries[int]"}, "params_schema": _WINDOW_PARAMS, "numeric": "object", "temporal": "trailing window; min_history_bars derives from window parameter"},
    {"name": "window.zscore", "family": "window", "category": "rolling_window", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "trailing window; zero variance outputs 0"},
    {"name": "norm.range_position", "family": "signal", "category": "normalization", "inputs": {"value": _TS_FLOAT, "high": _TS_FLOAT, "low": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}},
    {"name": "smooth.linear_recursive", "family": "signal", "category": "smoothing", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "stateful": True},
    {"name": "signal.cross_above", "family": "signal", "category": "cross", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.cross_below", "family": "signal", "category": "cross", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.crosses", "family": "signal", "category": "cross", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.threshold_above", "family": "signal", "category": "threshold", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _THRESHOLD_PARAMS, "numeric": "bool"},
    {"name": "signal.threshold_below", "family": "signal", "category": "threshold", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _THRESHOLD_PARAMS, "numeric": "bool"},
    {"name": "indicator.sma", "family": "indicator", "category": "trend", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS},
    {"name": "indicator.ema", "family": "indicator", "category": "trend", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "stateful": True},
    {"name": "indicator.rsi", "family": "indicator", "category": "oscillator", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS},
    {"name": "indicator.bollinger", "family": "indicator", "category": "band", "inputs": {"series": _TS_FLOAT}, "outputs": {"middle": _TS_FLOAT, "upper": _TS_FLOAT, "lower": _TS_FLOAT}, "params_schema": _BOLLINGER_PARAMS},
    {"name": "indicator.channel_breakout", "family": "indicator", "category": "breakout", "inputs": {"high": _TS_FLOAT, "low": _TS_FLOAT, "close": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _WINDOW_PARAMS, "numeric": "bool", "temporal": "uses previous trailing window only to avoid current-bar lookahead"},
    {"name": "indicator.rolling_mean", "family": "indicator", "category": "rolling_stat", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS},
    {"name": "indicator.rolling_std", "family": "indicator", "category": "rolling_stat", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS},
    {"name": "indicator.rolling_zscore", "family": "indicator", "category": "rolling_stat", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS},
    {"name": "indicator.macd", "family": "indicator", "category": "trend", "inputs": {"series": _TS_FLOAT}, "outputs": {"macd": _TS_FLOAT, "signal": _TS_FLOAT, "histogram": _TS_FLOAT}, "params_schema": _MACD_PARAMS, "stateful": True},
    {"name": "indicator.bollinger_band", "family": "indicator", "category": "band", "inputs": {"series": _TS_FLOAT}, "outputs": {"middle": _TS_FLOAT, "upper": _TS_FLOAT, "lower": _TS_FLOAT}, "params_schema": _BOLLINGER_PARAMS},
    {"name": "indicator.atr", "family": "indicator", "category": "volatility", "inputs": {"high": _TS_FLOAT, "low": _TS_FLOAT, "close": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "failure_mode": "missing high/low/close or invalid window emits diagnostic error"},
    {"name": "indicator.donchian_channel", "family": "indicator", "category": "breakout", "inputs": {"high": _TS_FLOAT, "low": _TS_FLOAT}, "outputs": {"upper": _TS_FLOAT, "lower": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "uses previous trailing window only to avoid current-bar lookahead"},
    {"name": "indicator.volatility", "family": "indicator", "category": "volatility", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "temporal": "rolling population standard deviation of percentage returns"},
    {"name": "indicator.linear_regression_slope", "family": "indicator", "category": "regression", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "failure_mode": "insufficient observations emits diagnostic error"},
    {"name": "indicator.beta", "family": "indicator", "category": "regression", "inputs": {"series": _TS_FLOAT, "benchmark": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "failure_mode": "zero benchmark variance emits diagnostic error"},
    {"name": "indicator.residual", "family": "indicator", "category": "regression", "inputs": {"series": _TS_FLOAT, "benchmark": _TS_FLOAT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "failure_mode": "zero benchmark variance emits diagnostic error"},
    {"name": "signal.greater_than", "family": "signal", "category": "comparison_signal", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.less_than", "family": "signal", "category": "comparison_signal", "inputs": {"a": _TS_FLOAT, "b": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.and", "family": "signal", "category": "boolean_signal", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.or", "family": "signal", "category": "boolean_signal", "inputs": {"a": _TS_BOOL, "b": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.not", "family": "signal", "category": "boolean_signal", "inputs": {"x": _TS_BOOL}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.between", "family": "signal", "category": "band_signal", "inputs": {"x": _TS_FLOAT, "lower": _TS_FLOAT, "upper": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _BETWEEN_PARAMS, "numeric": "bool"},
    {"name": "signal.outside_band", "family": "signal", "category": "band_signal", "inputs": {"x": _TS_FLOAT, "lower": _TS_FLOAT, "upper": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.breakout_up", "family": "signal", "category": "breakout", "inputs": {"series": _TS_FLOAT, "upper": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.breakout_down", "family": "signal", "category": "breakout", "inputs": {"series": _TS_FLOAT, "lower": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "numeric": "bool"},
    {"name": "signal.zscore_revert", "family": "signal", "category": "mean_reversion", "inputs": {"zscore": _TS_FLOAT}, "outputs": {"value": _TS_BOOL}, "params_schema": _ZSCORE_REVERT_PARAMS, "numeric": "bool"},
    {"name": "signal.rank_top_k", "family": "signal", "category": "panel_signal", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"selection": "Panel[bool]"}, "params_schema": _PANEL_RANK_PARAMS, "numeric": "bool", "panel_aware": True},
    {"name": "signal.rank_bottom_k", "family": "signal", "category": "panel_signal", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"selection": "Panel[bool]"}, "params_schema": _PANEL_RANK_PARAMS, "numeric": "bool", "panel_aware": True},
    # PR8 panel, factor, and weight record aliases.
    {"name": "panel.cross_sectional_rank", "family": "panel", "category": "panel_operator_alias", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"ranked": "Panel[int]"}, "panel_aware": True, "usage_notes": ("Alias for panel.rank reference semantics.",)},
    {"name": "panel.zscore_by_universe", "family": "panel", "category": "panel_operator_alias", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"panel": _PANEL_FLOAT}, "panel_aware": True, "usage_notes": ("Alias for panel.zscore over the active universe.",)},
    {"name": "panel.neutralize_group", "family": "panel", "category": "panel_operator_alias", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"panel": _PANEL_FLOAT}, "params_schema": _GROUP_NEUTRAL_PARAMS, "panel_aware": True, "failure_mode": "missing explicit group metadata emits diagnostic error", "usage_notes": ("Alias for panel.group_demean; group metadata must be explicit.",)},
    {"name": "selection.top_k", "family": "panel", "category": "selection_operator_alias", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"selection": _PANEL_BOOL}, "params_schema": _PANEL_RANK_PARAMS, "numeric": "bool", "panel_aware": True, "usage_notes": ("Alias for panel.top_k selection reference semantics.",)},
    {"name": "selection.bottom_k", "family": "panel", "category": "selection_operator_alias", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"selection": _PANEL_BOOL}, "params_schema": _PANEL_RANK_PARAMS, "numeric": "bool", "panel_aware": True, "usage_notes": ("Alias for panel.bottom_k selection reference semantics.",)},
    {"name": "factor.sector_neutral_rank", "family": "factor", "category": "sector_neutral_factor", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"ranked": "Panel[int]"}, "params_schema": _GROUP_NEUTRAL_PARAMS, "panel_aware": True, "failure_mode": "missing explicit group metadata emits diagnostic error", "usage_notes": ("Demean within explicit groups, then rank cross-sectionally.", "Does not infer sector metadata.")},
    {"name": "factor.residualize", "family": "factor", "category": "factor_residualization", "inputs": {"panel": _PANEL_FLOAT, "factor": _TS_FLOAT}, "outputs": {"panel": _PANEL_FLOAT}, "panel_aware": True, "usage_notes": ("Alias over panel.residualize reference semantics.",)},
    {"name": "factor.beta_neutral_signal", "family": "factor", "category": "beta_neutral_signal_record", "inputs": {"series": _TS_FLOAT, "benchmark": _TS_FLOAT}, "outputs": {"score": _TS_FLOAT}, "params_schema": _WINDOW_PARAMS, "panel_aware": True, "failure_mode": "zero benchmark variance emits diagnostic error", "usage_notes": ("Beta/residual signal record only; no portfolio beta-neutral construction.",)},
    {"name": "weight.equal_weight", "family": "weight", "category": "weight_record", "inputs": {"selection": _PANEL_BOOL}, "outputs": {"weights": _PANEL_DECIMAL}, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Alias for equal-long selection.to_weights reference semantics.",)},
    {"name": "weight.rank_weight", "family": "weight", "category": "weight_record", "inputs": {"ranked": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _RANK_WEIGHT_PARAMS, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Transforms positive ranks into deterministic raw weights; no optimizer.",)},
    {"name": "weight.inverse_vol_weight", "family": "weight", "category": "weight_record", "inputs": {"volatility": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _INVERSE_VOL_WEIGHT_PARAMS, "numeric": "decimal", "panel_aware": True, "failure_mode": "missing, bool, non-finite, or nonpositive volatility emits diagnostic error", "usage_notes": ("Inverse volatility transform only; no rebalance or optimizer engine.",)},
    {"name": "weight.vol_target_weight", "family": "weight", "category": "weight_record", "inputs": {"weights": _PANEL_DECIMAL, "volatility": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _VOLATILITY_TARGET_PARAMS, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Wrapper over risk.volatility_target scaling semantics; no portfolio engine.",)},
    {"name": "weight.market_neutral_weight", "family": "weight", "category": "weight_record", "inputs": {"weights": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Alias for weight.market_neutral reference semantics.",)},
    {"name": "weight.group_neutral_weight", "family": "weight", "category": "weight_record", "inputs": {"weights": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _GROUP_NEUTRAL_PARAMS, "numeric": "decimal", "panel_aware": True, "failure_mode": "missing explicit group metadata emits diagnostic error", "usage_notes": ("Neutralizes net exposure within explicit groups only.", "Does not infer sector metadata or optimize exposures.")},
    {"name": "weight.max_weight_clip", "family": "weight", "category": "weight_record", "inputs": {"weights": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _MAX_WEIGHT_CLIP_PARAMS, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Alias for weight.cap_per_symbol clip-no-redistribute semantics.",)},
    {"name": "weight.normalize_net", "family": "weight", "category": "weight_record", "inputs": {"weights": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _NET_NORMALIZE_PARAMS, "numeric": "decimal", "panel_aware": True, "usage_notes": ("Adds a deterministic per-timestamp net shift; no gross normalization or optimizer.",)},
    # Decision, gate, risk, and plan surface tokens.
    {"name": "decision.lift_bool", "family": "decision", "category": "decision_bridge", "inputs": {"series": _TS_BOOL}, "outputs": {"decision": _DECISION}, "numeric": "object"},
    {"name": "decision.long_flat", "family": "decision", "category": "rule_decision", "inputs": {"series": _TS_BOOL}, "outputs": {"decision": _DECISION}, "numeric": "object"},
    {"name": "decision.long_short", "family": "decision", "category": "rule_decision", "inputs": {"long_signal": _TS_BOOL, "short_signal": _TS_BOOL}, "outputs": {"decision": _DECISION}, "numeric": "object"},
    {"name": "decision.entry_exit_to_position", "family": "decision", "category": "rule_decision", "inputs": {"entry": _TS_BOOL, "exit": _TS_BOOL}, "outputs": {"decision": _DECISION}, "numeric": "object", "stateful": True},
    {"name": "decision.signal_to_decision", "family": "decision", "category": "rule_decision", "inputs": {"score": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _SIGNAL_TO_DECISION_PARAMS, "numeric": "object"},
    {"name": "decision.rank_to_selection", "family": "decision", "category": "selection_decision", "inputs": {"panel": _PANEL_FLOAT}, "outputs": {"selection": "Panel[bool]"}, "params_schema": _PANEL_RANK_PARAMS, "numeric": "object", "panel_aware": True},
    {"name": "decision.selection_to_weight", "family": "decision", "category": "selection_decision", "inputs": {"selection": "Panel[bool]"}, "outputs": {"weights": _PANEL_DECIMAL}, "numeric": "decimal", "panel_aware": True},
    {"name": "decision.gate_decision", "family": "decision", "category": "rule_decision", "inputs": {"decision": _DECISION, "gate": _DECISION}, "outputs": {"decision": _DECISION}, "numeric": "object"},
    {"name": "gate.cooldown", "family": "gate", "category": "state_gate", "inputs": {"decision": _DECISION}, "outputs": {"decision": _DECISION}, "params_schema": _COOLDOWN_GATE_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "gate block emits DecisionKind block; errors remain diagnostics"},
    {"name": "gate.market_freeze", "family": "gate", "category": "state_gate", "inputs": {"decision": _DECISION}, "outputs": {"decision": _DECISION}, "numeric": "object", "stateful": True, "failure_mode": "market freeze emits DecisionKind block; errors remain diagnostics"},
    {"name": "gate.circuit_breaker", "family": "gate", "category": "state_gate", "inputs": {"decision": _DECISION}, "outputs": {"decision": _DECISION}, "params_schema": _BREACH_THRESHOLD_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "breach threshold emits DecisionKind block; errors remain diagnostics"},
    {"name": "gate.observe_period", "family": "gate", "category": "state_gate", "inputs": {"decision": _DECISION}, "outputs": {"decision": _DECISION}, "params_schema": _OBSERVE_PERIOD_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "observe warmup emits DecisionKind block; errors remain diagnostics"},
    {"name": "gate.slot_budget", "family": "gate", "category": "state_gate", "inputs": {"decision": _DECISION}, "outputs": {"decision": _DECISION}, "params_schema": _SLOT_BUDGET_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "slot budget breach emits DecisionKind block; errors remain diagnostics"},
    {"name": "gate.volatility_regime", "family": "gate", "category": "risk_gate_record", "inputs": {"volatility": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _VOLATILITY_REGIME_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "volatility above regime threshold emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Record-layer volatility regime gate only; no live risk engine.",)},
    {"name": "gate.drawdown", "family": "gate", "category": "risk_gate_record", "inputs": {"drawdown": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _DRAWDOWN_GATE_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "drawdown breach emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Drawdown record gate only; no portfolio accounting runtime.",)},
    {"name": "gate.time_window", "family": "gate", "category": "temporal_gate_record", "inputs": {"timestamp": _TS_OBJECT}, "outputs": {"decision": _DECISION}, "params_schema": _TIME_WINDOW_GATE_PARAMS, "numeric": "object", "stateful": True, "temporal": "UTC timestamp time-of-day inclusive window", "failure_mode": "outside time window emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Time-window record gate only; Calendar TypeSpec remains deferred.",)},
    {"name": "gate.rebalance", "family": "gate", "category": "rebalance_gate_record", "inputs": {"drift": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _REBALANCE_GATE_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "drift below rebalance band emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Threshold record only; no rebalance scheduler or execution engine.",)},
    {"name": "gate.min_hold", "family": "gate", "category": "hold_gate_record", "inputs": {"age": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _HOLD_GATE_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "age below minimum hold emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Hold-age record gate only; no position lifecycle runtime.",)},
    {"name": "gate.max_hold", "family": "gate", "category": "hold_gate_record", "inputs": {"age": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _HOLD_GATE_PARAMS, "numeric": "object", "stateful": True, "failure_mode": "age below maximum hold threshold emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Exit eligibility record only; no order lifecycle runtime.",)},
    {"name": "risk.position_cap", "family": "risk", "category": "risk_gate", "inputs": {"decision": _DECISION, "state": "State[object]"}, "outputs": {"decision": _DECISION}, "params_schema": _POSITION_CAP_PARAMS, "numeric": "object", "risk_level": "medium", "failure_mode": "position cap breach emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Reference helper does not place orders or enforce broker limits.",)},
    {"name": "risk.volatility_target", "family": "risk", "category": "weight_risk", "inputs": {"weights": _PANEL_DECIMAL, "volatility": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _VOLATILITY_TARGET_PARAMS, "numeric": "decimal", "panel_aware": True, "risk_level": "medium", "failure_mode": "missing or nonpositive volatility emits diagnostic error", "usage_notes": ("Scales weights only; no gross/net normalization or portfolio engine.",)},
    {"name": "risk.turnover_cap", "family": "risk", "category": "weight_risk", "inputs": {"weights": _PANEL_DECIMAL, "previous": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _TURNOVER_CAP_PARAMS, "numeric": "decimal", "panel_aware": True, "risk_level": "medium", "failure_mode": "missing previous weight emits diagnostic error", "usage_notes": ("Clips per-symbol deltas only; no redistribution or rebalance plan.",)},
    {"name": "risk.stop_loss_record", "family": "risk", "category": "risk_record", "inputs": {"entry_price": _TS_FLOAT, "current_price": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _STOP_LOSS_PARAMS, "numeric": "object", "risk_level": "medium", "failure_mode": "stop loss threshold emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Record-layer stop-loss helper only; no broker-side stop order.",)},
    {"name": "risk.take_profit_record", "family": "risk", "category": "risk_record", "inputs": {"entry_price": _TS_FLOAT, "current_price": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _TAKE_PROFIT_PARAMS, "numeric": "object", "risk_level": "medium", "failure_mode": "take-profit threshold emits DecisionKind accept; errors remain diagnostics", "usage_notes": ("Record-layer take-profit helper only; no order placement.",)},
    {"name": "risk.trailing_stop_record", "family": "risk", "category": "risk_record", "inputs": {"price": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _TRAILING_STOP_PARAMS, "numeric": "object", "stateful": True, "risk_level": "medium", "failure_mode": "trailing stop threshold emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Record-layer trailing stop helper only; no broker-side stop order.",)},
    {"name": "risk.max_drawdown_record", "family": "risk", "category": "risk_record", "inputs": {"equity": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _MAX_DRAWDOWN_RECORD_PARAMS, "numeric": "object", "stateful": True, "risk_level": "medium", "failure_mode": "max drawdown breach emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Equity-curve record helper only; no account runtime.",)},
    {"name": "risk.volatility_target_record", "family": "risk", "category": "weight_risk_record", "inputs": {"weights": _PANEL_DECIMAL, "volatility": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _VOLATILITY_TARGET_PARAMS, "numeric": "decimal", "panel_aware": True, "risk_level": "medium", "failure_mode": "missing or nonpositive volatility emits diagnostic error", "usage_notes": ("Alias over risk.volatility_target record semantics; no rebalance engine.",)},
    {"name": "risk.exposure_cap_record", "family": "risk", "category": "risk_record", "inputs": {"decision": _DECISION, "exposure": _TS_FLOAT}, "outputs": {"decision": _DECISION}, "params_schema": _EXPOSURE_CAP_PARAMS, "numeric": "object", "risk_level": "medium", "failure_mode": "exposure cap breach emits DecisionKind block; errors remain diagnostics", "usage_notes": ("Exposure cap record only; no portfolio execution engine.",)},
    {"name": "risk.turnover_limit_record", "family": "risk", "category": "weight_risk_record", "inputs": {"weights": _PANEL_DECIMAL, "previous": _PANEL_DECIMAL}, "outputs": {"weights": _PANEL_DECIMAL}, "params_schema": _TURNOVER_CAP_PARAMS, "numeric": "decimal", "panel_aware": True, "risk_level": "medium", "failure_mode": "missing previous weight emits diagnostic error", "usage_notes": ("Alias over risk.turnover_cap clip-no-redistribute semantics; no rebalance engine.",)},
    {"name": "plan.noop", "family": "execution", "category": "plan_shell", "inputs": {"decision": _DECISION}, "outputs": {"plan": _PLAN}, "numeric": "object", "execution_support": "metadata_only"},
    {"name": "plan.order_intent", "family": "execution", "category": "plan_shell", "inputs": {"decision": _DECISION, "sizing": "Scalar[float]"}, "outputs": {"plan": _PLAN}, "numeric": "object", "execution_support": "metadata_only"},
    # Experimental optimizer and reserved design boundaries.
    {"name": "optimizer.mean_variance", "family": "optimizer", "category": "portfolio_optimizer", "inputs": {"expected_return": _PANEL_FLOAT, "risk": _PANEL_FLOAT}, "outputs": {"weights": _PANEL_DECIMAL}, "numeric": "decimal", "maturity": "experimental", "execution_support": "metadata_only", "panel_aware": True, "solver_backed": True, "risk_level": "high", "usage_notes": ("Requires explicit solver determinism contract before promotion.", "Metadata-only in Stage 3A.4; no executable optimizer path is exposed.")},
    {"name": "event.join_asof", "family": "event", "category": "event_stream", "inputs": {"events": _EVENT_OBJECT}, "outputs": {"events": _EVENT_OBJECT}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "event.filter", "family": "event", "category": "event_stream", "inputs": {"events": _EVENT_OBJECT, "predicate": _EVENT_BOOL}, "outputs": {"events": _EVENT_OBJECT}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "event.window_count", "family": "event", "category": "event_aggregation", "inputs": {"events": _EVENT_OBJECT}, "outputs": {"count": "TimeSeries[int]"}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "distribution.normal_fit", "family": "distribution", "category": "distribution_model", "inputs": {"series": _TS_FLOAT}, "outputs": {"value": _TS_OBJECT}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "distribution.quantile", "family": "distribution", "category": "distribution_metric", "inputs": {"distribution": _TS_OBJECT}, "outputs": {"value": _TS_FLOAT}, "params_schema": _QUANTILE_PARAMS, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "distribution.tail_probability", "family": "distribution", "category": "distribution_metric", "inputs": {"distribution": _TS_OBJECT, "threshold": _TS_FLOAT}, "outputs": {"probability": _TS_FLOAT}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "execution.submit_order", "family": "execution", "category": "execution_boundary", "inputs": {"plan": _PLAN}, "outputs": {"event": _EVENT_OBJECT_SINGLE}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "risk_level": "high", "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "execution.cancel_order", "family": "execution", "category": "execution_boundary", "inputs": {"plan": _PLAN}, "outputs": {"event": _EVENT_OBJECT_SINGLE}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "risk_level": "high", "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "execution.fill_report", "family": "execution", "category": "execution_feedback", "inputs": {"event": _EVENT_OBJECT_SINGLE}, "outputs": {"event": _EVENT_OBJECT_SINGLE}, "numeric": "object", "maturity": "reserved_design", "execution_support": "metadata_only", "reserved_only": True, "risk_level": "high", "usage_notes": _RESERVED_DESIGN_USAGE},
    {"name": "score.zscore", "family": "continuous_score", "category": "score_transform", "inputs": {"series": _TS_FLOAT}, "outputs": {"score": _TS_FLOAT}},
    {"name": "score.calibrate", "family": "continuous_score", "category": "score_calibration", "inputs": {"score": _TS_FLOAT}, "outputs": {"score": _TS_FLOAT}, "maturity": "experimental", "execution_support": "metadata_only", "usage_notes": ("Calibration has no accepted reference helper in Stage 3A.5.", "Continuous-score tokens do not produce DecisionKind or execution plans.")},
)
