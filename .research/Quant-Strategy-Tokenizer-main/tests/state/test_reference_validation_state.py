from __future__ import annotations

import pytest

from qst.ir import load_ir_v04
from qst.state import (
    StatePVAFixture,
    diagnostics_state_pv_a_v04,
    run_state_pv_a_case,
    trace_state_pv_a_v04,
)


def test_pv_a_runner_dispatches_cooldown_case() -> None:
    ir = load_ir_v04(
        {
            "ir_version": "qst-ir/0.4",
            "canonical_version": "qst-canonical/0.4",
            "strategy": {"id": "state_cooldown", "nodes": [], "outputs": {}},
            "metadata": {"reference_case": "state_cooldown"},
        }
    )
    fixture = StatePVAFixture(
        case="state_cooldown",
        events=["signal", "fill", "signal", "cooldown_expired"],
    )

    result = run_state_pv_a_case(ir, fixture)

    assert result.validation_result.ok
    assert result.outputs["states"] == ["ready", "cooldown", "cooldown", "ready"]
    assert result.outputs["decisions"] == ["active", "active", "blocked", "active"]
    assert result.replay_checks == {"fsm": True}


def test_pv_a_runner_reports_case_mismatch_diagnostic() -> None:
    ir = load_ir_v04(
        {
            "ir_version": "qst-ir/0.4",
            "canonical_version": "qst-canonical/0.4",
            "strategy": {"id": "bad_case", "nodes": [], "outputs": {}},
            "metadata": {"reference_case": "state_missing"},
        }
    )
    fixture = StatePVAFixture(case="state_cooldown", events=["signal"])

    result = run_state_pv_a_case(ir, fixture)

    assert not result.validation_result.ok
    assert [diagnostic.code for diagnostic in result.diagnostics] == [
        "QST_V2_STATE_PVA_CASE_MISMATCH"
    ]


def test_pv_a_fixture_payload_must_be_canonical_json() -> None:
    with pytest.raises(ValueError, match="canonical JSON-compatible"):
        StatePVAFixture(
            case="state_cooldown",
            events=[float("nan")],
        )


def test_pv_a_trace_and_diagnostics_artifacts_include_hashes() -> None:
    ir = load_ir_v04(
        {
            "ir_version": "qst-ir/0.4",
            "canonical_version": "qst-canonical/0.4",
            "strategy": {"id": "state_slot_budget_minimal", "nodes": [], "outputs": {}},
            "metadata": {"reference_case": "state_slot_budget_minimal"},
        }
    )
    fixture = StatePVAFixture(
        case="state_slot_budget_minimal",
        values=[1, 1, 1],
        params={"budget": 2},
    )

    trace = trace_state_pv_a_v04(ir, fixture)
    diagnostics = diagnostics_state_pv_a_v04(ir, fixture)

    assert trace.expected_artifact_hash.startswith("sha256:")
    assert diagnostics["expected_artifact_hash"].startswith("sha256:")
    assert trace.outputs["decisions"] == ["active", "active", "blocked"]
