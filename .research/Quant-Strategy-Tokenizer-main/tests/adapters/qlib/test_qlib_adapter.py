from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import yaml

from qst.adapters.qlib.extractor import (
    collect_unsupported,
    extract_dataset,
    extract_model,
    extract_records,
    extract_strategy_and_backtest,
)
from qst.adapters.qlib.importer import import_qlib_workflow
from qst.adapters.qlib.workflow_loader import load_workflow_config
from qst.hash import compute_hashes_v2
from qst.ir import canonical_bytes_v04, load_ir_v04_file, validate_ir_v04

ROOT = Path(__file__).resolve().parents[3]
QLIB_EXAMPLES = ROOT / "examples" / "adapters" / "qlib"
QLIB_EXAMPLES_REL = Path("examples") / "adapters" / "qlib"
EXPECTED = QLIB_EXAMPLES / "expected"


def _example(name: str) -> Path:
    return QLIB_EXAMPLES_REL / f"workflow_config_{name}.yaml"


def _expected(name: str) -> dict[str, object]:
    return json.loads((EXPECTED / f"{name}.coverage.json").read_text(encoding="utf-8"))


def test_loader_normalizes_yaml_dates_without_importing_qlib() -> None:
    config = load_workflow_config(_example("lightgbm_alpha158"))

    handler_kwargs = config["dataset"]["kwargs"]["handler"]["kwargs"]
    segments = config["dataset"]["kwargs"]["segments"]

    assert handler_kwargs["start_time"] == "2008-01-01"
    assert segments["train"] == ["2008-01-01", "2014-12-31"]
    assert config["model"]["class"] == "LGBModel"


def test_extractor_captures_supported_lightgbm_alpha158_workflow() -> None:
    config = load_workflow_config(_example("lightgbm_alpha158"))
    model = extract_model(config)
    dataset = extract_dataset(config)
    records = extract_records(config)
    strategy, backtest = extract_strategy_and_backtest(records)
    unsupported = collect_unsupported(
        model=model,
        dataset=dataset,
        strategy=strategy,
        records=records,
    )

    assert model is not None
    assert model.class_name == "LGBModel"
    assert dataset is not None
    assert dataset.handler_class == "Alpha158"
    assert [record.class_name for record in records] == ["SignalRecord", "PortAnaRecord"]
    assert strategy is not None
    assert strategy.class_name == "TopkDropoutStrategy"
    assert backtest is not None
    assert backtest.deal_price == "close"
    assert unsupported == []


def test_importer_writes_valid_candidate_gkr_and_coverage(tmp_path: Path) -> None:
    output = tmp_path / "qlib_candidate.gkr.yaml"
    coverage_path = tmp_path / "qlib_candidate.coverage.json"

    result = import_qlib_workflow(
        _example("lightgbm_alpha158"),
        output_path=output,
        coverage_path=coverage_path,
    )

    assert result.strategy_path == output
    assert result.coverage_path == coverage_path
    assert result.coverage.classification == "supported"
    assert output.exists()
    assert coverage_path.exists()

    ir = load_ir_v04_file(output)
    validation = validate_ir_v04(ir)
    assert validation.ok, validation.diagnostics

    hashes = compute_hashes_v2(ir)
    canonical = canonical_bytes_v04(ir)
    assert hashes.graph_hash.startswith("sha256:")
    assert hashes.param_hash.startswith("sha256:")
    assert hashes.instance_hash.startswith("sha256:")
    assert canonical.startswith(b"{")

    token_refs = [node.token_ref for node in ir.strategy.nodes]
    assert token_refs
    assert all(token_ref is not None for token_ref in token_refs)
    assert all(token_ref.namespace == "adapter" for token_ref in token_refs if token_ref)


def test_expected_coverage_fixtures_match_adapter_output() -> None:
    cases = {
        "lightgbm_alpha158": "lightgbm_alpha158",
        "custom_model": "custom_model",
        "custom_processor": "custom_processor",
    }
    for workflow_name, expected_name in cases.items():
        result = import_qlib_workflow(_example(workflow_name))
        assert result.coverage.model_dump(mode="json") == _expected(expected_name)


def test_custom_model_and_processor_are_partial_not_supported() -> None:
    custom_model = import_qlib_workflow(_example("custom_model")).coverage
    custom_processor = import_qlib_workflow(_example("custom_processor")).coverage

    assert custom_model.classification == "partially_supported"
    assert [item.kind for item in custom_model.unsupported_components] == ["custom_model"]
    assert custom_processor.classification == "partially_supported"
    assert [item.kind for item in custom_processor.unsupported_components] == [
        "custom_processor"
    ]


def test_cli_import_validate_hash_and_canonicalize(tmp_path: Path) -> None:
    output = tmp_path / "cli_candidate.gkr.yaml"
    coverage = tmp_path / "cli_candidate.coverage.json"
    canonical = tmp_path / "cli_candidate.canonical.json"

    subprocess.run(
        [
            sys.executable,
            "-m",
            "qst.cli",
            "adapter",
            "qlib",
            "import",
            str(_example("lightgbm_alpha158")),
            "--output",
            str(output),
            "--coverage",
            str(coverage),
        ],
        cwd=ROOT,
        check=True,
        text=True,
        capture_output=True,
    )

    for command in [
        ["validate", str(output)],
        ["hash", str(output)],
        ["canonicalize", str(output), "--output", str(canonical)],
    ]:
        subprocess.run(
            [sys.executable, "-m", "qst.cli", *command],
            cwd=ROOT,
            check=True,
            text=True,
            capture_output=True,
        )

    assert canonical.exists()
    coverage_data = json.loads(coverage.read_text(encoding="utf-8"))
    assert coverage_data["classification"] == "supported"


def test_adapter_source_does_not_import_or_execute_qlib() -> None:
    for path in (ROOT / "qst" / "adapters" / "qlib").glob("*.py"):
        text = path.read_text(encoding="utf-8")
        assert "import qlib" not in text
        assert "from qlib" not in text
        assert "subprocess" not in text


def test_candidate_yaml_is_canonical_json_compatible(tmp_path: Path) -> None:
    output = tmp_path / "candidate.gkr.yaml"
    result = import_qlib_workflow(_example("lightgbm_alpha158"), output_path=output)

    loaded = yaml.safe_load(output.read_text(encoding="utf-8"))
    assert loaded["metadata"]["source_adapter"] == "qlib"
    assert loaded["metadata"]["lossless"] is False
    assert loaded["metadata"]["runtime_execution"] is False
    assert loaded["metadata"]["classification"] == result.coverage.classification
