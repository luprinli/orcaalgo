from __future__ import annotations

import json

import pyarrow as pa
import pyarrow.parquet as pq
import pytest
from pydantic import ValidationError

from qst.frames import MarketFrame, OHLCVBar, compute_frame_hash
from qst.frames.io import (
    arrow_table_to_frame,
    dataframe_to_frame,
    frame_from_csv_text,
    frame_from_json_bytes,
    frame_to_arrow_table,
    frame_to_csv_text,
    frame_to_dataframe,
    frame_to_json_bytes,
    read_parquet_frame,
    write_parquet_frame,
)


def _aligned_market_frame() -> MarketFrame:
    return MarketFrame(
        bars=[
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
        ]
    )


def _missing_grid_rows() -> dict[str, list[str]]:
    return {
        "timestamp": ["2026-05-14T09:00:00Z", "2026-05-14T10:00:00Z"],
        "symbol": ["BTC/USD", "ETH/USD"],
        "open": ["1", "1"],
        "high": ["2", "2"],
        "low": ["1", "1"],
        "close": ["2", "2"],
        "volume": ["3", "3"],
    }


def _missing_grid_arrow_table() -> pa.Table:
    table = pa.table(_missing_grid_rows())
    metadata = dict(table.schema.metadata or {})
    metadata[b"qst_frame_version"] = b"qst-market-frame/1"
    return table.replace_schema_metadata(metadata)


def test_aligned_multi_symbol_market_frame_roundtrips_across_io_formats(tmp_path) -> None:  # type: ignore[no-untyped-def]
    frame = _aligned_market_frame()
    baseline = compute_frame_hash(frame)

    json_restored = frame_from_json_bytes(frame_to_json_bytes(frame))
    csv_restored = frame_from_csv_text(frame_to_csv_text(frame), "qst-market-frame/1")
    dataframe_restored = dataframe_to_frame(frame_to_dataframe(frame), "qst-market-frame/1")
    arrow_restored = arrow_table_to_frame(frame_to_arrow_table(frame))

    parquet_path = tmp_path / "market.parquet"
    write_parquet_frame(frame, parquet_path)
    parquet_restored = read_parquet_frame(parquet_path)

    for restored in [json_restored, csv_restored, dataframe_restored, arrow_restored, parquet_restored]:
        assert isinstance(restored, MarketFrame)
        assert compute_frame_hash(restored) == baseline


def test_missing_market_grid_is_rejected_from_json() -> None:
    payload = {
        "frame_version": "qst-market-frame/1",
        "bars": [
            {key: values[index] for key, values in _missing_grid_rows().items()}
            for index in range(2)
        ],
    }

    with pytest.raises(ValidationError, match="strict alignment"):
        frame_from_json_bytes(json.dumps(payload).encode("utf-8"))


def test_missing_market_grid_is_rejected_from_csv() -> None:
    text = "\n".join(
        [
            "timestamp,symbol,open,high,low,close,volume",
            "2026-05-14T09:00:00Z,BTC/USD,1,2,1,2,3",
            "2026-05-14T10:00:00Z,ETH/USD,1,2,1,2,3",
            "",
        ]
    )

    with pytest.raises(ValidationError, match="strict alignment"):
        frame_from_csv_text(text, "qst-market-frame/1")


def test_missing_market_grid_is_rejected_from_arrow() -> None:
    with pytest.raises(ValidationError, match="strict alignment"):
        arrow_table_to_frame(_missing_grid_arrow_table())


def test_missing_market_grid_is_rejected_from_parquet(tmp_path) -> None:  # type: ignore[no-untyped-def]
    path = tmp_path / "missing.parquet"
    pq.write_table(_missing_grid_arrow_table(), path)

    with pytest.raises(ValidationError, match="strict alignment"):
        read_parquet_frame(path)
