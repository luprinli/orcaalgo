"""Execution context for local runs."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class ExecutionContext:
    """Runtime context passed through the local executor."""

    run_id: str
    trace_path: Path | None = None
