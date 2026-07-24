from __future__ import annotations

from datetime import UTC, datetime

import pytest

from orca.models.strategy import (
    Capability,
    Node,
    StrategyBody,
    StrategyIRV04,
    TokenRef,
)
from orca.models.trade import TradeSignal


@pytest.fixture
def sample_trade_signal() -> TradeSignal:
    return TradeSignal(
        symbol="AAPL",
        signal="BUY",
        confidence=0.75,
        reason="zscore entry",
        timestamp=datetime(2025, 1, 15, 10, 30, tzinfo=UTC),
    )


@pytest.fixture
def sample_strategy_ir() -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBody(
            id="test-strategy-v1",
            version="1.0.0",
            nodes=[
                Node(
                    id="signal_gen",
                    token_ref=TokenRef(token_id="signal.threshold", version=">=1.0"),
                    params={"entry_threshold": 2.0, "exit_threshold": 0.5},
                ),
                Node(
                    id="position_sizer",
                    token_ref=TokenRef(token_id="size.kelly_fractional", version=">=1.0"),
                    params={"multiplier": 0.25, "per_trade_cap_pct": 0.02},
                ),
            ],
            outputs={"signal": "signal_gen.signal", "size": "position_sizer.contracts"},
        ),
        capabilities=[Capability(name="core")],
    )


@pytest.fixture
def sample_trade_records() -> list[dict]:
    return [
        {
            "symbol": "SPY",
            "side": "BUY",
            "entry_price": 450.0,
            "exit_price": 455.0,
            "quantity": 10,
            "pnl": 50.0,
            "cost": 4500.0,
            "confidence": 0.65,
            "placed_at": datetime(2025, 1, 10, 9, 30, tzinfo=UTC),
            "outcome": "win",
        },
        {
            "symbol": "QQQ",
            "side": "SELL",
            "entry_price": 380.0,
            "exit_price": 385.0,
            "quantity": 5,
            "pnl": -25.0,
            "cost": 1900.0,
            "confidence": 0.60,
            "placed_at": datetime(2025, 1, 12, 14, 0, tzinfo=UTC),
            "outcome": "loss",
        },
        {
            "symbol": "TSLA",
            "side": "BUY",
            "entry_price": 250.0,
            "exit_price": 260.0,
            "quantity": 20,
            "pnl": 200.0,
            "cost": 5000.0,
            "confidence": 0.80,
            "placed_at": datetime(2025, 1, 14, 11, 15, tzinfo=UTC),
            "outcome": "win",
        },
    ]
