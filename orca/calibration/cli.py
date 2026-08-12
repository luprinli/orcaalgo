from __future__ import annotations

from datetime import datetime

from orca.common.trade_loader import load_trades_from_db_or_synthetic


def _load_trades_for_calibration(since_dt: datetime) -> list[dict]:
    """Load trade records from the trade_executions table or from fixtures."""
    return load_trades_from_db_or_synthetic(
        since_dt,
        synthetic_count=200,
        logger_name="orca.calibration",
        synthetic_generator="calibration",
    )
