from __future__ import annotations

from pathlib import Path
from typing import ClassVar

from orca.ir.loader import load_ir
from orca.ir.validator import validate_ir
from orca.models.strategy import (
    Capability,
    Node,
    OutputSpec,
    PortSignature,
    PortTemporalSpec,
    StrategyBody,
    StrategyIRV04,
    TokenRef,
    TypeSpec,
)

STRATEGIES_DIR = Path(__file__).parent.parent / "configs" / "strategies"


class TestStrategyLoading:
    def test_load_intraday_mr(self):
        ir = load_ir(STRATEGIES_DIR / "intraday_mr.gkr.yaml")
        assert ir.strategy.id == "orca-intraday-mr-v1"
        assert len(ir.strategy.nodes) == 6
        assert ir.ir_version == "qst-ir/0.4"
        assert ir.canonical_version == "qst-canonical/0.4"
        node_ids = {n.id for n in ir.strategy.nodes}
        assert node_ids == {
            "bar_agg",
            "zscore_calc",
            "signal_gen",
            "regime_gate",
            "position_sizer",
            "risk_gate",
        }

    def test_load_opening_range_breakout(self):
        ir = load_ir(STRATEGIES_DIR / "opening_range_breakout.gkr.yaml")
        assert ir.strategy.id == "orca-opening-range-breakout-v1"
        assert len(ir.strategy.nodes) == 5
        node_ids = {n.id for n in ir.strategy.nodes}
        assert node_ids == {
            "range_detect",
            "breakout_signal",
            "regime_gate",
            "position_sizer",
            "risk_gate",
        }

    def test_load_trend_following(self):
        ir = load_ir(STRATEGIES_DIR / "trend_following.gkr.yaml")
        assert ir.strategy.id == "orca-trend-following-v1"
        assert len(ir.strategy.nodes) == 6
        node_ids = {n.id for n in ir.strategy.nodes}
        assert node_ids == {
            "ma_crossover",
            "atr_filter",
            "trend_signal",
            "regime_gate",
            "position_sizer",
            "risk_gate",
        }

    def test_all_strategies_have_core_capability(self):
        for f in STRATEGIES_DIR.glob("*.gkr.yaml"):
            ir = load_ir(f)
            has_core = any(c.name == "core" for c in ir.capabilities)
            assert has_core, f"{f.name} missing core capability"

    def test_all_strategies_pass_research_validation(self):
        for f in STRATEGIES_DIR.glob("*.gkr.yaml"):
            ir = load_ir(f)
            diags = validate_ir(ir, profile="research")
            errors = [d for d in diags if d.severity == "error"]
            assert not errors, f"{f.name} has errors: {[(e.code, e.message) for e in errors]}"

    def test_all_strategies_pass_production_guarded(self):
        for f in STRATEGIES_DIR.glob("*.gkr.yaml"):
            ir = load_ir(f)
            diags = validate_ir(ir, profile="production_guarded")
            errors = [d for d in diags if d.severity == "error"]
            assert not errors, f"{f.name} has errors: {[(e.code, e.message) for e in errors]}"

    def test_strategies_have_valid_output_refs(self):
        for f in STRATEGIES_DIR.glob("*.gkr.yaml"):
            ir = load_ir(f)
            node_ids = {n.id for n in ir.strategy.nodes}
            for out_name, out_ref in ir.strategy.outputs.items():
                parts = out_ref.split(".")
                assert len(parts) == 2, (
                    f"{f.name}: output '{out_name}' ref '{out_ref}' invalid format"
                )
                assert parts[0] in node_ids, (
                    f"{f.name}: output '{out_name}' references unknown node '{parts[0]}'"
                )

    def test_strategies_have_valid_input_refs(self):
        for f in STRATEGIES_DIR.glob("*.gkr.yaml"):
            ir = load_ir(f)
            node_ids = {n.id for n in ir.strategy.nodes}
            for node in ir.strategy.nodes:
                for inp_port, inp_ref in node.inputs.items():
                    parts = inp_ref.split(".")
                    assert len(parts) == 2, (
                        f"{f.name}: node '{node.id}' input '{inp_port}' ref '{inp_ref}' invalid"
                    )
                    assert parts[0] in node_ids, (
                        f"{f.name}: node '{node.id}' references unknown upstream '{parts[0]}'"
                    )

    def test_output_refs_integrity(self):
        ir = load_ir(STRATEGIES_DIR / "intraday_mr.gkr.yaml")
        assert ir.strategy.outputs == {
            "signal": "signal_gen.signal",
            "size": "position_sizer.contracts",
            "final": "risk_gate.approved_size",
        }

    def test_kelly_params_enforce_fractional(self):
        ir = load_ir(STRATEGIES_DIR / "intraday_mr.gkr.yaml")
        sizer = next(n for n in ir.strategy.nodes if n.id == "position_sizer")
        assert sizer.params["multiplier"] == 0.25
        assert sizer.params["multiplier"] < 1.0


class TestValidatorErrors:
    def test_missing_core_capability(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="no-core",
                version="1.0.0",
                nodes=[
                    Node(
                        id="n1",
                        token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                    )
                ],
                outputs={"out": "n1.output"},
            ),
            capabilities=[],
        )
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "MISSING_CORE_CAPABILITY" in codes

    def test_duplicate_node_ids(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="dup-nodes",
                version="1.0.0",
                nodes=[
                    Node(id="n1", token_ref=TokenRef(token_id="t1", version=">=1.0")),
                    Node(id="n2", token_ref=TokenRef(token_id="t2", version=">=1.0")),
                    Node(id="n1", token_ref=TokenRef(token_id="t3", version=">=1.0")),
                ],
                outputs={"out": "n1.output"},
            ),
            capabilities=[Capability(name="core")],
        )
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "DUPLICATE_NODE_ID" in codes

    def test_invalid_output_ref(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="bad-output",
                version="1.0.0",
                nodes=[
                    Node(
                        id="n1",
                        token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                    )
                ],
                outputs={"out": "nonexistent_node.signal"},
            ),
            capabilities=[Capability(name="core")],
        )
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "INVALID_OUTPUT_REF" in codes
        assert "MISSING_INPUT_NODE" not in codes

    def test_missing_input_node(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="missing-input",
                version="1.0.0",
                nodes=[
                    Node(
                        id="n1",
                        token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                    ),
                    Node(
                        id="n2",
                        token_ref=TokenRef(token_id="signal.threshold", version=">=1.0"),
                        inputs={"value": "no_such_node.zscore"},
                    ),
                ],
                outputs={"out": "n1.output"},
            ),
            capabilities=[Capability(name="core")],
        )
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "MISSING_INPUT_NODE" in codes
        assert "INVALID_OUTPUT_REF" not in codes


class TestProfileGating:
    def test_research_allows_unsafe_future(self):
        ir = _make_unsafe_future_strategy()
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        err_detail = [(e.code, e.message) for e in errors]
        assert not errors, f"research profile should allow unsafe_future, got: {err_detail}"

    def test_paper_allows_unsafe_future(self):
        ir = _make_unsafe_future_strategy()
        diags = validate_ir(ir, profile="paper")
        errors = [d for d in diags if d.severity == "error"]
        assert not errors, "paper profile should allow unsafe_future"

    def test_pretrade_rejects_unsafe_future(self):
        ir = _make_unsafe_future_strategy()
        diags = validate_ir(ir, profile="pretrade")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "UNSAFE_FUTURE" in codes

    def test_production_guarded_rejects_unsafe_future(self):
        ir = _make_unsafe_future_strategy()
        diags = validate_ir(ir, profile="production_guarded")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "UNSAFE_FUTURE" in codes

    def test_invalid_profile_returns_error(self):
        ir = _make_simple_strategy()
        diags = validate_ir(ir, profile="nonexistent")
        errors = [d for d in diags if d.severity == "error"]
        assert any(d.code == "INVALID_PROFILE" for d in errors)

    def test_clean_strategy_passes_all_profiles(self):
        ir = _make_simple_strategy()
        for profile in ["research", "paper", "pretrade", "production_guarded"]:
            diags = validate_ir(ir, profile=profile)
            errors = [d for d in diags if d.severity == "error"]
            detail = [(e.code, e.message) for e in errors]
            assert not errors, f"profile '{profile}' has errors: {detail}"


class TestIRVersion:
    def test_wrong_ir_version_rejected(self):
        ir = _make_simple_strategy()
        object.__setattr__(ir, "ir_version", "qst-ir/0.3")
        diags = validate_ir(ir, profile="research")
        errors = [d for d in diags if d.severity == "error"]
        codes = {d.code for d in errors}
        assert "IR_VERSION_MISMATCH" in codes


def _make_simple_strategy() -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBody(
            id="simple-v1",
            version="1.0.0",
            nodes=[
                Node(
                    id="n1",
                    token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                )
            ],
            outputs={"out": "n1.output"},
        ),
        capabilities=[Capability(name="core")],
    )


def _make_unsafe_future_strategy() -> StrategyIRV04:
    unsafe_sig = PortSignature(
        outputs={
            "prediction": OutputSpec(
                type=TypeSpec(kind="Scalar", value_type="float"),
                port_temporal=PortTemporalSpec(unsafe_future=True),
                temporal_rule=None,
            )
        },
        inputs={},
    )
    return StrategyIRV04(
        strategy=StrategyBody(
            id="unsafe-v1",
            version="1.0.0",
            nodes=[
                Node(
                    id="predictor",
                    token_ref=TokenRef(token_id="model.future_lookahead", version=">=1.0"),
                    signature=unsafe_sig,
                ),
                Node(
                    id="sizer",
                    token_ref=TokenRef(token_id="size.uniform", version=">=1.0"),
                    inputs={"prediction": "predictor.prediction"},
                ),
            ],
            outputs={"size": "sizer.size"},
        ),
        capabilities=[Capability(name="core")],
    )


class TestGKRRegimeGates:
    """Verify all 8 GKR configs have correct blocked_states aligned with RegimeActivationMatrix."""

    KNOWN_STRATEGIES: ClassVar[dict[str, str]] = {
        "grid.gkr.yaml": "orca-grid-trading-v1",
        "intraday_mr.gkr.yaml": "orca-intraday-mr-v1",
        "opening_range_breakout.gkr.yaml": "orca-opening-range-breakout-v1",
        "session_scalp.gkr.yaml": "orca-session-scalp-v1",
        "trend_following.gkr.yaml": "orca-trend-following-v1",
        "volatility_harvesting.gkr.yaml": "orca-volatility-harvesting-v1",
        "pairs_trading.gkr.yaml": "orca-pairs-trading-v1",
        "dragon_trend.gkr.yaml": "orca-dragon-trend-v1",
        "vwap_mr.gkr.yaml": "orca-vwap-mr-v1",
        "orb_15m.gkr.yaml": "orca-orb-15m-v1",
        "volume_scalp.gkr.yaml": "orca-volume-scalp-v1",
    }

    EXPECTED_BLOCKED: ClassVar[dict[str, list[int]]] = {
        "grid.gkr.yaml": [1, 2, 3],
        "intraday_mr.gkr.yaml": [1, 2, 3],
        "opening_range_breakout.gkr.yaml": [0, 3],
        "session_scalp.gkr.yaml": [3],
        "trend_following.gkr.yaml": [0, 2, 3],
        "volatility_harvesting.gkr.yaml": [0, 1, 3],
        "pairs_trading.gkr.yaml": [1, 3],
        "dragon_trend.gkr.yaml": [0, 3],
        "vwap_mr.gkr.yaml": [1, 2, 3],
        "orb_15m.gkr.yaml": [0, 3],
        "volume_scalp.gkr.yaml": [2, 3],
    }

    @staticmethod
    def _find_regime_gate(nodes):
        for node in nodes:
            if node.token_ref and node.token_ref.token_id == "risk.regime_filter":
                return node
        return None

    def test_all_configs_load_and_have_regime_gate(self):
        for filename, strategy_id in self.KNOWN_STRATEGIES.items():
            path = STRATEGIES_DIR / filename
            ir = load_ir(path)
            assert ir.strategy.id == strategy_id, f"{filename}: wrong strategy ID"
            gate = self._find_regime_gate(ir.strategy.nodes)
            assert gate is not None, f"{filename}: missing regime_gate node"

    def test_blocked_states_align_with_matrix(self):
        for filename, expected in self.EXPECTED_BLOCKED.items():
            path = STRATEGIES_DIR / filename
            ir = load_ir(path)
            gate = self._find_regime_gate(ir.strategy.nodes)
            assert gate is not None, f"{filename}: no regime gate found"
            blocked = gate.params.get("blocked_states", [])
            assert blocked == expected, f"{filename}: blocked_states={blocked}, expected={expected}"

    def test_crisis_is_always_blocked(self):
        """Crisis (regime 3) should be blocked for ALL strategies."""
        for filename in self.KNOWN_STRATEGIES:
            path = STRATEGIES_DIR / filename
            ir = load_ir(path)
            gate = self._find_regime_gate(ir.strategy.nodes)
            if gate is None:
                continue
            blocked = gate.params.get("blocked_states", [])
            assert 3 in blocked, f"{filename}: Crisis (regime 3) must be blocked, got {blocked}"
