from __future__ import annotations

from qst.hash import compute_hashes_v2
from qst.ir import load_ir_v04_file, validate_ir_v04
from tests.token_acceptance._helpers import (
    DEMO_CASES,
    DEMO_ROOT,
    REFERENCE_ROOT,
    TRACE_CASES,
    read_json,
)


def test_stage3b_demo_fixture_manifests_are_complete() -> None:
    for case in DEMO_CASES:
        reference = REFERENCE_ROOT / case
        fixture = read_json(reference / "fixture.json")

        assert fixture == {
            "artifact_kind": "stage3b_demo_acceptance_fixture",
            "case": case,
            "diagnostics_path": f"tests/reference/strategies/{case}/diagnostics.json",
            "hashes_path": f"tests/reference/strategies/{case}/hashes.json",
            "readme_path": f"examples/strategies/{case}/README.md",
            "strategy_path": f"examples/strategies/{case}/strategy.gkr.yaml",
            "trace_path": f"tests/reference/strategies/{case}/trace.json"
            if case in TRACE_CASES
            else None,
        }
        assert (DEMO_ROOT / case / "strategy.gkr.yaml").is_file()
        assert (DEMO_ROOT / case / "README.md").read_text(encoding="utf-8").strip()
        assert (reference / "diagnostics.json").is_file()
        assert (reference / "hashes.json").is_file()


def test_stage3b_all_demo_hash_and_diagnostic_sentinels_match() -> None:
    for case in DEMO_CASES:
        ir = load_ir_v04_file(DEMO_ROOT / case / "strategy.gkr.yaml")
        validation = validate_ir_v04(ir)
        expected_diagnostics = read_json(REFERENCE_ROOT / case / "diagnostics.json")
        expected_hashes = read_json(REFERENCE_ROOT / case / "hashes.json")

        assert [diagnostic.model_dump(mode="json") for diagnostic in validation.diagnostics] == (
            expected_diagnostics["diagnostics"]
        )
        assert validation.ok
        assert compute_hashes_v2(ir).__dict__ == expected_hashes


def test_stage3b_demo_trace_set_is_exact() -> None:
    traced_cases = {path.parent.name for path in REFERENCE_ROOT.glob("*/trace.json")}

    assert traced_cases == TRACE_CASES
