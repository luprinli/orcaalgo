from __future__ import annotations

import pyarrow as pa
import pyarrow.parquet as pq

from qst.frames import (
    FeatureFrame,
    FeatureRow,
    MarketFrame,
    OHLCVBar,
    compute_frame_hash,
)
from qst.frames.io import read_parquet_frame, write_parquet_frame


def test_parquet_market_frame_semantic_roundtrip_preserves_hash_and_schema_order(tmp_path) -> None:  # type: ignore[no-untyped-def]
    frame = MarketFrame(
        bars=[
            OHLCVBar(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                open="1",
                high="2",
                low="0.5",
                close="1.5",
                volume="10",
            )
        ]
    )
    path = tmp_path / "market.parquet"

    write_parquet_frame(frame, path)

    table = pq.read_table(path)
    assert table.schema.names == ["timestamp", "symbol", "open", "high", "low", "close", "volume"]
    assert table.schema.field("open").type == pa.string()

    restored = read_parquet_frame(path)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_parquet_feature_frame_semantic_roundtrip_preserves_hash(tmp_path) -> None:  # type: ignore[no-untyped-def]
    frame = FeatureFrame(
        rows=[
            FeatureRow(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                features={"zscore": "1.5", "ema": "10"},
            )
        ]
    )
    path = tmp_path / "features.parquet"

    write_parquet_frame(frame, path)

    table = pq.read_table(path)
    assert table.schema.names == ["timestamp", "symbol", "ema", "zscore"]

    restored = read_parquet_frame(path)
    assert compute_frame_hash(restored) == compute_frame_hash(frame)
