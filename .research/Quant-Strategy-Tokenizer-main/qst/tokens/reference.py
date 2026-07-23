"""Deterministic reference helpers for primitive token contracts.

These helpers are conformance aids for TokenSpec contracts. They are not a
strategy runtime and are intentionally not called by IR validation.
"""

from __future__ import annotations

import math
from collections.abc import Iterable, Mapping, Sequence
from decimal import Decimal
from typing import Any, Literal, cast, overload

from pydantic import BaseModel, ConfigDict, Field

from qst.artifacts.decimal_string import normalize_to_canonical
from qst.validation import Diagnostic, ValidationResult

SeriesRow = tuple[str, Any]
NumericSeries = list[tuple[str, float]]
BoolSeries = list[tuple[str, bool]]


class TokenReferenceError(ValueError):
    """Reference helper failure with a stable diagnostic-like code."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


class TokenReferenceResult(BaseModel):
    """Structured conformance helper result for non-runtime token facades."""

    model_config = ConfigDict(extra="forbid", frozen=True, arbitrary_types_allowed=True)

    decisions: tuple[Any, ...] = Field(default_factory=tuple)
    weights: Any | None = None
    diagnostics: ValidationResult = Field(default_factory=ValidationResult)
    trace: dict[str, Any] = Field(default_factory=dict)


def evaluate_math_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate a Stage 3A.1 `math.*` primitive reference helper."""

    token = _local_name(name, "math.")

    if token in {"add", "sub", "mul", "div", "pow"}:
        left, right = _expect_args(token, args, 2)
        a = _coerce_number(left)
        b = _coerce_number(right)
        if token == "add":
            return a + b
        if token == "sub":
            return a - b
        if token == "mul":
            return a * b
        if token == "pow":
            return _finite_result(a**b)
        if b == 0:
            raise TokenReferenceError(
                "QST_TOKEN_MATH_DIVIDE_BY_ZERO",
                "math.div cannot divide by zero.",
            )
        return _finite_result(a / b)

    if token in {"neg", "abs", "sqrt", "log", "exp", "floor", "ceil", "round", "sign"}:
        (raw,) = _expect_args(token, args, 1)
        x = _coerce_number(raw)
        if token == "neg":
            return -x
        if token == "abs":
            return abs(x)
        if token == "sqrt":
            if x < 0:
                raise TokenReferenceError(
                    "QST_TOKEN_MATH_DOMAIN_ERROR",
                    "math.sqrt requires a non-negative input.",
                )
            return _finite_result(math.sqrt(x))
        if token == "log":
            if x <= 0:
                raise TokenReferenceError(
                    "QST_TOKEN_MATH_DOMAIN_ERROR",
                    "math.log requires a positive input.",
                )
            return _finite_result(math.log(x))
        if token == "exp":
            return _finite_result(math.exp(x))
        if token == "floor":
            return math.floor(x)
        if token == "ceil":
            return math.ceil(x)
        if token == "round":
            ndigits = params.get("ndigits", 0)
            if isinstance(ndigits, bool) or not isinstance(ndigits, int):
                raise TokenReferenceError(
                    "QST_TOKEN_PARAM_TYPE_INVALID",
                    "math.round ndigits must be an integer.",
                )
            return round(x, ndigits)
        return 1 if x > 0 else -1 if x < 0 else 0

    if token in {"min", "max"}:
        (raw_values,) = _expect_args(token, args, 1)
        values = [_coerce_number(value) for value in _iter_values(raw_values)]
        if not values and not bool(params.get("allow_empty", False)):
            raise TokenReferenceError(
                "QST_TOKEN_EMPTY_INPUT",
                f"math.{token} requires at least one input.",
            )
        if not values:
            return None
        return min(values) if token == "min" else max(values)

    if token == "clip":
        (raw,) = _expect_args(token, args, 1)
        x = _coerce_number(raw)
        lower = _coerce_number(params.get("lower"))
        upper = _coerce_number(params.get("upper"))
        if lower > upper:
            raise TokenReferenceError(
                "QST_TOKEN_MATH_RANGE_INVALID",
                "math.clip lower must be less than or equal to upper.",
            )
        return min(max(x, lower), upper)

    if token in {"isnan", "isfinite"}:
        (raw,) = _expect_args(token, args, 1)
        x = _coerce_number(raw, allow_nonfinite=True)
        return math.isnan(x) if token == "isnan" else math.isfinite(x)

    if token == "where":
        condition, if_true, if_false = _expect_args(token, args, 3)
        if not isinstance(condition, bool):
            raise TokenReferenceError(
                "QST_TOKEN_BOOL_TYPE_INVALID",
                "math.where condition must be boolean.",
            )
        return _coerce_number(if_true if condition else if_false)

    if token == "fill_nan":
        (raw,) = _expect_args(token, args, 1)
        x = _coerce_number(raw, allow_nonfinite=True)
        replacement = _coerce_number(params.get("replacement"))
        if math.isnan(x):
            return replacement
        if not math.isfinite(x):
            raise TokenReferenceError(
                "QST_TOKEN_NUMERIC_NONFINITE",
                "math.fill_nan only handles NaN, not Infinity.",
            )
        return x

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported math token: {name}")


def evaluate_bool_token(name: str, *args: Any, allow_empty: bool = False) -> object:
    """Evaluate a Stage 3A.1 `bool.*` primitive reference helper."""

    token = _local_name(name, "bool.")

    if token in {"and", "or", "xor"}:
        left, right = _expect_args(token, args, 2)
        a = _coerce_bool(left)
        b = _coerce_bool(right)
        if token == "and":
            return a and b
        if token == "or":
            return a or b
        return a != b

    if token == "not":
        (raw,) = _expect_args(token, args, 1)
        return not _coerce_bool(raw)

    if token in {"any", "all", "count_true"}:
        (raw_values,) = _expect_args(token, args, 1)
        values = [_coerce_bool(value) for value in _iter_values(raw_values)]
        if not values and not allow_empty:
            raise TokenReferenceError(
                "QST_TOKEN_EMPTY_INPUT",
                f"bool.{token} requires at least one input.",
            )
        if token == "any":
            return any(values)
        if token == "all":
            return all(values)
        return sum(1 for value in values if value)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported bool token: {name}")


def evaluate_cmp_token(name: str, *args: Any, inclusive: bool = True) -> bool:
    """Evaluate a Stage 3A.1 `cmp.*` primitive reference helper."""

    token = _local_name(name, "cmp.")

    if token in {"eq", "ne", "gt", "gte", "lt", "lte"}:
        left, right = _expect_args(token, args, 2)
        a = _coerce_number(left)
        b = _coerce_number(right)
        return {
            "eq": a == b,
            "ne": a != b,
            "gt": a > b,
            "gte": a >= b,
            "lt": a < b,
            "lte": a <= b,
        }[token]

    if token in {"between", "outside"}:
        raw_x, raw_lower, raw_upper = _expect_args(token, args, 3)
        x = _coerce_number(raw_x)
        lower = _coerce_number(raw_lower)
        upper = _coerce_number(raw_upper)
        if lower > upper:
            raise TokenReferenceError(
                "QST_TOKEN_CMP_RANGE_INVALID",
                "cmp.between/cmp.outside require lower <= upper.",
            )
        inside = lower <= x <= upper if inclusive else lower < x < upper
        return inside if token == "between" else not inside

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported cmp token: {name}")


def evaluate_decision_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate Stage 3A.3 decision facade helpers."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    if name == "decision.lift_bool":
        (series,) = _expect_args("decision.lift_bool", args, 1)
        accept_reason = str(params.get("accept_reason", "decision_lift_bool_accept"))
        reject_reason = str(params.get("reject_reason", "decision_lift_bool_reject"))
        from qst.decision import DecisionV2

        return [
            DecisionV2(
                kind="accept" if value else "reject",
                reasons=(accept_reason if value else reject_reason,),
            )
            for value in _bool_values(series)
        ]

    if name == "decision.long_flat":
        (series,) = _expect_args(name, args, 1)
        return _bool_series_to_decisions(
            _bool_series(series),
            true_kind="accept",
            false_kind="reject",
            true_reason="DECISION_LONG",
            false_reason="DECISION_FLAT",
        )

    if name == "decision.long_short":
        long_signal, short_signal = _expect_args(name, args, 2)
        return _long_short_decisions(_bool_series(long_signal), _bool_series(short_signal))

    if name == "decision.entry_exit_to_position":
        entry_signal, exit_signal = _expect_args(name, args, 2)
        return _entry_exit_decisions(_bool_series(entry_signal), _bool_series(exit_signal))

    if name == "decision.signal_to_decision":
        (series,) = _expect_args(name, args, 1)
        threshold = _coerce_number(params.get("threshold", 0))
        return _numeric_series_to_decisions(
            _numeric_series(series),
            threshold=threshold,
            accept_reason=str(params.get("accept_reason", "SIGNAL_ACCEPTED")),
            reject_reason=str(params.get("reject_reason", "SIGNAL_REJECTED")),
        )

    if name == "decision.rank_to_selection":
        (panel,) = _expect_args(name, args, 1)
        side = str(params.get("side", "top"))
        selection_params = {key: value for key, value in params.items() if key != "side"}
        if side == "bottom":
            return evaluate_panel_token("panel.bottom_k", panel, **selection_params)
        if side != "top":
            raise TokenReferenceError(
                "QST_TOKEN_DECISION_SIDE_INVALID",
                "decision.rank_to_selection side must be 'top' or 'bottom'.",
            )
        return evaluate_panel_token("panel.top_k", panel, **selection_params)

    if name == "decision.selection_to_weight":
        (selection,) = _expect_args(name, args, 1)
        return evaluate_panel_token("selection.to_weights", selection, **params)

    if name == "decision.gate_decision":
        decision_values, gate_values = _expect_args(name, args, 2)
        return _gate_decisions(list(decision_values), list(gate_values))

    if name.startswith("decision."):
        (decisions,) = _expect_args(name, args, 1)
        from qst.decision import (
            aggregate_decisions,
            combine_decisions,
            fold_decisions,
            is_aggregator_id,
            is_fold_policy_id,
            is_monoid_id,
        )

        if is_monoid_id(name):
            return combine_decisions(name, list(decisions), score_policy=params.get("score_policy"))
        if is_fold_policy_id(name):
            return fold_decisions(name, list(decisions), score_policy=params.get("score_policy"))
        if is_aggregator_id(name):
            return aggregate_decisions(name, list(decisions), params=params)

    raise TokenReferenceError(
        "QST_TOKEN_REFERENCE_UNSUPPORTED",
        f"Unsupported decision token: {name}",
    )


def evaluate_state_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate Stage 3A.3 state facade helpers."""

    token = _local_name(name, "state.")

    if token == "delay":
        from qst.state import state_delay

        (values,) = _expect_args(token, args, 1)
        return state_delay(values, **params)

    if token == "accumulate":
        from qst.state import state_accumulate

        (values,) = _expect_args(token, args, 1)
        return state_accumulate(values, **params)

    if token == "edge_detect":
        from qst.state import state_edge_detect

        (values,) = _expect_args(token, args, 1)
        return state_edge_detect(values, **params)

    if token == "fsm":
        from qst.state import FSMDefinition, state_fsm

        (events,) = _expect_args(token, args, 1)
        active_params = dict(params)
        raw_definition = active_params.pop("definition", None)
        if raw_definition is None:
            raise TokenReferenceError(
                "QST_TOKEN_STATE_FSM_DEFINITION_MISSING",
                "state.fsm facade requires a definition parameter.",
            )
        definition = (
            raw_definition
            if isinstance(raw_definition, FSMDefinition)
            else FSMDefinition.model_validate(raw_definition)
        )
        return state_fsm(events, definition, **active_params)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported state token: {name}")


def evaluate_gate_token(name: str, values: Sequence[Any], **params: Any) -> list[Any]:
    """Evaluate Stage 3A.3 gate facade helpers and return DecisionV2 outputs."""

    token = _local_name(name, "gate.")

    if token == "cooldown":
        events = _string_values(values, field_name="gate.cooldown events")
        return _cooldown_gate(events)

    if token == "market_freeze":
        events = _string_values(values, field_name="gate.market_freeze events")
        return _market_freeze_gate(events)

    if token == "circuit_breaker":
        threshold = _coerce_int_param(params.get("threshold", 2), "threshold")
        if threshold <= 0:
            raise TokenReferenceError("QST_TOKEN_GATE_PARAM_INVALID", "threshold must be positive.")
        breaches = [_coerce_gate_int(value) for value in values]
        return _threshold_gate(breaches, threshold=threshold, reason="GATE_BLOCKED_CIRCUIT_BREAKER")

    if token == "observe_period":
        window = _coerce_int_param(params.get("window", 3), "window")
        if window <= 0:
            raise TokenReferenceError("QST_TOKEN_GATE_PARAM_INVALID", "window must be positive.")
        ticks = [_coerce_gate_int(value) for value in values]
        return _observe_period_gate(ticks, window=window)

    if token == "slot_budget":
        slot_budget = _coerce_int_param(params.get("slot_budget", 2), "slot_budget")
        if slot_budget < 0:
            raise TokenReferenceError(
                "QST_TOKEN_GATE_PARAM_INVALID",
                "slot_budget must be non-negative.",
            )
        consumed = [_coerce_gate_int(value) for value in values]
        return _slot_budget_gate(consumed, slot_budget=slot_budget)

    if token == "volatility_regime":
        max_volatility = _coerce_number(params.get("max_volatility"))
        if max_volatility < 0:
            raise TokenReferenceError(
                "QST_TOKEN_GATE_PARAM_INVALID",
                "max_volatility must be non-negative.",
            )
        return [
            _gate_decision(
                blocked=value > max_volatility,
                block_reason="GATE_BLOCKED_VOLATILITY_REGIME",
            )
            for value in _numeric_values(values, field_name="gate.volatility_regime volatility")
        ]

    if token == "drawdown":
        max_drawdown = _coerce_number(params.get("max_drawdown"))
        if max_drawdown < 0:
            raise TokenReferenceError(
                "QST_TOKEN_GATE_PARAM_INVALID",
                "max_drawdown must be non-negative.",
            )
        return [
            _gate_decision(
                blocked=abs(value) >= max_drawdown,
                block_reason="GATE_BLOCKED_DRAWDOWN",
            )
            for value in _numeric_values(values, field_name="gate.drawdown values")
        ]

    if token == "time_window":
        start = _coerce_hhmm(params.get("start_hhmm"), "start_hhmm")
        end = _coerce_hhmm(params.get("end_hhmm"), "end_hhmm")
        return [
            _gate_decision(
                blocked=not _hhmm_in_window(_timestamp_hhmm(timestamp), start=start, end=end),
                block_reason="GATE_BLOCKED_TIME_WINDOW",
            )
            for timestamp in _timestamp_values(values)
        ]

    if token == "rebalance":
        band = _coerce_decimal_param(params.get("band"), "band")
        if band < 0:
            raise TokenReferenceError(
                "QST_TOKEN_GATE_PARAM_INVALID",
                "band must be non-negative.",
            )
        return [
            _gate_decision(
                blocked=abs(value) < band,
                block_reason="GATE_BLOCKED_REBALANCE_BAND",
            )
            for value in _decimal_values(values, field_name="gate.rebalance drift")
        ]

    if token in {"min_hold", "max_hold"}:
        bars = _coerce_int_param(params.get("bars"), "bars")
        if bars <= 0:
            raise TokenReferenceError("QST_TOKEN_GATE_PARAM_INVALID", "bars must be positive.")
        reason = "GATE_BLOCKED_MIN_HOLD" if token == "min_hold" else "GATE_BLOCKED_MAX_HOLD_WAIT"
        return [
            _gate_decision(blocked=value < bars, block_reason=reason)
            for value in _numeric_values(values, field_name=f"gate.{token} age")
        ]

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported gate token: {name}")


def evaluate_panel_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate Stage 3A.4 panel facade helpers for conformance tests only."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    from qst.panel import (
        panel_bottom_k,
        panel_demean,
        panel_group_demean,
        panel_mask,
        panel_rank,
        panel_residualize,
        panel_top_k,
        panel_winsorize,
        panel_zscore,
        selection_to_weights,
    )

    if name == "panel.mask":
        panel, mask = _expect_args(name, args, 2)
        return panel_mask(panel, mask)
    if name in {"panel.rank", "panel.cross_sectional_rank"}:
        (panel,) = _expect_args(name, args, 1)
        return panel_rank(panel, **params)
    if name in {"panel.zscore", "panel.zscore_by_universe"}:
        (panel,) = _expect_args(name, args, 1)
        return panel_zscore(panel, **params)
    if name in {"panel.top_k", "selection.top_k"}:
        (panel,) = _expect_args(name, args, 1)
        return panel_top_k(panel, **params)
    if name in {"panel.bottom_k", "selection.bottom_k"}:
        (panel,) = _expect_args(name, args, 1)
        return panel_bottom_k(panel, **params)
    if name == "panel.demean":
        (panel,) = _expect_args(name, args, 1)
        return panel_demean(panel, **params)
    if name in {"panel.group_demean", "panel.neutralize_group"}:
        (panel,) = _expect_args(name, args, 1)
        return panel_group_demean(panel, **_strip_group_spec_ref(params))
    if name == "panel.winsorize":
        (panel,) = _expect_args(name, args, 1)
        return panel_winsorize(panel, **params)
    if name == "panel.residualize":
        (panel,) = _expect_args(name, args, 1)
        return panel_residualize(panel, **params)
    if name == "selection.to_weights":
        (selection,) = _expect_args(name, args, 1)
        return selection_to_weights(selection, **params)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported panel token: {name}")


def evaluate_factor_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate PR8 factor record helpers for conformance tests only."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    if name == "factor.sector_neutral_rank":
        (panel,) = _expect_args(name, args, 1)
        groups = _required_groups(params)
        cleaned_params = _strip_group_spec_ref(params)
        cleaned_params.pop("groups", None)
        neutral = cast(
            Any,
            evaluate_panel_token("panel.group_demean", panel, groups=groups, **cleaned_params),
        )
        if getattr(neutral, "diagnostics", None) and not neutral.diagnostics.ok:
            return neutral
        return evaluate_panel_token("panel.rank", neutral.panel, order=params.get("order", "descending"))

    if name == "factor.residualize":
        (panel,) = _expect_args(name, args, 1)
        return evaluate_panel_token("panel.residualize", panel, **params)

    if name == "factor.beta_neutral_signal":
        series, benchmark = _expect_args(name, args, 2)
        return evaluate_indicator_token("indicator.residual", series, benchmark, **params)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported factor token: {name}")


def evaluate_weight_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate Stage 3A.4 weight facade helpers for conformance tests only."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    from qst.panel import (
        PanelValue,
        SelectionPanelValue,
        WeightPanelValue,
        selection_to_weights,
        weight_cap_per_symbol,
        weight_market_neutral,
        weight_normalize_gross,
    )

    if name == "weight.equal_weight":
        (selection,) = _expect_args(name, args, 1)
        selection_panel = (
            selection
            if isinstance(selection, SelectionPanelValue)
            else SelectionPanelValue.model_validate(selection)
        )
        return selection_to_weights(selection_panel, method=params.get("method", "equal_long"))
    if name == "weight.rank_weight":
        (ranked,) = _expect_args(name, args, 1)
        panel = ranked if isinstance(ranked, PanelValue) else PanelValue.model_validate(ranked)
        return _weight_rank_weight(panel, **params)
    if name == "weight.inverse_vol_weight":
        (volatility,) = _expect_args(name, args, 1)
        panel = volatility if isinstance(volatility, PanelValue) else PanelValue.model_validate(volatility)
        return _weight_inverse_vol_weight(panel, **params)
    if name == "weight.vol_target_weight":
        weights, volatility = _expect_args(name, args, 2)
        return evaluate_risk_token("risk.volatility_target", weights, volatility, **params)
    if name in {"weight.normalize_gross"}:
        (weights,) = _expect_args(name, args, 1)
        return weight_normalize_gross(weights, **params)
    if name in {"weight.cap_per_symbol", "weight.max_weight_clip"}:
        (weights,) = _expect_args(name, args, 1)
        active_params = dict(params)
        if "max_abs_weight" not in active_params and "max_abs_weight" not in params:
            raise TokenReferenceError(
                "QST_TOKEN_WEIGHT_PARAM_MISSING",
                "weight.max_weight_clip requires max_abs_weight.",
            )
        return weight_cap_per_symbol(weights, **active_params)
    if name in {"weight.market_neutral", "weight.market_neutral_weight"}:
        (weights,) = _expect_args(name, args, 1)
        return weight_market_neutral(weights, **params)
    if name == "weight.group_neutral_weight":
        (weights,) = _expect_args(name, args, 1)
        weight_panel = weights if isinstance(weights, WeightPanelValue) else WeightPanelValue.model_validate(weights)
        return _weight_group_neutral(weight_panel, groups=_required_groups(params))
    if name == "weight.normalize_net":
        (weights,) = _expect_args(name, args, 1)
        weight_panel = weights if isinstance(weights, WeightPanelValue) else WeightPanelValue.model_validate(weights)
        return _weight_normalize_net(weight_panel, target_net=params.get("target_net", "0"))

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported weight token: {name}")


def evaluate_risk_token(name: str, *args: Any, **params: Any) -> TokenReferenceResult:
    """Evaluate Stage 3A.4 risk reference helpers without portfolio execution."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    if name == "risk.position_cap":
        decisions, positions = _expect_args(name, args, 2)
        return _risk_position_cap(decisions, positions, **params)
    if name == "risk.volatility_target":
        weights, volatility = _expect_args(name, args, 2)
        return _risk_volatility_target(weights, volatility, **params)
    if name == "risk.turnover_cap":
        weights, previous = _expect_args(name, args, 2)
        return _risk_turnover_cap(weights, previous, **params)
    if name == "risk.stop_loss_record":
        entry_price, current_price = _expect_args(name, args, 2)
        return _risk_stop_loss_record(entry_price, current_price, **params)
    if name == "risk.take_profit_record":
        entry_price, current_price = _expect_args(name, args, 2)
        return _risk_take_profit_record(entry_price, current_price, **params)
    if name == "risk.trailing_stop_record":
        (price,) = _expect_args(name, args, 1)
        return _risk_trailing_stop_record(price, **params)
    if name == "risk.max_drawdown_record":
        (equity,) = _expect_args(name, args, 1)
        return _risk_max_drawdown_record(equity, **params)
    if name == "risk.volatility_target_record":
        weights, volatility = _expect_args(name, args, 2)
        return _risk_volatility_target(weights, volatility, **params)
    if name == "risk.exposure_cap_record":
        decisions, exposures = _expect_args(name, args, 2)
        return _risk_position_cap(
            decisions,
            exposures,
            max_abs_position=params.get("max_abs_exposure"),
        )
    if name == "risk.turnover_limit_record":
        weights, previous = _expect_args(name, args, 2)
        return _risk_turnover_cap(weights, previous, **params)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported risk token: {name}")


def evaluate_data_token(name: str, series: Sequence[SeriesRow], **params: Any) -> NumericSeries:
    """Evaluate Stage 3A.2 `data.*` series reference helpers."""

    token = _local_name(name, "data.")
    rows = _numeric_series(series)

    if token == "identity":
        return rows

    if token == "shift":
        periods = _coerce_int_param(params.get("periods", 1), "periods")
        if periods < 0 and not bool(params.get("allow_unsafe_future", False)):
            raise TokenReferenceError(
                "QST_TOKEN_UNSAFE_FUTURE_SHIFT",
                "data.shift with negative periods uses future data.",
            )
        if periods == 0:
            return rows
        if periods > 0:
            return [(rows[index][0], rows[index - periods][1]) for index in range(periods, len(rows))]
        offset = abs(periods)
        return [(rows[index][0], rows[index + offset][1]) for index in range(0, len(rows) - offset)]

    if token in {"diff", "pct_change", "log_return"}:
        output: NumericSeries = []
        for index in range(1, len(rows)):
            timestamp, current = rows[index]
            previous = rows[index - 1][1]
            if token == "diff":
                value = current - previous
            elif token == "pct_change":
                if previous == 0:
                    raise TokenReferenceError(
                        "QST_TOKEN_DATA_DIVIDE_BY_ZERO",
                        "data.pct_change previous value cannot be zero.",
                    )
                value = current / previous - 1
            else:
                if current <= 0 or previous <= 0:
                    raise TokenReferenceError(
                        "QST_TOKEN_DATA_LOG_RETURN_DOMAIN_ERROR",
                        "data.log_return requires positive current and previous values.",
                    )
                value = math.log(current / previous)
            output.append((timestamp, _finite_result(value)))
        return output

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported data token: {name}")


def evaluate_time_token(name: str, series: Sequence[SeriesRow], **params: Any) -> NumericSeries:
    """Evaluate Stage 3A.2 `time.*` series reference helpers."""

    token = _local_name(name, "time.")
    rows = _numeric_series(series)

    if token == "session_filter":
        start = _coerce_hhmm(params.get("start_hhmm"), "start_hhmm")
        end = _coerce_hhmm(params.get("end_hhmm"), "end_hhmm")
        if start > end:
            raise TokenReferenceError(
                "QST_TOKEN_TIME_RANGE_INVALID",
                "time.session_filter requires start_hhmm <= end_hhmm.",
            )
        return [(timestamp, value) for timestamp, value in rows if start <= _timestamp_hhmm(timestamp) <= end]

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported time token: {name}")


def evaluate_align_token(
    name: str,
    series: Sequence[SeriesRow],
    other: Sequence[SeriesRow] | None = None,
) -> object:
    """Evaluate Stage 3A.2 `align.*` series reference helpers."""

    token = _local_name(name, "align.")

    if token in {"forward_fill", "drop_missing"}:
        rows = _numeric_series(series, allow_none=True)
        if token == "drop_missing":
            return [(timestamp, value) for timestamp, value in rows if value is not None]
        output: list[tuple[str, float | None]] = []
        last_seen: float | None = None
        for timestamp, value in rows:
            if value is not None:
                last_seen = value
            output.append((timestamp, last_seen))
        return output

    if other is None:
        raise TokenReferenceError(
            "QST_TOKEN_ARITY_INVALID",
            f"align.{token} requires left and right series.",
        )

    left_rows = _numeric_series(series)
    right_rows = _numeric_series(other)
    right_by_timestamp = dict(right_rows)

    if token == "inner_join":
        left: NumericSeries = []
        right: NumericSeries = []
        for timestamp, value in left_rows:
            if timestamp in right_by_timestamp:
                left.append((timestamp, value))
                right.append((timestamp, right_by_timestamp[timestamp]))
        return left, right

    if token == "left_join":
        left_output: NumericSeries = []
        right_output: list[tuple[str, float | None]] = []
        for timestamp, value in left_rows:
            left_output.append((timestamp, value))
            right_output.append((timestamp, right_by_timestamp.get(timestamp)))
        return left_output, right_output

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported align token: {name}")


def evaluate_window_token(name: str, series: Sequence[SeriesRow], **params: Any) -> object:
    """Evaluate Stage 3A.2 `window.*` trailing-window reference helpers."""

    token = _local_name(name, "window.")
    rows = _numeric_series(series)
    window, min_periods = _window_params(params)
    output: list[tuple[str, float | int]] = []

    for index, (timestamp, current) in enumerate(rows):
        values = [value for _, value in rows[max(0, index - window + 1) : index + 1]]
        if len(values) < min_periods:
            continue
        if token == "max":
            value: float | int = max(values)
        elif token == "min":
            value = min(values)
        elif token == "sum":
            value = sum(values)
        elif token == "count":
            value = len(values)
        elif token == "mean":
            value = sum(values) / len(values)
        elif token == "std":
            value = _population_std(values)
        elif token == "zscore":
            std = _population_std(values)
            value = 0 if std == 0 else (current - sum(values) / len(values)) / std
        else:
            raise TokenReferenceError(
                "QST_TOKEN_REFERENCE_UNSUPPORTED",
                f"Unsupported window token: {name}",
            )
        output.append((timestamp, value))
    return output


def evaluate_signal_token(name: str, *args: Any, **params: Any) -> object:
    """Evaluate Stage 3A.2 `signal.*`, `norm.*`, and `smooth.*` helpers."""

    if name.startswith("core."):
        name = name.removeprefix("core.")

    if name.startswith("signal."):
        token = name.removeprefix("signal.")
        if token in {"cross_above", "cross_below", "crosses"}:
            left, right = _expect_args(token, args, 2)
            joined_left, joined_right = cast(
                tuple[NumericSeries, NumericSeries],
                evaluate_align_token("align.inner_join", left, right),
            )
            output: BoolSeries = []
            for index in range(1, len(joined_left)):
                timestamp, current_left = joined_left[index]
                current_right = joined_right[index][1]
                previous_left = joined_left[index - 1][1]
                previous_right = joined_right[index - 1][1]
                crossed_above = previous_left <= previous_right and current_left > current_right
                crossed_below = previous_left >= previous_right and current_left < current_right
                if token == "cross_above":
                    value = crossed_above
                elif token == "cross_below":
                    value = crossed_below
                else:
                    value = crossed_above or crossed_below
                output.append((timestamp, value))
            return output

        if token in {"threshold_above", "threshold_below"}:
            (series,) = _expect_args(token, args, 1)
            threshold = _coerce_number(params.get("threshold"))
            inclusive = bool(params.get("inclusive", True))
            output = []
            for timestamp, numeric_value in _numeric_series(series):
                if token == "threshold_above":
                    passed = numeric_value >= threshold if inclusive else numeric_value > threshold
                else:
                    passed = numeric_value <= threshold if inclusive else numeric_value < threshold
                output.append((timestamp, passed))
            return output

        if token in {"greater_than", "less_than"}:
            left, right = _expect_args(token, args, 2)
            return _compare_series(left, right, op="gt" if token == "greater_than" else "lt")

        if token in {"and", "or"}:
            left, right = _expect_args(token, args, 2)
            return _combine_bool_series(left, right, op=token)

        if token == "not":
            (series,) = _expect_args(token, args, 1)
            return [(timestamp, not value) for timestamp, value in _bool_series(series)]

        if token in {"between", "outside_band"}:
            value, lower, upper = _expect_args(token, args, 3)
            return _band_signal(value, lower, upper, outside=token == "outside_band", inclusive=bool(params.get("inclusive", True)))

        if token in {"breakout_up", "breakout_down"}:
            series, band = _expect_args(token, args, 2)
            return _breakout_signal(series, band, direction="up" if token == "breakout_up" else "down")

        if token == "zscore_revert":
            (series,) = _expect_args(token, args, 1)
            threshold = _coerce_number(params.get("threshold", 2))
            if threshold <= 0:
                raise TokenReferenceError(
                    "QST_TOKEN_SIGNAL_THRESHOLD_INVALID",
                    "signal.zscore_revert threshold must be positive.",
                )
            return _zscore_revert_signal(series, threshold=threshold)

        if token == "rank_top_k":
            (panel,) = _expect_args(token, args, 1)
            return evaluate_panel_token("panel.top_k", panel, **params)

        if token == "rank_bottom_k":
            (panel,) = _expect_args(token, args, 1)
            return evaluate_panel_token("panel.bottom_k", panel, **params)

    if name.startswith("norm."):
        token = name.removeprefix("norm.")
        if token == "range_position":
            value_rows, high_rows, low_rows = _expect_args(token, args, 3)
            value_joined, high_joined = cast(
                tuple[NumericSeries, NumericSeries],
                evaluate_align_token("align.inner_join", value_rows, high_rows),
            )
            value_joined, low_joined = cast(
                tuple[NumericSeries, NumericSeries],
                evaluate_align_token("align.inner_join", value_joined, low_rows),
            )
            high_by_timestamp = dict(high_joined)
            low_by_timestamp = dict(low_joined)
            output_num: NumericSeries = []
            for timestamp, numeric_value in value_joined:
                high = high_by_timestamp[timestamp]
                low = low_by_timestamp[timestamp]
                if high == low:
                    raise TokenReferenceError(
                        "QST_TOKEN_SIGNAL_RANGE_ZERO",
                        "norm.range_position requires high != low.",
                    )
                output_num.append((timestamp, (numeric_value - low) / (high - low)))
            return output_num

    if name.startswith("smooth."):
        token = name.removeprefix("smooth.")
        if token == "linear_recursive":
            (series,) = _expect_args(token, args, 1)
            alpha = _coerce_number(params.get("alpha", 0.5))
            if alpha <= 0 or alpha > 1:
                raise TokenReferenceError(
                    "QST_TOKEN_SMOOTH_ALPHA_INVALID",
                    "smooth.linear_recursive requires 0 < alpha <= 1.",
                )
            output_num = []
            previous: float | None = None
            for timestamp, numeric_value in _numeric_series(series):
                previous = (
                    numeric_value
                    if previous is None
                    else alpha * numeric_value + (1 - alpha) * previous
                )
                output_num.append((timestamp, previous))
            return output_num

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported signal token: {name}")


def evaluate_indicator_token(
    name: str,
    series: Sequence[SeriesRow],
    *args: Sequence[SeriesRow],
    **params: Any,
) -> object:
    """Evaluate Stage 3A.2 `indicator.*` reference helpers."""

    token = _local_name(name, "indicator.")
    rows = _numeric_series(series)

    if token == "sma":
        return evaluate_window_token("window.mean", rows, **params)

    if token == "rolling_mean":
        return evaluate_window_token("window.mean", rows, **params)

    if token == "rolling_std":
        return evaluate_window_token("window.std", rows, **params)

    if token == "rolling_zscore":
        return evaluate_window_token("window.zscore", rows, **params)

    if token == "ema":
        window, _ = _window_params(params)
        alpha = 2 / (window + 1)
        output: NumericSeries = []
        previous: float | None = None
        for timestamp, value in rows:
            previous = value if previous is None else alpha * value + (1 - alpha) * previous
            output.append((timestamp, previous))
        return output

    if token == "rsi":
        window, _ = _window_params(params)
        if len(rows) <= window:
            return []
        gains: list[float] = []
        losses: list[float] = []
        output = []
        avg_gain: float | None = None
        avg_loss: float | None = None
        for index in range(1, len(rows)):
            change = rows[index][1] - rows[index - 1][1]
            gain = max(change, 0)
            loss = max(-change, 0)
            gains.append(gain)
            losses.append(loss)
            if index < window:
                continue
            if index == window:
                avg_gain = sum(gains[:window]) / window
                avg_loss = sum(losses[:window]) / window
            else:
                assert avg_gain is not None and avg_loss is not None
                avg_gain = (avg_gain * (window - 1) + gain) / window
                avg_loss = (avg_loss * (window - 1) + loss) / window
            timestamp = rows[index][0]
            output.append((timestamp, _rsi_from_avgs(avg_gain, avg_loss)))
        return output

    if token == "bollinger":
        width = _coerce_number(params.get("width", 2))
        mean_rows = cast(NumericSeries, evaluate_window_token("window.mean", rows, **params))
        std_rows = cast(NumericSeries, evaluate_window_token("window.std", rows, **params))
        std_by_timestamp = dict(std_rows)
        middle: NumericSeries = []
        upper: NumericSeries = []
        lower: NumericSeries = []
        for timestamp, mean_value in mean_rows:
            std = std_by_timestamp[timestamp]
            middle.append((timestamp, mean_value))
            upper.append((timestamp, mean_value + width * std))
            lower.append((timestamp, mean_value - width * std))
        return {"middle": middle, "upper": upper, "lower": lower}

    if token == "bollinger_band":
        return evaluate_indicator_token("indicator.bollinger", rows, **params)

    if token == "macd":
        return _macd_indicator(rows, **params)

    if token == "atr":
        low, close = _expect_args(token, tuple(args), 2)
        return _atr_indicator(rows, low, close, **params)

    if token == "donchian_channel":
        (low,) = _expect_args(token, tuple(args), 1)
        return _donchian_channel_indicator(rows, low, **params)

    if token == "volatility":
        return _volatility_indicator(rows, **params)

    if token == "linear_regression_slope":
        return _linear_regression_slope_indicator(rows, **params)

    if token == "beta":
        (benchmark,) = _expect_args(token, tuple(args), 1)
        return _beta_indicator(rows, benchmark, **params)

    if token == "residual":
        (benchmark,) = _expect_args(token, tuple(args), 1)
        return _residual_indicator(rows, benchmark, **params)

    raise TokenReferenceError("QST_TOKEN_REFERENCE_UNSUPPORTED", f"Unsupported indicator token: {name}")


def evaluate_channel_breakout_token(
    high: Sequence[SeriesRow],
    low: Sequence[SeriesRow],
    close: Sequence[SeriesRow],
    **params: Any,
) -> BoolSeries:
    """Evaluate `indicator.channel_breakout` using previous trailing high/low only."""

    window, _ = _window_params(params)
    joined_high, joined_low = cast(
        tuple[NumericSeries, NumericSeries],
        evaluate_align_token("align.inner_join", high, low),
    )
    joined_high, joined_close = cast(
        tuple[NumericSeries, NumericSeries],
        evaluate_align_token("align.inner_join", joined_high, close),
    )
    low_by_timestamp = dict(joined_low)
    close_by_timestamp = dict(joined_close)
    aligned = [
        (timestamp, high_value, low_by_timestamp[timestamp], close_by_timestamp[timestamp])
        for timestamp, high_value in joined_high
    ]
    output: BoolSeries = []
    for index in range(window, len(aligned)):
        previous = aligned[index - window : index]
        timestamp, _, _, close_value = aligned[index]
        previous_high = max(row[1] for row in previous)
        previous_low = min(row[2] for row in previous)
        output.append((timestamp, close_value > previous_high or close_value < previous_low))
    return output


def _macd_indicator(rows: NumericSeries, **params: Any) -> dict[str, NumericSeries]:
    fast_window = _coerce_int_param(params.get("fast_window", 12), "fast_window")
    slow_window = _coerce_int_param(params.get("slow_window", 26), "slow_window")
    signal_window = _coerce_int_param(params.get("signal_window", 9), "signal_window")
    if fast_window <= 0 or slow_window <= 0 or signal_window <= 0 or fast_window >= slow_window:
        raise TokenReferenceError(
            "QST_TOKEN_INDICATOR_PARAM_INVALID",
            "indicator.macd requires 0 < fast_window < slow_window and signal_window > 0.",
        )
    fast = dict(_ema_rows(rows, fast_window))
    slow_rows = _ema_rows(rows, slow_window)
    macd_rows = [(timestamp, fast[timestamp] - slow_value) for timestamp, slow_value in slow_rows]
    signal_rows = _ema_rows(macd_rows, signal_window)
    signal_by_timestamp = dict(signal_rows)
    histogram = [
        (timestamp, macd_value - signal_by_timestamp[timestamp])
        for timestamp, macd_value in macd_rows
        if timestamp in signal_by_timestamp
    ]
    return {"macd": macd_rows, "signal": signal_rows, "histogram": histogram}


def _ema_rows(rows: NumericSeries, window: int) -> NumericSeries:
    if window <= 0:
        raise TokenReferenceError("QST_TOKEN_WINDOW_INVALID", "EMA window must be positive.")
    alpha = 2 / (window + 1)
    output: NumericSeries = []
    previous: float | None = None
    for timestamp, value in rows:
        previous = value if previous is None else alpha * value + (1 - alpha) * previous
        output.append((timestamp, previous))
    return output


def _atr_indicator(
    high: NumericSeries,
    low: Sequence[SeriesRow],
    close: Sequence[SeriesRow],
    **params: Any,
) -> NumericSeries:
    window, _ = _window_params(params)
    high_rows, low_rows = _align_numeric_pair(high, low)
    high_rows, close_rows = _align_numeric_pair(high_rows, close)
    low_by_timestamp = dict(low_rows)
    close_by_timestamp = dict(close_rows)
    aligned = [
        (timestamp, high_value, low_by_timestamp[timestamp], close_by_timestamp[timestamp])
        for timestamp, high_value in high_rows
    ]
    true_ranges: NumericSeries = []
    previous_close: float | None = None
    for timestamp, high_value, low_value, close_value in aligned:
        if high_value < low_value:
            raise TokenReferenceError(
                "QST_TOKEN_INDICATOR_RANGE_INVALID",
                "indicator.atr requires high >= low.",
            )
        candidates = [high_value - low_value]
        if previous_close is not None:
            candidates.extend((abs(high_value - previous_close), abs(low_value - previous_close)))
        true_ranges.append((timestamp, max(candidates)))
        previous_close = close_value
    if len(true_ranges) < window:
        return []
    output: NumericSeries = []
    atr = sum(value for _, value in true_ranges[:window]) / window
    output.append((true_ranges[window - 1][0], atr))
    for timestamp, tr_value in true_ranges[window:]:
        atr = (atr * (window - 1) + tr_value) / window
        output.append((timestamp, atr))
    return output


def _donchian_channel_indicator(
    high: NumericSeries,
    low: Sequence[SeriesRow],
    **params: Any,
) -> dict[str, NumericSeries]:
    window, _ = _window_params(params)
    high_rows, low_rows = _align_numeric_pair(high, low)
    low_by_timestamp = dict(low_rows)
    aligned = [(timestamp, high_value, low_by_timestamp[timestamp]) for timestamp, high_value in high_rows]
    upper: NumericSeries = []
    lower: NumericSeries = []
    for index in range(window, len(aligned)):
        previous = aligned[index - window : index]
        timestamp = aligned[index][0]
        upper.append((timestamp, max(row[1] for row in previous)))
        lower.append((timestamp, min(row[2] for row in previous)))
    return {"upper": upper, "lower": lower}


def _volatility_indicator(rows: NumericSeries, **params: Any) -> NumericSeries:
    returns = evaluate_data_token("data.pct_change", rows)
    return cast(NumericSeries, evaluate_window_token("window.std", returns, **params))


def _linear_regression_slope_indicator(rows: NumericSeries, **params: Any) -> NumericSeries:
    window, _ = _window_params(params)
    output: NumericSeries = []
    for index in range(window - 1, len(rows)):
        values = [value for _, value in rows[index - window + 1 : index + 1]]
        output.append((rows[index][0], _ols_slope(values)))
    return output


def _beta_indicator(
    series: NumericSeries,
    benchmark: Sequence[SeriesRow],
    **params: Any,
) -> NumericSeries:
    return [(timestamp, beta) for timestamp, beta, _, _ in _rolling_beta_fit(series, benchmark, **params)]


def _residual_indicator(
    series: NumericSeries,
    benchmark: Sequence[SeriesRow],
    **params: Any,
) -> NumericSeries:
    return [
        (timestamp, y_value - (alpha + beta * x_value))
        for timestamp, beta, alpha, (x_value, y_value) in _rolling_beta_fit(series, benchmark, **params)
    ]


def _rolling_beta_fit(
    series: NumericSeries,
    benchmark: Sequence[SeriesRow],
    **params: Any,
) -> list[tuple[str, float, float, tuple[float, float]]]:
    window, _ = _window_params(params)
    series_rows, benchmark_rows = _align_numeric_pair(series, benchmark)
    benchmark_by_timestamp = dict(benchmark_rows)
    aligned = [(timestamp, benchmark_by_timestamp[timestamp], value) for timestamp, value in series_rows]
    output: list[tuple[str, float, float, tuple[float, float]]] = []
    for index in range(window - 1, len(aligned)):
        current_window = aligned[index - window + 1 : index + 1]
        xs = [row[1] for row in current_window]
        ys = [row[2] for row in current_window]
        mean_x = sum(xs) / len(xs)
        mean_y = sum(ys) / len(ys)
        var_x = sum((x - mean_x) ** 2 for x in xs)
        if var_x == 0:
            raise TokenReferenceError(
                "QST_TOKEN_INDICATOR_ZERO_VARIANCE",
                "indicator.beta/residual require nonzero benchmark variance.",
            )
        beta = sum((x - mean_x) * (y - mean_y) for x, y in zip(xs, ys, strict=True)) / var_x
        alpha = mean_y - beta * mean_x
        timestamp, current_x, current_y = aligned[index]
        output.append((timestamp, beta, alpha, (current_x, current_y)))
    return output


def _ols_slope(values: Sequence[float]) -> float:
    if len(values) < 2:
        raise TokenReferenceError(
            "QST_TOKEN_INDICATOR_INSUFFICIENT_OBSERVATIONS",
            "linear regression slope requires at least two observations.",
        )
    xs = list(range(len(values)))
    mean_x = sum(xs) / len(xs)
    mean_y = sum(values) / len(values)
    var_x = sum((x - mean_x) ** 2 for x in xs)
    if var_x == 0:
        raise TokenReferenceError(
            "QST_TOKEN_INDICATOR_ZERO_VARIANCE",
            "linear regression slope requires nonzero x variance.",
        )
    return sum((x - mean_x) * (y - mean_y) for x, y in zip(xs, values, strict=True)) / var_x


def _compare_series(left: Any, right: Any, *, op: Literal["gt", "lt"]) -> BoolSeries:
    left_rows, right_rows = _align_numeric_pair(left, right)
    right_by_timestamp = dict(right_rows)
    if op == "gt":
        return [(timestamp, value > right_by_timestamp[timestamp]) for timestamp, value in left_rows]
    return [(timestamp, value < right_by_timestamp[timestamp]) for timestamp, value in left_rows]


def _combine_bool_series(left: Any, right: Any, *, op: str) -> BoolSeries:
    left_rows, right_rows = _align_bool_pair(left, right)
    right_by_timestamp = dict(right_rows)
    if op == "and":
        return [(timestamp, value and right_by_timestamp[timestamp]) for timestamp, value in left_rows]
    return [(timestamp, value or right_by_timestamp[timestamp]) for timestamp, value in left_rows]


def _band_signal(value: Any, lower: Any, upper: Any, *, outside: bool, inclusive: bool) -> BoolSeries:
    value_rows, lower_rows = _align_numeric_pair(value, lower)
    value_rows, upper_rows = _align_numeric_pair(value_rows, upper)
    lower_by_timestamp = dict(lower_rows)
    upper_by_timestamp = dict(upper_rows)
    output: BoolSeries = []
    for timestamp, numeric_value in value_rows:
        low = lower_by_timestamp[timestamp]
        high = upper_by_timestamp[timestamp]
        if low > high:
            raise TokenReferenceError(
                "QST_TOKEN_SIGNAL_BAND_INVALID",
                "signal.between/outside_band require lower <= upper.",
            )
        inside = low <= numeric_value <= high if inclusive else low < numeric_value < high
        output.append((timestamp, not inside if outside else inside))
    return output


def _breakout_signal(series: Any, band: Any, *, direction: Literal["up", "down"]) -> BoolSeries:
    rows, band_rows = _align_numeric_pair(series, band)
    band_by_timestamp = dict(band_rows)
    if direction == "up":
        return [(timestamp, value > band_by_timestamp[timestamp]) for timestamp, value in rows]
    return [(timestamp, value < band_by_timestamp[timestamp]) for timestamp, value in rows]


def _zscore_revert_signal(series: Any, *, threshold: float) -> BoolSeries:
    rows = _numeric_series(series)
    output: BoolSeries = []
    for index in range(1, len(rows)):
        timestamp, current = rows[index]
        previous = rows[index - 1][1]
        same_side = previous == 0 or current == 0 or previous * current > 0
        output.append((timestamp, abs(previous) >= threshold and abs(current) < abs(previous) and same_side))
    return output


def _bool_series_to_decisions(
    rows: BoolSeries,
    *,
    true_kind: Literal["accept", "reject", "unknown", "block"],
    false_kind: Literal["accept", "reject", "unknown", "block"],
    true_reason: str,
    false_reason: str,
) -> list[Any]:
    from qst.decision import DecisionV2

    return [
        DecisionV2(kind=true_kind if value else false_kind, reasons=(true_reason if value else false_reason,))
        for _, value in rows
    ]


def _numeric_series_to_decisions(
    rows: NumericSeries,
    *,
    threshold: float,
    accept_reason: str,
    reject_reason: str,
) -> list[Any]:
    from qst.decision import DecisionV2

    return [
        DecisionV2(
            kind="accept" if value >= threshold else "reject",
            reasons=(accept_reason if value >= threshold else reject_reason,),
        )
        for _, value in rows
    ]


def _long_short_decisions(long_signal: BoolSeries, short_signal: BoolSeries) -> list[Any]:
    from qst.decision import DecisionV2

    long_rows, short_rows = _align_bool_pair(long_signal, short_signal)
    short_by_timestamp = dict(short_rows)
    output: list[DecisionV2] = []
    for timestamp, long_value in long_rows:
        short_value = short_by_timestamp[timestamp]
        if long_value and short_value:
            output.append(DecisionV2(kind="block", reasons=("DECISION_LONG_SHORT_CONFLICT",)))
        elif long_value:
            output.append(DecisionV2(kind="accept", reasons=("DECISION_LONG",)))
        elif short_value:
            output.append(DecisionV2(kind="accept", reasons=("DECISION_SHORT",)))
        else:
            output.append(DecisionV2(kind="reject", reasons=("DECISION_FLAT",)))
    return output


def _entry_exit_decisions(entry_signal: BoolSeries, exit_signal: BoolSeries) -> list[Any]:
    from qst.decision import DecisionV2

    entry_rows, exit_rows = _align_bool_pair(entry_signal, exit_signal)
    exit_by_timestamp = dict(exit_rows)
    in_position = False
    output: list[DecisionV2] = []
    for timestamp, entry_value in entry_rows:
        exit_value = exit_by_timestamp[timestamp]
        if entry_value and exit_value:
            output.append(DecisionV2(kind="block", reasons=("DECISION_ENTRY_EXIT_CONFLICT",)))
            continue
        if exit_value:
            in_position = False
            output.append(DecisionV2(kind="reject", reasons=("DECISION_EXIT",)))
            continue
        if entry_value:
            in_position = True
            output.append(DecisionV2(kind="accept", reasons=("DECISION_ENTRY",)))
            continue
        output.append(
            DecisionV2(
                kind="accept" if in_position else "reject",
                reasons=("DECISION_HOLD_LONG" if in_position else "DECISION_FLAT",),
            )
        )
    return output


def _gate_decisions(decision_values: list[Any], gate_values: list[Any]) -> list[Any]:
    from qst.decision import DecisionV2

    if len(decision_values) != len(gate_values):
        raise TokenReferenceError(
            "QST_TOKEN_DECISION_INPUT_LENGTH_MISMATCH",
            "decision.gate_decision inputs must have the same length.",
        )
    output: list[DecisionV2] = []
    for decision_raw, gate_raw in zip(decision_values, gate_values, strict=True):
        decision = decision_raw if isinstance(decision_raw, DecisionV2) else DecisionV2.model_validate(decision_raw)
        gate = gate_raw if isinstance(gate_raw, DecisionV2) else DecisionV2.model_validate(gate_raw)
        if gate.kind == "block":
            output.append(DecisionV2(kind="block", reasons=(*decision.reasons, *gate.reasons)))
        else:
            output.append(decision)
    return output


def _cooldown_gate(events: Sequence[str]) -> list[Any]:
    from qst.state import FSMDefinition, state_fsm

    definition = FSMDefinition.model_validate(
        {
            "states": ["ready", "cooldown"],
            "events": ["signal", "fill", "cooldown_expired"],
            "initial_state": "ready",
            "transitions": [
                {"from_state": "ready", "event": "signal", "to_state": "ready"},
                {"from_state": "ready", "event": "fill", "to_state": "cooldown"},
                {"from_state": "ready", "event": "cooldown_expired", "to_state": "ready"},
                {"from_state": "cooldown", "event": "signal", "to_state": "cooldown"},
                {"from_state": "cooldown", "event": "fill", "to_state": "cooldown"},
                {"from_state": "cooldown", "event": "cooldown_expired", "to_state": "ready"},
            ],
            "failure_policy": "raise",
        }
    )
    result = state_fsm(events, definition)
    return [
        _gate_decision(
            blocked=event == "signal" and state == "cooldown",
            block_reason="GATE_BLOCKED_COOLDOWN",
        )
        for event, state in zip(events, result.outputs, strict=True)
    ]


def _market_freeze_gate(events: Sequence[str]) -> list[Any]:
    from qst.state import FSMDefinition, state_fsm

    definition = FSMDefinition.model_validate(
        {
            "states": ["active", "frozen"],
            "events": ["signal", "freeze_on", "freeze_off"],
            "initial_state": "active",
            "transitions": [
                {"from_state": "active", "event": "signal", "to_state": "active"},
                {"from_state": "active", "event": "freeze_on", "to_state": "frozen"},
                {"from_state": "active", "event": "freeze_off", "to_state": "active"},
                {"from_state": "frozen", "event": "signal", "to_state": "frozen"},
                {"from_state": "frozen", "event": "freeze_on", "to_state": "frozen"},
                {"from_state": "frozen", "event": "freeze_off", "to_state": "active"},
            ],
            "failure_policy": "raise",
        }
    )
    result = state_fsm(events, definition)
    return [
        _gate_decision(
            blocked=event == "signal" and state == "frozen",
            block_reason="GATE_BLOCKED_MARKET_FREEZE",
        )
        for event, state in zip(events, result.outputs, strict=True)
    ]


def _threshold_gate(values: Sequence[int], *, threshold: int, reason: str) -> list[Any]:
    from qst.state import state_accumulate

    result = state_accumulate(values, reducer="sum", initial=0)
    return [
        _gate_decision(blocked=int(output) >= threshold, block_reason=reason)
        for output in result.outputs
    ]


def _observe_period_gate(ticks: Sequence[int], *, window: int) -> list[Any]:
    from qst.state import state_accumulate

    result = state_accumulate(ticks, reducer="sum", initial=0)
    return [
        _gate_decision(
            blocked=int(output) < window,
            block_reason="GATE_BLOCKED_OBSERVE_PERIOD",
        )
        for output in result.outputs
    ]


def _slot_budget_gate(consumed: Sequence[int], *, slot_budget: int) -> list[Any]:
    from qst.state import state_accumulate

    result = state_accumulate(consumed, reducer="sum", initial=0)
    return [
        _gate_decision(
            blocked=int(output) > slot_budget,
            block_reason="GATE_BLOCKED_SLOT_BUDGET",
        )
        for output in result.outputs
    ]


def _gate_decision(*, blocked: bool, block_reason: str) -> Any:
    from qst.decision import DecisionV2

    return DecisionV2(
        kind="block" if blocked else "accept",
        reasons=(block_reason if blocked else "GATE_ACCEPTED",),
    )


def _strip_group_spec_ref(params: dict[str, Any]) -> dict[str, Any]:
    return {
        key: value
        for key, value in params.items()
        if key not in {"group_spec_ref", "order"}
    }


def _required_groups(params: dict[str, Any]) -> Mapping[str, str]:
    groups = params.get("groups")
    if not isinstance(groups, Mapping) or not groups:
        raise TokenReferenceError(
            "QST_TOKEN_FACTOR_GROUPS_REQUIRED",
            "Factor/group neutral helpers require explicit groups metadata.",
        )
    output: dict[str, str] = {}
    for symbol, group in groups.items():
        if not isinstance(symbol, str) or not isinstance(group, str) or not symbol or not group:
            raise TokenReferenceError(
                "QST_TOKEN_FACTOR_GROUPS_INVALID",
                "groups must map non-empty symbol strings to non-empty group strings.",
            )
        output[symbol] = group
    return output


def _weight_rank_weight(panel: Any, **params: Any) -> Any:
    from qst.panel import PanelOperatorResult, WeightPanelValue, WeightPoint

    target_gross = _coerce_decimal_param(params.get("target_gross", "1"), "target_gross")
    if target_gross < 0:
        raise TokenReferenceError(
            "QST_TOKEN_WEIGHT_PARAM_INVALID",
            "weight.rank_weight target_gross must be non-negative.",
        )
    side = str(params.get("side", "long"))
    if side not in {"long", "short"}:
        raise TokenReferenceError("QST_TOKEN_WEIGHT_SIDE_INVALID", "side must be 'long' or 'short'.")
    sign = Decimal("1") if side == "long" else Decimal("-1")
    output: list[WeightPoint] = []
    for timestamp, rows in _panel_rows_by_time(panel).items():
        active = [row for row in rows if row.in_universe]
        ranks = [_coerce_decimal_value(row.value, f"{row.symbol} rank") for row in active]
        if any(rank <= 0 for rank in ranks):
            raise TokenReferenceError(
                "QST_TOKEN_WEIGHT_RANK_INVALID",
                "weight.rank_weight requires positive rank values.",
            )
        scores = [Decimal(len(ranks)) - rank + Decimal("1") for rank in ranks]
        score_sum = sum(scores, Decimal("0"))
        for row, score in zip(active, scores, strict=True):
            weight = Decimal("0") if score_sum == 0 else sign * target_gross * score / score_sum
            output.append(
                WeightPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    weight=_canonical_decimal(weight),
                    in_universe=row.in_universe,
                )
            )
        output.extend(
            WeightPoint(timestamp=row.timestamp, symbol=row.symbol, weight="0", in_universe=False)
            for row in rows
            if not row.in_universe
        )
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="raw", normalized=False),
        trace={"operator_id": "weight.rank_weight", "target_gross": _canonical_decimal(target_gross), "side": side},
    )


def _weight_inverse_vol_weight(panel: Any, **params: Any) -> Any:
    from qst.panel import PanelOperatorResult, WeightPanelValue, WeightPoint

    target_gross = _coerce_decimal_param(params.get("target_gross", "1"), "target_gross")
    if target_gross < 0:
        raise TokenReferenceError(
            "QST_TOKEN_WEIGHT_PARAM_INVALID",
            "weight.inverse_vol_weight target_gross must be non-negative.",
        )
    output: list[WeightPoint] = []
    for timestamp, rows in _panel_rows_by_time(panel).items():
        active = [row for row in rows if row.in_universe]
        vols = [_coerce_decimal_value(row.value, f"{row.symbol} volatility") for row in active]
        if any(vol <= 0 for vol in vols):
            raise TokenReferenceError(
                "QST_TOKEN_WEIGHT_VOLATILITY_NONPOSITIVE",
                "weight.inverse_vol_weight requires positive volatility values.",
            )
        inverse = [Decimal("1") / vol for vol in vols]
        inverse_sum = sum(inverse, Decimal("0"))
        for row, inv_value in zip(active, inverse, strict=True):
            weight = Decimal("0") if inverse_sum == 0 else target_gross * inv_value / inverse_sum
            output.append(
                WeightPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    weight=_canonical_decimal(weight),
                    in_universe=row.in_universe,
                )
            )
        output.extend(
            WeightPoint(timestamp=row.timestamp, symbol=row.symbol, weight="0", in_universe=False)
            for row in rows
            if not row.in_universe
        )
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="raw", normalized=False),
        trace={"operator_id": "weight.inverse_vol_weight", "target_gross": _canonical_decimal(target_gross)},
    )


def _weight_group_neutral(weights: Any, *, groups: Mapping[str, str]) -> Any:
    from qst.panel import PanelOperatorResult, WeightPanelValue, WeightPoint

    output: list[WeightPoint] = []
    for _timestamp, rows in _weight_rows_by_time(weights).items():
        active = [row for row in rows if row.in_universe]
        by_group: dict[str, list[Any]] = {}
        for row in active:
            group = groups.get(row.symbol)
            if group is None:
                raise TokenReferenceError(
                    "QST_TOKEN_FACTOR_GROUP_MISSING",
                    f"Missing group for symbol {row.symbol!r}.",
                )
            by_group.setdefault(group, []).append(row)
        for group_rows in by_group.values():
            mean = sum((_coerce_decimal_value(row.weight, row.symbol) for row in group_rows), Decimal("0")) / Decimal(
                len(group_rows)
            )
            for row in group_rows:
                output.append(row.model_copy(update={"weight": _canonical_decimal(_coerce_decimal_value(row.weight, row.symbol) - mean)}))
        output.extend(row.model_copy(update={"weight": "0"}) for row in rows if not row.in_universe)
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="raw", normalized=False),
        trace={"operator_id": "weight.group_neutral_weight", "group_count": len(set(groups.values()))},
    )


def _weight_normalize_net(weights: Any, *, target_net: Any) -> Any:
    from qst.panel import PanelOperatorResult, WeightPanelValue

    target = _coerce_decimal_param(target_net, "target_net")
    output = []
    for _timestamp, rows in _weight_rows_by_time(weights).items():
        active = [row for row in rows if row.in_universe]
        inactive = [row for row in rows if not row.in_universe]
        if not active:
            output.extend(inactive)
            continue
        net = sum((_coerce_decimal_value(row.weight, row.symbol) for row in active), Decimal("0"))
        shift = (target - net) / Decimal(len(active))
        output.extend(
            row.model_copy(update={"weight": _canonical_decimal(_coerce_decimal_value(row.weight, row.symbol) + shift)})
            for row in active
        )
        output.extend(inactive)
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="normalized", normalized=True),
        trace={"operator_id": "weight.normalize_net", "target_net": _canonical_decimal(target)},
    )


def _panel_rows_by_time(panel: Any) -> dict[str, list[Any]]:
    grouped: dict[str, list[Any]] = {}
    for row in panel.rows:
        grouped.setdefault(row.timestamp, []).append(row)
    return {timestamp: sorted(rows, key=lambda row: row.symbol) for timestamp, rows in sorted(grouped.items())}


def _weight_rows_by_time(weights: Any) -> dict[str, list[Any]]:
    grouped: dict[str, list[Any]] = {}
    for row in weights.rows:
        grouped.setdefault(row.timestamp, []).append(row)
    return {timestamp: sorted(rows, key=lambda row: row.symbol) for timestamp, rows in sorted(grouped.items())}


def _risk_position_cap(decisions: Any, positions: Any, **params: Any) -> TokenReferenceResult:
    from qst.decision import DecisionV2

    max_abs_position = _coerce_decimal_param(
        params.get("max_abs_position"),
        "max_abs_position",
    )
    if max_abs_position < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.position_cap max_abs_position must be non-negative.",
        )
    decision_values = list(decisions)
    position_values = [_coerce_decimal_value(value, "position") for value in positions]
    if len(decision_values) != len(position_values):
        raise TokenReferenceError(
            "QST_TOKEN_RISK_INPUT_LENGTH_MISMATCH",
            "risk.position_cap decisions and positions must have the same length.",
        )

    output: list[DecisionV2] = []
    capped_count = 0
    for decision, position in zip(decision_values, position_values, strict=True):
        if not isinstance(decision, DecisionV2):
            decision = DecisionV2.model_validate(decision)
        if decision.kind == "block" or abs(position) <= max_abs_position:
            output.append(decision)
            continue
        capped_count += 1
        output.append(
            DecisionV2(
                kind="block",
                reasons=(*decision.reasons, "RISK_POSITION_CAP_EXCEEDED"),
                score=decision.score,
            )
        )
    return TokenReferenceResult(
        decisions=tuple(output),
        trace={
            "operator_id": "risk.position_cap",
            "max_abs_position": _canonical_decimal(max_abs_position),
            "blocked_count": capped_count,
        },
    )


def _risk_volatility_target(weights: Any, volatility: Any, **params: Any) -> TokenReferenceResult:
    from qst.panel import WeightPanelValue, WeightPoint

    weight_panel = weights if isinstance(weights, WeightPanelValue) else WeightPanelValue.model_validate(weights)
    volatility_by_key = _panel_numeric_map(volatility)
    target = _coerce_decimal_param(params.get("target_volatility", "1"), "target_volatility")
    if target < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.volatility_target target_volatility must be non-negative.",
        )

    output: list[WeightPoint] = []
    diagnostics: list[Diagnostic] = []
    scaled_count = 0
    for row in weight_panel.rows:
        if not row.in_universe:
            output.append(row)
            continue
        key = (row.timestamp, row.symbol)
        vol = volatility_by_key.get(key)
        if vol is None:
            diagnostics.append(
                _reference_diagnostic(
                    "QST_TOKEN_RISK_VOLATILITY_MISSING",
                    f"Missing volatility for {key!r}.",
                )
            )
            continue
        if vol <= 0:
            diagnostics.append(
                _reference_diagnostic(
                    "QST_TOKEN_RISK_VOLATILITY_NONPOSITIVE",
                    f"Volatility must be positive for {key!r}.",
                )
            )
            continue
        scaled_count += 1
        scaled = Decimal(row.weight) * target / vol
        output.append(row.model_copy(update={"weight": _canonical_decimal(scaled)}))
    if diagnostics:
        return TokenReferenceResult(
            diagnostics=ValidationResult(diagnostics=diagnostics),
            trace={"operator_id": "risk.volatility_target"},
        )
    return TokenReferenceResult(
        weights=WeightPanelValue(
            rows=tuple(output),
            weight_kind=weight_panel.weight_kind,
            normalized=weight_panel.normalized,
        ),
        trace={
            "operator_id": "risk.volatility_target",
            "target_volatility": _canonical_decimal(target),
            "scaled_count": scaled_count,
            "normalization": "none",
        },
    )


def _risk_turnover_cap(weights: Any, previous: Any, **params: Any) -> TokenReferenceResult:
    from qst.panel import WeightPanelValue, WeightPoint

    weight_panel = weights if isinstance(weights, WeightPanelValue) else WeightPanelValue.model_validate(weights)
    previous_panel = (
        previous if isinstance(previous, WeightPanelValue) else WeightPanelValue.model_validate(previous)
    )
    previous_by_key = {
        (row.timestamp, row.symbol): Decimal(row.weight)
        for row in previous_panel.rows
        if row.in_universe
    }
    max_turnover = _coerce_decimal_param(params.get("max_turnover"), "max_turnover")
    if max_turnover < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.turnover_cap max_turnover must be non-negative.",
        )

    output: list[WeightPoint] = []
    diagnostics: list[Diagnostic] = []
    clipped_count = 0
    for row in weight_panel.rows:
        if not row.in_universe:
            output.append(row)
            continue
        key = (row.timestamp, row.symbol)
        if key not in previous_by_key:
            diagnostics.append(
                _reference_diagnostic(
                    "QST_TOKEN_RISK_PREVIOUS_WEIGHT_MISSING",
                    f"Missing previous weight for {key!r}.",
                )
            )
            continue
        previous_weight = previous_by_key[key]
        current_weight = Decimal(row.weight)
        delta = current_weight - previous_weight
        clipped_delta = min(max(delta, -max_turnover), max_turnover)
        if clipped_delta != delta:
            clipped_count += 1
        output.append(row.model_copy(update={"weight": _canonical_decimal(previous_weight + clipped_delta)}))
    if diagnostics:
        return TokenReferenceResult(
            diagnostics=ValidationResult(diagnostics=diagnostics),
            trace={"operator_id": "risk.turnover_cap"},
        )
    return TokenReferenceResult(
        weights=WeightPanelValue(
            rows=tuple(output),
            weight_kind=weight_panel.weight_kind,
            normalized=weight_panel.normalized,
        ),
        trace={
            "operator_id": "risk.turnover_cap",
            "max_turnover": _canonical_decimal(max_turnover),
            "clipped_count": clipped_count,
            "redistribution": "none",
        },
    )


def _risk_stop_loss_record(entry_price: Any, current_price: Any, **params: Any) -> TokenReferenceResult:
    from qst.decision import DecisionV2

    stop_loss_pct = _coerce_number(params.get("stop_loss_pct"))
    if stop_loss_pct < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.stop_loss_record stop_loss_pct must be non-negative.",
        )
    entries, currents = _aligned_numeric_values(entry_price, current_price)
    decisions: list[DecisionV2] = []
    triggered = 0
    for entry, current in zip(entries, currents, strict=True):
        breach = current <= entry * (1 - stop_loss_pct)
        if breach:
            triggered += 1
        decisions.append(
            DecisionV2(
                kind="block" if breach else "accept",
                reasons=("RISK_STOP_LOSS_BREACH" if breach else "RISK_ACCEPTED",),
            )
        )
    return TokenReferenceResult(
        decisions=tuple(decisions),
        trace={"operator_id": "risk.stop_loss_record", "triggered_count": triggered},
    )


def _risk_take_profit_record(entry_price: Any, current_price: Any, **params: Any) -> TokenReferenceResult:
    from qst.decision import DecisionV2

    take_profit_pct = _coerce_number(params.get("take_profit_pct"))
    if take_profit_pct < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.take_profit_record take_profit_pct must be non-negative.",
        )
    entries, currents = _aligned_numeric_values(entry_price, current_price)
    decisions: list[DecisionV2] = []
    triggered = 0
    for entry, current in zip(entries, currents, strict=True):
        hit = current >= entry * (1 + take_profit_pct)
        if hit:
            triggered += 1
        decisions.append(
            DecisionV2(
                kind="accept" if hit else "reject",
                reasons=("RISK_TAKE_PROFIT_TRIGGERED" if hit else "RISK_TAKE_PROFIT_NOT_TRIGGERED",),
            )
        )
    return TokenReferenceResult(
        decisions=tuple(decisions),
        trace={"operator_id": "risk.take_profit_record", "triggered_count": triggered},
    )


def _risk_trailing_stop_record(price: Any, **params: Any) -> TokenReferenceResult:
    from qst.decision import DecisionV2

    trail_pct = _coerce_number(params.get("trail_pct"))
    if trail_pct < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.trailing_stop_record trail_pct must be non-negative.",
        )
    prices = _numeric_values(price, field_name="risk.trailing_stop_record price")
    high_water: float | None = None
    decisions: list[DecisionV2] = []
    triggered = 0
    for value in prices:
        high_water = value if high_water is None else max(high_water, value)
        trigger_price = high_water * (1 - trail_pct)
        hit = value <= trigger_price and value < high_water
        if hit:
            triggered += 1
        decisions.append(
            DecisionV2(
                kind="block" if hit else "accept",
                reasons=("RISK_TRAILING_STOP_TRIGGERED" if hit else "RISK_ACCEPTED",),
            )
        )
    return TokenReferenceResult(
        decisions=tuple(decisions),
        trace={
            "operator_id": "risk.trailing_stop_record",
            "triggered_count": triggered,
            "order_execution": "none",
        },
    )


def _risk_max_drawdown_record(equity: Any, **params: Any) -> TokenReferenceResult:
    from qst.decision import DecisionV2

    max_drawdown = _coerce_number(params.get("max_drawdown"))
    if max_drawdown < 0:
        raise TokenReferenceError(
            "QST_TOKEN_RISK_PARAM_INVALID",
            "risk.max_drawdown_record max_drawdown must be non-negative.",
        )
    values = _numeric_values(equity, field_name="risk.max_drawdown_record equity")
    peak: float | None = None
    decisions: list[DecisionV2] = []
    blocked_count = 0
    for value in values:
        if value <= 0:
            raise TokenReferenceError(
                "QST_TOKEN_RISK_EQUITY_NONPOSITIVE",
                "risk.max_drawdown_record requires positive equity values.",
            )
        peak = value if peak is None else max(peak, value)
        drawdown = (peak - value) / peak
        breach = drawdown >= max_drawdown
        if breach:
            blocked_count += 1
        decisions.append(
            DecisionV2(
                kind="block" if breach else "accept",
                reasons=("RISK_MAX_DRAWDOWN_EXCEEDED" if breach else "RISK_ACCEPTED",),
            )
        )
    return TokenReferenceResult(
        decisions=tuple(decisions),
        trace={"operator_id": "risk.max_drawdown_record", "blocked_count": blocked_count},
    )


def _local_name(name: str, prefix: str) -> str:
    if name.startswith("core."):
        name = name.removeprefix("core.")
    if name.startswith(prefix):
        return name.removeprefix(prefix)
    raise TokenReferenceError(
        "QST_TOKEN_REFERENCE_UNSUPPORTED",
        f"Expected token name with prefix {prefix}: {name}",
    )


def _expect_args(token: str, args: tuple[Any, ...], count: int) -> tuple[Any, ...]:
    if len(args) != count:
        raise TokenReferenceError(
            "QST_TOKEN_ARITY_INVALID",
            f"{token} expected {count} argument(s), got {len(args)}.",
        )
    return args


def _iter_values(raw_values: Any) -> Iterable[Any]:
    if isinstance(raw_values, (str, bytes)):
        raise TokenReferenceError(
            "QST_TOKEN_INPUT_TYPE_INVALID",
            "Reduction input must be an iterable of values.",
        )
    try:
        iter(raw_values)
    except TypeError as exc:
        raise TokenReferenceError(
            "QST_TOKEN_INPUT_TYPE_INVALID",
            "Reduction input must be an iterable of values.",
        ) from exc
    return cast(Iterable[Any], raw_values)


def _coerce_number(value: Any, *, allow_nonfinite: bool = False) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise TokenReferenceError(
            "QST_TOKEN_NUMERIC_TYPE_INVALID",
            "Numeric token input must be int or float, not bool.",
        )
    result = float(value)
    if not allow_nonfinite and not math.isfinite(result):
        raise TokenReferenceError(
            "QST_TOKEN_NUMERIC_NONFINITE",
            "Numeric token input must be finite.",
        )
    return result


def _coerce_decimal_param(value: Any, name: str) -> Decimal:
    if value is None:
        raise TokenReferenceError(
            "QST_TOKEN_PARAM_TYPE_INVALID",
            f"{name} is required.",
        )
    return _coerce_decimal_value(value, name)


def _coerce_decimal_value(value: Any, name: str) -> Decimal:
    if isinstance(value, bool) or not isinstance(value, (int, float, str, Decimal)):
        raise TokenReferenceError(
            "QST_TOKEN_NUMERIC_TYPE_INVALID",
            f"{name} must be int, float, Decimal, or DecimalString, not bool.",
        )
    try:
        canonical = normalize_to_canonical(value)
    except ValueError as exc:
        raise TokenReferenceError(
            "QST_TOKEN_NUMERIC_NONFINITE",
            f"{name} must be finite canonical decimal material.",
        ) from exc
    return Decimal(canonical)


def _canonical_decimal(value: Decimal) -> str:
    return normalize_to_canonical(value)


def _coerce_int_param(value: Any, name: str) -> int:
    if isinstance(value, bool) or not isinstance(value, int):
        raise TokenReferenceError(
            "QST_TOKEN_PARAM_TYPE_INVALID",
            f"{name} must be an integer.",
        )
    return value


def _coerce_bool(value: Any) -> bool:
    if not isinstance(value, bool):
        raise TokenReferenceError(
            "QST_TOKEN_BOOL_TYPE_INVALID",
            "Boolean token input must be bool.",
        )
    return value


def _bool_values(values: Any) -> list[bool]:
    if isinstance(values, Sequence) and not isinstance(values, (str, bytes)):
        if all(isinstance(item, tuple) and len(item) == 2 for item in values):
            return [_coerce_bool(value) for _, value in _series_rows(cast(Sequence[SeriesRow], values))]
        return [_coerce_bool(value) for value in values]
    raise TokenReferenceError(
        "QST_TOKEN_INPUT_TYPE_INVALID",
        "Boolean series input must be a sequence.",
    )


def _bool_series(values: Any) -> BoolSeries:
    if isinstance(values, Sequence) and not isinstance(values, (str, bytes)):
        if all(isinstance(item, tuple) and len(item) == 2 for item in values):
            return [
                (timestamp, _coerce_bool(value))
                for timestamp, value in _series_rows(cast(Sequence[SeriesRow], values))
            ]
        return [(str(index), _coerce_bool(value)) for index, value in enumerate(values)]
    raise TokenReferenceError(
        "QST_TOKEN_INPUT_TYPE_INVALID",
        "Boolean series input must be a sequence.",
    )


def _align_numeric_pair(left: Any, right: Any) -> tuple[NumericSeries, NumericSeries]:
    left_rows = _numeric_series(left)
    right_rows = _numeric_series(right)
    right_by_timestamp = dict(right_rows)
    left_output: NumericSeries = []
    right_output: NumericSeries = []
    for timestamp, value in left_rows:
        if timestamp in right_by_timestamp:
            left_output.append((timestamp, value))
            right_output.append((timestamp, right_by_timestamp[timestamp]))
    return left_output, right_output


def _align_bool_pair(left: Any, right: Any) -> tuple[BoolSeries, BoolSeries]:
    left_rows = _bool_series(left)
    right_rows = _bool_series(right)
    right_by_timestamp = dict(right_rows)
    left_output: BoolSeries = []
    right_output: BoolSeries = []
    for timestamp, value in left_rows:
        if timestamp in right_by_timestamp:
            left_output.append((timestamp, value))
            right_output.append((timestamp, right_by_timestamp[timestamp]))
    return left_output, right_output


def _string_values(values: Sequence[Any], *, field_name: str) -> list[str]:
    output: list[str] = []
    for value in values:
        if not isinstance(value, str) or not value:
            raise TokenReferenceError(
                "QST_TOKEN_GATE_EVENT_INVALID",
                f"{field_name} must contain non-empty string events.",
            )
        output.append(value)
    return output


def _coerce_gate_int(value: Any) -> int:
    if isinstance(value, bool) or not isinstance(value, int):
        raise TokenReferenceError(
            "QST_TOKEN_GATE_INPUT_INVALID",
            "Gate numeric inputs must be integers, not bool.",
        )
    return value


def _numeric_values(values: Sequence[Any], *, field_name: str) -> list[float]:
    if isinstance(values, (str, bytes)):
        raise TokenReferenceError(
            "QST_TOKEN_INPUT_TYPE_INVALID",
            f"{field_name} must be a sequence of numeric values.",
        )
    if all(isinstance(item, tuple) and len(item) == 2 for item in values):
        return [
            value
            for _timestamp, value in _numeric_series(cast(Sequence[SeriesRow], values))
        ]
    return [_coerce_number(value) for value in values]


def _decimal_values(values: Sequence[Any], *, field_name: str) -> list[Decimal]:
    if isinstance(values, (str, bytes)):
        raise TokenReferenceError(
            "QST_TOKEN_INPUT_TYPE_INVALID",
            f"{field_name} must be a sequence of decimal-compatible values.",
        )
    if all(isinstance(item, tuple) and len(item) == 2 for item in values):
        return [
            _coerce_decimal_value(value, field_name)
            for _timestamp, value in _series_rows(cast(Sequence[SeriesRow], values))
        ]
    return [_coerce_decimal_value(value, field_name) for value in values]


def _timestamp_values(values: Sequence[Any]) -> list[str]:
    if isinstance(values, (str, bytes)):
        raise TokenReferenceError(
            "QST_TOKEN_INPUT_TYPE_INVALID",
            "Timestamp gate input must be a sequence.",
        )
    timestamps: list[str] = []
    for item in values:
        if isinstance(item, tuple) and len(item) == 2:
            timestamp = item[0]
        else:
            timestamp = item
        if not isinstance(timestamp, str) or not timestamp:
            raise TokenReferenceError(
                "QST_TOKEN_SERIES_TIMESTAMP_INVALID",
                "Timestamp gate input must contain non-empty timestamp strings.",
            )
        timestamps.append(timestamp)
    return timestamps


def _hhmm_in_window(value: str, *, start: str, end: str) -> bool:
    if start <= end:
        return start <= value <= end
    return value >= start or value <= end


def _aligned_numeric_values(left: Any, right: Any) -> tuple[list[float], list[float]]:
    left_is_series = isinstance(left, Sequence) and not isinstance(left, (str, bytes)) and all(
        isinstance(item, tuple) and len(item) == 2 for item in left
    )
    right_is_series = isinstance(right, Sequence) and not isinstance(right, (str, bytes)) and all(
        isinstance(item, tuple) and len(item) == 2 for item in right
    )
    if left_is_series and right_is_series:
        left_rows, right_rows = _align_numeric_pair(left, right)
        return [value for _timestamp, value in left_rows], [value for _timestamp, value in right_rows]
    left_values = _numeric_values(left, field_name="left risk input")
    right_values = _numeric_values(right, field_name="right risk input")
    if len(left_values) != len(right_values):
        raise TokenReferenceError(
            "QST_TOKEN_RISK_INPUT_LENGTH_MISMATCH",
            "Risk record numeric inputs must have the same length.",
        )
    return left_values, right_values


def _finite_result(value: float) -> float:
    if not math.isfinite(value):
        raise TokenReferenceError(
            "QST_TOKEN_NUMERIC_NONFINITE",
            "Numeric token output must be finite.",
        )
    return value


@overload
def _numeric_series(series: Sequence[SeriesRow], *, allow_none: Literal[False] = False) -> NumericSeries:
    ...


@overload
def _numeric_series(series: Sequence[SeriesRow], *, allow_none: Literal[True]) -> list[tuple[str, float | None]]:
    ...


def _numeric_series(series: Sequence[SeriesRow], *, allow_none: bool = False) -> Any:
    rows = _series_rows(series)
    output: list[tuple[str, float | None]] = []
    for timestamp, value in rows:
        if value is None and allow_none:
            output.append((timestamp, None))
        else:
            output.append((timestamp, _coerce_number(value)))
    return output


def _panel_numeric_map(panel: Any) -> dict[tuple[str, str], Decimal]:
    from qst.panel import PanelValue

    panel_value = panel if isinstance(panel, PanelValue) else PanelValue.model_validate(panel)
    output: dict[tuple[str, str], Decimal] = {}
    for row in panel_value.rows:
        if not row.in_universe:
            continue
        if row.value is None:
            raise TokenReferenceError(
                "QST_TOKEN_RISK_PANEL_VALUE_MISSING",
                f"Missing active risk panel value for {(row.timestamp, row.symbol)!r}.",
            )
        output[(row.timestamp, row.symbol)] = _coerce_decimal_value(
            row.value,
            f"{row.timestamp}/{row.symbol}",
        )
    return output


def _series_rows(series: Sequence[SeriesRow]) -> list[SeriesRow]:
    rows: list[SeriesRow] = []
    seen: set[str] = set()
    for raw_timestamp, value in series:
        if not isinstance(raw_timestamp, str) or not raw_timestamp:
            raise TokenReferenceError(
                "QST_TOKEN_SERIES_TIMESTAMP_INVALID",
                "Series timestamp must be a non-empty string.",
            )
        if raw_timestamp in seen:
            raise TokenReferenceError(
                "QST_TOKEN_SERIES_DUPLICATE_TIMESTAMP",
                f"Duplicate timestamp: {raw_timestamp}",
            )
        seen.add(raw_timestamp)
        rows.append((raw_timestamp, value))
    return sorted(rows, key=lambda row: row[0])


def _coerce_hhmm(value: Any, name: str) -> str:
    if not isinstance(value, str) or len(value) != 5 or value[2] != ":":
        raise TokenReferenceError(
            "QST_TOKEN_TIME_FORMAT_INVALID",
            f"{name} must use HH:MM format.",
        )
    hour, minute = value.split(":")
    if not hour.isdigit() or not minute.isdigit():
        raise TokenReferenceError("QST_TOKEN_TIME_FORMAT_INVALID", f"{name} must use HH:MM format.")
    if not (0 <= int(hour) <= 23 and 0 <= int(minute) <= 59):
        raise TokenReferenceError("QST_TOKEN_TIME_FORMAT_INVALID", f"{name} must be a valid time.")
    return value


def _timestamp_hhmm(timestamp: str) -> str:
    if "T" not in timestamp:
        raise TokenReferenceError(
            "QST_TOKEN_SERIES_TIMESTAMP_INVALID",
            "Timestamp must contain a T separator for session filtering.",
        )
    time_part = timestamp.split("T", 1)[1]
    if len(time_part) < 5:
        raise TokenReferenceError(
            "QST_TOKEN_SERIES_TIMESTAMP_INVALID",
            "Timestamp time component must include HH:MM.",
        )
    return _coerce_hhmm(time_part[:5], "timestamp")


def _window_params(params: dict[str, Any]) -> tuple[int, int]:
    window = _coerce_int_param(params.get("window"), "window")
    if window <= 0:
        raise TokenReferenceError("QST_TOKEN_WINDOW_INVALID", "window must be positive.")
    raw_min_periods = params.get("min_periods", window)
    min_periods = _coerce_int_param(raw_min_periods, "min_periods")
    if min_periods <= 0:
        raise TokenReferenceError("QST_TOKEN_WINDOW_INVALID", "min_periods must be positive.")
    return window, min_periods


def _population_std(values: Sequence[float]) -> float:
    mean = sum(values) / len(values)
    return math.sqrt(sum((value - mean) ** 2 for value in values) / len(values))


def _rsi_from_avgs(avg_gain: float, avg_loss: float) -> float:
    if avg_gain == 0 and avg_loss == 0:
        return 50
    if avg_loss == 0:
        return 100
    rs = avg_gain / avg_loss
    return 100 - 100 / (1 + rs)


def _reference_diagnostic(code: str, message: str) -> Diagnostic:
    return Diagnostic(code=code, severity="error", phase="runtime", message=message)
