"""Exit optimization model training.

Trains a LightGBM regressor to predict exit urgency (0=hold, 1=exit now)
based on trade-state features. The model learns when to tighten stops
early (mean-reversion trades) vs when to widen them (trend trades).
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

import numpy as np

logger = logging.getLogger("orca.ml.train.exit_model")

try:
    import lightgbm as lgb
    HAS_LIGHTGBM = True
except ImportError:
    lgb = None
    HAS_LIGHTGBM = False


@dataclass(frozen=True)
class ExitTrainingResult:
    model: object | None
    mse: float
    mae: float
    r2: float
    feature_importance: dict[str, float]
    passed_gate: bool
    params: dict = field(default_factory=dict)
    metadata: dict = field(default_factory=dict)


FEATURE_NAMES = [
    "pnl_atr", "stop_distance", "bars_since_entry",
    "vol_change", "hmm_state", "cvd_trend", "volume_trend",
    "adx", "mae", "mfe", "hour_sin", "signal_confidence",
]


class ExitModelTrainer:
    def __init__(
        self,
        n_estimators: int = 100,
        max_depth: int = 5,
        learning_rate: float = 0.05,
    ):
        self.n_estimators = n_estimators
        self.max_depth = max_depth
        self.learning_rate = learning_rate

    def train(
        self,
        X: np.ndarray,
        y: np.ndarray,
        feature_names: list[str] | None = None,
        random_state: int = 42,
    ) -> ExitTrainingResult:
        if not HAS_LIGHTGBM:
            raise ImportError("lightgbm is required. Install: pip install lightgbm")
        if len(X) < 50:
            raise ValueError(f"insufficient samples: {len(X)}")

        split = int(len(X) * 0.8)
        X_train, X_test = X[:split], X[split:]
        y_train, y_test = y[:split], y[split:]

        model = lgb.LGBMRegressor(
            n_estimators=self.n_estimators,
            max_depth=self.max_depth,
            learning_rate=self.learning_rate,
            subsample=0.8,
            colsample_bytree=0.8,
            random_state=random_state,
            verbosity=-1,
        )
        model.fit(X_train, y_train)

        y_pred = model.predict(X_test)
        mse = float(np.mean((y_test - y_pred) ** 2))
        mae = float(np.mean(np.abs(y_test - y_pred)))
        ss_res = np.sum((y_test - y_pred) ** 2)
        ss_tot = np.sum((y_test - np.mean(y_test)) ** 2)
        r2 = 1.0 - ss_res / max(ss_tot, 1e-12)

        names = feature_names or FEATURE_NAMES
        importance = {}
        for i, imp in enumerate(model.feature_importances_):
            if i < len(names):
                importance[names[i]] = float(imp)

        passed = mse < 0.15 and r2 > 0.1

        logger.info("exit model: mse=%.4f mae=%.4f r2=%.4f passed=%v", mse, mae, r2, passed)

        return ExitTrainingResult(
            model=model,
            mse=mse,
            mae=mae,
            r2=r2,
            feature_importance=importance,
            passed_gate=passed,
            params={
                "n_estimators": self.n_estimators,
                "max_depth": self.max_depth,
                "learning_rate": self.learning_rate,
            },
            metadata={
                "train_samples": len(X_train),
                "test_samples": len(X_test),
            },
        )

    def save_model(self, result: ExitTrainingResult, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        if result.model is not None and HAS_LIGHTGBM:
            result.model.booster_.save_model(str(path))
            logger.info("exit model saved to %s", path)

    def save_results(self, result: ExitTrainingResult, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        data = {
            "mse": result.mse,
            "mae": result.mae,
            "r2": result.r2,
            "passed_gate": result.passed_gate,
            "feature_importance": result.feature_importance,
            "params": result.params,
            "metadata": result.metadata,
            "trained_at": datetime.now(UTC).isoformat(),
        }
        with open(path, "w") as f:
            json.dump(data, f, indent=2)


def urgency_to_stop_multiplier(
    urgency: float,
    base_multiplier: float = 2.0,
    adjustment_factor: float = 0.5,
) -> float:
    """Convert exit urgency to a dynamic stop multiplier.

    urgency=0.0 (hold) → stopMultiplier = base * (1 + 0*0.5) = base (wider stop)
    urgency=0.5 (neutral) → stopMultiplier = base * (1 + 0.5*0.5) = 1.25*base
    urgency=1.0 (exit now) → stopMultiplier = base * (1 + 1.0*0.5) = 1.5*base (tighter stop)

    This means: higher urgency = tighter stop = exit sooner.
    """
    return base_multiplier * (1.0 + urgency * adjustment_factor)
