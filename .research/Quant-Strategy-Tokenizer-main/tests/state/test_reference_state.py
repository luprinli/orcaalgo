from __future__ import annotations

from qst.state import (
    StatePolicy,
    state_accumulate,
    state_delay,
    state_edge_detect,
)


def diagnostic_codes(result: object) -> list[str]:
    return [diagnostic.code for diagnostic in result.result.diagnostics]  # type: ignore[attr-defined]


def test_state_delay_is_deterministic_with_warmup_and_reset_trace() -> None:
    result = state_delay(
        [10, 20, 30, 40],
        steps=2,
        initial=0,
        policy=StatePolicy(warmup_policy="emit_initial", reset_policy="on_event"),
        reset_events=[False, False, True, False],
    )

    assert result.outputs == [0, 0, 0, 0]
    assert result.result.ok
    assert result.trace.events[2].reset is True
    assert result.trace.events[2].state_before == [10, 20]
    assert result.trace.events[2].state_after == [30]
    assert "reset" in result.trace.events[2].policy_decision


def test_state_delay_missing_default_policy_is_error() -> None:
    result = state_delay([1, None, 3])

    assert result.outputs == [None, None, 1]
    assert diagnostic_codes(result) == ["QST_V2_STATE_MISSING_EVENT"]
    assert not result.result.ok


def test_state_accumulate_sum_count_last_min_max() -> None:
    assert state_accumulate([1, 2, 3], reducer="sum").outputs == [1, 3, 6]
    assert state_accumulate([1, 2, 3], reducer="count").outputs == [1, 2, 3]
    assert state_accumulate([1, 2, 3], reducer="last").outputs == [1, 2, 3]
    assert state_accumulate([3, 1, 2], reducer="min").outputs == [3, 1, 1]
    assert state_accumulate([3, 1, 2], reducer="max").outputs == [3, 3, 3]


def test_state_accumulate_unknown_reducer_fails_without_events() -> None:
    result = state_accumulate([1, 2, 3], reducer="median")

    assert result.outputs == []
    assert result.trace.events == []
    assert diagnostic_codes(result) == ["QST_V2_STATE_REDUCER_UNKNOWN"]


def test_state_accumulate_reset_before_current_input() -> None:
    result = state_accumulate(
        [1, 2, 3],
        reducer="sum",
        policy=StatePolicy(reset_policy="on_event"),
        reset_events=[False, True, False],
    )

    assert result.outputs == [1, 2, 5]
    assert result.trace.events[1].state_before == 1
    assert result.trace.events[1].state_after == 2
    assert result.trace.events[1].reset is True


def test_state_edge_detect_modes() -> None:
    values = [False, True, True, False]

    assert state_edge_detect(values, mode="rising").outputs == [False, True, False, False]
    assert state_edge_detect(values, mode="falling").outputs == [False, False, False, True]
    assert state_edge_detect(values, mode="change").outputs == [False, True, False, True]


def test_state_edge_detect_missing_policy() -> None:
    result = state_edge_detect([False, None, True], policy=StatePolicy(missing_event_policy="skip"))

    assert result.outputs == [False, False, True]
    assert result.result.ok
    assert result.trace.events[1].policy_decision.endswith("missing_skip")
