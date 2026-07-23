from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.frames import TraceEvent, TraceLog


def test_trace_log_accepts_canonical_json_payload_and_sorts_events() -> None:
    log = TraceLog(
        events=[
            TraceEvent(timestamp="2026-05-14T10:00:00Z", node_id="b", event="done", payload={"y": 2}),
            TraceEvent(timestamp="2026-05-14T09:00:00Z", node_id="a", event="start", payload={"x": 1}),
        ]
    )

    assert [event.node_id for event in log.events] == ["a", "b"]


def test_trace_event_rejects_non_canonical_json_payload() -> None:
    with pytest.raises(ValidationError):
        TraceEvent(timestamp="2026-05-14T09:00:00Z", node_id="a", event="bad", payload=(1, 2))
