"""Purged walk-forward cross-validation for time-series data.

Implements the Lopez de Prado (2018) purged K-fold CV method to prevent
look-ahead bias in financial ML. Key differences from standard CV:

1. Purge: Remove training samples whose labels overlap with test samples.
2. Embargo: Remove training samples immediately after test boundary.
3. Temporal ordering: Splits are chronological, never randomized.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import datetime

import numpy as np

from orca.ml.config import CV_EMBARGO_PCT, CV_N_SPLITS

logger = logging.getLogger("orca.ml.purge_cv")


@dataclass
class PurgedFold:
    train_indices: np.ndarray
    test_indices: np.ndarray
    train_start: datetime | None = None
    train_end: datetime | None = None
    test_start: datetime | None = None
    test_end: datetime | None = None


@dataclass
class PurgedKFold:
    """Purged K-Fold cross-validator for time-series data.

    Splits data into K folds with purge and embargo to prevent information leakage.

    Args:
        n_splits: Number of folds.
        embargo_pct: Fraction of samples to embargo after test boundary.
    """
    n_splits: int = CV_N_SPLITS
    embargo_pct: float = CV_EMBARGO_PCT

    def split(
        self,
        X: np.ndarray,
        y: np.ndarray | None = None,
        groups: np.ndarray | None = None,
        timestamps: list[datetime] | None = None,
        t1: np.ndarray | None = None,
    ) -> list[PurgedFold]:
        """Generate purged train/test splits.

        Args:
            X: Feature matrix, shape (n_samples, n_features).
            y: Target labels (optional).
            groups: Group labels (optional) — samples in the same group
                    are kept together.
            timestamps: Timestamp per sample for temporal ordering.
            t1: Optional array of label end-time bar indices per sample.
                When provided, training samples whose label horizon
                (t1[i]) extends into the test window are purged from
                the training set. This prevents label overlap leakage.

        Returns:
            List of PurgedFold objects.
        """
        n_samples = len(X)
        if n_samples < self.n_splits * 2:
            raise ValueError(
                f"Too few samples ({n_samples}) for {self.n_splits} splits"
            )

        # Determine indices in temporal order
        if timestamps is not None:
            indices = np.array(sorted(range(n_samples), key=lambda i: timestamps[i]))
        else:
            indices = np.arange(n_samples)

        split_size = n_samples // self.n_splits
        embargo_samples = max(1, int(split_size * self.embargo_pct))

        folds: list[PurgedFold] = []

        for k in range(self.n_splits):
            test_start = k * split_size
            test_end = min((k + 1) * split_size, n_samples)
            test_idx = indices[test_start:test_end]

            # Train indices: all samples before test_start, minus embargo
            train_end_idx = max(0, test_start - embargo_samples)

            # Purge: remove training samples whose label horizon (t1)
            # extends into the test window
            if t1 is not None:
                purge_mask = np.zeros(n_samples, dtype=bool)
                for i in range(train_end_idx):
                    orig_i = indices[i]
                    if orig_i < len(t1) and int(t1[orig_i]) >= test_start:
                        purge_mask[i] = True
                train_indices_raw = indices[:train_end_idx]
                train_idx = train_indices_raw[~purge_mask[:train_end_idx]]
            else:
                train_idx = indices[:train_end_idx]

            if len(train_idx) == 0:
                logger.warning("Fold %d: empty training set, skipping", k)
                continue

            fold = PurgedFold(
                train_indices=train_idx,
                test_indices=test_idx,
                train_start=timestamps[train_idx[0]] if timestamps is not None else None,
                train_end=timestamps[train_idx[-1]] if timestamps is not None else None,
                test_start=timestamps[test_idx[0]] if timestamps is not None else None,
                test_end=timestamps[test_idx[-1]] if timestamps is not None else None,
            )
            folds.append(fold)

            logger.debug(
                "Fold %d: train=%d samples test=%d samples embargo=%d purge=%s",
                k, len(train_idx), len(test_idx), embargo_samples,
                "t1" if t1 is not None else "none",
            )

        return folds

    def split_with_groups(
        self,
        X: np.ndarray,
        timestamps: list[datetime],
        group_ids: np.ndarray,
        y: np.ndarray | None = None,
    ) -> list[PurgedFold]:
        """Split with group constraint — samples in same group stay together.

        This prevents, for example, splitting a single trade's lifecycle
        across train and test sets.
        """
        n_samples = len(X)
        unique_groups = np.unique(group_ids)

        # Map each group to its earliest timestamp for ordering
        group_times: dict[int, datetime] = {}
        for g in unique_groups:
            group_mask = group_ids == g
            group_ts = [timestamps[i] for i in range(n_samples) if group_mask[i]]
            if group_ts:
                group_times[g] = min(group_ts)

        sorted_groups = sorted(unique_groups, key=lambda g: group_times.get(g, datetime.max))
        n_groups = len(sorted_groups)
        split_size = max(1, n_groups // self.n_splits)
        embargo_groups = max(1, int(split_size * self.embargo_pct))

        folds: list[PurgedFold] = []

        for k in range(self.n_splits):
            test_start = k * split_size
            test_end = min((k + 1) * split_size, n_groups)
            test_groups = set(sorted_groups[test_start:test_end])

            train_end_group_idx = max(0, test_start - embargo_groups)
            train_groups = set(sorted_groups[:train_end_group_idx])

            train_idx = np.array([i for i in range(n_samples) if group_ids[i] in train_groups])
            test_idx = np.array([i for i in range(n_samples) if group_ids[i] in test_groups])

            if len(train_idx) == 0 or len(test_idx) == 0:
                logger.warning("Fold %d: empty set, skipping", k)
                continue

            folds.append(PurgedFold(
                train_indices=train_idx,
                test_indices=test_idx,
                train_start=timestamps[train_idx[0]] if timestamps is not None else None,
                test_start=timestamps[test_idx[0]] if timestamps is not None else None,
            ))

        return folds


def purged_cross_val_score(
    model,
    X: np.ndarray,
    y: np.ndarray,
    timestamps: list[datetime],
    n_splits: int = CV_N_SPLITS,
    embargo_pct: float = CV_EMBARGO_PCT,
    scoring: str = "accuracy",
) -> tuple[list[float], list[PurgedFold]]:
    """Evaluate a model using purged walk-forward cross-validation.

    Args:
        model: Any object with fit(X, y) and predict(X) methods.
        X: Feature matrix.
        y: Target labels.
        timestamps: Timestamp per sample.
        n_splits: Number of folds.
        embargo_pct: Embargo fraction.
        scoring: "accuracy", "brier", or custom.

    Returns:
        (scores, folds) — list of per-fold scores and fold definitions.
    """
    cv = PurgedKFold(n_splits=n_splits, embargo_pct=embargo_pct)
    folds = cv.split(X, y, timestamps=timestamps)

    scores: list[float] = []

    for fold in folds:
        X_train = X[fold.train_indices]
        y_train = y[fold.train_indices]
        X_test = X[fold.test_indices]
        y_test = y[fold.test_indices]

        model_clone = model.__class__(**model.get_params())
        model_clone.fit(X_train, y_train)

        if scoring == "brier":
            y_prob = model_clone.predict_proba(X_test)[:, 1]
            score = 1.0 - float(np.mean((y_prob - y_test) ** 2))
        elif scoring == "accuracy":
            y_pred = model_clone.predict(X_test)
            score = float(np.mean(y_pred == y_test))
        else:
            y_pred = model_clone.predict(X_test)
            score = float(np.mean(y_pred == y_test))

        scores.append(score)

    if len(scores) > 0:
        logger.info(
            "purged CV: %d folds, scores=%s, mean=%.4f, std=%.4f",
            len(folds),
            [round(s, 4) for s in scores],
            float(np.mean(scores)),
            float(np.std(scores)),
        )
    else:
        logger.warning("purged CV: no valid folds produced scores")

    return scores, folds
