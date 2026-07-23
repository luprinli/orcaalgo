"""CSV IO for frame models."""

from __future__ import annotations

import csv
import io
import json
from pathlib import Path
from typing import Literal, TypeAlias

from qst.frames.base import timestamp_to_string
from qst.frames.feature_frame import FeatureFrame, FeatureRow
from qst.frames.market_frame import MarketFrame, OHLCVBar
from qst.frames.signal_frame import SignalFrame, SignalRow
from qst.frames.trace_log import TraceEvent, TraceLog

Frame: TypeAlias = MarketFrame | SignalFrame | FeatureFrame | TraceLog
FrameVersion: TypeAlias = Literal[
    "qst-market-frame/1",
    "qst-signal-frame/1",
    "qst-feature-frame/1",
    "qst-trace-log/1",
]


def _write_dicts(fieldnames: list[str], rows: list[dict[str, str]]) -> str:
    handle = io.StringIO(newline="")
    writer = csv.DictWriter(handle, fieldnames=fieldnames, lineterminator="\n")
    writer.writeheader()
    writer.writerows(rows)
    return handle.getvalue()


def _read_dicts(text: str) -> tuple[list[str], list[dict[str, str]]]:
    reader = csv.DictReader(io.StringIO(text, newline=""))
    fieldnames = reader.fieldnames
    if fieldnames is None:
        raise ValueError("CSV frame is missing a header")
    fieldnames_list = list(fieldnames)

    rows: list[dict[str, str]] = []
    for row in reader:
        rows.append({key: _required(row, key) for key in fieldnames_list})
    return fieldnames_list, rows


def _required(row: dict[str, str | None], key: str) -> str:
    value = row.get(key)
    if value is None:
        raise ValueError(f"CSV row missing required field {key!r}")
    return value


def frame_to_csv_text(frame: Frame) -> str:
    if isinstance(frame, MarketFrame):
        return _market_to_csv(frame)
    if isinstance(frame, SignalFrame):
        return _signal_to_csv(frame)
    if isinstance(frame, FeatureFrame):
        return _feature_to_csv(frame)
    return _trace_to_csv(frame)


def frame_from_csv_text(text: str, frame_version: FrameVersion) -> Frame:
    if frame_version == "qst-market-frame/1":
        return _market_from_csv(text)
    if frame_version == "qst-signal-frame/1":
        return _signal_from_csv(text)
    if frame_version == "qst-feature-frame/1":
        return _feature_from_csv(text)
    return _trace_from_csv(text)


def write_csv_frame(frame: Frame, path: str | Path) -> None:
    Path(path).write_text(frame_to_csv_text(frame), encoding="utf-8")


def read_csv_frame(path: str | Path, frame_version: FrameVersion) -> Frame:
    return frame_from_csv_text(Path(path).read_text(encoding="utf-8"), frame_version)


def _market_to_csv(frame: MarketFrame) -> str:
    rows = [
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
    ]
    return _write_dicts(["timestamp", "symbol", "open", "high", "low", "close", "volume"], rows)


def _market_from_csv(text: str) -> MarketFrame:
    _, rows = _read_dicts(text)
    return MarketFrame(
        bars=[
            OHLCVBar.model_validate(
                {
                    "timestamp": row["timestamp"],
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                }
            )
            for row in rows
        ]
    )


def _signal_to_csv(frame: SignalFrame) -> str:
    rows = [
        {
            "timestamp": timestamp_to_string(row.timestamp),
            "symbol": row.symbol,
            "signal": row.signal,
            "size": row.size,
        }
        for row in frame.rows
    ]
    return _write_dicts(["timestamp", "symbol", "signal", "size"], rows)


def _signal_from_csv(text: str) -> SignalFrame:
    _, rows = _read_dicts(text)
    return SignalFrame(
        rows=[
            SignalRow.model_validate(
                {
                    "timestamp": row["timestamp"],
                    "symbol": row["symbol"],
                    "signal": row["signal"],
                    "size": row["size"],
                }
            )
            for row in rows
        ]
    )


def _feature_to_csv(frame: FeatureFrame) -> str:
    feature_names = sorted({name for row in frame.rows for name in row.features})
    fieldnames = ["timestamp", "symbol", *feature_names]
    rows = [
        {
            "timestamp": timestamp_to_string(row.timestamp),
            "symbol": row.symbol,
            **{name: row.features.get(name, "") for name in feature_names},
        }
        for row in frame.rows
    ]
    return _write_dicts(fieldnames, rows)


def _feature_from_csv(text: str) -> FeatureFrame:
    fieldnames, rows = _read_dicts(text)
    feature_names = [name for name in fieldnames if name not in {"timestamp", "symbol"}]
    return FeatureFrame(
        rows=[
            FeatureRow.model_validate(
                {
                    "timestamp": row["timestamp"],
                    "symbol": row["symbol"],
                    "features": {name: row[name] for name in feature_names if row[name] != ""},
                }
            )
            for row in rows
        ]
    )


def _trace_to_csv(frame: TraceLog) -> str:
    rows = [
        {
            "timestamp": timestamp_to_string(event.timestamp),
            "node_id": event.node_id,
            "event": event.event,
            "payload_json": json.dumps(event.payload, sort_keys=True, separators=(",", ":")),
        }
        for event in frame.events
    ]
    return _write_dicts(["timestamp", "node_id", "event", "payload_json"], rows)


def _trace_from_csv(text: str) -> TraceLog:
    _, rows = _read_dicts(text)
    return TraceLog(
        events=[
            TraceEvent.model_validate(
                {
                    "timestamp": row["timestamp"],
                    "node_id": row["node_id"],
                    "event": row["event"],
                    "payload": json.loads(row["payload_json"]),
                }
            )
            for row in rows
        ]
    )
