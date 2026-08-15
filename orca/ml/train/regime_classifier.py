"""6-class regime classifier training pipeline.

Trained on HMM alpha vectors + market features to produce fine-grained
regime probabilities. Output is a continuous regime score (0.0-1.0) that
replaces the step-function Kelly multipliers in the position sizer.

States: CALM, ACCUMULATION, TRENDING, DISTRIBUTION, HIGH_VOL, CRISIS

Training labels come from rule-based pre-labeling (existing in
orca/simulation/calibrate_regime.py), synthetic augmentation
(orca/simulation/regime.py), and manual verification.

The model learns:
  - HMM alpha[4] → what the 1D HMM already thinks
  - VIX → macro fear gauge
  - Sentiment → crowd positioning
  - CVD trend → smart money direction
  - Vol structure → regime transition signals
  - Time features → seasonality in regime changes
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np

from orca.ml.config import REGIME_ACCURACY_MIN

logger = logging.getLogger("orca.ml.train.regime_classifier")

try:
    import xgboost as xgb
except ImportError:
    xgb = None

REGIME_LABELS = [
    "calm",  # 0
    "accumulation",  # 1
    "trending",  # 2
    "distribution",  # 3
    "high_vol",  # 4
    "crisis",  # 5
]

REGIME_SCORE_WEIGHTS = [1.0, 0.9, 0.8, 0.7, 0.4, 0.0]


@dataclass(frozen=True)
class RegimeTrainingResult:
    model: object | None
    accuracy: float
    roc_auc_ovo: float
    confusion_matrix: np.ndarray | None
    feature_importance: dict[str, float]
    passed_gate: bool
    params: dict = field(default_factory=dict)
    metadata: dict = field(default_factory=dict)


@dataclass(frozen=True)
class RegimeClassifier:
    n_estimators: int = 100
    max_depth: int = 4
    learning_rate: float = 0.05
    subsample: float = 0.8
    accuracy_min: float = REGIME_ACCURACY_MIN

    def train(
        self,
        X: np.ndarray,
        y: np.ndarray,
        feature_names: list[str] | None = None,
        random_state: int = 42,
    ) -> RegimeTrainingResult:
        if xgb is None:
            raise ImportError("xgboost is required for regime classifier training")

        n_samples = len(X)
        n_classes = len(np.unique(y))
        if n_samples < 200:
            raise ValueError(f"insufficient samples: {n_samples} (need 200+)")

        split_idx = int(n_samples * 0.8)
        X_train, X_test = X[:split_idx], X[split_idx:]
        y_train, y_test = y[:split_idx], y[split_idx:]

        model = xgb.XGBClassifier(
            n_estimators=self.n_estimators,
            max_depth=self.max_depth,
            learning_rate=self.learning_rate,
            subsample=self.subsample,
            objective="multi:softprob",
            num_class=n_classes,
            eval_metric="mlogloss",
            random_state=random_state,
            verbosity=0,
        )

        model.fit(X_train, y_train, verbose=False)

        y_pred = model.predict(X_test)
        accuracy = float(np.mean(y_pred == y_test))

        y_prob = model.predict_proba(X_test)
        roc_auc = _multiclass_roc_auc(y_test, y_prob, n_classes)

        cm = None
        try:
            cm = np.zeros((n_classes, n_classes), dtype=int)
            for true_i, pred_i in zip(y_test.astype(int), y_pred.astype(int), strict=False):
                cm[true_i, pred_i] += 1
        except Exception:
            logger.debug("confusion matrix computation skipped")

        importance = {}
        if feature_names:
            for i, imp in enumerate(model.feature_importances_):
                if i < len(feature_names):
                    importance[feature_names[i]] = float(imp)
                else:
                    importance[f"f{i}"] = float(imp)

        passed = accuracy >= self.accuracy_min

        logger.info(
            "regime classifier: accuracy=%.3f roc_auc=%.3f passed=%v",
            accuracy,
            roc_auc,
            passed,
        )

        return RegimeTrainingResult(
            model=model,
            accuracy=accuracy,
            roc_auc_ovo=roc_auc,
            confusion_matrix=cm,
            feature_importance=importance,
            passed_gate=passed,
            params={
                "n_estimators": self.n_estimators,
                "max_depth": self.max_depth,
                "learning_rate": self.learning_rate,
                "n_classes": n_classes,
            },
            metadata={
                "train_samples": len(X_train),
                "test_samples": len(X_test),
                "n_classes": n_classes,
            },
        )

    def save_model(self, result: RegimeTrainingResult, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        if result.model is not None:
            result.model.save_model(str(path))
            logger.info("regime model saved to %s", path)


def continuous_regime_score(probs: np.ndarray, weights: list[float] | None = None) -> float:
    """Convert n-class probabilities to a continuous regime score [0, 1].

    Weighted sum: calm/accum/trending → higher score (trade more),
    distribution/high_vol → lower score (trade less),
    crisis → zero (no trading).

    1.0 = ideal trading conditions, 0.0 = stop trading.
    """
    if weights is None:
        weights = REGIME_SCORE_WEIGHTS
    n = len(probs)
    w = weights[:n] if len(weights) >= n else weights + [0.0] * (n - len(weights))
    return float(np.clip(np.dot(probs, w), 0.0, 1.0))


def score_to_kelly_multiplier(score: float) -> float:
    """Map continuous regime score to Kelly position size multiplier.

    Smooth interpolation between the old step-function values:
      score=1.0 → 1.5x (calm — trade at 150%)
      score=0.8 → 1.0x (trending — normal sizing)
      score=0.5 → 0.5x (high_vol — half sizing)
      score=0.3 → 0.25x (distribution — minimal sizing)
      score=0.0 → 0.0x (crisis — no trading)
    """
    return float(np.clip(1.5 * score, 0.0, 1.5))


def build_regime_features(
    hmm_alpha: np.ndarray,
    vix: float = 20.0,
    sentiment: int = 50,
    cvd_trend: float = 0.0,
    vol_structure: float = 0.5,
    hour: float = 12.0,
) -> np.ndarray:
    """Build feature vector for regime classification.

    Args:
        hmm_alpha: 4-element HMM alpha vector.
        vix: Current VIX value.
        sentiment: Sentiment score 0-100.
        cvd_trend: CVD trend (-1 to 1).
        vol_structure: Volatility term structure (0=backward, 0.5=flat, 1=contango).
        hour: Hour of day for session effects.

    Returns:
        Feature vector, shape (14,).
    """
    features = np.zeros(14)
    features[:4] = hmm_alpha
    features[4] = vix / 100.0
    features[5] = sentiment / 100.0
    features[6] = cvd_trend
    features[7] = vol_structure
    features[8] = np.sin(2 * np.pi * hour / 24.0)
    features[9] = np.cos(2 * np.pi * hour / 24.0)
    features[10] = np.sin(2 * np.pi * hour / 24.0 * 2)
    features[11] = np.cos(2 * np.pi * hour / 24.0 * 2)
    features[12] = float(vix > 25)
    features[13] = float(vix > 35)
    return features


def _multiclass_roc_auc(y_true: np.ndarray, y_prob: np.ndarray, n_classes: int) -> float:
    try:
        from sklearn.metrics import roc_auc_score

        return float(roc_auc_score(y_true, y_prob, multi_class="ovo"))
    except ImportError:
        return 0.5


def should_retrain(
    trade_logs: list[dict],
    stale_accuracy: float = 0.55,
    min_trades_since_retrain: int = 500,
    regime_drift_threshold: float = 0.30,
) -> bool:
    """Determine if the regime classifier should be retrained.

    Args:
        trade_logs: Recent trade executions with regime state info.
        stale_accuracy: Accuracy below which retraining is forced.
        min_trades_since_retrain: Minimum trades before retraining considered.
        regime_drift_threshold: KL-like divergence threshold for regime
            distribution drift from the calibrated baseline.

    Returns:
        True if retraining is recommended.
    """
    n = len(trade_logs)
    if n < min_trades_since_retrain:
        logger.debug("should_retrain: insufficient trades (%d < %d)", n, min_trades_since_retrain)
        return False

    # Check for significant drift in regime exposure
    regime_counts = [0] * 6
    regime_labels = REGIME_LABELS
    for t in trade_logs:
        r = t.get("regime_state", t.get("regime", -1))
        if isinstance(r, int) and 0 <= r < len(regime_counts):
            regime_counts[r] += 1
        elif isinstance(r, str) and r in regime_labels:
            regime_counts[regime_labels.index(r)] += 1

    total = sum(regime_counts)
    if total == 0:
        return False

    observed = np.array([c / total for c in regime_counts])
    expected = np.array([0.40, 0.20, 0.15, 0.10, 0.10, 0.05])
    div = float(
        np.sum(observed * np.log(np.clip(observed / np.clip(expected, 0.001, 1), 0.001, 10)))
    )

    if div > regime_drift_threshold:
        logger.info(
            "should_retrain: regime drift detected (div=%.3f > %.3f)", div, regime_drift_threshold
        )
        return True

    # Check stale accuracy from win rate
    winners = sum(1 for t in trade_logs if float(t.get("pnl", 0)) > 0)
    win_rate = winners / n if n > 0 else 0
    if win_rate < stale_accuracy:
        logger.info("should_retrain: win rate degraded (%.3f < %.3f)", win_rate, stale_accuracy)
        return True

    logger.debug("should_retrain: no retrain needed (div=%.3f, win_rate=%.3f)", div, win_rate)
    return False
