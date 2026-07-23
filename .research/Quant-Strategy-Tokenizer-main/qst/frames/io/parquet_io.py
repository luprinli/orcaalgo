"""Parquet IO for QST frames."""

from __future__ import annotations

import importlib
from pathlib import Path
from typing import Any

from qst.frames.io.arrow_io import arrow_table_to_frame, frame_to_arrow_table
from qst.frames.io.csv_io import FrameVersion
from qst.frames.io.json_io import Frame


def _load_pyarrow_parquet() -> Any:
    try:
        return importlib.import_module("pyarrow.parquet")
    except ModuleNotFoundError as exc:
        raise ImportError(
            "pyarrow is required for QST Parquet frame IO. Install with "
            "`pip install -e .[parquet]` or `pip install pyarrow>=14`."
        ) from exc


def write_parquet_frame(frame: Frame, path: str | Path) -> None:
    pq = _load_pyarrow_parquet()
    table = frame_to_arrow_table(frame)
    pq.write_table(table, path, compression="NONE")


def read_parquet_frame(path: str | Path, frame_version: FrameVersion | None = None) -> Frame:
    pq = _load_pyarrow_parquet()
    table = pq.read_table(path)
    return arrow_table_to_frame(table, frame_version)
