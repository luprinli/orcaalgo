from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from qst.hash import expected_artifact_hash_v2
from qst.ir import load_ir_v04_file, validate_ir_v04
from qst.panel.reference_validation import (
    diagnostics_panel_pv_b_v04,
    load_panel_pv_b_fixture,
    trace_panel_pv_b_v04,
)

ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "tests" / "reference" / "panel"
STRATEGIES = REFERENCE / "strategies"
FIXTURES = REFERENCE / "fixtures"
DIAGNOSTICS = REFERENCE / "diagnostics"
TRACES = REFERENCE / "traces"


def test_pv_b_strategies_validate_before_reference_execution() -> None:
    for path in sorted(STRATEGIES.glob("panel_*.gkr.yaml")):
        ir = load_ir_v04_file(path)
        if ir.metadata.get("reference_validation") != "panel-reference":
            continue

        result = validate_ir_v04(ir)

        assert result.ok, [diagnostic.code for diagnostic in result.diagnostics]


def test_pv_b_reference_diagnostics_match_and_hash() -> None:
    for path in sorted(DIAGNOSTICS.glob("*.json")):
        case = path.stem
        ir = load_ir_v04_file(STRATEGIES / f"{case}.gkr.yaml")
        fixture = load_panel_pv_b_fixture(FIXTURES / f"{case}.json")
        expected = _load_json(path)
        artifact = diagnostics_panel_pv_b_v04(ir, fixture)

        assert artifact == expected
        assert expected["diagnostics"] == []
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(
            _without_hash(expected)
        )


def test_pv_b_reference_traces_match_and_hash() -> None:
    for path in sorted(TRACES.glob("*.json")):
        case = path.stem
        ir = load_ir_v04_file(STRATEGIES / f"{case}.gkr.yaml")
        fixture = load_panel_pv_b_fixture(FIXTURES / f"{case}.json")
        expected = _load_json(path)
        trace = trace_panel_pv_b_v04(ir, fixture).model_dump(mode="json")

        assert trace == expected
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(
            _without_hash(expected)
        )

        mutated_hash = {**expected, "expected_artifact_hash": "sha256:" + "f" * 64}
        assert expected_artifact_hash_v2(_without_hash(mutated_hash)) == expected[
            "expected_artifact_hash"
        ]


def test_pv_b_gate_cases_have_expected_weight_outputs() -> None:
    top_bottom = _load_json(TRACES / "panel_top_bottom_market_neutral.json")
    residual = _load_json(TRACES / "panel_btc_residual_meanrev.json")

    assert top_bottom["outputs"]["raw_exposure"] == {"gross": "2", "net": "0"}
    assert top_bottom["outputs"]["final_exposure"] == {"gross": "1", "net": "0"}
    assert top_bottom["outputs"]["final_weights"]["weight_kind"] == "normalized"
    assert top_bottom["outputs"]["final_weights"]["normalized"] is True
    assert top_bottom["outputs"]["long_symbols"] == ["BTC/USDT"]
    assert top_bottom["outputs"]["short_symbols"] == ["ADA/USDT"]

    assert residual["outputs"]["final_weights"]["weight_kind"] == "normalized"
    assert residual["outputs"]["final_weights"]["normalized"] is True
    assert residual["outputs"]["final_weights"]["rows"]
    assert residual["outputs"]["selected_symbols"] == ["ETH/USDT", "SOL/USDT"]


def _load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    assert isinstance(loaded, dict)
    return loaded


def _without_hash(value: dict[str, Any]) -> dict[str, Any]:
    return {key: item for key, item in value.items() if key != "expected_artifact_hash"}
