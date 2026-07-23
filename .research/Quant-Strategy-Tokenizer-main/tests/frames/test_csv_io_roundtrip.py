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
from qst.frames.io import frame_from_csv_text, frame_to_csv_text


def test_market_csv_roundtrip_preserves_semantic_hash() -> None:
    frame = MarketFrame(
        bars=[
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3")
        ]
    )
    restored = frame_from_csv_text(frame_to_csv_text(frame), "qst-market-frame/1")
    assert isinstance(restored, MarketFrame)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_signal_csv_roundtrip_preserves_semantic_hash() -> None:
    frame = SignalFrame(
        rows=[SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="short", size="0.5")]
    )
    restored = frame_from_csv_text(frame_to_csv_text(frame), "qst-signal-frame/1")
    assert isinstance(restored, SignalFrame)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_feature_csv_roundtrip_preserves_semantic_hash_and_sorted_columns() -> None:
    frame = FeatureFrame(
        rows=[
            FeatureRow(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                features={"zscore": "1", "ema": "2"},
            )
        ]
    )
    csv_text = frame_to_csv_text(frame)
    assert csv_text.splitlines()[0] == "timestamp,symbol,ema,zscore"
    restored = frame_from_csv_text(csv_text, "qst-feature-frame/1")
    assert isinstance(restored, FeatureFrame)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_trace_csv_roundtrip_preserves_semantic_hash() -> None:
    frame = TraceLog(
        events=[TraceEvent(timestamp="2026-05-14T09:00:00Z", node_id="n1", event="done", payload={"ok": True})]
    )
    restored = frame_from_csv_text(frame_to_csv_text(frame), "qst-trace-log/1")
    assert isinstance(restored, TraceLog)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)
