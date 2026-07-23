"""Importer entrypoint for Qlib workflow configs."""

from __future__ import annotations

from pathlib import Path

import yaml

from qst.adapters.qlib.coverage import build_coverage
from qst.adapters.qlib.extractor import (
    collect_unsupported,
    extract_dataset,
    extract_model,
    extract_records,
    extract_strategy_and_backtest,
)
from qst.adapters.qlib.mapper import build_candidate_gkr
from qst.adapters.qlib.models import QlibImportResult
from qst.adapters.qlib.workflow_loader import load_workflow_config


def import_qlib_workflow(
    source_path: Path,
    *,
    output_path: Path | None = None,
    coverage_path: Path | None = None,
) -> QlibImportResult:
    config = load_workflow_config(source_path)
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
    source = source_path.as_posix()
    coverage = build_coverage(
        source=source,
        model=model,
        dataset=dataset,
        records=records,
        strategy=strategy,
        backtest=backtest,
        unsupported=unsupported,
    )
    if output_path is not None:
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(
            yaml.safe_dump(build_candidate_gkr(coverage), sort_keys=False),
            encoding="utf-8",
        )
    if coverage_path is not None:
        coverage_path.parent.mkdir(parents=True, exist_ok=True)
        coverage_path.write_text(coverage.model_dump_json(indent=2), encoding="utf-8")
    return QlibImportResult(
        ok=True,
        source=source,
        strategy_path=output_path,
        coverage_path=coverage_path,
        coverage=coverage,
        warnings=coverage.warnings,
    )

