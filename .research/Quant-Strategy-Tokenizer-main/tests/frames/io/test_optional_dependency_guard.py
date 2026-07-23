from __future__ import annotations

import importlib

import pytest

from qst.frames import MarketFrame
from qst.frames.io import arrow_io, parquet_io


def _raise_for_pyarrow(name: str):  # type: ignore[no-untyped-def]
    if name == "pyarrow" or name == "pyarrow.parquet":
        raise ModuleNotFoundError(name)
    return importlib.import_module(name)


def test_arrow_module_imports_without_pyarrow_but_call_raises_install_hint(monkeypatch) -> None:  # type: ignore[no-untyped-def]
    monkeypatch.setattr(arrow_io.importlib, "import_module", _raise_for_pyarrow)

    with pytest.raises(ImportError, match=r"pip install.*pyarrow"):
        arrow_io.frame_to_arrow_table(MarketFrame())


def test_parquet_module_imports_without_pyarrow_but_call_raises_install_hint(monkeypatch, tmp_path) -> None:  # type: ignore[no-untyped-def]
    monkeypatch.setattr(parquet_io.importlib, "import_module", _raise_for_pyarrow)

    with pytest.raises(ImportError, match=r"pip install.*pyarrow"):
        parquet_io.write_parquet_frame(MarketFrame(), tmp_path / "frame.parquet")
