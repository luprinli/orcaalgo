"""artifact safety helpers."""

from __future__ import annotations

import re
from typing import Annotated

from pydantic import AfterValidator

POSIX_RELATIVE_PATH_PATTERN = re.compile(r"^(?!/)(?!.*\.\.)(?!.*\\)[A-Za-z0-9._\-/]+$")


def validate_posix_relative_path(path: str) -> str:
    """Validate a package-internal POSIX relative path."""

    if not isinstance(path, str):
        raise TypeError(f"path must be str, got {type(path).__name__}")
    if not path:
        raise ValueError("Empty path not allowed")
    if not POSIX_RELATIVE_PATH_PATTERN.match(path):
        raise ValueError(f"Path {path!r} violates POSIXRelativePath constraints")
    return path


POSIXRelativePath = Annotated[str, AfterValidator(validate_posix_relative_path)]
