"""Portfolio snapshot artifact."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from qst.artifacts.base import QSTArtifact
from qst.artifacts.decimal_string import DecimalString


class Position(BaseModel):
    """One portfolio position."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    symbol: str
    quantity: DecimalString
    market_value: DecimalString
    average_price: DecimalString | None = None


class PortfolioSnapshot(QSTArtifact):
    """Portfolio state artifact."""

    artifact_version: Literal["qst-portfolio-snapshot/1"] = "qst-portfolio-snapshot/1"
    timestamp: str
    base_currency: str
    cash: DecimalString
    equity: DecimalString
    positions: list[Position] = Field(default_factory=list)
