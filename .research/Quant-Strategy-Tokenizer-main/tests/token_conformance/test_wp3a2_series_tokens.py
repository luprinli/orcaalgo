from __future__ import annotations

import math

import pytest

from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_align_token,
    evaluate_channel_breakout_token,
    evaluate_data_token,
    evaluate_indicator_token,
    evaluate_signal_token,
    evaluate_time_token,
    evaluate_window_token,
)

SERIES = [
    ("2026-05-16T09:00:00Z", 1),
    ("2026-05-16T09:01:00Z", 2),
    ("2026-05-16T09:02:00Z", 4),
    ("2026-05-16T09:03:00Z", 8),
]

WP3A2_TOKENS = {
    "data.identity",
    "data.shift",
    "data.diff",
    "data.pct_change",
    "data.log_return",
    "time.session_filter",
    "align.inner_join",
    "align.left_join",
    "align.forward_fill",
    "align.drop_missing",
    "window.max",
    "window.min",
    "window.mean",
    "window.std",
    "window.sum",
    "window.count",
    "window.zscore",
    "signal.cross_above",
    "signal.cross_below",
    "signal.crosses",
    "signal.threshold_above",
    "signal.threshold_below",
    "norm.range_position",
    "smooth.linear_recursive",
    "indicator.sma",
    "indicator.ema",
    "indicator.rsi",
    "indicator.bollinger",
    "indicator.channel_breakout",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _assert_code(exc: pytest.ExceptionInfo[TokenReferenceError], code: str) -> None:
    assert exc.value.code == code


def test_wp3a2_tokens_have_surface_contracts() -> None:
    specs = _spec_by_name()

    assert WP3A2_TOKENS <= set(specs)
    for name in sorted(WP3A2_TOKENS):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.temporal
        assert spec.surface.contract.numeric
        assert spec.surface.contract.missing_data
        assert spec.surface.contract.failure_mode

    assert specs["signal.crosses"].surface.family == "signal"
    assert specs["indicator.sma"].surface.category == "trend"
    assert specs["window.zscore"].surface.capabilities.deterministic_level == "semantic_float64"


def test_data_helpers_cover_shift_diff_returns_and_duplicate_timestamps() -> None:
    assert evaluate_data_token("data.identity", list(reversed(SERIES))) == SERIES
    assert evaluate_data_token("data.shift", SERIES, periods=1) == [
        ("2026-05-16T09:01:00Z", 1.0),
        ("2026-05-16T09:02:00Z", 2.0),
        ("2026-05-16T09:03:00Z", 4.0),
    ]
    assert evaluate_data_token("data.shift", SERIES, periods=-1, allow_unsafe_future=True) == [
        ("2026-05-16T09:00:00Z", 2.0),
        ("2026-05-16T09:01:00Z", 4.0),
        ("2026-05-16T09:02:00Z", 8.0),
    ]
    assert evaluate_data_token("data.diff", SERIES) == [
        ("2026-05-16T09:01:00Z", 1.0),
        ("2026-05-16T09:02:00Z", 2.0),
        ("2026-05-16T09:03:00Z", 4.0),
    ]
    assert evaluate_data_token("data.pct_change", SERIES) == [
        ("2026-05-16T09:01:00Z", 1.0),
        ("2026-05-16T09:02:00Z", 1.0),
        ("2026-05-16T09:03:00Z", 1.0),
    ]

    with pytest.raises(TokenReferenceError) as unsafe_shift:
        evaluate_data_token("data.shift", SERIES, periods=-1)
    _assert_code(unsafe_shift, "QST_TOKEN_UNSAFE_FUTURE_SHIFT")

    with pytest.raises(TokenReferenceError) as duplicate_timestamp:
        evaluate_data_token("data.identity", [("2026-05-16T09:00:00Z", 1), ("2026-05-16T09:00:00Z", 2)])
    _assert_code(duplicate_timestamp, "QST_TOKEN_SERIES_DUPLICATE_TIMESTAMP")

    with pytest.raises(TokenReferenceError) as bool_numeric:
        evaluate_data_token("data.identity", [("2026-05-16T09:00:00Z", True)])
    _assert_code(bool_numeric, "QST_TOKEN_NUMERIC_TYPE_INVALID")


def test_data_return_domain_errors_are_stable() -> None:
    with pytest.raises(TokenReferenceError) as pct_zero:
        evaluate_data_token(
            "data.pct_change",
            [("2026-05-16T09:00:00Z", 0), ("2026-05-16T09:01:00Z", 1)],
        )
    _assert_code(pct_zero, "QST_TOKEN_DATA_DIVIDE_BY_ZERO")

    with pytest.raises(TokenReferenceError) as log_non_positive:
        evaluate_data_token(
            "data.log_return",
            [("2026-05-16T09:00:00Z", 1), ("2026-05-16T09:01:00Z", 0)],
        )
    _assert_code(log_non_positive, "QST_TOKEN_DATA_LOG_RETURN_DOMAIN_ERROR")


def test_time_and_align_helpers_have_deterministic_join_and_missing_behavior() -> None:
    session_rows = [
        ("2026-05-16T08:59:00Z", 1),
        ("2026-05-16T09:00:00Z", 2),
        ("2026-05-16T10:00:00Z", 3),
        ("2026-05-16T10:01:00Z", 4),
    ]
    assert evaluate_time_token("time.session_filter", session_rows, start_hhmm="09:00", end_hhmm="10:00") == [
        ("2026-05-16T09:00:00Z", 2.0),
        ("2026-05-16T10:00:00Z", 3.0),
    ]

    left = [("2026-05-16T09:00:00Z", 1), ("2026-05-16T09:01:00Z", 2), ("2026-05-16T09:02:00Z", 3)]
    right = [("2026-05-16T09:01:00Z", 20), ("2026-05-16T09:02:00Z", 30), ("2026-05-16T09:03:00Z", 40)]

    assert evaluate_align_token("align.inner_join", left, right) == (
        [("2026-05-16T09:01:00Z", 2.0), ("2026-05-16T09:02:00Z", 3.0)],
        [("2026-05-16T09:01:00Z", 20.0), ("2026-05-16T09:02:00Z", 30.0)],
    )
    assert evaluate_align_token("align.left_join", left, right) == (
        [("2026-05-16T09:00:00Z", 1.0), ("2026-05-16T09:01:00Z", 2.0), ("2026-05-16T09:02:00Z", 3.0)],
        [("2026-05-16T09:00:00Z", None), ("2026-05-16T09:01:00Z", 20.0), ("2026-05-16T09:02:00Z", 30.0)],
    )

    sparse = [
        ("2026-05-16T09:00:00Z", None),
        ("2026-05-16T09:01:00Z", 1),
        ("2026-05-16T09:02:00Z", None),
        ("2026-05-16T09:03:00Z", 3),
    ]
    assert evaluate_align_token("align.forward_fill", sparse) == [
        ("2026-05-16T09:00:00Z", None),
        ("2026-05-16T09:01:00Z", 1.0),
        ("2026-05-16T09:02:00Z", 1.0),
        ("2026-05-16T09:03:00Z", 3.0),
    ]
    assert evaluate_align_token("align.drop_missing", sparse) == [
        ("2026-05-16T09:01:00Z", 1.0),
        ("2026-05-16T09:03:00Z", 3.0),
    ]


def test_window_helpers_cover_trailing_window_semantics() -> None:
    assert evaluate_window_token("window.sum", SERIES, window=3) == [
        ("2026-05-16T09:02:00Z", 7.0),
        ("2026-05-16T09:03:00Z", 14.0),
    ]
    assert evaluate_window_token("window.count", SERIES, window=3) == [
        ("2026-05-16T09:02:00Z", 3),
        ("2026-05-16T09:03:00Z", 3),
    ]
    assert evaluate_window_token("window.min", SERIES, window=3) == [
        ("2026-05-16T09:02:00Z", 1.0),
        ("2026-05-16T09:03:00Z", 2.0),
    ]
    assert evaluate_window_token("window.max", SERIES, window=3) == [
        ("2026-05-16T09:02:00Z", 4.0),
        ("2026-05-16T09:03:00Z", 8.0),
    ]
    assert evaluate_window_token("window.mean", SERIES, window=3, min_periods=2)[0] == (
        "2026-05-16T09:01:00Z",
        1.5,
    )
    assert evaluate_window_token("window.std", [(timestamp, index) for index, (timestamp, _) in enumerate(SERIES, 1)], window=3)[0][1] == pytest.approx(math.sqrt(2 / 3))
    assert evaluate_window_token(
        "window.zscore",
        [("2026-05-16T09:00:00Z", 2), ("2026-05-16T09:01:00Z", 2), ("2026-05-16T09:02:00Z", 2)],
        window=3,
    ) == [("2026-05-16T09:02:00Z", 0)]

    with pytest.raises(TokenReferenceError) as invalid_window:
        evaluate_window_token("window.mean", SERIES, window=0)
    _assert_code(invalid_window, "QST_TOKEN_WINDOW_INVALID")


def test_signal_helpers_cover_crosses_thresholds_and_range_position() -> None:
    left = [
        ("2026-05-16T09:00:00Z", 1),
        ("2026-05-16T09:01:00Z", 1),
        ("2026-05-16T09:02:00Z", 3),
        ("2026-05-16T09:03:00Z", 1),
    ]
    right = [(timestamp, 2) for timestamp, _ in left]

    assert evaluate_signal_token("signal.cross_above", left, right) == [
        ("2026-05-16T09:01:00Z", False),
        ("2026-05-16T09:02:00Z", True),
        ("2026-05-16T09:03:00Z", False),
    ]
    assert evaluate_signal_token("signal.cross_below", left, right)[-1] == (
        "2026-05-16T09:03:00Z",
        True,
    )
    assert evaluate_signal_token("signal.crosses", left, right) == [
        ("2026-05-16T09:01:00Z", False),
        ("2026-05-16T09:02:00Z", True),
        ("2026-05-16T09:03:00Z", True),
    ]
    assert evaluate_signal_token("signal.threshold_above", left, threshold=3) == [
        ("2026-05-16T09:00:00Z", False),
        ("2026-05-16T09:01:00Z", False),
        ("2026-05-16T09:02:00Z", True),
        ("2026-05-16T09:03:00Z", False),
    ]
    assert evaluate_signal_token("signal.threshold_below", left, threshold=1)[:2] == [
        ("2026-05-16T09:00:00Z", True),
        ("2026-05-16T09:01:00Z", True),
    ]
    assert evaluate_signal_token(
        "norm.range_position",
        [("2026-05-16T09:00:00Z", 5)],
        [("2026-05-16T09:00:00Z", 10)],
        [("2026-05-16T09:00:00Z", 0)],
    ) == [("2026-05-16T09:00:00Z", 0.5)]
    assert evaluate_signal_token("smooth.linear_recursive", SERIES, alpha=0.5)[-1] == (
        "2026-05-16T09:03:00Z",
        5.375,
    )


def test_indicator_helpers_are_deterministic_and_current_bar_safe() -> None:
    assert evaluate_indicator_token("indicator.sma", SERIES, window=2) == [
        ("2026-05-16T09:01:00Z", 1.5),
        ("2026-05-16T09:02:00Z", 3.0),
        ("2026-05-16T09:03:00Z", 6.0),
    ]
    assert evaluate_indicator_token("indicator.ema", SERIES, window=3) == [
        ("2026-05-16T09:00:00Z", 1.0),
        ("2026-05-16T09:01:00Z", 1.5),
        ("2026-05-16T09:02:00Z", 2.75),
        ("2026-05-16T09:03:00Z", 5.375),
    ]
    assert evaluate_indicator_token("indicator.rsi", SERIES, window=2) == [
        ("2026-05-16T09:02:00Z", 100),
        ("2026-05-16T09:03:00Z", 100),
    ]
    assert evaluate_indicator_token(
        "indicator.rsi",
        [("2026-05-16T09:00:00Z", 1), ("2026-05-16T09:01:00Z", 1), ("2026-05-16T09:02:00Z", 1)],
        window=2,
    ) == [("2026-05-16T09:02:00Z", 50)]

    bands = evaluate_indicator_token("indicator.bollinger", SERIES[:3], window=3, width=2)
    assert bands["middle"] == [("2026-05-16T09:02:00Z", pytest.approx(7 / 3))]
    assert bands["upper"][0][1] == pytest.approx(7 / 3 + 2 * math.sqrt(14 / 9))
    assert bands["lower"][0][1] == pytest.approx(7 / 3 - 2 * math.sqrt(14 / 9))

    high = [("2026-05-16T09:00:00Z", 10), ("2026-05-16T09:01:00Z", 11), ("2026-05-16T09:02:00Z", 100)]
    low = [("2026-05-16T09:00:00Z", 5), ("2026-05-16T09:01:00Z", 4), ("2026-05-16T09:02:00Z", 3)]
    close = [("2026-05-16T09:00:00Z", 7), ("2026-05-16T09:01:00Z", 8), ("2026-05-16T09:02:00Z", 50)]
    assert evaluate_channel_breakout_token(high, low, close, window=2) == [
        ("2026-05-16T09:02:00Z", True)
    ]
