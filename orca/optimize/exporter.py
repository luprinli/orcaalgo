"""Export optimized parameters to .gkr.yaml with content-addressable hash."""

import hashlib
import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import yaml


def export_best_params(
    strategy_id: str,
    best_params: dict[str, Any],
    metrics: dict[str, Any],
    validation: dict[str, Any] | None = None,
    output_dir: str | Path = "configs/strategies",
) -> Path:
    """Write optimized strategy config to .gkr.yaml with content hash."""
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    config = {
        "version": 1,
        "strategy_id": strategy_id,
        "parameters": {k: v for k, v in best_params.items()},
        "metrics": {
            "sharpe_ratio": metrics.get("sharpe_ratio"),
            "max_drawdown": metrics.get("max_drawdown"),
            "total_return": metrics.get("total_return"),
            "win_rate": metrics.get("win_rate"),
            "num_trades": metrics.get("num_trades"),
        },
        "optimized_at": datetime.now(UTC).isoformat(),
        "optimized_by": "vectorbt",
    }

    if validation:
        config["validation"] = validation

    canonical = json.dumps(config, sort_keys=True, default=str)
    config["content_hash"] = hashlib.sha256(canonical.encode()).hexdigest()

    output_path = output_dir / f"{strategy_id}.gkr.yaml"
    with open(output_path, "w") as f:
        yaml.dump(config, f, sort_keys=False, default_flow_style=False)

    return output_path
