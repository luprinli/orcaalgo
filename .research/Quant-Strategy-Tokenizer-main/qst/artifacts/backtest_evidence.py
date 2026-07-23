"""Backtest evidence artifacts."""

from __future__ import annotations

import math
from typing import Annotated, Literal

from pydantic import AfterValidator, BaseModel, ConfigDict, Field

from qst.artifacts.base import QSTArtifact
from qst.artifacts.safety import POSIXRelativePath
from qst.hash.common import HashString


def _validate_finite(value: float) -> float:
    if not math.isfinite(value):
        raise ValueError(f"Statistic must be finite, got {value!r}")
    return value


def _validate_finite_or_none(value: float | None) -> float | None:
    if value is None:
        return value
    return _validate_finite(value)


FiniteRequiredFloat = Annotated[float, AfterValidator(_validate_finite)]
FiniteOptionalFloat = Annotated[float | None, AfterValidator(_validate_finite_or_none)]


class ArtifactRef(BaseModel):
    """Reference to a package-internal artifact file."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    path: POSIXRelativePath
    hash: HashString


class BacktestStats(BaseModel):
    """Finite backtest statistics."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    total_return: FiniteRequiredFloat
    num_trades: int = Field(ge=0)
    annual_return: FiniteOptionalFloat = None
    annual_volatility: FiniteOptionalFloat = None
    sharpe_ratio: FiniteOptionalFloat = None
    sortino_ratio: FiniteOptionalFloat = None
    max_drawdown: FiniteOptionalFloat = None
    calmar_ratio: FiniteOptionalFloat = None
    win_rate: FiniteOptionalFloat = None
    profit_factor: FiniteOptionalFloat = None


class BacktestEvidence(QSTArtifact):
    """Backtest result evidence artifact."""

    artifact_version: Literal["qst-backtest-evidence/1"] = "qst-backtest-evidence/1"
    strategy_instance_hash: HashString
    market_frame_hash: HashString | None = None
    stats: BacktestStats
    equity_curve: ArtifactRef | None = None
    execution_reports: list[ArtifactRef] = Field(default_factory=list)
    portfolio_snapshots: list[ArtifactRef] = Field(default_factory=list)
