from __future__ import annotations

from qst.frames import (
    FeatureFrame,
    FeatureRow,
    MarketFrame,
    OHLCVBar,
    SignalFrame,
    SignalRow,
    TraceEvent,
    TraceLog,
    compute_frame_hash,
)
from qst.frames.io import frame_from_json_bytes, frame_to_json_bytes


def test_json_roundtrip_all_frame_types_preserves_hash_and_frame_hash_field() -> None:
    frames = [
        MarketFrame(bars=[OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3")]),
        SignalFrame(rows=[SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="long", size="1")]),
        FeatureFrame(rows=[FeatureRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", features={"ema": "1"})]),
        TraceLog(events=[TraceEvent(timestamp="2026-05-14T09:00:00Z", node_id="n1", event="done", payload={"ok": True})]),
    ]

    for frame in frames:
        with_hash = frame.model_copy(update={"frame_hash": compute_frame_hash(frame)})
        restored = frame_from_json_bytes(frame_to_json_bytes(with_hash))
        assert restored.frame_hash == with_hash.frame_hash
        assert compute_frame_hash(restored) == compute_frame_hash(frame)
