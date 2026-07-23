from __future__ import annotations

from qst.frames import (
    FeatureFrame,
    FeatureRow,
    MarketFrame,
    OHLCVBar,
    compute_frame_hash,
)
from qst.frames.io import dataframe_to_frame, frame_to_dataframe


def test_market_frame_pandas_roundtrip_preserves_decimal_strings_as_object_dtype() -> None:
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

    dataframe = frame_to_dataframe(frame)

    assert str(dataframe["open"].dtype) == "object"
    assert dataframe.loc[0, "open"] == "1"
    assert isinstance(dataframe.loc[0, "open"], str)

    restored = dataframe_to_frame(dataframe, "qst-market-frame/1")
    assert compute_frame_hash(restored) == compute_frame_hash(frame)


def test_feature_frame_pandas_roundtrip_preserves_sorted_feature_columns() -> None:
    frame = FeatureFrame(
        rows=[
            FeatureRow(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                features={"zscore": "1.5", "ema": "10"},
            )
        ]
    )

    dataframe = frame_to_dataframe(frame)

    assert list(dataframe.columns) == ["timestamp", "symbol", "ema", "zscore"]
    restored = dataframe_to_frame(dataframe, "qst-feature-frame/1")
    assert compute_frame_hash(restored) == compute_frame_hash(frame)
