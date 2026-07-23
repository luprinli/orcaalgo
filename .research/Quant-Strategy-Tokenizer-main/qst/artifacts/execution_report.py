"""ExecutionReport artifact."""

from __future__ import annotations

from decimal import Decimal
from typing import Literal

from pydantic import model_validator

from qst.artifacts.base import QSTArtifact
from qst.artifacts.decimal_string import DecimalString


class ExecutionReport(QSTArtifact):
    """Order/execution event artifact."""

    artifact_version: Literal["qst-execution-report/1"] = "qst-execution-report/1"
    event_type: Literal["new", "trade", "cancel", "replace", "reject", "expire"]
    state: Literal[
        "pending_new",
        "acknowledged",
        "partially_filled",
        "filled",
        "cancelled",
        "rejected",
        "expired",
    ]
    qty_intended: DecimalString
    qty_last: DecimalString
    qty_filled: DecimalString
    qty_remaining: DecimalString
    last_fill_price: DecimalString | None = None
    avg_fill_price: DecimalString | None = None
    time_in_force: Literal["gtc", "ioc", "fok", "day", "post_only"] | None = None
    reject_reason: str | None = None
    cancel_reason: str | None = None
    liquidity_side: Literal["maker", "taker", "unknown"] | None = None
    source_protocol: str
    source_system: str | None = None
    venue: str

    @model_validator(mode="after")
    def validate_state_machine(self) -> ExecutionReport:
        if self.event_type != "trade":
            if Decimal(self.qty_last) != Decimal("0"):
                raise ValueError(
                    f"event_type={self.event_type} requires qty_last='0', got {self.qty_last!r}"
                )
        else:
            if Decimal(self.qty_last) <= Decimal("0"):
                raise ValueError("event_type=trade requires qty_last > 0")
            if self.last_fill_price is None:
                raise ValueError("event_type=trade requires last_fill_price")
        return self
