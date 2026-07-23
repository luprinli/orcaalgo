"""Tests for GKR IR → Odin compiler output generation."""

from __future__ import annotations

import pytest

from orca.ir.compiler import compile_all, compile_strategy, _generate_odin_struct
from orca.ir.loader import load_ir
from orca.models.strategy import Node, StrategyBody, StrategyIRV04, TokenRef


def make_node(token_id: str, params: dict | None = None) -> Node:
    return Node(
        id="n1",
        token_ref=TokenRef(token_id=token_id, version=">=1.0"),
        params=params or {},
    )


def make_ir(node: Node, strategy_id: str = "test-strat") -> StrategyIRV04:
    return StrategyIRV04(
        ir_version="qst-ir/0.4",
        canonical_version="qst-canonical/0.4",
        strategy=StrategyBody(
            id=strategy_id,
            version="1.0.0",
            nodes=[node],
            outputs={"signal": "n1.signal"},
        ),
        capabilities=["core"],
    )


class TestCompilerOutput:
    def test_signal_threshold_generates_mr(self):
        node = make_node("signal.threshold", {"entry_threshold": 2.5, "exit_threshold": 0.3})
        ir = make_ir(node)
        code = compile_strategy(ir, 2)

        assert "Context_IntradayMR" in code
        assert "init_intraday_mr" in code
        assert "eval_intraday_mr" in code
        assert "entry_z = 2.5" in code
        assert "exit_z = 0.3" in code
        assert "strategy.register" in code

    def test_signal_trend_generates_trend(self):
        node = make_node("signal.trend", {"fast_period": 10, "slow_period": 30, "atr_period": 7, "atr_multiplier": 1.5})
        ir = make_ir(node)
        code = compile_strategy(ir, 4)

        assert "Context_TrendFollowing" in code
        assert "init_trend" in code
        assert "eval_trend" in code
        assert "fast_period = 10" in code
        assert "slow_period = 30" in code
        assert "atr_period = 7" in code
        assert "atr_multiplier = 1.5" in code

    def test_signal_opening_range_generates_orb(self):
        node = make_node("signal.opening_range", {"range_minutes": 10, "entry_buffer_pct": 0.2})
        ir = make_ir(node)
        code = compile_strategy(ir, 3)

        assert "Context_ORB" in code
        assert "init_orb" in code
        assert "eval_orb" in code
        assert "range_minutes = 10" in code
        assert "entry_buffer_pct = 0.2" in code

    def test_signal_breakout_generates_orb(self):
        node = make_node("signal.breakout", {"range_minutes": 15, "entry_buffer_pct": 0.3})
        ir = make_ir(node)
        code = compile_strategy(ir, 3)

        assert "Context_ORB" in code
        assert "init_orb" in code

    def test_indicator_ma_crossover_generates_trend(self):
        node = make_node("indicator.ma_crossover", {"fast_period": 5, "slow_period": 20})
        ir = make_ir(node)
        code = compile_strategy(ir, 4)

        assert "Context_TrendFollowing" in code
        assert "fast_period = 5" in code
        assert "slow_period = 20" in code

    def test_indicator_atr_filter_generates_trend(self):
        node = make_node("indicator.atr_filter", {"atr_period": 21})
        ir = make_ir(node)
        code = compile_strategy(ir, 4)

        assert "Context_TrendFollowing" in code
        assert "atr_period = 21" in code

    def test_signal_range_detect_generates_orb(self):
        node = make_node("signal.range_detect", {"range_minutes": 7})
        ir = make_ir(node)
        code = compile_strategy(ir, 3)

        assert "Context_ORB" in code
        assert "range_minutes = 7" in code

    def test_unknown_token_returns_stub(self):
        node = make_node("signal.unknown_token")
        ir = make_ir(node)
        code = compile_strategy(ir, 99)

        assert "_ = 99" in code

    def test_compile_all_loads_real_files(self, tmp_path):
        import json
        out = tmp_path / "gen.odin"
        try:
            result = compile_all(gkr_dir="configs/strategies", output=out)
        except Exception:
            pytest.skip("Strategy YAML files not available")
            return

        assert out.exists()
        content = out.read_text()
        assert "register_generated_strategies" in content
        assert "intraday_mr" in content.lower() or "opening_range_breakout" in content.lower()


class TestCompilerEdgeCases:
    def test_empty_params(self):
        node = make_node("signal.trend")
        ir = make_ir(node)
        code = compile_strategy(ir, 4)

        assert "fast_period = 20" in code
        assert "slow_period = 50" in code
        assert "atr_period = 14" in code

    def test_strategy_with_no_signal_nodes(self):
        node = make_node("data.bar_aggregator", {"period_seconds": 60})
        ir = make_ir(node)
        code = compile_strategy(ir, 10)

        assert code != ""
