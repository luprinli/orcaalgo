"""File-based progress reporter for long-running batch processes.

Writes a JSON progress file that both Python and Go can read,
enabling real-time status monitoring across process boundaries.

Usage:
    from orca.simulation.progress import ProgressReporter

    reporter = ProgressReporter("batch-123")
    reporter.start(total_items=5000, description="Generating SPY candles")
    for chunk in chunks:
        if reporter.should_halt():
            break
        process(chunk)
        reporter.advance(len(chunk), extra={"current_price": 450.23})

    reporter.finish()
"""

from __future__ import annotations

import json
import os
import signal
import time
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

DEFAULT_PROGRESS_DIR = Path(os.environ.get("ORCA_PROGRESS_DIR", "data/progress"))


class ProgressReporter:
    """Thread-safe, file-based progress reporter.

    Writes JSON to {progress_dir}/{batch_id}.json.
    Reads .halt_{batch_id} flag file for graceful shutdown.
    """

    def __init__(
        self,
        batch_id: str,
        progress_dir: str | Path = DEFAULT_PROGRESS_DIR,
        auto_signal: bool = True,
    ):
        self.batch_id = batch_id
        self.progress_dir = Path(progress_dir)
        self.progress_dir.mkdir(parents=True, exist_ok=True)
        self.path = self.progress_dir / f"{batch_id}.json"
        self.halt_path = self.progress_dir / f".halt_{batch_id}"
        self.lock_path = self.progress_dir / f"{batch_id}.lock"

        self._started_at: datetime | None = None
        self._total: int = 0
        self._completed: int = 0
        self._failed: int = 0
        self._status: str = "pending"
        self._description: str = ""
        self._extra: dict[str, Any] = {}
        self._last_write: float = 0.0
        self._write_interval: float = 1.0

        if auto_signal:
            signal.signal(signal.SIGINT, self._handle_signal)
            signal.signal(signal.SIGTERM, self._handle_signal)

    def start(
        self, total_items: int, description: str = "", extra: dict[str, Any] | None = None
    ) -> None:
        self._started_at = datetime.now(UTC)
        self._total = total_items
        self._status = "running"
        self._description = description
        self._extra = extra or {}
        self._write()

    def advance(self, items: int = 1, extra: dict[str, Any] | None = None) -> None:
        self._completed += items
        if extra:
            self._extra.update(extra)
        self._maybe_write()

    def advance_failed(self, items: int = 1) -> None:
        self._failed += items
        self._maybe_write()

    def finish(self, status: str = "completed", extra: dict[str, Any] | None = None) -> None:
        self._status = status
        if extra:
            self._extra.update(extra)
        self._write()
        if self.lock_path.exists():
            self.lock_path.unlink()

    def should_halt(self) -> bool:
        if self.halt_path.exists():
            self.halt_path.unlink(missing_ok=True)
            self._status = "halted"
            self._write()
            return True
        return False

    def request_halt(self) -> None:
        self.halt_path.touch()

    @property
    def progress_pct(self) -> float:
        if self._total == 0:
            return 100.0 if self._status == "completed" else 0.0
        return min(100.0, (self._completed + self._failed) / self._total * 100)

    @property
    def elapsed_seconds(self) -> float:
        if self._started_at is None:
            return 0.0
        return (datetime.now(UTC) - self._started_at).total_seconds()

    @property
    def eta_seconds(self) -> float:
        done = self._completed + self._failed
        if done == 0 or self._started_at is None:
            return float("inf")
        rate = done / max(0.001, self.elapsed_seconds)
        remaining = self._total - done
        return remaining / max(0.0001, rate)

    def snapshot(self) -> dict[str, Any]:
        return {
            "batch_id": self.batch_id,
            "status": self._status,
            "description": self._description,
            "progress_pct": round(self.progress_pct, 1),
            "completed": self._completed,
            "failed": self._failed,
            "total": self._total,
            "elapsed_s": round(self.elapsed_seconds, 1),
            "eta_s": round(self.eta_seconds, 1) if self.eta_seconds < 1e9 else None,
            "started_at": self._started_at.isoformat() if self._started_at else None,
            "extra": self._extra,
            "halt_file_exists": self.halt_path.exists(),
        }

    def _maybe_write(self) -> None:
        now = time.monotonic()
        if now - self._last_write >= self._write_interval:
            self._write()

    def _write(self) -> None:
        self._last_write = time.monotonic()
        with self.lock_path.open("w") as f:
            f.write(str(os.getpid()))
        with self.path.open("w") as f:
            json.dump(self.snapshot(), f, indent=2, default=str)

    def _handle_signal(self, signum: int, frame: Any) -> None:
        self.finish(status="interrupted")
        raise SystemExit(1)


def read_progress(
    batch_id: str, progress_dir: str | Path = DEFAULT_PROGRESS_DIR
) -> dict[str, Any] | None:
    """Read the current progress of a batch from its JSON file."""
    path = Path(progress_dir) / f"{batch_id}.json"
    if not path.exists():
        return None
    with open(path) as f:
        return json.load(f)


def list_active_batches(progress_dir: str | Path = DEFAULT_PROGRESS_DIR) -> list[dict[str, Any]]:
    """List all active (running/halted/pending) batches."""
    pd = Path(progress_dir)
    if not pd.exists():
        return []
    results: list[dict[str, Any]] = []
    for f in sorted(pd.glob("*.json")):
        try:
            data = json.loads(f.read_text())
            if data.get("status") in ("running", "halted", "pending"):
                results.append(data)
        except (json.JSONDecodeError, OSError):
            continue
    return results


def halt_batch(batch_id: str, progress_dir: str | Path = DEFAULT_PROGRESS_DIR) -> bool:
    """Request graceful halt of a running batch."""
    pd = Path(progress_dir)
    pd.mkdir(parents=True, exist_ok=True)
    halt_path = pd / f".halt_{batch_id}"
    halt_path.touch()
    return halt_path.exists()
