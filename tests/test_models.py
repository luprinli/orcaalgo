from __future__ import annotations

from datetime import datetime, timezone

import pytest
from pydantic import ValidationError

from orca.models.risk import BreachCondition, DrawdownLevel, KillSwitchState, RiskSnapshot
from orca.models.strategy import (
    Node,
    StrategyBody,
    StrategyIRV04,
    TokenRef,
)
from orca.models.trade import Order, OrderSide, OrderState, TradeSignal


class TestTradeSignal:
    def test_create_valid_signal(self):
        s = TradeSignal(symbol="SPY", signal="BUY", confidence=0.75, timestamp=datetime.now(timezone.utc))
        assert s.signal == "BUY"
        assert s.confidence == 0.75

    def test_rejects_invalid_confidence(self):
        with pytest.raises(ValidationError):
            TradeSignal(symbol="SPY", signal="BUY", confidence=1.5, timestamp=datetime.now(timezone.utc))

    def test_rejects_invalid_signal(self):
        with pytest.raises(ValidationError):
            TradeSignal(symbol="SPY", signal="INVALID", confidence=0.5, timestamp=datetime.now(timezone.utc))

    def test_rejects_extra_fields(self):
        with pytest.raises(ValidationError):
            TradeSignal(symbol="SPY", signal="BUY", confidence=0.5, timestamp=datetime.now(timezone.utc), extra="nope")

    def test_frozen_prevents_mutation(self, sample_trade_signal):
        with pytest.raises(ValidationError):
            sample_trade_signal.confidence = 0.9


class TestStrategyIR:
    def test_minimal_strategy(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="test",
                version="1.0.0",
                nodes=[
                    Node(
                        id="n1",
                        token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                    )
                ],
                outputs={"out": "n1.output"},
            ),
        )
        assert ir.ir_version == "qst-ir/0.4"
        assert ir.strategy.id == "test"

    def test_rejects_empty_nodes(self):
        with pytest.raises(ValidationError):
            StrategyIRV04(
                strategy=StrategyBody(
                    id="test",
                    version="1.0.0",
                    nodes=[],
                    outputs={},
                ),
            )

    def test_graph_hash_is_deterministic(self, sample_strategy_ir):
        from orca.hash.graph import graph_hash_v2

        h1 = graph_hash_v2(sample_strategy_ir)
        h2 = graph_hash_v2(sample_strategy_ir)
        assert h1 == h2
        assert h1.startswith("sha256:")

    def test_param_hash_differs_from_graph_hash(self, sample_strategy_ir):
        from orca.hash.graph import graph_hash_v2, param_hash_v2

        gh = graph_hash_v2(sample_strategy_ir)
        ph = param_hash_v2(sample_strategy_ir)
        assert gh != ph

    def test_node_order_does_not_affect_hash(self):
        ir1 = StrategyIRV04(
            strategy=StrategyBody(
                id="test",
                version="1.0.0",
                nodes=[
                    Node(id="a", token_ref=TokenRef(token_id="t1", version=">=1.0"), params={"p": 1}),
                    Node(id="b", token_ref=TokenRef(token_id="t2", version=">=1.0"), params={"p": 2}),
                ],
                outputs={"out": "a.output"},
            ),
        )
        ir2 = StrategyIRV04(
            strategy=StrategyBody(
                id="test",
                version="1.0.0",
                nodes=[
                    Node(id="b", token_ref=TokenRef(token_id="t2", version=">=1.0"), params={"p": 2}),
                    Node(id="a", token_ref=TokenRef(token_id="t1", version=">=1.0"), params={"p": 1}),
                ],
                outputs={"out": "a.output"},
            ),
        )
        from orca.hash.graph import graph_hash_v2

        assert graph_hash_v2(ir1) == graph_hash_v2(ir2)


class TestRiskModels:
    def test_risk_snapshot_defaults(self):
        snap = RiskSnapshot(
            timestamp=datetime.now(timezone.utc),
            balance=100000,
            equity=99500,
            daily_drawdown_pct=0.5,
            absolute_drawdown=500,
            high_water_mark=100000,
            open_position_count=2,
            daily_trade_count=5,
            margin_used=15000,
        )
        assert snap.drawdown_level == "CLEAR"
        assert snap.kill_switch.is_locked is False

    def test_breach_condition(self):
        bc = BreachCondition(code="DAILY_DD", current_value=5.2, threshold=5.0, message="DD exceeded")
        assert bc.code == "DAILY_DD"
        assert bc.current_value >= bc.threshold
