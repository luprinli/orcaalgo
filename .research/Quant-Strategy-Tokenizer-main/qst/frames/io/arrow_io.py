"""Arrow Table interop for QST frames.

The module is importable without pyarrow installed. Functions that need
pyarrow raise an actionable ImportError at call time.
"""

from __future__ import annotations

import importlib
from typing import Any

from qst.frames.io.csv_io import FrameVersion
from qst.frames.io.json_io import Frame
from qst.frames.io.pandas_io import dataframe_to_frame, frame_to_dataframe

_FRAME_VERSION_METADATA_KEY = b"qst_frame_version"


def _load_pyarrow() -> Any:
    try:
        return importlib.import_module("pyarrow")
    except ModuleNotFoundError as exc:
        raise ImportError(
            "pyarrow is required for QST Arrow frame IO. Install with "
            "`pip install -e .[parquet]` or `pip install pyarrow>=14`."
        ) from exc


def frame_to_arrow_table(frame: Frame) -> Any:
    pa = _load_pyarrow()
    dataframe = frame_to_dataframe(frame)
    table = pa.Table.from_pandas(dataframe, preserve_index=False)
    metadata = dict(table.schema.metadata or {})
    metadata[_FRAME_VERSION_METADATA_KEY] = frame.frame_version.encode("utf-8")
    return table.replace_schema_metadata(metadata)


def arrow_table_to_frame(table: Any, frame_version: FrameVersion | None = None) -> Frame:
    if frame_version is None:
        metadata = table.schema.metadata or {}
        raw_version = metadata.get(_FRAME_VERSION_METADATA_KEY)
        if raw_version is None:
            raise ValueError("Arrow table is missing qst_frame_version metadata")
        frame_version = raw_version.decode("utf-8")

    dataframe = table.to_pandas()
    return dataframe_to_frame(dataframe, frame_version)
