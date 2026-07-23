"""Model evaluation utilities for ML model quality gates.

Evaluates trained models using:
  - Brier score (calibration quality)
  - Murphy decomposition (reliability, resolution, uncertainty)
  - ROC-AUC (discrimination)
  - Profit curve (cumulative PnL of gated vs ungated signals)
"""

from __future__ import annotations

import logging
from dataclasses import dataclass

import numpy as np

from orca.math.brier import brier_score as _canonical_brier
from orca.math.brier import murphy_decomposition as _canonical_murphy
from orca.ml.config import MIN_SAMPLES_PER_BIN

logger = logging.getLogger("orca.ml.train.evaluate")


@dataclass(frozen=True)
class MurphyDecomposition:
    brier: float
    reliability: float
    resolution: float
    uncertainty: float
    n_bins: int
    bins_sufficient: bool


@dataclass(frozen=True)
class ModelEvaluation:
    brier_score: float
    roc_auc: float
    accuracy: float
    murphy: MurphyDecomposition
    precision: float
    recall: float
    f1_score: float
    passed_gates: bool
    gate_results: dict


def brier_score(y_true: np.ndarray, y_prob: np.ndarray) -> float:
    """NumPy convenience wrapper around canonical `orca.math.brier.brier_score`."""
    return _canonical_brier(y_prob.tolist(), y_true.astype(int).tolist())


def murphy_decomposition(
    y_true: np.ndarray,
    y_prob: np.ndarray,
    n_bins: int = 10,
    min_per_bin: int = MIN_SAMPLES_PER_BIN,
) -> MurphyDecomposition:
    result = _canonical_murphy(y_prob.tolist(), y_true.astype(int).tolist(), n_bins)
    return MurphyDecomposition(
        brier=result.brier,
        reliability=result.reliability,
        resolution=result.resolution,
        uncertainty=result.uncertainty,
        n_bins=n_bins,
        bins_sufficient=len([b for b in result.bin_stats if b.count > 0]) >= 3,
    )


def evaluate_model(
    y_true: np.ndarray,
    y_prob: np.ndarray,
    threshold: float = 0.55,
    brier_max: float = 0.20,
    roc_auc_min: float = 0.65,
) -> ModelEvaluation:
    y_pred = (y_prob >= threshold).astype(int)

    brier = brier_score(y_true, y_prob)
    murphy = murphy_decomposition(y_true, y_prob)
    roc_auc = _compute_roc_auc(y_true, y_prob)
    accuracy = float(np.mean(y_pred == y_true))

    tp = int(np.sum((y_pred == 1) & (y_true == 1)))
    fp = int(np.sum((y_pred == 1) & (y_true == 0)))
    fn = int(np.sum((y_pred == 0) & (y_true == 1)))

    precision = tp / max(tp + fp, 1)
    recall = tp / max(tp + fn, 1)
    f1 = 2 * precision * recall / max(precision + recall, 1e-12)

    gate_results = {
        "brier": brier < brier_max,
        "roc_auc": roc_auc > roc_auc_min,
        "reliability": murphy.reliability < 0.10,
    }
    passed = all(gate_results.values())

    return ModelEvaluation(
        brier_score=brier,
        roc_auc=roc_auc,
        accuracy=accuracy,
        murphy=murphy,
        precision=precision,
        recall=recall,
        f1_score=f1,
        passed_gates=passed,
        gate_results=gate_results,
    )


def profit_curve(
    y_true: np.ndarray,
    y_prob: np.ndarray,
    trade_pnls: np.ndarray | None = None,
    n_thresholds: int = 50,
) -> dict[str, np.ndarray]:
    thresholds = np.linspace(0.3, 0.9, n_thresholds)
    n_signals = np.zeros(n_thresholds, dtype=int)
    n_accepted = np.zeros(n_thresholds, dtype=int)
    n_wins = np.zeros(n_thresholds, dtype=int)
    cumulative_pnl = np.zeros(n_thresholds)

    for i, thresh in enumerate(thresholds):
        accepted = y_prob >= thresh
        n_signals[i] = len(y_true)
        n_accepted[i] = int(np.sum(accepted))
        if n_accepted[i] > 0:
            n_wins[i] = int(np.sum(y_true[accepted]))
        if trade_pnls is not None:
            cumulative_pnl[i] = float(np.sum(trade_pnls[accepted]))

    return {
        "thresholds": thresholds,
        "n_signals": n_signals,
        "n_accepted": n_accepted,
        "n_wins": n_wins,
        "win_rate": n_wins / np.maximum(n_accepted, 1),
        "rejection_rate": 1.0 - n_accepted / n_signals,
        "cumulative_pnl": cumulative_pnl,
    }


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
