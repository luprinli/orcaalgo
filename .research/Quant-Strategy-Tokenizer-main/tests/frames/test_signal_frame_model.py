from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.frames import SignalFrame, SignalRow


def test_signal_frame_canonicalizes_rows_and_symbols() -> None:
    frame = SignalFrame(
        rows=[
            SignalRow(timestamp="2026-05-14T10:00:00Z", symbol="ETH/USD", signal="flat", size="0"),
            SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="long", size="1"),
        ]
    )

    assert frame.symbols == ["BTC/USD", "ETH/USD"]
    assert [row.symbol for row in frame.rows] == ["BTC/USD", "ETH/USD"]


def test_signal_frame_rejects_invalid_signal() -> None:
    with pytest.raises(ValidationError):
        SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="buy", size="1")


def test_signal_frame_rejects_duplicate_timestamp_symbol() -> None:
    row = SignalRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", signal="long", size="1")
    with pytest.raises(ValidationError, match="Duplicate signal"):
        SignalFrame(rows=[row, row])
