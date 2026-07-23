"""panel-reference Panel reference strategies for Token System v2 WP8e."""

from __future__ import annotations

import json
from decimal import Decimal
from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.canonical_json import stable_json_bytes
from qst.hash import expected_artifact_hash_v2
from qst.ir import StrategyIRV04, validate_ir_v04
from qst.panel.operators import (
    PANEL_OPERATOR_TOKENS,
    WEIGHT_OPERATOR_TOKENS,
    PanelOperatorResult,
    PanelValue,
    SelectionPanelValue,
    SelectionPoint,
    WeightPanelValue,
    panel_bottom_k,
    panel_residualize,
    panel_top_k,
    panel_zscore,
    selection_to_weights,
    weight_market_neutral,
    weight_normalize_gross,
)
from qst.validation import Diagnostic, ValidationResult

PANEL_PV_B_FIXTURE_VERSION: Literal["qst-v04-panel-fixture/0.1"] = (
    "qst-v04-panel-fixture/0.1"
)
PANEL_PV_B_TRACE_ARTIFACT_VERSION: Literal["qst-v04-panel-validation-trace/0.1"] = (
    "qst-v04-panel-validation-trace/0.1"
)
PANEL_PV_B_DIAGNOSTICS_ARTIFACT_VERSION: Literal[
    "qst-v04-panel-expected-diagnostics/0.1"
] = "qst-v04-panel-expected-diagnostics/0.1"

PanelPVBCase = Literal[
    "panel_top_bottom_market_neutral",
    "panel_btc_residual_meanrev",
]


class LongShortSelectionFixture(BaseModel):
    """Explicit long/short selection material used by panel-reference without a combine operator."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    selection_kind: Literal["long_short"] = "long_short"
    long_symbols: tuple[str, ...]
    short_symbols: tuple[str, ...]
    universe_ref: str = Field(min_length=1)
    source: dict[str, Literal["panel.top_k", "panel.bottom_k"]]

    @model_validator(mode="before")
    @classmethod
    def _sort_symbols(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        value = dict(value)
        value["long_symbols"] = tuple(sorted(value.get("long_symbols", ())))
        value["short_symbols"] = tuple(sorted(value.get("short_symbols", ())))
        return value

    @model_validator(mode="after")
    def _validate_shape(self) -> LongShortSelectionFixture:
        if set(self.long_symbols) & set(self.short_symbols):
            raise ValueError("long_symbols and short_symbols must be disjoint")
        if self.source.get("long_from") != "panel.top_k":
            raise ValueError("long_short selection source.long_from must be panel.top_k")
        if self.source.get("short_from") != "panel.bottom_k":
            raise ValueError("long_short selection source.short_from must be panel.bottom_k")
        return self


class PanelPVBFixture(BaseModel):
    """Fixture payload for one panel-reference Panel reference case."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-panel-fixture/0.1"] = PANEL_PV_B_FIXTURE_VERSION
    case: PanelPVBCase
    panel: PanelValue
    params: dict[str, Any] = Field(default_factory=dict)
    factor: dict[str, str] = Field(default_factory=dict)
    factor_symbol: str | None = None
    factor_symbol_tradable: bool = False
    long_short_selection: LongShortSelectionFixture | None = None

    @field_validator("params", "factor")
    @classmethod
    def _payload_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="panel-reference fixture payload")
        return value


class PanelPVBResult(BaseModel):
    """Deterministic panel-reference result before artifact wrapping."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    case: str
    outputs: dict[str, Any] = Field(default_factory=dict)
    diagnostics: list[Diagnostic] = Field(default_factory=list)
    operator_traces: dict[str, Any] = Field(default_factory=dict)

    @field_validator("outputs", "operator_traces")
    @classmethod
    def _payload_is_json(cls, value: Any) -> Any:
        _ensure_json(value, field_name="panel-reference result payload")
        return value

    @property
    def validation_result(self) -> ValidationResult:
        """Validation result derived from diagnostics."""

        return ValidationResult(diagnostics=self.diagnostics)


class PanelPVBTraceArtifact(BaseModel):
    """Serializable trace artifact for one panel-reference case."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-panel-validation-trace/0.1"] = (
        PANEL_PV_B_TRACE_ARTIFACT_VERSION
    )
    strategy: str
    case: PanelPVBCase
    outputs: dict[str, Any]
    diagnostics: list[dict[str, Any]]
    operator_traces: dict[str, Any]
    expected_artifact_hash: str


def load_panel_pv_b_fixture(path: str | Path) -> PanelPVBFixture:
    """Load a panel-reference fixture from JSON."""

    with Path(path).open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    return PanelPVBFixture.model_validate(loaded)


def run_panel_pv_b_case(ir: StrategyIRV04, fixture: PanelPVBFixture) -> PanelPVBResult:
    """Run one deterministic panel-reference reference case after static IR validation."""

    validation = validate_ir_v04(ir)
    if not validation.ok:
        return PanelPVBResult(case=fixture.case, diagnostics=validation.errors)

    metadata_case = ir.metadata.get("reference_case")
    if metadata_case != fixture.case:
        return _error_result(
            fixture.case,
            "QST_V2_PANEL_PVB_CASE_MISMATCH",
            f"Strategy {ir.strategy.id!r} declares case {metadata_case!r}, fixture is {fixture.case!r}.",
        )

    token_diagnostics = _validate_case_token_refs(ir, fixture.case)
    if token_diagnostics:
        return PanelPVBResult(case=fixture.case, diagnostics=token_diagnostics)

    if fixture.case == "panel_top_bottom_market_neutral":
        return _panel_top_bottom_market_neutral(fixture)
    if fixture.case == "panel_btc_residual_meanrev":
        return _panel_btc_residual_meanrev(fixture)

    return _error_result(
        fixture.case,
        "QST_V2_PANEL_PVB_CASE_UNKNOWN",
        f"Unsupported panel-reference Panel case {fixture.case!r}.",
    )


def trace_panel_pv_b_v04(ir: StrategyIRV04, fixture: PanelPVBFixture) -> PanelPVBTraceArtifact:
    """Return a hash-bearing panel-reference trace artifact."""

    result = run_panel_pv_b_case(ir, fixture)
    material = {
        "artifact_version": PANEL_PV_B_TRACE_ARTIFACT_VERSION,
        "strategy": ir.strategy.id,
        "case": result.case,
        "outputs": result.outputs,
        "diagnostics": [
            diagnostic.model_dump(mode="json", exclude_none=True)
            for diagnostic in result.diagnostics
        ],
        "operator_traces": result.operator_traces,
    }
    return PanelPVBTraceArtifact.model_validate(
        {
            **material,
            "expected_artifact_hash": expected_artifact_hash_v2(material),
        }
    )


def diagnostics_panel_pv_b_v04(ir: StrategyIRV04, fixture: PanelPVBFixture) -> dict[str, Any]:
    """Return a hash-bearing panel-reference diagnostics artifact."""

    result = run_panel_pv_b_case(ir, fixture)
    material = {
        "artifact_version": PANEL_PV_B_DIAGNOSTICS_ARTIFACT_VERSION,
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


def _panel_top_bottom_market_neutral(fixture: PanelPVBFixture) -> PanelPVBResult:
    if fixture.long_short_selection is None:
        return _error_result(
            fixture.case,
            "QST_V2_PANEL_PVB_SELECTION_REQUIRED",
            "panel_top_bottom_market_neutral requires explicit long_short selection fixture.",
        )
    k = _int_param(fixture, "k", default=1)
    top = panel_top_k(fixture.panel, k=k)
    bottom = panel_bottom_k(fixture.panel, k=k)
    early = _first_error(fixture.case, top, bottom)
    if early is not None:
        return early

    selection_diagnostics = _validate_long_short_selection_fixture(
        fixture.panel,
        fixture.long_short_selection,
        top.selection,
        bottom.selection,
    )
    if selection_diagnostics:
        return PanelPVBResult(case=fixture.case, diagnostics=selection_diagnostics)

    selection = _long_short_selection_value(fixture.long_short_selection, fixture.panel)
    raw = selection_to_weights(selection, method="equal_long_short")
    neutral = weight_market_neutral(raw.weights or WeightPanelValue(), target_gross="1")
    early = _first_error(fixture.case, raw, neutral)
    if early is not None:
        return early

    return PanelPVBResult(
        case=fixture.case,
        outputs={
            "long_symbols": list(fixture.long_short_selection.long_symbols),
            "short_symbols": list(fixture.long_short_selection.short_symbols),
            "raw_weights": _weights_json(raw.weights),
            "raw_exposure": _exposure(raw.weights),
            "final_weights": _weights_json(neutral.weights),
            "final_exposure": _exposure(neutral.weights),
        },
        diagnostics=neutral.diagnostics.diagnostics,
        operator_traces={
            "panel.top_k": top.trace,
            "panel.bottom_k": bottom.trace,
            "selection.long_short_fixture": {
                "selection_kind": "long_short",
                "long_symbols": list(fixture.long_short_selection.long_symbols),
                "short_symbols": list(fixture.long_short_selection.short_symbols),
                "universe_ref": fixture.long_short_selection.universe_ref,
                "source": fixture.long_short_selection.source,
            },
            "selection.to_weights": raw.trace,
            "weight.market_neutral": neutral.trace,
        },
    )


def _panel_btc_residual_meanrev(fixture: PanelPVBFixture) -> PanelPVBResult:
    factor_diagnostics = _validate_external_factor_fixture(fixture)
    if factor_diagnostics:
        return PanelPVBResult(case=fixture.case, diagnostics=factor_diagnostics)

    k = _int_param(fixture, "k", default=1)
    residual = panel_residualize(fixture.panel, factor=fixture.factor)
    zscore = panel_zscore(residual.panel or PanelValue())
    bottom = panel_bottom_k(zscore.panel or PanelValue(), k=k)
    raw = selection_to_weights(bottom.selection or SelectionPanelValue(), method="equal_long")
    normalized = weight_normalize_gross(raw.weights or WeightPanelValue(), target_gross="1")
    early = _first_error(fixture.case, residual, zscore, bottom, raw, normalized)
    if early is not None:
        return early

    return PanelPVBResult(
        case=fixture.case,
        outputs={
            "selected_symbols": _selected_symbols(bottom.selection),
            "residual_panel": _panel_json(residual.panel),
            "zscore_panel": _panel_json(zscore.panel),
            "final_weights": _weights_json(normalized.weights),
            "final_exposure": _exposure(normalized.weights),
        },
        diagnostics=[
            *residual.diagnostics.diagnostics,
            *zscore.diagnostics.diagnostics,
            *bottom.diagnostics.diagnostics,
            *raw.diagnostics.diagnostics,
            *normalized.diagnostics.diagnostics,
        ],
        operator_traces={
            "panel.residualize": residual.trace,
            "panel.zscore": zscore.trace,
            "panel.bottom_k": bottom.trace,
            "selection.to_weights": raw.trace,
            "weight.normalize_gross": normalized.trace,
        },
    )


def _validate_case_token_refs(ir: StrategyIRV04, case: PanelPVBCase) -> list[Diagnostic]:
    required_by_case = {
        "panel_top_bottom_market_neutral": {
            "panel.top_k",
            "panel.bottom_k",
            "selection.to_weights",
            "weight.market_neutral",
        },
        "panel_btc_residual_meanrev": {
            "panel.residualize",
            "panel.zscore",
            "panel.bottom_k",
            "selection.to_weights",
            "weight.normalize_gross",
        },
    }
    required = required_by_case.get(case)
    if required is None:
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_CASE_UNKNOWN",
                f"Unsupported panel-reference Panel case {case!r}.",
            )
        ]
    allowed = {*PANEL_OPERATOR_TOKENS, *WEIGHT_OPERATOR_TOKENS}
    present: set[str] = set()
    diagnostics: list[Diagnostic] = []
    for node in ir.strategy.nodes:
        name = node.token_ref.name if node.token_ref is not None else node.token
        if name is None or not name.startswith(("panel.", "selection.", "weight.")):
            continue
        present.add(name)
        if name not in allowed:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_PVB_TOKEN_REF_UNSUPPORTED",
                    f"panel-reference strategy references unsupported token {name!r}.",
                    node_id=node.id,
                )
            )
    missing = sorted(required - present)
    if missing:
        diagnostics.append(
            _diagnostic(
                "QST_V2_PANEL_PVB_TOKEN_REF_MISSING",
                f"panel-reference strategy is missing required token refs: {missing}.",
            )
        )
    return diagnostics


def _validate_long_short_selection_fixture(
    panel: PanelValue,
    fixture: LongShortSelectionFixture,
    top: SelectionPanelValue | None,
    bottom: SelectionPanelValue | None,
) -> list[Diagnostic]:
    active_symbols = {row.symbol for row in panel.rows if row.in_universe}
    selected = set(fixture.long_symbols) | set(fixture.short_symbols)
    out_of_universe = sorted(selected - active_symbols)
    if out_of_universe:
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_SELECTION_OUT_OF_UNIVERSE",
                f"long_short selection contains out-of-universe symbols: {out_of_universe}.",
            )
        ]

    top_symbols = set(_selected_symbols(top))
    bottom_symbols = set(_selected_symbols(bottom))
    if top_symbols != set(fixture.long_symbols):
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_TOP_SELECTION_MISMATCH",
                f"Top-k symbols {sorted(top_symbols)} do not match long_symbols {list(fixture.long_symbols)}.",
            )
        ]
    if bottom_symbols != set(fixture.short_symbols):
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_BOTTOM_SELECTION_MISMATCH",
                f"Bottom-k symbols {sorted(bottom_symbols)} do not match short_symbols {list(fixture.short_symbols)}.",
            )
        ]
    return []


def _long_short_selection_value(
    fixture: LongShortSelectionFixture,
    panel: PanelValue,
) -> SelectionPanelValue:
    long_symbols = set(fixture.long_symbols)
    short_symbols = set(fixture.short_symbols)
    rows = [
        SelectionPoint(
            timestamp=row.timestamp,
            symbol=row.symbol,
            selected=row.symbol in long_symbols or row.symbol in short_symbols,
            side="long" if row.symbol in long_symbols else "short" if row.symbol in short_symbols else None,
            in_universe=row.in_universe,
        )
        for row in panel.rows
        if row.symbol in long_symbols or row.symbol in short_symbols
    ]
    return SelectionPanelValue(rows=tuple(rows), selection_kind="long_short")


def _validate_external_factor_fixture(fixture: PanelPVBFixture) -> list[Diagnostic]:
    if not fixture.factor_symbol:
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_FACTOR_SYMBOL_REQUIRED",
                "panel_btc_residual_meanrev requires external factor_symbol metadata.",
            )
        ]
    if not fixture.factor:
        return [
            _diagnostic(
                "QST_V2_PANEL_PVB_FACTOR_REQUIRED",
                "panel_btc_residual_meanrev requires external BTC factor values.",
            )
        ]
    if not fixture.factor_symbol_tradable:
        active_factor_rows = [
            row
            for row in fixture.panel.rows
            if row.symbol == fixture.factor_symbol and row.in_universe
        ]
        if active_factor_rows:
            return [
                _diagnostic(
                    "QST_V2_PANEL_PVB_FACTOR_SYMBOL_TRADABLE",
                    f"Factor symbol {fixture.factor_symbol!r} is active in the selection universe.",
                )
            ]
    return []


def _first_error(case: PanelPVBCase, *results: PanelOperatorResult) -> PanelPVBResult | None:
    diagnostics = [
        diagnostic
        for result in results
        for diagnostic in result.diagnostics.diagnostics
        if diagnostic.severity == "error"
    ]
    if diagnostics:
        return PanelPVBResult(case=case, diagnostics=diagnostics)
    return None


def _panel_json(panel: PanelValue | None) -> dict[str, Any]:
    return panel.model_dump(mode="json") if panel is not None else {"rows": []}


def _weights_json(weights: WeightPanelValue | None) -> dict[str, Any]:
    return weights.model_dump(mode="json") if weights is not None else {"rows": []}


def _selected_symbols(selection: SelectionPanelValue | None) -> list[str]:
    if selection is None:
        return []
    return sorted({row.symbol for row in selection.rows if row.in_universe and row.selected})


def _exposure(weights: WeightPanelValue | None) -> dict[str, str]:
    if weights is None:
        return {"gross": "0", "net": "0"}
    active = [row for row in weights.rows if row.in_universe]
    gross = sum((abs(Decimal(row.weight)) for row in active), Decimal("0"))
    net = sum((Decimal(row.weight) for row in active), Decimal("0"))
    return {"gross": _canonical_decimal(gross), "net": _canonical_decimal(net)}


def _int_param(fixture: PanelPVBFixture, name: str, *, default: int) -> int:
    raw = fixture.params.get(name, default)
    if not isinstance(raw, int) or raw < 1:
        raise ValueError(f"panel-reference fixture param {name!r} must be a positive integer")
    return raw


def _error_result(case: PanelPVBCase, code: str, message: str) -> PanelPVBResult:
    return PanelPVBResult(case=case, diagnostics=[_diagnostic(code, message)])


def _diagnostic(code: str, message: str, *, node_id: str | None = None) -> Diagnostic:
    return Diagnostic(code=code, severity="error", phase="signature", message=message, node_id=node_id)


def _canonical_decimal(value: Decimal) -> str:
    normalized = value.normalize()
    if normalized == 0:
        return "0"
    return format(normalized, "f")


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
