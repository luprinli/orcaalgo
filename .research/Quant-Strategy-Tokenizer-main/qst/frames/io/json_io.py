"""Canonical JSON IO for QST frames."""

from __future__ import annotations

import json
from collections.abc import Mapping
from pathlib import Path
from typing import Any, TypeAlias

from qst.canonical_json import stable_json_bytes
from qst.frames.feature_frame import FeatureFrame
from qst.frames.market_frame import MarketFrame
from qst.frames.signal_frame import SignalFrame
from qst.frames.trace_log import TraceLog

Frame: TypeAlias = MarketFrame | SignalFrame | FeatureFrame | TraceLog


def frame_to_json_bytes(frame: Frame) -> bytes:
    return stable_json_bytes(frame.model_dump(mode="json"))


def frame_from_mapping(payload: Mapping[str, Any]) -> Frame:
    frame_version = payload.get("frame_version")
    if frame_version == "qst-market-frame/1":
        return MarketFrame.model_validate(payload)
    if frame_version == "qst-signal-frame/1":
        return SignalFrame.model_validate(payload)
    if frame_version == "qst-feature-frame/1":
        return FeatureFrame.model_validate(payload)
    if frame_version == "qst-trace-log/1":
        return TraceLog.model_validate(payload)
    raise ValueError(f"Unsupported frame_version: {frame_version!r}")


def frame_from_json_bytes(payload: bytes) -> Frame:
    raw = json.loads(payload.decode("utf-8"))
    if not isinstance(raw, dict):
        raise ValueError("Frame JSON must contain an object")
    return frame_from_mapping(raw)


def write_json_frame(frame: Frame, path: str | Path) -> None:
    Path(path).write_bytes(frame_to_json_bytes(frame))


def read_json_frame(path: str | Path) -> Frame:
    return frame_from_json_bytes(Path(path).read_bytes())
