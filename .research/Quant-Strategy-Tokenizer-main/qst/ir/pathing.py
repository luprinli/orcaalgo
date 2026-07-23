"""Path helpers for public GKR file forms."""

from __future__ import annotations

from pathlib import Path


def is_gkr_source(path: str | Path) -> bool:
    """Return true when *path* names an editable GKR source document."""

    return Path(path).name.endswith(".gkr.yaml")


def is_gkr_package(path: str | Path) -> bool:
    """Return true when *path* names a packaged Graph Kernel Record."""

    return Path(path).suffix == ".gkr"
