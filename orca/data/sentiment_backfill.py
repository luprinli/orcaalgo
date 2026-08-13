"""Historical sentiment backfill from Alternative.me Fear & Greed Index.

Fetches the full historical Fear & Greed Index from https://api.alternative.me/fng/
and upserts into the sentiment_logs table for backfill completeness.
"""

from __future__ import annotations

import datetime
from typing import Any


def fetch_fear_greed_index(limit: int = 0) -> list[dict[str, Any]]:
    """Fetch historical Fear & Greed Index from Alternative.me.

    Args:
        limit: Number of records to fetch (0 = full history).

    Returns:
        List of dicts with {timestamp, value, value_classification}.
    """
    import urllib.request
    import json as _json

    url = f"https://api.alternative.me/fng/?limit={limit}&format=json"
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            data = _json.loads(resp.read().decode())
    except Exception as e:
        raise RuntimeError(f"Failed to fetch Fear & Greed Index: {e}") from e

    results = []
    for entry in data.get("data", []):
        timestamp = int(entry.get("timestamp", 0))
        value_str = entry.get("value", "50")
        classification = entry.get("value_classification", "Neutral")

        ts = datetime.datetime.fromtimestamp(timestamp, tz=datetime.timezone.utc)
        value = int(value_str)
        score = value

        label = classification
        label_map = {
            "Extreme Fear": "Extreme Fear",
            "Fear": "Fear",
            "Neutral": "Neutral",
            "Greed": "Greed",
            "Extreme Greed": "Extreme Greed",
        }
        label = label_map.get(classification, "Neutral")

        results.append({
            "timestamp": ts,
            "score": score,
            "label": label,
        })

    return results


def backfill_sentiment(
    limit: int = 0,
    verbose: bool = True,
) -> dict[str, Any]:
    """Fetch Fear & Greed Index from Alternative.me and upsert into sentiment_logs.

    Args:
        limit: Number of records (0 = full history).
        verbose: Print progress.

    Returns:
        Dict with {rows_inserted, rows_skipped, errors}.
    """
    import psycopg2
    import psycopg2.extras
    from orca.data.db_integration import get_connection

    stats = {"rows_inserted": 0, "rows_skipped": 0, "errors": []}

    try:
        entries = fetch_fear_greed_index(limit=limit)
        if verbose:
            print(f"Fetched {len(entries)} Fear & Greed records from Alternative.me")
    except Exception as e:
        stats["errors"].append(str(e))
        return stats

    if not entries:
        return stats

    conn = get_connection()
    try:
        rows = [
            (e["timestamp"], e["score"], e["label"])
            for e in entries
            if 0 <= e["score"] <= 100
        ]
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(cur, """
                INSERT INTO sentiment_logs (timestamp, score, label)
                VALUES %s
                ON CONFLICT (timestamp) DO UPDATE SET
                    score = EXCLUDED.score,
                    label = EXCLUDED.label
            """, rows, page_size=500)
            stats["rows_inserted"] = cur.rowcount
        conn.commit()
        if verbose:
            print(f"Upserted {stats['rows_inserted']} rows into sentiment_logs")
    except Exception as e:
        conn.rollback()
        stats["errors"].append(str(e))
    finally:
        conn.close()

    return stats
