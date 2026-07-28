from __future__ import annotations

from orca.models.strategy import (
    Node,
    OutputSpec,
    PortSignature,
    PortTemporalSpec,
    StrategyBody,
    StrategyIRV04,
    TemporalRule,
    TokenRef,
    TypeSpec,
)
from orca.ports.temporal import trace_temporal_validation


def _make_node(node_id, *, unsafe_future_declared=False, unsafe_future_rule=False):
    return Node(
        id=node_id,
        token_ref=TokenRef(token_id="core.threshold", version=">=1.0"),
        params={"threshold": 2.0},
        signature=PortSignature(
            outputs={
                "signal": OutputSpec(
                    type=TypeSpec(kind="Decision", value_type="float"),
                    port_temporal=PortTemporalSpec(
                        unsafe_future=unsafe_future_declared,
                    ),
                    temporal_rule=TemporalRule(
                        kind="constant",
                        value={"unsafe_future": unsafe_future_rule},
                    ),
                ),
            },
        ),
    )


class TestTraceTemporalValidation:
    def test_empty_nodes_returns_no_diagnostics(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="empty",
                version="1.0.0",
                nodes=[
                    Node(
                        id="bare",
                        token_ref=TokenRef(token_id="core.passthrough", version=">=1.0"),
                    ),
                ],
            ),
        )
        result = trace_temporal_validation(ir, "research")
        assert result == []

    def test_node_without_signature_returns_no_diagnostics(self):
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="no-sig",
                version="1.0.0",
                nodes=[
                    Node(
                        id="no_sig",
                        token_ref=TokenRef(token_id="core.threshold", version=">=1.0"),
                        signature=None,
                    ),
                ],
            ),
        )
        result = trace_temporal_validation(ir, "research")
        assert result == []

    def test_matching_temporal_rules_returns_no_diagnostics(self):
        node = _make_node("match", unsafe_future_declared=False, unsafe_future_rule=False)
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="matching",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "production_guarded")
        assert result == []

    def test_both_true_matching_rules_no_diagnostics(self):
        node = _make_node("both_true", unsafe_future_declared=True, unsafe_future_rule=True)
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="both-true",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "production_guarded")
        assert result == []

    def test_declared_false_rule_true_produces_warning(self):
        node = _make_node("conflict", unsafe_future_declared=False, unsafe_future_rule=True)
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="conflict-false-true",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "production_guarded")
        assert len(result) == 1
        assert result[0].code == "TEMPORAL_CONFLICT"
        assert result[0].severity == "warning"
        assert result[0].node_id == "conflict"
        assert result[0].port == "signal"

    def test_declared_true_rule_false_produces_warning(self):
        node = _make_node("conflict2", unsafe_future_declared=True, unsafe_future_rule=False)
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="conflict-true-false",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "production_guarded")
        assert len(result) == 1
        assert result[0].code == "TEMPORAL_CONFLICT"

    def test_multiple_nodes_with_mixed_results(self):
        node_ok = _make_node("ok", unsafe_future_declared=True, unsafe_future_rule=True)
        node_bad = _make_node("bad", unsafe_future_declared=False, unsafe_future_rule=True)
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="mixed",
                version="1.0.0",
                nodes=[node_ok, node_bad],
            ),
        )
        result = trace_temporal_validation(ir, "research")
        assert len(result) == 1
        assert result[0].node_id == "bad"

    def test_port_without_port_temporal_skipped(self):
        node = Node(
            id="no_pt",
            token_ref=TokenRef(token_id="core.threshold", version=">=1.0"),
            signature=PortSignature(
                outputs={
                    "signal": OutputSpec(
                        type=TypeSpec(kind="Decision", value_type="float"),
                        port_temporal=None,
                        temporal_rule=TemporalRule(kind="constant", value={"unsafe_future": True}),
                    ),
                },
            ),
        )
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="no-pt",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "research")
        assert result == []

    def test_port_without_temporal_rule_skipped(self):
        node = Node(
            id="no_tr",
            token_ref=TokenRef(token_id="core.threshold", version=">=1.0"),
            signature=PortSignature(
                outputs={
                    "signal": OutputSpec(
                        type=TypeSpec(kind="Decision", value_type="float"),
                        port_temporal=PortTemporalSpec(unsafe_future=True),
                        temporal_rule=None,
                    ),
                },
            ),
        )
        ir = StrategyIRV04(
            strategy=StrategyBody(
                id="no-tr",
                version="1.0.0",
                nodes=[node],
            ),
        )
        result = trace_temporal_validation(ir, "research")
        assert result == []
