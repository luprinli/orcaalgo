"""Convert VectorBT best parameters to GKR YAML format.

CRITICAL: v1 output format MUST match orca/optimize/exporter.py v1 schema.
This ensures the Go engine can load configs regardless of which pipeline produced them.

v2 format (with GKR node-graph mapping) is planned but not yet supported by the
Go engine loader.
"""

import hashlib
import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import yaml

SUPPORTED_SCHEMA_VERSIONS = (1, 2)


def params_to_gkr(
    strategy_name: str,
    symbol: str,
    params: dict[str, Any],
    metrics: dict[str, Any] | None = None,
    validation: dict[str, Any] | None = None,
    output_dir: str = "configs/strategies",
    schema_version: int = 1,
) -> Path:
    """Generate GKR YAML file from parameters.

    Args:
        strategy_name: Strategy ID (e.g. "intraday_mr")
        symbol: Ticker symbol used for discovery
        params: Best parameters dict
        metrics: Optional metrics from sweep
        validation: Optional walk-forward validation results
        output_dir: Destination directory for .gkr.yaml
        schema_version: 1 = flat format (matching exporter.py), 2 = node-graph (planned)

    Returns:
        Path to generated .gkr.yaml file.
    """
    if schema_version not in SUPPORTED_SCHEMA_VERSIONS:
        raise ValueError(
            f"Unsupported schema_version: {schema_version}. "
            f"Supported: {SUPPORTED_SCHEMA_VERSIONS}"
        )

    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    if schema_version == 2:
        config = _build_v2_config(strategy_name, params, metrics, validation)
    else:
        config = _build_v1_config(strategy_name, params, metrics, validation)

    canonical = json.dumps(
        {k: v for k, v in config.items() if k != "content_hash"},
        sort_keys=True,
        default=str,
    )
    config["content_hash"] = "sha256:" + hashlib.sha256(canonical.encode()).hexdigest()

    filename = f"{strategy_name}.gkr.yaml"
    output_path = output_dir / filename
    with open(output_path, "w", encoding="utf-8") as f:
        yaml.dump(config, f, sort_keys=False, default_flow_style=False)

    return output_path


def _build_v1_config(
    strategy_name: str,
    params: dict[str, Any],
    metrics: dict[str, Any] | None,
    validation: dict[str, Any] | None,
) -> dict[str, Any]:
    """Build v1 flat config — identical schema to orca/optimize/exporter.py output."""
    config: dict[str, Any] = {
        "version": 1,
        "strategy_id": strategy_name,
        "parameters": {
            k: float(v) if isinstance(v, (int, float, bool)) else v
            for k, v in params.items()
        },
        "metrics": {
            "sharpe_ratio": metrics.get("sharpe_ratio") if metrics else None,
            "max_drawdown": metrics.get("max_drawdown") if metrics else None,
            "total_return": metrics.get("total_return") if metrics else None,
            "win_rate": metrics.get("win_rate") if metrics else None,
            "num_trades": metrics.get("num_trades") if metrics else None,
        },
        "optimized_at": datetime.now(UTC).isoformat(),
        "optimized_by": "vectorbt",
    }
    if validation:
        config["validation"] = validation
    return config


def _build_v2_config(
    strategy_name: str,
    params: dict[str, Any],
    metrics: dict[str, Any] | None,
    validation: dict[str, Any] | None,
) -> dict[str, Any]:
    """Build v2 config with GKR node-graph mapping.

    Planned format — enables direct consumption by orca validate.
    Not yet supported by Go engine loader.
    """
    gkr_mapping = _generate_gkr_mapping(strategy_name, params)

    config: dict[str, Any] = {
        "version": 2,
        "param_schema_version": "1.0",
        "strategy_id": strategy_name,
        "parameters": params,
        "gkr_mapping": gkr_mapping,
        "metrics": {
            "sharpe_ratio": metrics.get("sharpe_ratio") if metrics else None,
            "max_drawdown": metrics.get("max_drawdown") if metrics else None,
            "total_return": metrics.get("total_return") if metrics else None,
            "win_rate": metrics.get("win_rate") if metrics else None,
            "num_trades": metrics.get("num_trades") if metrics else None,
        },
        "optimized_at": datetime.now(UTC).isoformat(),
        "optimized_by": "vectorbt",
    }
    if validation:
        config["validation"] = validation
    return config


def _generate_gkr_mapping(strategy_name: str, params: dict[str, Any]) -> dict[str, Any]:
    """Map flat parameter names to GKR node paths.

    Uses the canonical GKR templates from configs/strategies/ for reference.
    """
    mappings: dict[str, dict[str, dict[str, str]]] = {
        "intraday_mr": {
            "rsi_period": {"node": "zscore_calc", "param": "lookback"},
            "entry_threshold": {"node": "signal_gen", "param": "entry_threshold"},
            "exit_threshold": {"node": "signal_gen", "param": "exit_threshold"},
        },
        "trend_following": {
            "ema_fast": {"node": "ema_crossover", "param": "fast_period"},
            "ema_slow": {"node": "ema_crossover", "param": "slow_period"},
            "adx_threshold": {"node": "adx_filter", "param": "threshold"},
        },
        "opening_range_breakout": {
            "range_minutes": {"node": "orb_signal", "param": "range_minutes"},
            "atr_mult": {"node": "orb_signal", "param": "atr_mult"},
            "volume_mult": {"node": "orb_signal", "param": "volume_mult"},
        },
        "grid_trading": {
            "grid_levels": {"node": "grid_signal", "param": "levels"},
            "grid_spacing_pct": {"node": "grid_signal", "param": "spacing_pct"},
            "max_open": {"node": "grid_signal", "param": "max_open"},
        },
    }
    strategy_map = mappings.get(strategy_name, {})
    return {k: strategy_map[k] for k in params if k in strategy_map}
