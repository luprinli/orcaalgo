"""Deterministic reference semantics for Token System v2 WP8c Panel operators."""

from __future__ import annotations

import math
from collections import defaultdict
from collections.abc import Mapping
from decimal import Decimal
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.artifacts.decimal_string import DecimalString, normalize_to_canonical
from qst.panel.model import MissingPolicyKind, UniverseMask
from qst.validation import Diagnostic, ValidationResult

PanelOperatorName = Literal[
    "panel.mask",
    "panel.rank",
    "panel.zscore",
    "panel.top_k",
    "panel.bottom_k",
    "panel.demean",
    "panel.group_demean",
    "panel.winsorize",
    "panel.residualize",
    "selection.to_weights",
]
WeightOperatorName = Literal[
    "weight.normalize_gross",
    "weight.cap_per_symbol",
    "weight.market_neutral",
]
TiePolicy = Literal["stable_symbol_order"]
PanelOrder = Literal["ascending", "descending"]
ZeroVariancePolicy = Literal["output_zero"]
SelectionSizePolicy = Literal["allow_smaller"]
InsufficientObservationsPolicy = Literal["unknown", "error"]
RawWeightMethod = Literal["equal_long", "equal_short", "equal_long_short"]
ZeroGrossPolicy = Literal["keep_zero", "error"]

PANEL_OPERATOR_TOKENS: tuple[PanelOperatorName, ...] = (
    "panel.mask",
    "panel.rank",
    "panel.zscore",
    "panel.top_k",
    "panel.bottom_k",
    "panel.demean",
    "panel.group_demean",
    "panel.winsorize",
    "panel.residualize",
    "selection.to_weights",
)
PANEL_OPS_PACK_ID = "qst-tokenpack-panel-ops"
PANEL_OPS_PACK_VERSION = "0.1.0"
WEIGHT_OPERATOR_TOKENS: tuple[WeightOperatorName, ...] = (
    "weight.normalize_gross",
    "weight.cap_per_symbol",
    "weight.market_neutral",
)
PANEL_WEIGHTS_PACK_ID = "qst-tokenpack-panel-weights"
PANEL_WEIGHTS_PACK_VERSION = "0.1.0"


class PanelPoint(BaseModel):
    """One long-format Panel value cell."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    timestamp: str = Field(min_length=1)
    symbol: str = Field(min_length=1)
    value: DecimalString | None
    in_universe: bool = True


class PanelValue(BaseModel):
    """Small deterministic long-format Panel value used by reference helpers."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    rows: tuple[PanelPoint, ...] = Field(default_factory=tuple)

    @model_validator(mode="before")
    @classmethod
    def _sort_rows(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        rows = tuple(
            PanelPoint.model_validate(row) if not isinstance(row, PanelPoint) else row
            for row in value.get("rows", ())
        )
        seen: set[tuple[str, str]] = set()
        for row in rows:
            key = (row.timestamp, row.symbol)
            if key in seen:
                raise ValueError(f"Duplicate Panel row for timestamp/symbol {key!r}")
            seen.add(key)
        return {
            **value,
            "rows": tuple(sorted(rows, key=lambda row: (row.timestamp, row.symbol))),
        }


class SelectionPoint(BaseModel):
    """One long-format selection cell."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    timestamp: str = Field(min_length=1)
    symbol: str = Field(min_length=1)
    selected: bool
    side: Literal["long", "short", "both"] | None = None
    score: DecimalString | None = None
    in_universe: bool = True


class SelectionPanelValue(BaseModel):
    """Reference SelectionPanel value."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    rows: tuple[SelectionPoint, ...] = Field(default_factory=tuple)
    selection_kind: Literal["long_only", "short_only", "long_short", "ranked"] = "long_only"

    @model_validator(mode="before")
    @classmethod
    def _sort_rows(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        rows = tuple(
            SelectionPoint.model_validate(row) if not isinstance(row, SelectionPoint) else row
            for row in value.get("rows", ())
        )
        seen: set[tuple[str, str]] = set()
        for row in rows:
            key = (row.timestamp, row.symbol)
            if key in seen:
                raise ValueError(f"Duplicate SelectionPanel row for timestamp/symbol {key!r}")
            seen.add(key)
        return {
            **value,
            "rows": tuple(sorted(rows, key=lambda row: (row.timestamp, row.symbol))),
        }


class WeightPoint(BaseModel):
    """One long-format weight cell."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    timestamp: str = Field(min_length=1)
    symbol: str = Field(min_length=1)
    weight: DecimalString
    in_universe: bool = True


class WeightPanelValue(BaseModel):
    """Reference WeightPanel value used by WP8c/WP8d helpers."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    rows: tuple[WeightPoint, ...] = Field(default_factory=tuple)
    weight_kind: Literal["raw", "normalized"] = "raw"
    normalized: bool = False

    @model_validator(mode="before")
    @classmethod
    def _sort_rows(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        rows = tuple(
            WeightPoint.model_validate(row) if not isinstance(row, WeightPoint) else row
            for row in value.get("rows", ())
        )
        seen: set[tuple[str, str]] = set()
        for row in rows:
            key = (row.timestamp, row.symbol)
            if key in seen:
                raise ValueError(f"Duplicate WeightPanel row for timestamp/symbol {key!r}")
            seen.add(key)
        return {
            **value,
            "rows": tuple(sorted(rows, key=lambda row: (row.timestamp, row.symbol))),
        }


class PanelOperatorResult(BaseModel):
    """Reference helper result with structured diagnostics and trace material."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    panel: PanelValue | None = None
    selection: SelectionPanelValue | None = None
    weights: WeightPanelValue | None = None
    diagnostics: ValidationResult = Field(default_factory=ValidationResult)
    trace: dict[str, Any] = Field(default_factory=dict)


def panel_mask(panel: PanelValue, mask: UniverseMask | SelectionPanelValue) -> PanelOperatorResult:
    """Apply a UniverseMask or SelectionPanel to a Panel without changing value type."""

    if isinstance(mask, UniverseMask):
        included = set(mask.included)
        rows = tuple(
            row.model_copy(update={"in_universe": row.in_universe and row.symbol in included})
            for row in panel.rows
        )
        mask_kind = "universe_mask"
    else:
        selected = {
            (row.timestamp, row.symbol)
            for row in mask.rows
            if row.in_universe and row.selected
        }
        rows = tuple(
            row.model_copy(update={"in_universe": row.in_universe and (row.timestamp, row.symbol) in selected})
            for row in panel.rows
        )
        mask_kind = "selection_panel"
    return _panel_result(
        "panel.mask",
        PanelValue(rows=rows),
        trace_extra={"mask_kind": mask_kind},
    )


def panel_rank(
    panel: PanelValue,
    *,
    missing_policy: MissingPolicyKind = "error_on_missing",
    order: PanelOrder = "descending",
    tie_policy: TiePolicy = "stable_symbol_order",
    rank_base: int = 1,
) -> PanelOperatorResult:
    """Rank values cross-sectionally per timestamp."""

    if rank_base != 1:
        return _error("QST_V2_PANEL_RANK_BASE_UNSUPPORTED", "panel.rank rank_base must be 1.")
    if tie_policy != "stable_symbol_order":
        return _error("QST_V2_PANEL_TIE_POLICY_UNSUPPORTED", "WP8c only supports stable_symbol_order.")
    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    if diagnostics:
        return _diagnostics("panel.rank", diagnostics)
    output: list[PanelPoint] = []
    reverse = order == "descending"
    for timestamp, rows in grouped.items():
        sorted_rows = sorted(rows, key=lambda item: ((-item[1] if reverse else item[1]), item[0].symbol))
        for offset, (row, _value) in enumerate(sorted_rows):
            output.append(
                PanelPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    value=str(rank_base + offset),
                    in_universe=row.in_universe,
                )
            )
    return _panel_result(
        "panel.rank",
        PanelValue(rows=tuple(output)),
        trace_extra={"order": order, "tie_policy": tie_policy, "rank_base": rank_base},
    )


def panel_zscore(
    panel: PanelValue,
    *,
    missing_policy: MissingPolicyKind = "error_on_missing",
    ddof: int = 0,
    zero_variance_policy: ZeroVariancePolicy = "output_zero",
) -> PanelOperatorResult:
    """Compute per-timestamp cross-sectional z-scores."""

    if ddof != 0:
        return _error("QST_V2_PANEL_ZSCORE_DDOF_UNSUPPORTED", "WP8c panel.zscore requires ddof=0.")
    if zero_variance_policy != "output_zero":
        return _error(
            "QST_V2_PANEL_ZERO_VARIANCE_POLICY_UNSUPPORTED",
            "WP8c panel.zscore requires zero_variance_policy=output_zero.",
        )
    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    if diagnostics:
        return _diagnostics("panel.zscore", diagnostics)
    output: list[PanelPoint] = []
    for timestamp, rows in grouped.items():
        values = [value for _row, value in rows]
        if not values:
            continue
        mean = sum(values) / len(values)
        variance = sum((value - mean) ** 2 for value in values) / len(values)
        std = math.sqrt(variance)
        for row, value in rows:
            zscore = 0.0 if std == 0.0 else (value - mean) / std
            output.append(
                PanelPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    value=_canonical_number(zscore),
                    in_universe=row.in_universe,
                )
            )
    return _panel_result(
        "panel.zscore",
        PanelValue(rows=tuple(output)),
        trace_extra={"ddof": ddof, "zero_variance_policy": zero_variance_policy},
    )


def panel_demean(
    panel: PanelValue,
    *,
    missing_policy: MissingPolicyKind = "error_on_missing",
) -> PanelOperatorResult:
    """Subtract the per-timestamp cross-sectional mean."""

    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    if diagnostics:
        return _diagnostics("panel.demean", diagnostics)
    output: list[PanelPoint] = []
    for timestamp, rows in grouped.items():
        values = [value for _row, value in rows]
        if not values:
            continue
        mean = sum(values) / len(values)
        output.extend(
            PanelPoint(
                timestamp=timestamp,
                symbol=row.symbol,
                value=_canonical_number(value - mean),
                in_universe=row.in_universe,
            )
            for row, value in rows
        )
    return _panel_result("panel.demean", PanelValue(rows=tuple(output)))


def panel_top_k(
    panel: PanelValue,
    *,
    k: int,
    missing_policy: MissingPolicyKind = "error_on_missing",
    tie_policy: TiePolicy = "stable_symbol_order",
    selection_size_policy: SelectionSizePolicy = "allow_smaller",
) -> PanelOperatorResult:
    """Select the top-k values per timestamp."""

    return _top_bottom_k(
        panel,
        k=k,
        operator_id="panel.top_k",
        descending=True,
        missing_policy=missing_policy,
        tie_policy=tie_policy,
        selection_size_policy=selection_size_policy,
    )


def panel_bottom_k(
    panel: PanelValue,
    *,
    k: int,
    missing_policy: MissingPolicyKind = "error_on_missing",
    tie_policy: TiePolicy = "stable_symbol_order",
    selection_size_policy: SelectionSizePolicy = "allow_smaller",
) -> PanelOperatorResult:
    """Select the bottom-k values per timestamp."""

    return _top_bottom_k(
        panel,
        k=k,
        operator_id="panel.bottom_k",
        descending=False,
        missing_policy=missing_policy,
        tie_policy=tie_policy,
        selection_size_policy=selection_size_policy,
    )


def panel_group_demean(
    panel: PanelValue,
    *,
    groups: Mapping[str, str],
    missing_policy: MissingPolicyKind = "error_on_missing",
    missing_group_policy: Literal["error", "drop", "assign_unknown"] = "error",
) -> PanelOperatorResult:
    """Subtract per-timestamp, per-group means."""

    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    diagnostics = list(diagnostics)
    if diagnostics:
        return _diagnostics("panel.group_demean", diagnostics)
    output: list[PanelPoint] = []
    for timestamp, rows in grouped.items():
        by_group: dict[str, list[tuple[PanelPoint, float]]] = defaultdict(list)
        for row, value in rows:
            group = groups.get(row.symbol)
            if group is None:
                if missing_group_policy == "error":
                    diagnostics.append(
                        _diagnostic(
                            "QST_V2_PANEL_GROUP_MISSING",
                            f"Missing group for symbol {row.symbol!r}.",
                        )
                    )
                    continue
                if missing_group_policy == "drop":
                    continue
                group = "__unknown__"
            by_group[group].append((row, value))
        for group_rows in by_group.values():
            mean = sum(value for _row, value in group_rows) / len(group_rows)
            output.extend(
                PanelPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    value=_canonical_number(value - mean),
                    in_universe=row.in_universe,
                )
                for row, value in group_rows
            )
    if diagnostics:
        return _diagnostics("panel.group_demean", diagnostics)
    return _panel_result(
        "panel.group_demean",
        PanelValue(rows=tuple(output)),
        trace_extra={"group_count": len(set(groups.values())), "missing_group_policy": missing_group_policy},
    )


def panel_winsorize(
    panel: PanelValue,
    *,
    lower_quantile: DecimalString = "0.01",
    upper_quantile: DecimalString = "0.99",
    missing_policy: MissingPolicyKind = "error_on_missing",
) -> PanelOperatorResult:
    """Winsorize per timestamp with deterministic nearest-rank quantile bounds."""

    lower = float(Decimal(lower_quantile))
    upper = float(Decimal(upper_quantile))
    if lower < 0 or upper > 1 or lower > upper:
        return _error(
            "QST_V2_PANEL_WINSORIZE_QUANTILE_INVALID",
            "winsorize requires 0 <= lower_quantile <= upper_quantile <= 1.",
        )
    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    if diagnostics:
        return _diagnostics("panel.winsorize", diagnostics)
    output: list[PanelPoint] = []
    for timestamp, rows in grouped.items():
        sorted_rows = sorted(rows, key=lambda item: (item[1], item[0].symbol))
        n = len(sorted_rows)
        if n == 0:
            continue
        lower_index = _nearest_rank_index(lower, n)
        upper_index = _nearest_rank_index(upper, n)
        lower_bound = sorted_rows[lower_index][1]
        upper_bound = sorted_rows[upper_index][1]
        for row, value in rows:
            clipped = min(max(value, lower_bound), upper_bound)
            output.append(
                PanelPoint(
                    timestamp=timestamp,
                    symbol=row.symbol,
                    value=_canonical_number(clipped),
                    in_universe=row.in_universe,
                )
            )
    return _panel_result(
        "panel.winsorize",
        PanelValue(rows=tuple(output)),
        trace_extra={
            "lower_quantile": lower_quantile,
            "upper_quantile": upper_quantile,
            "interpolation": "nearest_rank",
        },
    )


def panel_residualize(
    panel: PanelValue,
    *,
    factor: Mapping[str, DecimalString],
    missing_policy: MissingPolicyKind = "error_on_missing",
    include_intercept: bool = True,
    min_observations: int = 3,
    insufficient_observations_policy: InsufficientObservationsPolicy = "unknown",
) -> PanelOperatorResult:
    """Single-factor per-symbol OLS residualization over time."""

    if not include_intercept:
        return _error(
            "QST_V2_PANEL_RESIDUALIZE_INTERCEPT_REQUIRED",
            "WP8c panel.residualize requires include_intercept=true.",
        )
    if min_observations < 3:
        return _error(
            "QST_V2_PANEL_RESIDUALIZE_MIN_OBSERVATIONS_INVALID",
            "panel.residualize min_observations must be at least 3.",
        )
    factor_values: dict[str, float] = {}
    diagnostics: list[Diagnostic] = []
    for timestamp, value in factor.items():
        parsed = _finite_decimal(value, context=f"factor[{timestamp!r}]", diagnostics=diagnostics)
        if parsed is not None:
            factor_values[timestamp] = parsed
    grouped, numeric_diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    diagnostics.extend(numeric_diagnostics)
    if any(diagnostic.severity == "error" for diagnostic in diagnostics):
        return _diagnostics("panel.residualize", diagnostics)

    by_symbol: dict[str, list[tuple[PanelPoint, float, float]]] = defaultdict(list)
    output: list[PanelPoint] = []
    for timestamp, rows in grouped.items():
        if timestamp not in factor_values:
            if missing_policy == "error_on_missing":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_PANEL_RESIDUALIZE_FACTOR_MISSING",
                        f"Missing factor value for timestamp {timestamp!r}.",
                    )
                )
            continue
        x = factor_values[timestamp]
        for row, y in rows:
            by_symbol[row.symbol].append((row, x, y))
    if any(diagnostic.severity == "error" for diagnostic in diagnostics):
        return _diagnostics("panel.residualize", diagnostics)

    insufficient_symbols: list[str] = []
    for symbol, observations in sorted(by_symbol.items()):
        if len(observations) < min_observations:
            insufficient_symbols.append(symbol)
            if insufficient_observations_policy == "error":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_PANEL_RESIDUALIZE_INSUFFICIENT_OBSERVATIONS",
                        f"Symbol {symbol!r} has fewer than min_observations observations.",
                    )
                )
            else:
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_PANEL_RESIDUALIZE_INSUFFICIENT_OBSERVATIONS",
                        f"Symbol {symbol!r} has fewer than min_observations observations.",
                        severity="warning",
                    )
                )
                output.extend(
                    PanelPoint(
                        timestamp=row.timestamp,
                        symbol=row.symbol,
                        value=None,
                        in_universe=row.in_universe,
                    )
                    for row, _x, _y in observations
                )
            continue
        xs = [x for _row, x, _y in observations]
        ys = [y for _row, _x, y in observations]
        mean_x = sum(xs) / len(xs)
        mean_y = sum(ys) / len(ys)
        denom = sum((x - mean_x) ** 2 for x in xs)
        if denom == 0:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_RESIDUALIZE_DEGENERATE_FACTOR",
                    f"Symbol {symbol!r} has a degenerate factor vector.",
                    severity="warning" if insufficient_observations_policy == "unknown" else "error",
                )
            )
            if insufficient_observations_policy == "unknown":
                output.extend(
                    PanelPoint(
                        timestamp=row.timestamp,
                        symbol=row.symbol,
                        value=None,
                        in_universe=row.in_universe,
                    )
                    for row, _x, _y in observations
                )
            continue
        beta = sum((x - mean_x) * (y - mean_y) for _row, x, y in observations) / denom
        alpha = mean_y - beta * mean_x
        output.extend(
            PanelPoint(
                timestamp=row.timestamp,
                symbol=row.symbol,
                value=_canonical_number(y - (alpha + beta * x)),
                in_universe=row.in_universe,
            )
            for row, x, y in observations
        )
    if any(diagnostic.severity == "error" for diagnostic in diagnostics):
        return _diagnostics("panel.residualize", diagnostics)
    return _panel_result(
        "panel.residualize",
        PanelValue(rows=tuple(output)),
        diagnostics=diagnostics,
        trace_extra={
            "include_intercept": include_intercept,
            "min_observations": min_observations,
            "insufficient_observations_policy": insufficient_observations_policy,
            "insufficient_symbols": insufficient_symbols,
        },
    )


def selection_to_weights(
    selection: SelectionPanelValue,
    *,
    method: RawWeightMethod = "equal_long",
) -> PanelOperatorResult:
    """Convert a SelectionPanel to raw, unnormalized WeightPanel."""

    rows: list[WeightPoint] = []
    for timestamp, points in _selection_by_time(selection).items():
        selected = [point for point in points if point.in_universe and point.selected]
        if method == "equal_long":
            rows.extend(_equal_weight_points(timestamp, selected, sign=1))
        elif method == "equal_short":
            rows.extend(_equal_weight_points(timestamp, selected, sign=-1))
        else:
            both_points = [point for point in selected if point.side == "both"]
            if both_points:
                return _diagnostics(
                    "selection.to_weights",
                    [
                        _diagnostic(
                            "QST_V2_SELECTION_SIDE_BOTH_UNSUPPORTED",
                            "equal_long_short does not support side='both' in WP8.",
                        )
                    ],
                )
            long_points = [point for point in selected if point.side in {None, "long", "both"}]
            short_points = [point for point in selected if point.side in {"short", "both"}]
            rows.extend(_equal_weight_points(timestamp, long_points, sign=1))
            rows.extend(_equal_weight_points(timestamp, short_points, sign=-1))
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(rows)),
        diagnostics=ValidationResult(),
        trace={
            "operator_id": "selection.to_weights",
            "method": method,
            "weight_kind": "raw",
            "normalized": False,
        },
    )


def weight_normalize_gross(
    weights: WeightPanelValue,
    *,
    target_gross: str = "1",
    zero_gross_policy: ZeroGrossPolicy = "keep_zero",
) -> PanelOperatorResult:
    """Scale eligible weights so gross exposure equals target_gross."""

    target = _canonical_param_decimal(target_gross, param_name="target_gross")
    if target < 0:
        return _error("QST_V2_WEIGHT_TARGET_GROSS_INVALID", "target_gross must be >= 0.")
    output: list[WeightPoint] = []
    trace: dict[str, Any] = {
        "operator_id": "weight.normalize_gross",
        "target_gross": _canonical_decimal(target),
        "zero_gross_policy": zero_gross_policy,
        "gross_before": {},
        "gross_after": {},
        "net_before": {},
        "net_after": {},
        "zero_gross_policy_applied": {},
    }
    diagnostics: list[Diagnostic] = []
    for timestamp, rows in _weights_by_time(weights).items():
        active = [row for row in rows if row.in_universe]
        inactive = [row for row in rows if not row.in_universe]
        gross = _gross(active)
        net = _net(active)
        trace["gross_before"][timestamp] = _canonical_decimal(gross)
        trace["net_before"][timestamp] = _canonical_decimal(net)
        if gross == 0:
            trace["zero_gross_policy_applied"][timestamp] = True
            if zero_gross_policy == "error":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_WEIGHT_ZERO_GROSS",
                        f"Cannot normalize zero gross weights at timestamp {timestamp!r}.",
                    )
                )
                continue
            normalized = [row.model_copy(update={"weight": "0"}) for row in active]
        else:
            trace["zero_gross_policy_applied"][timestamp] = False
            scale = target / gross
            normalized = [
                row.model_copy(update={"weight": _canonical_decimal(_weight_decimal(row) * scale)})
                for row in active
            ]
        output.extend(normalized)
        output.extend(inactive)
        trace["gross_after"][timestamp] = _canonical_decimal(_gross(normalized))
        trace["net_after"][timestamp] = _canonical_decimal(_net(normalized))
    if diagnostics:
        return _diagnostics("weight.normalize_gross", diagnostics)
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="normalized", normalized=True),
        diagnostics=ValidationResult(),
        trace=trace,
    )


def weight_cap_per_symbol(
    weights: WeightPanelValue,
    *,
    max_abs_weight: str,
    mode: str = "clip_no_redistribute",
) -> PanelOperatorResult:
    """Clip per-symbol absolute weights without redistributing clipped exposure."""

    cap = _canonical_param_decimal(max_abs_weight, param_name="max_abs_weight")
    if cap < 0:
        return _error("QST_V2_WEIGHT_MAX_ABS_WEIGHT_INVALID", "max_abs_weight must be >= 0.")
    if mode != "clip_no_redistribute":
        return _error(
            "QST_V2_WEIGHT_CAP_MODE_UNSUPPORTED",
            "WP8d only supports mode=clip_no_redistribute.",
        )
    output: list[WeightPoint] = []
    capped_symbols: list[dict[str, str]] = []
    trace: dict[str, Any] = {
        "operator_id": "weight.cap_per_symbol",
        "max_abs_weight": _canonical_decimal(cap),
        "mode": mode,
        "capped_symbols": capped_symbols,
        "gross_before": {},
        "gross_after": {},
        "net_before": {},
        "net_after": {},
    }
    for timestamp, rows in _weights_by_time(weights).items():
        active = [row for row in rows if row.in_universe]
        inactive = [row for row in rows if not row.in_universe]
        trace["gross_before"][timestamp] = _canonical_decimal(_gross(active))
        trace["net_before"][timestamp] = _canonical_decimal(_net(active))
        clipped_rows: list[WeightPoint] = []
        for row in active:
            before = _weight_decimal(row)
            after = min(max(before, -cap), cap)
            if after != before:
                capped_symbols.append(
                    {
                        "timestamp": timestamp,
                        "symbol": row.symbol,
                        "before": _canonical_decimal(before),
                        "after": _canonical_decimal(after),
                    }
                )
            clipped_rows.append(row.model_copy(update={"weight": _canonical_decimal(after)}))
        output.extend(clipped_rows)
        output.extend(inactive)
        trace["gross_after"][timestamp] = _canonical_decimal(_gross(clipped_rows))
        trace["net_after"][timestamp] = _canonical_decimal(_net(clipped_rows))
    return PanelOperatorResult(
        weights=WeightPanelValue(
            rows=tuple(output),
            weight_kind=weights.weight_kind,
            normalized=weights.normalized,
        ),
        diagnostics=ValidationResult(),
        trace=trace,
    )


def weight_market_neutral(
    weights: WeightPanelValue,
    *,
    target_net: str = "0",
    target_gross: str = "1",
    neutralization_method: str = "demean_then_gross_normalize",
    zero_gross_policy: ZeroGrossPolicy = "keep_zero",
) -> PanelOperatorResult:
    """Demean eligible weights and scale to target gross with target net zero."""

    canonical_net = _canonical_param_decimal(target_net, param_name="target_net")
    target = _canonical_param_decimal(target_gross, param_name="target_gross")
    if canonical_net != 0:
        return _error("QST_V2_WEIGHT_TARGET_NET_UNSUPPORTED", "WP8d only supports target_net=0.")
    if target < 0:
        return _error("QST_V2_WEIGHT_TARGET_GROSS_INVALID", "target_gross must be >= 0.")
    if neutralization_method != "demean_then_gross_normalize":
        return _error(
            "QST_V2_WEIGHT_NEUTRALIZATION_METHOD_UNSUPPORTED",
            "WP8d only supports demean_then_gross_normalize.",
        )

    output: list[WeightPoint] = []
    diagnostics: list[Diagnostic] = []
    trace: dict[str, Any] = {
        "operator_id": "weight.market_neutral",
        "neutralization_method": neutralization_method,
        "target_net": "0",
        "target_gross": _canonical_decimal(target),
        "zero_gross_policy": zero_gross_policy,
        "eligible_count": {},
        "net_before": {},
        "gross_before": {},
        "adjustment": {},
        "gross_after_demean": {},
        "net_after": {},
        "gross_after": {},
        "zero_gross_policy_applied": {},
    }
    for timestamp, rows in _weights_by_time(weights).items():
        active = [row for row in rows if row.in_universe]
        inactive = [row for row in rows if not row.in_universe]
        eligible_count = len(active)
        trace["eligible_count"][timestamp] = eligible_count
        if eligible_count == 0:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_MARKET_NEUTRAL_EMPTY_UNIVERSE",
                    f"market_neutral has no eligible weights at timestamp {timestamp!r}.",
                )
            )
            continue
        net = _net(active)
        gross = _gross(active)
        adjustment = net / Decimal(eligible_count)
        demeaned_values = [_weight_decimal(row) - adjustment for row in active]
        demeaned_gross = sum((abs(value) for value in demeaned_values), Decimal("0"))
        trace["net_before"][timestamp] = _canonical_decimal(net)
        trace["gross_before"][timestamp] = _canonical_decimal(gross)
        trace["adjustment"][timestamp] = _canonical_decimal(adjustment)
        trace["gross_after_demean"][timestamp] = _canonical_decimal(demeaned_gross)
        if demeaned_gross == 0:
            trace["zero_gross_policy_applied"][timestamp] = True
            if zero_gross_policy == "error":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_WEIGHT_MARKET_NEUTRAL_ZERO_GROSS",
                        f"market_neutral demeaned gross is zero at timestamp {timestamp!r}.",
                    )
                )
                continue
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_MARKET_NEUTRAL_ZEROED",
                    f"market_neutral produced zero weights at timestamp {timestamp!r}.",
                    severity="warning",
                )
            )
            neutral_rows = [row.model_copy(update={"weight": "0"}) for row in active]
        else:
            trace["zero_gross_policy_applied"][timestamp] = False
            scale = target / demeaned_gross
            neutral_rows = [
                row.model_copy(update={"weight": _canonical_decimal(value * scale)})
                for row, value in zip(active, demeaned_values, strict=True)
            ]
        output.extend(neutral_rows)
        output.extend(inactive)
        trace["net_after"][timestamp] = _canonical_decimal(_net(neutral_rows))
        trace["gross_after"][timestamp] = _canonical_decimal(_gross(neutral_rows))
    if any(diagnostic.severity == "error" for diagnostic in diagnostics):
        return _diagnostics("weight.market_neutral", diagnostics)
    return PanelOperatorResult(
        weights=WeightPanelValue(rows=tuple(output), weight_kind="normalized", normalized=True),
        diagnostics=ValidationResult(diagnostics=diagnostics),
        trace=trace,
    )


def _top_bottom_k(
    panel: PanelValue,
    *,
    k: int,
    operator_id: Literal["panel.top_k", "panel.bottom_k"],
    descending: bool,
    missing_policy: MissingPolicyKind,
    tie_policy: TiePolicy,
    selection_size_policy: SelectionSizePolicy,
) -> PanelOperatorResult:
    if k <= 0:
        return _error("QST_V2_PANEL_K_INVALID", f"{operator_id} requires k > 0.")
    if tie_policy != "stable_symbol_order":
        return _error("QST_V2_PANEL_TIE_POLICY_UNSUPPORTED", "WP8c only supports stable_symbol_order.")
    if selection_size_policy != "allow_smaller":
        return _error(
            "QST_V2_PANEL_SELECTION_SIZE_POLICY_UNSUPPORTED",
            "WP8c only supports selection_size_policy=allow_smaller.",
        )
    grouped, diagnostics = _numeric_groups(panel, missing_policy=missing_policy)
    if diagnostics:
        return _diagnostics(operator_id, diagnostics)
    output: list[SelectionPoint] = []
    actual_selected: dict[str, int] = {}
    eligible_count: dict[str, int] = {}
    for timestamp, rows in grouped.items():
        eligible_count[timestamp] = len(rows)
        sorted_rows = sorted(
            rows,
            key=lambda item: ((-item[1] if descending else item[1]), item[0].symbol),
        )
        selected_keys = {(row.timestamp, row.symbol) for row, _value in sorted_rows[:k]}
        actual_selected[timestamp] = len(selected_keys)
        output.extend(
            SelectionPoint(
                timestamp=timestamp,
                symbol=row.symbol,
                selected=(row.timestamp, row.symbol) in selected_keys,
                side="long" if operator_id == "panel.top_k" else "short",
                score=row.value,
                in_universe=row.in_universe,
            )
            for row, _value in rows
        )
    return PanelOperatorResult(
        selection=SelectionPanelValue(
            rows=tuple(output),
            selection_kind="long_only" if operator_id == "panel.top_k" else "short_only",
        ),
        diagnostics=ValidationResult(),
        trace={
            "operator_id": operator_id,
            "requested_k": k,
            "eligible_count": eligible_count,
            "actual_selected": actual_selected,
            "selection_size_policy": selection_size_policy,
            "tie_policy": tie_policy,
        },
    )


def _numeric_groups(
    panel: PanelValue,
    *,
    missing_policy: MissingPolicyKind,
) -> tuple[dict[str, list[tuple[PanelPoint, float]]], list[Diagnostic]]:
    diagnostics: list[Diagnostic] = []
    grouped: dict[str, list[tuple[PanelPoint, float]]] = defaultdict(list)
    for row in panel.rows:
        if not row.in_universe:
            continue
        if row.value is None:
            if missing_policy == "error_on_missing":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_PANEL_MISSING_VALUE",
                        f"Missing value for active universe cell {(row.timestamp, row.symbol)!r}.",
                    )
                )
            continue
        parsed = _finite_decimal(row.value, context=f"{row.timestamp}/{row.symbol}", diagnostics=diagnostics)
        if parsed is not None:
            grouped[row.timestamp].append((row, parsed))
    return {key: sorted(value, key=lambda item: item[0].symbol) for key, value in sorted(grouped.items())}, diagnostics


def _finite_decimal(value: str, *, context: str, diagnostics: list[Diagnostic]) -> float | None:
    number = float(Decimal(value))
    if not math.isfinite(number):
        diagnostics.append(
            _diagnostic(
                "QST_V2_PANEL_NON_FINITE_VALUE",
                f"Non-finite numeric value for {context}.",
            )
        )
        return None
    return number


def _selection_by_time(selection: SelectionPanelValue) -> dict[str, list[SelectionPoint]]:
    grouped: dict[str, list[SelectionPoint]] = defaultdict(list)
    for row in selection.rows:
        grouped[row.timestamp].append(row)
    return {key: sorted(value, key=lambda item: item.symbol) for key, value in sorted(grouped.items())}


def _weights_by_time(weights: WeightPanelValue) -> dict[str, list[WeightPoint]]:
    grouped: dict[str, list[WeightPoint]] = defaultdict(list)
    for row in weights.rows:
        grouped[row.timestamp].append(row)
    return {key: sorted(value, key=lambda item: item.symbol) for key, value in sorted(grouped.items())}


def _weight_decimal(row: WeightPoint) -> Decimal:
    return Decimal(row.weight)


def _gross(rows: list[WeightPoint]) -> Decimal:
    return sum((abs(_weight_decimal(row)) for row in rows), Decimal("0"))


def _net(rows: list[WeightPoint]) -> Decimal:
    return sum((_weight_decimal(row) for row in rows), Decimal("0"))


def _equal_weight_points(timestamp: str, points: list[SelectionPoint], *, sign: int) -> list[WeightPoint]:
    if not points:
        return []
    weight = _canonical_number(sign / len(points))
    return [
        WeightPoint(timestamp=timestamp, symbol=point.symbol, weight=weight, in_universe=point.in_universe)
        for point in sorted(points, key=lambda point: point.symbol)
    ]


def _nearest_rank_index(quantile: float, n: int) -> int:
    return min(max(math.ceil(quantile * n) - 1, 0), n - 1)


def _canonical_number(value: float) -> str:
    if isinstance(value, float) and abs(value) < 1e-12:
        value = 0.0
    return normalize_to_canonical(value)


def _canonical_param_decimal(raw: str | float | Decimal, *, param_name: str) -> Decimal:
    try:
        return Decimal(normalize_to_canonical(raw))
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{param_name} must be a finite canonicalizable DecimalString") from exc


def _canonical_decimal(value: Decimal) -> str:
    return normalize_to_canonical(value)


def _panel_result(
    operator_id: PanelOperatorName,
    panel: PanelValue,
    *,
    diagnostics: list[Diagnostic] | None = None,
    trace_extra: dict[str, Any] | None = None,
) -> PanelOperatorResult:
    return PanelOperatorResult(
        panel=panel,
        diagnostics=ValidationResult(diagnostics=diagnostics or []),
        trace={
            "operator_id": operator_id,
            "missing_policy": trace_extra.get("missing_policy") if trace_extra else "error_on_missing",
            "symbol_count_by_timestamp": _symbol_counts(panel),
            **(trace_extra or {}),
        },
    )


def _symbol_counts(panel: PanelValue) -> dict[str, int]:
    counts: dict[str, int] = defaultdict(int)
    for row in panel.rows:
        if row.in_universe:
            counts[row.timestamp] += 1
    return dict(sorted(counts.items()))


def _diagnostics(operator_id: str, diagnostics: list[Diagnostic]) -> PanelOperatorResult:
    return PanelOperatorResult(
        diagnostics=ValidationResult(diagnostics=diagnostics),
        trace={"operator_id": operator_id, "diagnostics": [item.model_dump(mode="json") for item in diagnostics]},
    )


def _error(code: str, message: str) -> PanelOperatorResult:
    return _diagnostics("panel.operator", [_diagnostic(code, message)])


def _diagnostic(code: str, message: str, *, severity: Literal["warning", "error"] = "error") -> Diagnostic:
    return Diagnostic(code=code, severity=severity, phase="signature", message=message)
