from __future__ import annotations

from datetime import datetime
from decimal import Decimal
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

DrawdownLevel = Literal["CLEAR", "WARN", "DERISK", "HALT"]


class BreachCondition(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    code: str
    current_value: float
    threshold: float
    message: str


class KillSwitchState(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    is_locked: bool = False
    reason: str = ""
    triggered_at: datetime | None = None


class RiskSnapshot(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    timestamp: datetime
    balance: Decimal = Field(ge=0)
    equity: Decimal = Field(ge=0)
    floating_pnl: Decimal = Decimal("0")
    high_water_mark: Decimal = Field(ge=0)
    daily_drawdown_pct: float = Field(ge=0.0)
    absolute_drawdown: Decimal = Field(ge=0)
    open_position_count: int = Field(ge=0)
    daily_trade_count: int = Field(ge=0)
    regime_multiplier: float = Field(default=1.0, ge=0.0, le=2.0)
    margin_used: Decimal = Field(ge=0)
    drawdown_level: DrawdownLevel = "CLEAR"
    kill_switch: KillSwitchState = Field(default_factory=KillSwitchState)
    breaches: list[BreachCondition] = Field(default_factory=list)
