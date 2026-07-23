from __future__ import annotations

from qst.ir import (
    NodeV04,
    StrategyBodyV04,
    StrategyIRV04,
    validate_temporal_v04,
)


def _source_node() -> NodeV04:
    return NodeV04(
        id="source",
        signature={
            "outputs": {
                "close": {
                    "type": "TimeSeries[float]",
                    "port_temporal": {"available_at": "bar_close"},
                }
            }
        },
    )


def test_shift_future_research_warning() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="shift_future",
            nodes=[
                _source_node(),
                NodeV04(
                    id="shift_future",
                    inputs={"series": "source.close"},
                    params={"periods": -1},
                    signature={
                        "inputs": {"series": {"type": "TimeSeries[float]"}},
                        "outputs": {
                            "value": {
                                "type": "TimeSeries[float]",
                                "temporal_rule": {
                                    "kind": "param_predicate",
                                    "param": "periods",
                                    "equals": -1,
                                    "when_true": {"unsafe_future": True},
                                    "when_false": {},
                                },
                            }
                        },
                    },
                ),
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="research")

    assert result.ok
    assert result.diagnostics[0].severity == "warning"
    assert result.diagnostics[0].code == "QST_V2_TEMPORAL_UNSAFE_FUTURE"


def test_shift_future_pretrade_error() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="shift_future",
            nodes=[
                NodeV04(
                    id="shift_future",
                    params={"periods": -1},
                    signature={
                        "outputs": {
                            "value": {
                                "type": "TimeSeries[float]",
                                "temporal_rule": {
                                    "kind": "param_predicate",
                                    "param": "periods",
                                    "equals": -1,
                                    "when_true": {"unsafe_future": True},
                                    "when_false": {},
                                },
                            }
                        },
                    },
                )
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="pretrade")

    assert not result.ok
    assert result.errors[0].code == "QST_V2_TEMPORAL_UNSAFE_FUTURE"


def test_trailing_window_sets_min_history() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="rolling",
            nodes=[
                NodeV04(
                    id="rolling",
                    params={"lookback": 14},
                    signature={
                        "outputs": {
                            "value": {
                                "type": "TimeSeries[float]",
                                "temporal_rule": {"kind": "window_min_history", "param": "lookback"},
                            }
                        }
                    },
                )
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="pretrade")

    assert result.ok


def test_centered_window_emits_unsafe_future() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="centered",
            nodes=[
                NodeV04(
                    id="centered",
                    params={"window": 5},
                    signature={
                        "outputs": {
                            "value": {
                                "type": "TimeSeries[float]",
                                "temporal_rule": {
                                    "kind": "centered_window_unsafe",
                                    "param": "window",
                                },
                            }
                        }
                    },
                )
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="pretrade")

    assert not result.ok
    assert result.errors[0].node_id == "centered"


def test_next_open_prediction_available_at_next_bar_open() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="next_open",
            nodes=[
                NodeV04(
                    id="predict",
                    signature={
                        "outputs": {
                            "score": {
                                "type": "TimeSeries[float]",
                                "temporal_rule": {
                                    "kind": "constant",
                                    "value": {"available_at": "next_bar_open"},
                                },
                            }
                        }
                    },
                )
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="pretrade")

    assert result.ok


def test_input_temporal_requirement_rejects_late_upstream_output() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="requirement",
            nodes=[
                NodeV04(
                    id="predict",
                    signature={
                        "outputs": {
                            "score": {
                                "type": "TimeSeries[float]",
                                "port_temporal": {"available_at": "next_bar_open"},
                            }
                        }
                    },
                ),
                NodeV04(
                    id="consumer",
                    inputs={"score": "predict.score"},
                    signature={
                        "inputs": {
                            "score": {
                                "type": "TimeSeries[float]",
                                "temporal_requirement": {"max_available_at": "bar_close"},
                            }
                        },
                        "outputs": {"value": {"type": "TimeSeries[float]"}},
                    },
                ),
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="pretrade")

    assert not result.ok
    assert result.errors[0].code == "QST_V2_TEMPORAL_REQUIREMENT_UNSATISFIED"


def test_temporal_rule_conflict_is_error() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="conflict",
            nodes=[
                NodeV04(
                    id="n",
                    signature={
                        "outputs": {
                            "value": {
                                "type": "TimeSeries[float]",
                                "port_temporal": {"available_at": "bar_close"},
                                "temporal_rule": {
                                    "kind": "constant",
                                    "value": {"available_at": "next_bar_open"},
                                },
                            }
                        }
                    },
                )
            ],
        )
    )

    result = validate_temporal_v04(ir, profile="research")

    assert not result.ok
    assert result.errors[0].code == "QST_V2_TEMPORAL_CONFLICT"
