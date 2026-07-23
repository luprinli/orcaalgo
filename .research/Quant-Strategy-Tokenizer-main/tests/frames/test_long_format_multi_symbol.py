from __future__ import annotations

from qst.frames import MarketFrame, OHLCVBar
from qst.frames.io import frame_to_csv_text


def test_multi_symbol_market_frame_serializes_sorted_long_format() -> None:
    frame = MarketFrame(
        bars=[
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
        ]
    )

    assert frame.symbols == ["BTC/USD", "ETH/USD"]
    assert [(bar.timestamp.isoformat(), bar.symbol) for bar in frame.bars] == [
        ("2026-05-14T09:00:00+00:00", "BTC/USD"),
        ("2026-05-14T09:00:00+00:00", "ETH/USD"),
        ("2026-05-14T10:00:00+00:00", "BTC/USD"),
        ("2026-05-14T10:00:00+00:00", "ETH/USD"),
    ]

    lines = frame_to_csv_text(frame).splitlines()
    assert lines[1].startswith("2026-05-14T09:00:00Z,BTC/USD,")
    assert lines[2].startswith("2026-05-14T09:00:00Z,ETH/USD,")
    assert lines[3].startswith("2026-05-14T10:00:00Z,BTC/USD,")
    assert lines[4].startswith("2026-05-14T10:00:00Z,ETH/USD,")
