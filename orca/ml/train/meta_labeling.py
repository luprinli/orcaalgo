"""XGBoost meta-labeling model training pipeline.

Trains a binary classifier that predicts whether a trade signal will result
in a winning trade. The model is trained on features computed at signal time
with triple-barrier labels, using purged walk-forward cross-validation to
prevent look-ahead bias.
"""

from __future__ import annotations

import hashlib
import json
import logging
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

import numpy as np

from orca.ml.config import (
    BRIER_MAX,
    MIN_SAMPLES_GLOBAL,
    ROC_AUC_MIN,
    XGB_COLSAMPLE_BYTREE,
    XGB_EARLY_STOPPING,
    XGB_LEARNING_RATE,
    XGB_MAX_DEPTH,
    XGB_N_ESTIMATORS,
    XGB_SCALE_POS_WEIGHT,
    XGB_SUBSAMPLE,
)
from orca.ml.dataset import FeatureDataset, split_temporal
from orca.ml.purge_cv import PurgedKFold

logger = logging.getLogger("orca.ml.train.meta_labeling")

try:
    import xgboost as xgb
except ImportError:
    xgb = None
    logger.warning(
        "xgboost not installed, meta-labeling training disabled. "
        "Install with: pip install xgboost"
    )


@dataclass
class TrainingResult:
    model: object | None
    brier_score: float
    roc_auc: float
    accuracy: float
    feature_importance: dict[str, float]
    cv_scores: list[float]
    cv_mean: float
    cv_std: float
    oos_brier: float
    passed_gate: bool
    model_hash: str = ""
    params: dict = field(default_factory=dict)
    metadata: dict = field(default_factory=dict)


@dataclass
class MetaLabelingTrainer:
    n_estimators: int = XGB_N_ESTIMATORS
    max_depth: int = XGB_MAX_DEPTH
    learning_rate: float = XGB_LEARNING_RATE
    subsample: float = XGB_SUBSAMPLE
    colsample_bytree: float = XGB_COLSAMPLE_BYTREE
    scale_pos_weight: float = XGB_SCALE_POS_WEIGHT
    early_stopping_rounds: int = XGB_EARLY_STOPPING
    cv_splits: int = 5
    validation_fraction: float = 0.15
    min_samples: int = MIN_SAMPLES_GLOBAL

    def train(
        self,
        dataset: FeatureDataset,
        timestamps: list[datetime] | None = None,
        feature_indices: list[int] | None = None,
        random_state: int = 42,
    ) -> TrainingResult:
        if xgb is None:
            raise ImportError("xgboost is required for meta-labeling training")

        X, y = dataset.to_numpy()

        if feature_indices is not None:
            X = X[:, feature_indices]

        if dataset.n_samples < self.min_samples:
            raise ValueError(
                f"insufficient samples: {dataset.n_samples} < {self.min_samples}"
            )

        valid, issues = dataset.validate()
        for issue in issues:
            if "insufficient samples" not in issue:
                if not valid:
                    raise ValueError(f"dataset validation failed: {issue}")

        if timestamps is None:
            timestamps = [s.timestamp for s in dataset.samples]

        train, val, test = split_temporal(
            dataset,
            train_ratio=0.70,
            val_ratio=self.validation_fraction,
        )
        X_train, y_train = train.to_numpy()
        X_val, y_val = val.to_numpy()
        X_test, y_test = test.to_numpy()

        if feature_indices is not None:
            X_train = X_train[:, feature_indices]
            X_val = X_val[:, feature_indices]
            X_test = X_test[:, feature_indices]

        pos_count = int(np.sum(y_train))
        neg_count = len(y_train) - pos_count
        scale_weight = max(neg_count, 1) / max(pos_count, 1)
        logger.info(
            "training: train=%d val=%d test=%d pos_ratio=%.2f scale_weight=%.2f",
            len(X_train), len(X_val), len(X_test),
            pos_count / max(len(y_train), 1), scale_weight,
        )

        model = xgb.XGBClassifier(
            n_estimators=self.n_estimators,
            max_depth=self.max_depth,
            learning_rate=self.learning_rate,
            subsample=self.subsample,
            colsample_bytree=self.colsample_bytree,
            scale_pos_weight=scale_weight,
            eval_metric="logloss",
            early_stopping_rounds=self.early_stopping_rounds,
            random_state=random_state,
            verbosity=0,
        )

        model.fit(
            X_train, y_train,
            eval_set=[(X_val, y_val)],
            verbose=False,
        )

        y_prob_train = model.predict_proba(X_train)[:, 1]
        y_prob_val = model.predict_proba(X_val)[:, 1]
        y_prob_test = model.predict_proba(X_test)[:, 1]
        y_pred_test = model.predict(X_test)

        brier_train = float(np.mean((y_prob_train - y_train) ** 2))
        brier_val = float(np.mean((y_prob_val - y_val) ** 2))
        brier_test = float(np.mean((y_prob_test - y_test) ** 2))
        accuracy = float(np.mean(y_pred_test == y_test))

        roc_auc = _compute_roc_auc(y_test, y_prob_test)

        importance = {dataset.feature_names[i]: float(imp)
                      for i, imp in enumerate(model.feature_importances_)
                      if i < len(dataset.feature_names)}

        cv = PurgedKFold(n_splits=self.cv_splits)
        folds = cv.split(X, y, timestamps=timestamps)
        cv_scores: list[float] = []

        for fold in folds:
            Xf_train, yf_train = X[fold.train_indices], y[fold.train_indices]
            Xf_test, yf_test = X[fold.test_indices], y[fold.test_indices]
            if feature_indices is not None:
                Xf_train = Xf_train[:, feature_indices]
                Xf_test = Xf_test[:, feature_indices]

            fold_model = xgb.XGBClassifier(
                n_estimators=self.n_estimators,
                max_depth=self.max_depth,
                learning_rate=self.learning_rate,
                subsample=self.subsample,
                colsample_bytree=self.colsample_bytree,
                scale_pos_weight=scale_weight,
                eval_metric="logloss",
                random_state=random_state,
                verbosity=0,
            )
            fold_model.fit(Xf_train, yf_train, verbose=False)
            yf_prob = fold_model.predict_proba(Xf_test)[:, 1]
            cv_score = 1.0 - float(np.mean((yf_prob - yf_test) ** 2))
            cv_scores.append(cv_score)

        passed = brier_test < BRIER_MAX and roc_auc > ROC_AUC_MIN

        hash_input = {
            "n_estimators": self.n_estimators,
            "max_depth": self.max_depth,
            "learning_rate": self.learning_rate,
            "n_train": len(X_train),
            "n_val": len(X_val),
            "n_test": len(X_test),
            "n_features": len(dataset.feature_names),
            "feature_names": dataset.feature_names[:10] if len(dataset.feature_names) > 10 else dataset.feature_names,
        }
        hash_raw = json.dumps(hash_input, sort_keys=True)
        model_hash = hashlib.sha256(hash_raw.encode()).hexdigest()[:16]

        if passed:
            logger.info(
                "model PASSED: brier=%.4f roc_auc=%.4f accuracy=%.2f cv=%.4f+/-%.4f",
                brier_test, roc_auc, accuracy,
                float(np.mean(cv_scores)), float(np.std(cv_scores)),
            )
        else:
            logger.warning(
                "model FAILED gates: brier=%.4f (max=%.2f) roc_auc=%.4f (min=%.2f)",
                brier_test, BRIER_MAX, roc_auc, ROC_AUC_MIN,
            )

        return TrainingResult(
            model=model,
            brier_score=brier_test,
            roc_auc=roc_auc,
            accuracy=accuracy,
            feature_importance=importance,
            cv_scores=cv_scores,
            cv_mean=float(np.mean(cv_scores)),
            cv_std=float(np.std(cv_scores)),
            oos_brier=brier_test,
            passed_gate=passed,
            model_hash=model_hash,
            params={
                "n_estimators": self.n_estimators,
                "max_depth": self.max_depth,
                "learning_rate": self.learning_rate,
                "subsample": self.subsample,
                "colsample_bytree": self.colsample_bytree,
                "scale_pos_weight": scale_weight,
            },
            metadata={
                "train_samples": len(X_train),
                "val_samples": len(X_val),
                "test_samples": len(X_test),
                "train_brier": brier_train,
                "val_brier": brier_val,
            },
        )

    def save_model(self, result: TrainingResult, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        if result.model is not None:
            result.model.save_model(str(path))
            logger.info("model saved to %s", path)

    def save_results(self, result: TrainingResult, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        data = {
            "brier_score": result.brier_score,
            "roc_auc": result.roc_auc,
            "accuracy": result.accuracy,
            "cv_mean": result.cv_mean,
            "cv_std": result.cv_std,
            "oos_brier": result.oos_brier,
            "passed_gate": result.passed_gate,
            "model_hash": result.model_hash,
            "feature_importance": result.feature_importance,
            "params": result.params,
            "metadata": result.metadata,
            "cv_scores": result.cv_scores,
            "trained_at": datetime.now(UTC).isoformat(),
        }
        with open(path, "w") as f:
            json.dump(data, f, indent=2)
        logger.info("results saved to %s", path)

    def save_model_with_metadata(self, result: TrainingResult, model_path: str | Path, meta_path: str | Path | None = None) -> None:
        self.save_model(result, model_path)
        meta_path = Path(meta_path) if meta_path else Path(model_path).with_suffix(".meta.json")
        meta = {
            "model_hash": result.model_hash,
            "brier_score": result.brier_score,
            "roc_auc": result.roc_auc,
            "passed_gate": result.passed_gate,
            "n_estimators": self.n_estimators,
            "max_depth": self.max_depth,
            "trained_at": datetime.now(UTC).isoformat(),
            "cv_mean": result.cv_mean,
            "cv_std": result.cv_std,
            "metadata": result.metadata,
        }
        meta_path.parent.mkdir(parents=True, exist_ok=True)
        with open(meta_path, "w") as f:
            json.dump(meta, f, indent=2)
        logger.info("model metadata saved to %s (hash=%s)", meta_path, result.model_hash)


def load_model(path: str | Path) -> object:
    if xgb is None:
        raise ImportError("xgboost is required")
    model = xgb.XGBClassifier()
    model.load_model(str(path))
    return model


def predict(model: object, X: np.ndarray) -> np.ndarray:
    return model.predict_proba(X)[:, 1]


def _compute_roc_auc(y_true: np.ndarray, y_prob: np.ndarray) -> float:
    try:
        from sklearn.metrics import roc_auc_score
        return float(roc_auc_score(y_true, y_prob))
    except ImportError:
        desc = np.argsort(y_prob)[::-1]
        y_sorted = y_true[desc]
        n_pos = int(np.sum(y_true))
        n_neg = len(y_true) - n_pos
        if n_pos == 0 or n_neg == 0:
            return 0.5
        tpr = np.cumsum(y_sorted) / n_pos
        fpr = np.cumsum(1 - y_sorted) / n_neg
        return float(np.trapz(tpr, fpr))
