"""Data and concept drift detection for ML model monitoring.

Uses Population Stability Index (PSI) to detect when the feature distribution
in production diverges from the training distribution. PSI is the primary
signal for proactive retraining.

PSI thresholds:
  < 0.10: No significant drift
  0.10-0.20: Moderate drift - alert, no immediate action
  > 0.20: Significant drift — trigger retraining
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from enum import Enum, auto

import numpy as np

logger = logging.getLogger("orca.ml.drift_detection")


class DriftStatus(Enum):
    NO_DRIFT = auto()
    MODERATE_DRIFT = auto()
    SIGNIFICANT_DRIFT = auto()


@dataclass(frozen=True)
class DriftReport:
    status: DriftStatus
    psi_total: float
    per_feature_psi: dict[str, float] = field(default_factory=dict)
    triggers: list[str] = field(default_factory=list)


def compute_psi(
    reference: np.ndarray,
    recent: np.ndarray,
    bins: int = 10,
    epsilon: float = 1e-6,
) -> float:
    """Compute Population Stability Index between reference and recent distributions.

    Args:
        reference: (n_samples, n_features) — training distribution.
        recent: (n_samples, n_features) — recent production distribution.
        bins: Number of bins for discretization.
        epsilon: Small value to prevent division by zero.

    Returns:
        Total PSI value across all features.
    """
    if reference.ndim == 1:
        reference = reference.reshape(-1, 1)
    if recent.ndim == 1:
        recent = recent.reshape(-1, 1)

    n_features = reference.shape[1]
    psi_total = 0.0

    for i in range(n_features):
        ref_col = reference[:, i]
        rec_col = recent[:, i]

        col_min = min(ref_col.min(), rec_col.min())
        col_max = max(ref_col.max(), rec_col.max())

        if col_max - col_min < epsilon:
            continue

        bin_edges = np.linspace(col_min, col_max, bins + 1)

        ref_hist, _ = np.histogram(ref_col, bins=bin_edges)
        rec_hist, _ = np.histogram(rec_col, bins=bin_edges)

        ref_hist = ref_hist.astype(np.float64) / max(len(ref_col), 1)
        rec_hist = rec_hist.astype(np.float64) / max(len(rec_col), 1)

        ref_hist = np.clip(ref_hist, epsilon, None)
        rec_hist = np.clip(rec_hist, epsilon, None)

        psi_i = np.sum((rec_hist - ref_hist) * np.log(rec_hist / ref_hist))
        psi_total += max(psi_i, 0.0)

    return psi_total


def compute_per_feature_psi(
    reference: np.ndarray,
    recent: np.ndarray,
    feature_names: list[str] | None = None,
    bins: int = 10,
    epsilon: float = 1e-6,
) -> dict[str, float]:
    """Compute PSI per feature for detailed drift diagnosis.

    Returns:
        Dict of {feature_name: psi_value}.
    """
    if reference.ndim == 1:
        reference = reference.reshape(-1, 1)
    if recent.ndim == 1:
        recent = recent.reshape(-1, 1)

    n_features = reference.shape[1]
    if feature_names is None:
        feature_names = [f"f{i}" for i in range(n_features)]

    result: dict[str, float] = {}
    for i in range(n_features):
        psi = compute_psi(reference[:, i:i + 1], recent[:, i:i + 1], bins, epsilon)
        name = feature_names[i] if i < len(feature_names) else f"f{i}"
        result[name] = float(psi)

    return result


def classify_drift(
    reference: np.ndarray,
    recent: np.ndarray,
    feature_names: list[str] | None = None,
    significant_threshold: float = 0.20,
    moderate_threshold: float = 0.10,
    vix_ratio: float | None = None,
) -> DriftReport:
    """Classify whether drift is significant enough to trigger retraining.

    Also checks VIX ratio as an additional signal.

    Args:
        reference: Training distribution features.
        recent: Recent production features.
        feature_names: Optional feature names for per-feature breakdown.
        significant_threshold: PSI threshold for retraining trigger.
        moderate_threshold: PSI threshold for alert only.
        vix_ratio: recent_20d_vix_avg / training_vix_avg — optional extra signal.

    Returns:
        DriftReport with status, psi_total, per-feature breakdown, and triggers.
    """
    psi_total = compute_psi(reference, recent)
    per_feature = compute_per_feature_psi(reference, recent, feature_names)

    triggers: list[str] = []

    if psi_total > significant_threshold:
        status = DriftStatus.SIGNIFICANT_DRIFT
        triggers.append(f"psi={psi_total:.3f} > {significant_threshold}")
    elif psi_total > moderate_threshold:
        status = DriftStatus.MODERATE_DRIFT
        triggers.append(f"psi={psi_total:.3f} > {moderate_threshold}")
    else:
        status = DriftStatus.NO_DRIFT

    if vix_ratio is not None and vix_ratio > 2.0:
        triggers.append(f"vix_ratio={vix_ratio:.2f} > 2.0")
        if status == DriftStatus.NO_DRIFT:
            status = DriftStatus.MODERATE_DRIFT

    logger.info(
        "drift check: status=%s psi=%.3f triggers=%s",
        status.name, psi_total, triggers,
    )

    return DriftReport(
        status=status,
        psi_total=float(psi_total),
        per_feature_psi=per_feature,
        triggers=triggers,
    )


def should_retrain(
    recent_features: np.ndarray,
    training_features: np.ndarray,
    current_win_rate: float,
    baseline_win_rate: float,
    current_brier: float | None = None,
    training_brier: float | None = None,
    vix_ratio: float | None = None,
    brier_degradation_ratio: float = 1.10,
    win_rate_degradation_pp: float = 0.05,
) -> tuple[bool, list[str], DriftReport]:
    """Determine whether a model should be retrained.

    Aggregates PSI, win rate degradation, and Brier score degradation.

    Returns:
        (should_retrain, triggers, drift_report)
    """
    drift = classify_drift(training_features, recent_features, vix_ratio=vix_ratio)
    triggers: list[str] = []

    if drift.status == DriftStatus.SIGNIFICANT_DRIFT:
        triggers.extend(drift.triggers)

    if current_win_rate < baseline_win_rate - win_rate_degradation_pp:
        triggers.append(
            f"win_rate_degradation: {current_win_rate:.3f} < baseline {baseline_win_rate:.3f}"
        )

    if current_brier is not None and training_brier is not None:
        if current_brier > training_brier * brier_degradation_ratio:
            triggers.append(
                f"brier_degradation: {current_brier:.4f} > "
                f"{training_brier * brier_degradation_ratio:.4f}"
            )

    should = len(triggers) > 0
    if should:
        logger.warning("retraining triggered: %s", triggers)

    return should, triggers, drift
