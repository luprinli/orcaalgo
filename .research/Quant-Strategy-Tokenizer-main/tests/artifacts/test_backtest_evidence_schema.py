from __future__ import annotations

import math

import pytest
from pydantic import ValidationError

from qst.artifacts import ArtifactRef, BacktestEvidence, BacktestStats
from tests.artifacts.schema_helpers import validate_schema

HASH = "sha256:" + "2" * 64


def test_backtest_evidence_schema_validates_model_dump() -> None:
    evidence = BacktestEvidence(
        strategy_instance_hash=HASH,
        stats=BacktestStats(total_return=0.12, num_trades=3, sharpe_ratio=None),
        equity_curve=ArtifactRef(path="artifacts/backtest/equity_curve.csv", hash=HASH),
    )

    validate_schema("backtest_evidence-0.4.schema.json", evidence.model_dump(mode="json"))


@pytest.mark.parametrize("value", [math.nan, math.inf, -math.inf])
def test_backtest_stats_reject_non_finite_values(value: float) -> None:
    with pytest.raises(ValidationError):
        BacktestStats(total_return=value, num_trades=0)

    with pytest.raises(ValidationError):
        BacktestStats(total_return=0.0, num_trades=0, sharpe_ratio=value)


@pytest.mark.parametrize("path", ["/absolute/file.csv", "../escape.csv", "nested\\windows.csv"])
def test_artifact_ref_rejects_unsafe_paths(path: str) -> None:
    with pytest.raises(ValidationError):
        ArtifactRef(path=path, hash=HASH)
