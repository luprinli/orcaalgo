from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from qst.hash import expected_artifact_hash_v2
from qst.ir import (
    load_ir_v04_file,
    trace_temporal_validation_v04,
    validate_temporal_v04,
)

ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "tests" / "reference" / "temporal"
STRATEGIES = REFERENCE / "strategies"
DIAGNOSTICS = REFERENCE / "diagnostics"
TRACES = REFERENCE / "traces"


def test_pv_c_reference_diagnostics_match_and_hash() -> None:
    for path in sorted(DIAGNOSTICS.glob("*.json")):
        strategy_name, profile = _case_from_expected_path(path)
        ir = load_ir_v04_file(STRATEGIES / f"{strategy_name}.gkr.yaml")
        expected = _load_json(path)
        result = validate_temporal_v04(ir, profile=profile)

        material = {
            "artifact_version": expected["artifact_version"],
            "strategy": ir.strategy.id,
            "profile": profile,
            "diagnostics": [
                diagnostic.model_dump(mode="json", exclude_none=True)
                for diagnostic in result.diagnostics
            ],
        }

        assert material == _without_hash(expected)
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(material)


def test_pv_c_reference_traces_match_and_hash() -> None:
    for path in sorted(TRACES.glob("*.json")):
        strategy_name, profile = _case_from_expected_path(path)
        ir = load_ir_v04_file(STRATEGIES / f"{strategy_name}.gkr.yaml")
        expected = _load_json(path)

        trace = trace_temporal_validation_v04(ir, profile=profile).to_artifact()

        assert trace == expected
        assert expected["expected_artifact_hash"] == expected_artifact_hash_v2(_without_hash(expected))


def test_pv_c_gate_cases_have_expected_outcomes() -> None:
    shift = load_ir_v04_file(STRATEGIES / "temporal_shift_future.gkr.yaml")
    centered = load_ir_v04_file(STRATEGIES / "temporal_centered_window.gkr.yaml")
    next_open = load_ir_v04_file(STRATEGIES / "temporal_next_open_prediction.gkr.yaml")

    assert validate_temporal_v04(shift, profile="research").ok
    assert not validate_temporal_v04(shift, profile="pretrade").ok
    assert not validate_temporal_v04(centered, profile="pretrade").ok
    assert validate_temporal_v04(next_open, profile="pretrade").ok


def _case_from_expected_path(path: Path) -> tuple[str, str]:
    strategy_name, profile = path.stem.rsplit(".", 1)
    return strategy_name, profile


def _load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    assert isinstance(loaded, dict)
    return loaded


def _without_hash(value: dict[str, Any]) -> dict[str, Any]:
    return {key: item for key, item in value.items() if key != "expected_artifact_hash"}
