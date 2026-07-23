from __future__ import annotations

from pathlib import Path

import pytest
from pydantic import ValidationError

from qst.ir import load_ir_v04_file
from qst.panel.operators import PanelValue
from qst.panel.reference_validation import (
    LongShortSelectionFixture,
    PanelPVBFixture,
    load_panel_pv_b_fixture,
    run_panel_pv_b_case,
)

ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "tests" / "reference" / "panel"
STRATEGIES = REFERENCE / "strategies"
FIXTURES = REFERENCE / "fixtures"


def test_pv_b_runner_refuses_invalid_ir_before_helper_composition() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_top_bottom_market_neutral.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_top_bottom_market_neutral.json")

    missing_weights = ir.model_copy(update={"capabilities": ["core", "panel_type", "panel_ops"]})
    panel_recipes = ir.model_copy(
        update={"capabilities": ["core", "panel_type", "panel_ops", "panel_weights", "panel_recipes"]}
    )

    assert run_panel_pv_b_case(missing_weights, fixture).diagnostics[0].code == (
        "QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRED"
    )
    assert run_panel_pv_b_case(panel_recipes, fixture).diagnostics[0].code == "capability_not_accepted"


def test_pv_b_runner_reports_strategy_fixture_case_mismatch() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_top_bottom_market_neutral.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_btc_residual_meanrev.json")

    result = run_panel_pv_b_case(ir, fixture)

    assert result.diagnostics[0].code == "QST_V2_PANEL_PVB_CASE_MISMATCH"


def test_pv_b_runner_reports_unknown_constructed_case() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_top_bottom_market_neutral.gkr.yaml")
    fixture = PanelPVBFixture.model_construct(
        artifact_version="qst-v04-panel-fixture/0.1",
        case="unknown_case",
        panel=PanelValue(),
        params={},
        factor={},
        factor_symbol=None,
        factor_symbol_tradable=False,
        long_short_selection=None,
    )
    ir = ir.model_copy(update={"metadata": {"reference_case": "unknown_case"}})

    result = run_panel_pv_b_case(ir, fixture)

    assert result.diagnostics[0].code == "QST_V2_PANEL_PVB_CASE_UNKNOWN"


def test_long_short_selection_fixture_rejects_overlap() -> None:
    with pytest.raises(ValidationError):
        LongShortSelectionFixture.model_validate(
            {
                "selection_kind": "long_short",
                "long_symbols": ["BTC/USDT"],
                "short_symbols": ["BTC/USDT"],
                "universe_ref": "tests/reference/panel/fixtures/top_bottom_universe.json",
                "source": {
                    "long_from": "panel.top_k",
                    "short_from": "panel.bottom_k",
                },
            }
        )


def test_long_short_selection_rejects_out_of_universe_and_top_mismatch() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_top_bottom_market_neutral.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_top_bottom_market_neutral.json")

    out_of_universe = fixture.model_copy(
        update={
            "long_short_selection": fixture.long_short_selection.model_copy(
                update={"long_symbols": ("XRP/USDT",)}
            )
        }
    )
    top_mismatch = fixture.model_copy(
        update={
            "long_short_selection": fixture.long_short_selection.model_copy(
                update={"long_symbols": ("ETH/USDT",)}
            )
        }
    )

    assert run_panel_pv_b_case(ir, out_of_universe).diagnostics[0].code == (
        "QST_V2_PANEL_PVB_SELECTION_OUT_OF_UNIVERSE"
    )
    assert run_panel_pv_b_case(ir, top_mismatch).diagnostics[0].code == (
        "QST_V2_PANEL_PVB_TOP_SELECTION_MISMATCH"
    )


def test_dynamic_universe_false_is_not_missing_but_active_missing_is_diagnostic() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_top_bottom_market_neutral.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_top_bottom_market_neutral.json")

    assert run_panel_pv_b_case(ir, fixture).diagnostics == []

    missing_rows = [
        row.model_copy(update={"value": None}) if row.symbol == "BTC/USDT" else row
        for row in fixture.panel.rows
    ]
    missing_fixture = fixture.model_copy(update={"panel": PanelValue(rows=tuple(missing_rows))})

    assert run_panel_pv_b_case(ir, missing_fixture).diagnostics[0].code == "QST_V2_PANEL_MISSING_VALUE"


def test_btc_residual_case_requires_external_nontradable_factor_symbol() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_btc_residual_meanrev.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_btc_residual_meanrev.json")
    active_rows = [
        row.model_copy(update={"in_universe": True}) if row.symbol == "BTC/USDT" else row
        for row in fixture.panel.rows
    ]
    active_btc_fixture = fixture.model_copy(update={"panel": PanelValue(rows=tuple(active_rows))})
    no_factor_fixture = fixture.model_copy(update={"factor_symbol": None})

    assert run_panel_pv_b_case(ir, fixture).diagnostics == []
    assert run_panel_pv_b_case(ir, active_btc_fixture).diagnostics[0].code == (
        "QST_V2_PANEL_PVB_FACTOR_SYMBOL_TRADABLE"
    )
    assert run_panel_pv_b_case(ir, no_factor_fixture).diagnostics[0].code == (
        "QST_V2_PANEL_PVB_FACTOR_SYMBOL_REQUIRED"
    )


def test_pv_b_result_requires_accepted_token_refs() -> None:
    ir = load_ir_v04_file(STRATEGIES / "panel_btc_residual_meanrev.gkr.yaml")
    fixture = load_panel_pv_b_fixture(FIXTURES / "panel_btc_residual_meanrev.json")
    bad_nodes = [
        node.model_copy(update={"token_ref": node.token_ref.model_copy(update={"name": "panel.select"})})
        if node.id == "meanrev_long"
        else node
        for node in ir.strategy.nodes
    ]
    bad_ir = ir.model_copy(update={"strategy": ir.strategy.model_copy(update={"nodes": bad_nodes})})

    result = run_panel_pv_b_case(bad_ir, fixture)

    assert result.diagnostics[0].code == "QST_V2_PANEL_OPERATOR_NOT_ACCEPTED"
