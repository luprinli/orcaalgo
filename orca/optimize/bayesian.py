#!/usr/bin/env python3
"""Bayesian parameter optimization using optuna. Calls the Go REST API for backtests."""
import json
import os
import time
from datetime import timedelta
from typing import Any

try:
    import optuna
    import requests
except ImportError:
    optuna = None
    requests = None
    print("Install dependencies: pip install optuna requests")


def _default_api_base() -> str:
    return os.environ.get("ORCA_API_BASE", "http://localhost:8080/api/v1")


class BayesianOptimizer:
    def __init__(
        self,
        strategy_id: str = "intraday_mr",
        symbol: str = "EURUSD",
        train_start: str = "2022-01-01",
        train_end: str = "2023-12-31",
        test_start: str = "2024-01-01",
        test_end: str = "2024-12-31",
        capital: float = 100000.0,
        api_base: str | None = None,
        n_trials: int = 50,
        timeout_minutes: int = 30,
        objective_metric: str = "sharpe",
        param_ranges: dict[str, tuple[float, float]] | None = None,
    ):
        self.strategy_id = strategy_id
        self.symbol = symbol
        self.train_start = train_start
        self.train_end = train_end
        self.test_start = test_start
        self.test_end = test_end
        self.capital = capital
        self.api_base = api_base or _default_api_base()
        self.n_trials = n_trials
        self.timeout = timedelta(minutes=timeout_minutes)
        self.objective_metric = objective_metric
        self.param_ranges = param_ranges or self._default_ranges()

    def _default_ranges(self) -> dict[str, tuple[float, float]]:
        return {
            "entry_z": (-3.0, 3.0),
            "exit_z": (0.1, 1.5),
            "lookback": (10, 50),
            "stop_loss_pct": (0.5, 5.0),
            "take_profit_pct": (0.5, 5.0),
        }

    def _run_backtest(self, params: dict[str, float]) -> float:
        body = {
            "strategy_id": self.strategy_id,
            "symbols": [self.symbol],
            "start_date": self.train_start,
            "end_date": self.test_end,
            "initial_capital": self.capital,
            "strategy_params": params,
            "timeframe": "1d",
        }
        try:
            resp = requests.post(f"{self.api_base}/backtests", json=body, timeout=120)
            if resp.status_code != 200:
                return -999.0
            data = resp.json()
            sharpe = data.get("sharpe_ratio", 0)
            max_dd = data.get("max_drawdown", 0)
            return sharpe / (1 + max_dd / 100) if max_dd > 0 else sharpe
        except Exception:
            return -999.0

    def objective(self, trial) -> float:
        params = {}
        for name, (lo, hi) in self.param_ranges.items():
            params[name] = trial.suggest_float(name, lo, hi)
        score = self._run_backtest(params)
        score = max(score, -999.0)
        return score

    def run(self, progress_callback=None) -> dict[str, Any]:
        if optuna is None:
            return {"error": "optuna not installed. pip install optuna"}

        if progress_callback:
            progress_callback({"status": "creating_study", "progress_pct": 0})

        study = optuna.create_study(direction="maximize")
        start_time = time.time()

        for trial_num in range(self.n_trials):
            if time.time() - start_time > self.timeout.total_seconds():
                break
            study.optimize(self.objective, n_trials=1)
            if progress_callback:
                pct = (trial_num + 1) / self.n_trials * 100
                progress_callback({
                    "status": "running",
                    "progress_pct": round(pct, 1),
                    "current_trial": trial_num + 1,
                    "total_trials": self.n_trials,
                    "best_value": study.best_value if study.best_trial else None,
                    "elapsed_s": round(time.time() - start_time, 1),
                })

        if progress_callback:
            progress_callback({"status": "completed", "progress_pct": 100})

        return {
            "best_params": study.best_params,
            "best_value": study.best_value,
            "n_trials": len(study.trials),
            "elapsed_s": round(time.time() - start_time, 1),
            "trials": [
                {"number": t.number, "value": t.value, "params": t.params}
                for t in study.trials if t.value is not None
            ],
        }


def run_optimization(**kwargs) -> dict[str, Any]:
    optimizer = BayesianOptimizer(**kwargs)
    return optimizer.run(progress_callback=lambda p: print(
        f"  [{p.get('current_trial',0)}/{p.get('total_trials',50)}] "
        f"best={p.get('best_value','?'):.4f}  elapsed={p.get('elapsed_s',0):.0f}s"
    ))


if __name__ == "__main__":
    result = run_optimization(
        strategy_id="intraday_mr", symbol="EURUSD",
        train_start="2022-01-01", train_end="2023-12-31",
        test_start="2024-01-01", test_end="2024-12-31",
        n_trials=20,
    )
    print(json.dumps(result, indent=2, default=str))
