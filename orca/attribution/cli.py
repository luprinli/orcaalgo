from __future__ import annotations

from datetime import datetime

from orca.common.trade_loader import load_trades_from_db_or_synthetic


def _load_trades_for_attribution(since_dt: datetime) -> list[dict]:
    return load_trades_from_db_or_synthetic(
        since_dt,
        synthetic_count=50,
        logger_name="orca.attribution",
        synthetic_generator="attribution",
    )
