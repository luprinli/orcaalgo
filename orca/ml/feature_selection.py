"""Feature selection and validation pipeline.

Validates candidate features before model training to prevent noisy or
redundant features from entering the model. Three-stage validation:

1. Correlation analysis — remove features with |r| > 0.9
2. Mutual information scoring — rank features by predictive power
3. SHAP summary — verify economic intuition (optional, requires xgboost + shap)
"""

from __future__ import annotations

import logging

import numpy as np
from scipy.stats import entropy

from orca.ml.config import FEATURE_NAMES

logger = logging.getLogger("orca.ml.feature_selection")

FEATURE_CORRELATION_THRESHOLD = 0.9


def correlation_filter(
    features: np.ndarray,
    threshold: float = FEATURE_CORRELATION_THRESHOLD,
) -> tuple[list[int], list[tuple[int, int, float]]]:
    """Remove features with pairwise correlation above threshold.

    Returns:
        keep_indices: List of feature indices to keep.
        redundant_pairs: List of (i, j, correlation) for flagging.
    """
    n_features = features.shape[1]
    corr = np.corrcoef(features, rowvar=False)
    redundant = set()
    pairs: list[tuple[int, int, float]] = []

    for i in range(n_features):
        if i in redundant:
            continue
        for j in range(i + 1, n_features):
            if j in redundant:
                continue
            if abs(corr[i, j]) > threshold:
                redundant.add(j)
                pairs.append((i, j, float(corr[i, j])))

    keep = [i for i in range(n_features) if i not in redundant]
    logger.info(
        "correlation_filter: %d features removed, %d kept (threshold=%.2f)",
        len(redundant),
        len(keep),
        threshold,
    )
    if pairs:
        for i, j, c in pairs:
            logger.info(
                "  removed feature %d (%s) — correlated with %d (%s), r=%.3f",
                j,
                FEATURE_NAMES[j] if j < len(FEATURE_NAMES) else f"f{j}",
                i,
                FEATURE_NAMES[i] if i < len(FEATURE_NAMES) else f"f{i}",
                c,
            )
    return keep, pairs


def mutual_information_score(
    features: np.ndarray,
    target: np.ndarray,
    n_bins: int = 20,
) -> np.ndarray:
    """Compute mutual information between each feature and the target.

    Uses histogram-based discretization (no sklearn dependency).

    Args:
        features: (n_samples, n_features)
        target: (n_samples,) — binary or continuous
        n_bins: Number of bins for discretization

    Returns:
        Array of MI scores, shape (n_features,).
    """
    n_features = features.shape[1]
    scores = np.zeros(n_features)

    # Discretize target if continuous
    if len(np.unique(target)) > 2:
        target_bins = np.digitize(
            target,
            np.percentile(target, np.linspace(0, 100, n_bins + 1)[1:-1]),
        )
    else:
        target_bins = target.astype(int)

    for i in range(n_features):
        col = features[:, i]
        if len(np.unique(col)) <= 3:
            feature_bins = col.astype(int)
        else:
            feature_bins = np.digitize(
                col,
                np.percentile(col, np.linspace(0, 100, n_bins + 1)[1:-1]),
            )

        # Joint histogram
        joint = np.zeros((n_bins, n_bins))
        for f_val, t_val in zip(feature_bins, target_bins, strict=False):
            f_idx = min(int(f_val), n_bins - 1)
            t_idx = min(int(t_val), n_bins - 1)
            joint[f_idx, t_idx] += 1

        joint /= joint.sum()
        marginal_f = joint.sum(axis=1)
        marginal_t = joint.sum(axis=0)

        # MI = H(f) + H(t) - H(f, t)
        h_f = entropy(marginal_f[marginal_f > 0])
        h_t = entropy(marginal_t[marginal_t > 0])
        h_joint = entropy(joint[joint > 0])
        scores[i] = max(h_f + h_t - h_joint, 0.0)

    return scores


def rank_features_by_mi(
    features: np.ndarray,
    target: np.ndarray,
    feature_names: list[str] | None = None,
) -> list[tuple[int, str, float]]:
    """Rank features by mutual information score.

    Returns:
        List of (index, name, mi_score) sorted descending.
    """
    scores = mutual_information_score(features, target)
    if feature_names is None:
        feature_names = FEATURE_NAMES[: features.shape[1]]

    ranked = sorted(
        [
            (i, feature_names[i] if i < len(feature_names) else f"f{i}", float(scores[i]))
            for i in range(len(scores))
        ],
        key=lambda x: x[2],
        reverse=True,
    )
    return ranked


def validate_features(
    features: np.ndarray,
    target: np.ndarray,
    feature_names: list[str] | None = None,
    correlation_threshold: float = FEATURE_CORRELATION_THRESHOLD,
    mi_bottom_pct: float = 0.10,
) -> dict:
    """Full feature validation pipeline.

    Args:
        features: (n_samples, n_features) array.
        target: (n_samples,) binary labels.
        feature_names: Optional names for reporting.
        correlation_threshold: Max allowed pairwise correlation.
        mi_bottom_pct: Fraction of features with lowest MI to flag.

    Returns:
        Dict with validation results: keep_indices, redundant_pairs, mi_ranked, flagged.
    """
    if feature_names is None:
        feature_names = [
            FEATURE_NAMES[i] if i < len(FEATURE_NAMES) else f"f{i}"
            for i in range(features.shape[1])
        ]

    keep_idx, redundant_pairs = correlation_filter(features, correlation_threshold)
    filtered = features[:, keep_idx]
    filtered_names = [feature_names[i] for i in keep_idx]

    mi_ranked = rank_features_by_mi(filtered, target, filtered_names)

    bottom_n = max(1, int(len(mi_ranked) * mi_bottom_pct))
    flagged = mi_ranked[-bottom_n:]

    logger.info(
        "feature validation complete: %d kept, %d redundant, %d low-MI flagged",
        len(keep_idx),
        len(redundant_pairs),
        len(flagged),
    )

    return {
        "keep_indices": keep_idx,
        "redundant_pairs": redundant_pairs,
        "mi_ranked": mi_ranked,
        "flagged_low_mi": flagged,
        "passed": len(keep_idx) >= 5,  # minimum 5 features to proceed
    }


def validate_feature_vector(
    fv: np.ndarray,
) -> bool:
    """Check a single feature vector for basic validity.

    Returns False if the vector contains NaN, Inf, or has wrong dimension.
    """
    if fv.shape[-1] != len(FEATURE_NAMES):
        return False
    if np.any(np.isnan(fv)):
        return False
    if np.any(np.isinf(fv)):
        return False
    return True
