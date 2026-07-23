from __future__ import annotations

from datetime import datetime
from decimal import Decimal
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

OrderSide = Literal["BUY", "SELL"]
OrderState = Literal["CREATED", "BOOKED", "PARTIAL_FILL", "FILLED", "CANCELLED", "REJECTED"]


class TradeSignal(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    symbol: str = Field(min_length=1, max_length=20)
    signal: Literal["BUY", "SELL", "HOLD"]
    confidence: float = Field(ge=0.0, le=1.0)
    reason: str = ""
    timestamp: datetime


class Order(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    order_id: str
    symbol: str
    side: OrderSide
    quantity: Decimal = Field(gt=0)
    price: Decimal | None = None
    order_type: Literal["MARKET", "LIMIT", "STOP", "STOP_LIMIT"]
    state: OrderState = "CREATED"
    filled_quantity: Decimal = Decimal("0")
    created_at: datetime
    updated_at: datetime | None = None


class Fill(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    fill_id: str
    order_id: str
    symbol: str
    side: OrderSide
    fill_quantity: Decimal = Field(gt=0)
    fill_price: Decimal = Field(gt=0)
    filled_at: datetime


class Position(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    symbol: str
    side: OrderSide
    quantity: Decimal = Field(gt=0)
    average_entry_price: Decimal = Field(gt=0)
    unrealized_pnl: Decimal = Decimal("0")
    last_updated: datetime
