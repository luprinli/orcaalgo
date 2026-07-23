"""FeatureFrame model."""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.artifacts.decimal_string import DecimalString
from qst.frames.base import FrameBase, QSTSymbol, normalize_timestamp


class FeatureRow(BaseModel):
    model_config = ConfigDict(extra="forbid")

    timestamp: datetime
    symbol: QSTSymbol
    features: dict[str, DecimalString] = Field(default_factory=dict)

    @field_validator("timestamp")
    @classmethod
    def _normalize_timestamp(cls, value: datetime) -> datetime:
        return normalize_timestamp(value)

    @model_validator(mode="after")
    def _canonicalize_features(self) -> FeatureRow:
        self.features = {key: self.features[key] for key in sorted(self.features)}
        return self


class FeatureFrame(FrameBase):
    frame_version: Literal["qst-feature-frame/1"] = "qst-feature-frame/1"
    symbols: list[QSTSymbol] = Field(default_factory=list)
    rows: list[FeatureRow] = Field(default_factory=list)

    @model_validator(mode="after")
    def _canonicalize(self) -> FeatureFrame:
        seen: set[tuple[datetime, str]] = set()
        for row in self.rows:
            key = (row.timestamp, row.symbol)
            if key in seen:
                raise ValueError(f"Duplicate feature row for {row.symbol} at {row.timestamp.isoformat()}")
            seen.add(key)

        self.rows = sorted(self.rows, key=lambda row: (row.timestamp, row.symbol))
        self.symbols = sorted(set(self.symbols) | {row.symbol for row in self.rows})
        return self
