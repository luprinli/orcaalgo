from __future__ import annotations

import pyarrow as pa

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
from qst.frames.io import arrow_table_to_frame, frame_to_arrow_table


def test_arrow_roundtrip_preserves_frame_hash_for_all_frame_types() -> None:
    frames = [
        MarketFrame(
            bars=[
                OHLCVBar(
                    timestamp="2026-05-14T09:00:00Z",
                    symbol="BTC/USD",
                    open="1",
                    high="2",
                    low="1",
                    close="2",
                    volume="3",
                )
            ]
        ),
        SignalFrame(rows=[SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="long", size="1")]),
        FeatureFrame(rows=[FeatureRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", features={"ema": "1"})]),
        TraceLog(events=[TraceEvent(timestamp="2026-05-14T09:00:00Z", node_id="n1", event="done", payload={"ok": True})]),
    ]

    for frame in frames:
        restored = arrow_table_to_frame(frame_to_arrow_table(frame))
        assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_arrow_schema_stores_decimal_strings_as_strings() -> None:
    frame = MarketFrame(
        bars=[
            OHLCVBar(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                open="1",
                high="2",
                low="1",
                close="2",
                volume="3",
            )
        ]
    )

    table = frame_to_arrow_table(frame)

    assert table.schema.field("open").type == pa.string()
    assert table.schema.field("high").type == pa.string()
    assert table.schema.field("low").type == pa.string()
    assert table.schema.field("close").type == pa.string()
    assert table.schema.field("volume").type == pa.string()
