from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from qst.hash import expected_artifact_hash_v2
from qst.ir import load_ir_v04_file
from qst.state import (
    diagnostics_state_pv_a_v04,
    load_state_pv_a_fixture,
    trace_state_pv_a_v04,
)

ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "tests" / "reference" / "state"
STRATEGIES = REFERENCE / "strategies"
FIXTURES = REFERENCE / "fixtures"
DIAGNOSTICS = REFERENCE / "diagnostics"
TRACES = REFERENCE / "traces"


def test_pv_a_reference_diagnostics_match_and_hash() -> None:
    for path in sorted(DIAGNOSTICS.glob("*.json")):
        case = path.stem
        ir = load_ir_v04_file(STRATEGIES / f"{case}.gkr.yaml")
        fixture = load_state_pv_a_fixture(FIXTURES / f"{case}.json")
        expected = _load_json(path)
        artifact = diagnostics_state_pv_a_v04(ir, fixture)

        assert artifact == expected
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(
            _without_hash(expected)
        )


def test_pv_a_reference_traces_match_and_hash() -> None:
    for path in sorted(TRACES.glob("*.json")):
        case = path.stem
        ir = load_ir_v04_file(STRATEGIES / f"{case}.gkr.yaml")
        fixture = load_state_pv_a_fixture(FIXTURES / f"{case}.json")
        expected = _load_json(path)
        trace = trace_state_pv_a_v04(ir, fixture).model_dump(mode="json")

        assert trace == expected
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(
            _without_hash(expected)
        )


def test_pv_a_gate_cases_have_expected_outcomes() -> None:
    expected_decisions = {
        "state_cooldown": ["active", "active", "blocked", "active", "active"],
        "state_market_freeze": ["active", "active", "blocked", "active", "active"],
        "state_circuit_breaker": ["active", "active", "blocked", "blocked", "blocked"],
        "state_observe_period": ["blocked", "blocked", "active", "active"],
        "state_slot_budget_minimal": ["active", "active", "blocked", "blocked"],
    }

    for case, decisions in expected_decisions.items():
        ir = load_ir_v04_file(STRATEGIES / f"{case}.gkr.yaml")
        fixture = load_state_pv_a_fixture(FIXTURES / f"{case}.json")
        trace = trace_state_pv_a_v04(ir, fixture)

        assert trace.diagnostics == []
        assert trace.outputs["decisions"] == decisions
        assert all(trace.replay_checks.values())


def _load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    assert isinstance(loaded, dict)
    return loaded


def _without_hash(value: dict[str, Any]) -> dict[str, Any]:
    return {key: item for key, item in value.items() if key != "expected_artifact_hash"}
