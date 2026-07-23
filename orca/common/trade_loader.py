from __future__ import annotations

import json
import logging
import os
import random
import subprocess
from datetime import UTC, datetime, timedelta

logger = logging.getLogger("orca.common.trade_loader")


def load_trades_from_db_or_synthetic(
    since_dt: datetime,
    synthetic_count: int = 50,
    logger_name: str = "orca.common",
    synthetic_generator: str = "attribution",
) -> list[dict]:
    """Load trade records from DB, falling back to synthetic data.

    Args:
        since_dt: UTC datetime to fetch trades since.
        synthetic_count: Number of synthetic records to generate on fallback.
        logger_name: Logger name for debug messages.
        synthetic_generator: One of 'attribution' or 'calibration' to control
            the shape of synthetic data.

    Returns:
        List of trade dicts.
    """
    local_logger = logging.getLogger(logger_name)
    db_url = os.environ.get("ORCA_DB_URL", "")

    if db_url:
        try:
            result = subprocess.run(
                [
                    "python", "-c",
                    f"""
import json, sys
sys.path.insert(0, ".")
try:
    from orca.db import fetch_trades
    trades = fetch_trades(since="{since_dt.isoformat()}")
    print(json.dumps(trades, default=str))
except Exception:
    print(json.dumps([]))
""",
                ],
                capture_output=True, text=True, timeout=15,
            )
            if result.stdout.strip():
                return json.loads(result.stdout)
        except Exception:
            local_logger.debug("subprocess trade fetch failed, using synthetic data")

    return _generate_synthetic_trades(synthetic_count, synthetic_generator)


def _generate_synthetic_trades(count: int, generator: str) -> list[dict]:
    random.seed(42)
    samples: list[dict] = []

    if generator == "attribution":
        symbols = ["SPY", "QQQ", "AAPL", "MSFT", "TSLA"]
        for _ in range(count):
            s = random.choice(symbols)
            side = random.choice(["BUY", "SELL"])
            entry = random.uniform(20, 500)
            conf = random.uniform(0.5, 0.9)
            pnl = random.uniform(-100, 200)
            samples.append({
                "symbol": s,
                "side": side,
                "entry_price": round(entry, 2),
                "exit_price": round(entry * (1 + pnl / (entry * 10)), 2),
                "quantity": random.randint(1, 100),
                "pnl": round(pnl, 2),
                "cost": round(entry * random.randint(1, 100), 2),
                "confidence": round(conf, 4),
                "placed_at": (datetime.now(UTC) - timedelta(days=random.randint(1, 90))).isoformat(),
                "outcome": "win" if pnl > 0 else "loss",
            })
    elif generator == "calibration":
        for _ in range(count):
            p = random.random()
            outcome = "win" if random.random() < p else "loss"
            samples.append({
                "confidence": p,
                "outcome": outcome,
                "side": random.choice(["BUY", "SELL"]),
            })

    return samples
