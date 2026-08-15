"""Maps OrcaAlgo strategy IDs to VectorBT indicator configurations."""

STRATEGY_INDICATORS: dict[str, dict] = {
    "intraday_mr": {
        "description": "RSI-based mean reversion with entry/exit thresholds",
        "default_params": {"rsi_period": 20, "entry_threshold": 30, "exit_threshold": 50},
        "param_grid": {
            "rsi_period": [14, 20, 25, 30],
            "entry_threshold": [20, 25, 30, 35],
            "exit_threshold": [45, 50, 55, 60, 70],
        },
    },
    "trend_following": {
        "description": "EMA crossover with ADX filter",
        "default_params": {"ema_fast": 20, "ema_slow": 50, "adx_threshold": 25},
        "param_grid": {
            "ema_fast": [10, 20, 30, 40],
            "ema_slow": [40, 50, 60, 80],
            "adx_threshold": [20, 25, 30],
        },
    },
    "opening_range_breakout": {
        "description": "First-N-minute range breakout",
        "default_params": {"range_minutes": 5, "atr_mult": 2.0, "volume_mult": 1.5},
        "param_grid": {
            "range_minutes": [5, 15, 30],
            "atr_mult": [1.5, 2.0, 2.5, 3.0],
        },
    },
    "grid_trading": {
        "description": "Grid of limit orders with fixed spacing",
        "default_params": {"grid_levels": 5, "grid_spacing_pct": 1.0, "max_open": 10},
        "param_grid": {
            "grid_levels": [3, 5, 7, 10],
            "grid_spacing_pct": [0.5, 1.0, 1.5, 2.0],
        },
    },
}


def get_config(strategy_id: str) -> dict:
    if strategy_id not in STRATEGY_INDICATORS:
        raise ValueError(f"Unknown strategy: {strategy_id}")
    return STRATEGY_INDICATORS[strategy_id]
