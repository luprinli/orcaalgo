"""MarketFrame model."""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.artifacts.decimal_string import DecimalString
from qst.frames.base import FrameBase, QSTSymbol, normalize_timestamp


class OHLCVBar(BaseModel):
    model_config = ConfigDict(extra="forbid")

    timestamp: datetime
    symbol: QSTSymbol
    open: DecimalString
    high: DecimalString
    low: DecimalString
    close: DecimalString
    volume: DecimalString

    @field_validator("timestamp")
    @classmethod
    def _normalize_timestamp(cls, value: datetime) -> datetime:
        return normalize_timestamp(value)


class MarketFrame(FrameBase):
    frame_version: Literal["qst-market-frame/1"] = "qst-market-frame/1"
    symbols: list[QSTSymbol] = Field(default_factory=list)
    bars: list[OHLCVBar] = Field(default_factory=list)

    @model_validator(mode="after")
    def _canonicalize(self) -> MarketFrame:
        seen: set[tuple[datetime, str]] = set()
        for bar in self.bars:
            key = (bar.timestamp, bar.symbol)
            if key in seen:
                raise ValueError(f"Duplicate OHLCV bar for {bar.symbol} at {bar.timestamp.isoformat()}")
            seen.add(key)

        self.bars = sorted(self.bars, key=lambda bar: (bar.timestamp, bar.symbol))
        self.symbols = sorted(set(self.symbols) | {bar.symbol for bar in self.bars})
        self._validate_strict_alignment(seen)
        return self

    def _validate_strict_alignment(self, seen: set[tuple[datetime, str]]) -> None:
        if len(self.symbols) <= 1 or not self.bars:
            return

        timestamps = sorted({bar.timestamp for bar in self.bars})
        missing = [
            (timestamp, symbol)
            for timestamp in timestamps
            for symbol in self.symbols
            if (timestamp, symbol) not in seen
        ]
        if not missing:
            return

        sample = ", ".join(f"{timestamp.isoformat()} {symbol}" for timestamp, symbol in missing[:5])
        raise ValueError(
            "MarketFrame strict alignment violation: "
            f"missing {len(missing)} OHLCV bar(s); sample=[{sample}]"
        )
