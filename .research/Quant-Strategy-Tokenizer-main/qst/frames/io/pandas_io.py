"""pandas DataFrame interop for QST frames."""

from __future__ import annotations

import json
from typing import TypeAlias

import pandas as pd

from qst.frames.base import timestamp_to_string
from qst.frames.feature_frame import FeatureFrame, FeatureRow
from qst.frames.io.csv_io import FrameVersion
from qst.frames.io.json_io import Frame
from qst.frames.market_frame import MarketFrame, OHLCVBar
from qst.frames.signal_frame import SignalFrame, SignalRow
from qst.frames.trace_log import TraceEvent, TraceLog

FrameDataFrame: TypeAlias = pd.DataFrame


def frame_to_dataframe(frame: Frame) -> pd.DataFrame:
    """Convert a frame to a deterministic pandas DataFrame.

    DecimalString fields are kept as Python strings so pandas preserves object
    dtype instead of coercing to floating point values.
    """

    if isinstance(frame, MarketFrame):
        return pd.DataFrame(
            [
                {
                    "timestamp": timestamp_to_string(bar.timestamp),
                    "symbol": bar.symbol,
                    "open": bar.open,
                    "high": bar.high,
                    "low": bar.low,
                    "close": bar.close,
                    "volume": bar.volume,
                }
                for bar in frame.bars
            ],
            columns=["timestamp", "symbol", "open", "high", "low", "close", "volume"],
            dtype=object,
        )

    if isinstance(frame, SignalFrame):
        return pd.DataFrame(
            [
                {
                    "timestamp": timestamp_to_string(row.timestamp),
                    "symbol": row.symbol,
                    "signal": row.signal,
                    "size": row.size,
                }
                for row in frame.rows
            ],
            columns=["timestamp", "symbol", "signal", "size"],
            dtype=object,
        )

    if isinstance(frame, FeatureFrame):
        feature_names = sorted({name for row in frame.rows for name in row.features})
        columns = ["timestamp", "symbol", *feature_names]
        return pd.DataFrame(
            [
                {
                    "timestamp": timestamp_to_string(row.timestamp),
                    "symbol": row.symbol,
                    **{name: row.features.get(name, None) for name in feature_names},
                }
                for row in frame.rows
            ],
            columns=columns,
            dtype=object,
        )

    return pd.DataFrame(
        [
            {
                "timestamp": timestamp_to_string(event.timestamp),
                "node_id": event.node_id,
                "event": event.event,
                "payload_json": json.dumps(event.payload, sort_keys=True, separators=(",", ":")),
            }
            for event in frame.events
        ],
        columns=["timestamp", "node_id", "event", "payload_json"],
        dtype=object,
    )


def dataframe_to_frame(dataframe: pd.DataFrame, frame_version: FrameVersion) -> Frame:
    """Convert a pandas DataFrame back into a QST frame."""

    if frame_version == "qst-market-frame/1":
        return MarketFrame(
            bars=[
                OHLCVBar.model_validate(
                    {
                        "timestamp": str(row["timestamp"]),
                        "symbol": str(row["symbol"]),
                        "open": str(row["open"]),
                        "high": str(row["high"]),
                        "low": str(row["low"]),
                        "close": str(row["close"]),
                        "volume": str(row["volume"]),
                    }
                )
                for row in dataframe.to_dict("records")
            ]
        )

    if frame_version == "qst-signal-frame/1":
        return SignalFrame(
            rows=[
                SignalRow.model_validate(
                    {
                        "timestamp": str(row["timestamp"]),
                        "symbol": str(row["symbol"]),
                        "signal": str(row["signal"]),
                        "size": str(row["size"]),
                    }
                )
                for row in dataframe.to_dict("records")
            ]
        )

    if frame_version == "qst-feature-frame/1":
        feature_names = [name for name in dataframe.columns if name not in {"timestamp", "symbol"}]
        return FeatureFrame(
            rows=[
                FeatureRow.model_validate(
                    {
                        "timestamp": str(row["timestamp"]),
                        "symbol": str(row["symbol"]),
                        "features": {
                            name: str(row[name])
                            for name in feature_names
                            if row[name] is not None and not pd.isna(row[name])
                        },
                    }
                )
                for row in dataframe.to_dict("records")
            ]
        )

    return TraceLog(
        events=[
            TraceEvent.model_validate(
                {
                    "timestamp": str(row["timestamp"]),
                    "node_id": str(row["node_id"]),
                    "event": str(row["event"]),
                    "payload": json.loads(str(row["payload_json"])),
                }
            )
            for row in dataframe.to_dict("records")
        ]
    )
