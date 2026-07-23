"""Test helpers shared by integration and e2e tests."""

from __future__ import annotations

from pathlib import Path

import pandas as pd


def load_sample_market(path: Path) -> pd.DataFrame:
    """Load the deterministic sample market CSV with a datetime index when present."""

    frame = pd.read_csv(path)
    for column in ("timestamp", "ts", "time", "date"):
        if column in frame.columns:
            frame[column] = pd.to_datetime(frame[column])
            return frame.set_index(column)
    return frame
