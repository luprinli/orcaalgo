from __future__ import annotations

import pytest

from qst.decision import DecisionV2
from qst.panel import PanelValue
from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_decision_token,
    evaluate_indicator_token,
    evaluate_signal_token,
)

SERIES = [
    ("2026-05-16T09:00:00Z", 1),
    ("2026-05-16T09:01:00Z", 2),
    ("2026-05-16T09:02:00Z", 3),
    ("2026-05-16T09:03:00Z", 4),
]

PR6_INDICATORS = {
    "indicator.rolling_mean",
    "indicator.rolling_std",
    "indicator.rolling_zscore",
    "indicator.macd",
    "indicator.bollinger_band",
    "indicator.atr",
    "indicator.donchian_channel",
    "indicator.volatility",
    "indicator.linear_regression_slope",
    "indicator.beta",
    "indicator.residual",
}

PR6_SIGNALS = {
    "signal.greater_than",
    "signal.less_than",
    "signal.and",
    "signal.or",
    "signal.not",
    "signal.between",
    "signal.outside_band",
    "signal.breakout_up",
    "signal.breakout_down",
    "signal.zscore_revert",
    "signal.rank_top_k",
    "signal.rank_bottom_k",
}

PR6_DECISIONS = {
    "decision.long_flat",
    "decision.long_short",
    "decision.entry_exit_to_position",
    "decision.signal_to_decision",
    "decision.rank_to_selection",
    "decision.selection_to_weight",
    "decision.gate_decision",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _kinds(decisions: object) -> list[str]:
    return [decision.kind for decision in decisions]  # type: ignore[union-attr]


def _panel() -> PanelValue:
    return PanelValue.model_validate(
        {
            "rows": [
                {"timestamp": "2026-05-16T09:00:00Z", "symbol": "AAA", "value": "3"},
                {"timestamp": "2026-05-16T09:00:00Z", "symbol": "BBB", "value": "1"},
                {"timestamp": "2026-05-16T09:00:00Z", "symbol": "CCC", "value": "2"},
            ]
        }
    )


def test_pr6_core_rule_tokens_have_surface_contracts() -> None:
    specs = _spec_by_name()
    expected = PR6_INDICATORS | PR6_SIGNALS | PR6_DECISIONS

    assert expected <= set(specs)
    for name in sorted(expected):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.temporal
        assert spec.surface.contract.numeric
        assert spec.surface.contract.missing_data
        assert spec.surface.contract.failure_mode

    assert specs["indicator.macd"].surface.family == "indicator"
    assert specs["signal.rank_top_k"].surface.capabilities.panel_aware
    assert specs["decision.selection_to_weight"].surface.capabilities.panel_aware


def test_indicator_helpers_cover_macd_atr_donchian_and_regression() -> None:
    macd = evaluate_indicator_token(
        "indicator.macd",
        SERIES,
        fast_window=2,
        slow_window=3,
        signal_window=2,
    )
    assert macd["macd"][-1][1] == pytest.approx(0.3935185)
    assert macd["signal"][-1][1] == pytest.approx(0.3425926)
    assert macd["histogram"][-1][1] == pytest.approx(0.0509259)

    high = [("t1", 10), ("t2", 12), ("t3", 13)]
    low = [("t1", 9), ("t2", 10), ("t3", 12)]
    close = [("t1", 9.5), ("t2", 11), ("t3", 12.5)]
    assert evaluate_indicator_token("indicator.atr", high, low, close, window=2) == [
        ("t2", 1.75),
        ("t3", 1.875),
    ]

    channels = evaluate_indicator_token("indicator.donchian_channel", high, low, window=2)
    assert channels["upper"] == [("t3", 12.0)]
    assert channels["lower"] == [("t3", 9.0)]

    assert evaluate_indicator_token("indicator.rolling_mean", SERIES, window=2)[-1] == (
        "2026-05-16T09:03:00Z",
        3.5,
    )
    assert evaluate_indicator_token("indicator.linear_regression_slope", SERIES, window=3) == [
        ("2026-05-16T09:02:00Z", 1.0),
        ("2026-05-16T09:03:00Z", 1.0),
    ]

    benchmark = [("t1", 1), ("t2", 2), ("t3", 3)]
    asset = [("t1", 3), ("t2", 5), ("t3", 7)]
    assert evaluate_indicator_token("indicator.beta", asset, benchmark, window=3) == [("t3", 2.0)]
    assert evaluate_indicator_token("indicator.residual", asset, benchmark, window=3) == [("t3", 0.0)]

    with pytest.raises(TokenReferenceError) as zero_variance:
        evaluate_indicator_token("indicator.beta", asset, [("t1", 1), ("t2", 1), ("t3", 1)], window=3)
    assert zero_variance.value.code == "QST_TOKEN_INDICATOR_ZERO_VARIANCE"


def test_signal_helpers_cover_truth_tables_and_panel_rank_selection() -> None:
    left = [("t1", 1), ("t2", 3), ("t3", 2)]
    right = [("t1", 2), ("t2", 2), ("t3", 2)]
    assert evaluate_signal_token("signal.greater_than", left, right) == [
        ("t1", False),
        ("t2", True),
        ("t3", False),
    ]
    assert evaluate_signal_token("signal.less_than", left, right) == [
        ("t1", True),
        ("t2", False),
        ("t3", False),
    ]

    a = [("t1", True), ("t2", False)]
    b = [("t1", False), ("t2", True)]
    assert evaluate_signal_token("signal.and", a, b) == [("t1", False), ("t2", False)]
    assert evaluate_signal_token("signal.or", a, b) == [("t1", True), ("t2", True)]
    assert evaluate_signal_token("signal.not", a) == [("t1", False), ("t2", True)]

    lower = [("t1", 0), ("t2", 2), ("t3", 2)]
    upper = [("t1", 2), ("t2", 4), ("t3", 3)]
    assert evaluate_signal_token("signal.between", left, lower, upper) == [
        ("t1", True),
        ("t2", True),
        ("t3", True),
    ]
    assert evaluate_signal_token("signal.outside_band", left, lower, upper) == [
        ("t1", False),
        ("t2", False),
        ("t3", False),
    ]
    assert evaluate_signal_token("signal.breakout_up", left, right) == [
        ("t1", False),
        ("t2", True),
        ("t3", False),
    ]
    assert evaluate_signal_token("signal.breakout_down", left, right) == [
        ("t1", True),
        ("t2", False),
        ("t3", False),
    ]
    assert evaluate_signal_token("signal.zscore_revert", [("t1", 2.5), ("t2", 1.8), ("t3", -1.0)]) == [
        ("t2", True),
        ("t3", False),
    ]

    top = evaluate_signal_token("signal.rank_top_k", _panel(), k=1)
    bottom = evaluate_signal_token("signal.rank_bottom_k", _panel(), k=1)
    assert [row.symbol for row in top.selection.rows if row.selected] == ["AAA"]
    assert [row.symbol for row in bottom.selection.rows if row.selected] == ["BBB"]


def test_decision_helpers_preserve_decision_kind_boundary() -> None:
    assert _kinds(evaluate_decision_token("decision.long_flat", [True, False])) == ["accept", "reject"]
    assert _kinds(evaluate_decision_token("decision.long_short", [True, False, True], [False, True, True])) == [
        "accept",
        "accept",
        "block",
    ]
    assert _kinds(evaluate_decision_token("decision.entry_exit_to_position", [True, False, False], [False, False, True])) == [
        "accept",
        "accept",
        "reject",
    ]
    assert _kinds(evaluate_decision_token("decision.signal_to_decision", [("t1", 0.1), ("t2", -0.2)], threshold=0)) == [
        "accept",
        "reject",
    ]

    selection = evaluate_decision_token("decision.rank_to_selection", _panel(), k=1, side="bottom")
    weights = evaluate_decision_token("decision.selection_to_weight", selection.selection, method="equal_short")
    assert [row.symbol for row in selection.selection.rows if row.selected] == ["BBB"]
    assert weights.weights.rows[0].weight == "-1"

    base = [DecisionV2(kind="accept", reasons=("BASE",)), DecisionV2(kind="reject", reasons=("BASE",))]
    gate = [DecisionV2(kind="block", reasons=("GATE",)), DecisionV2(kind="accept", reasons=("GATE",))]
    gated = evaluate_decision_token("decision.gate_decision", base, gate)
    assert _kinds(gated) == ["block", "reject"]
    assert "GATE" in gated[0].reasons
