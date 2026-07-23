"""Enhanced multi-dimensional HMM training.

Extends the existing 1D Gaussian HMM (returns only) to use 5-dimensional
observations for better regime discrimination:

  [log_return, vol20, vol_ratio, spread_pct, cvd_divergence]

Uses hmmlearn.GaussianHMM with full/diag covariance. Exports parameters
in the same format as the existing HMM training for backward compatibility.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np

logger = logging.getLogger("orca.ml.train.hmm_enhanced")


@dataclass(frozen=True)
class EnhancedHMMParams:
    n_states: int = 4
    n_dimensions: int = 5
    state_labels: list[str] = field(default_factory=lambda: [
        "CALM", "TRENDING", "HIGH_VOL", "CRISIS",
    ])
    transition: list[list[float]] = field(default_factory=list)
    initial_probs: list[float] = field(default_factory=list)
    emission_means: list[list[float]] = field(default_factory=list)
    emission_covars: list[list[float]] = field(default_factory=list)

    def to_json(self, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        data = {
            "n_states": self.n_states,
            "n_dimensions": self.n_dimensions,
            "state_labels": self.state_labels,
            "transition": self.transition,
            "initial_probs": self.initial_probs,
            "emission_means": self.emission_means,
            "emission_covars": self.emission_covars,
        }
        path.write_text(json.dumps(data, indent=2))


def train_enhanced_hmm(
    data: np.ndarray,
    n_states: int = 4,
    n_iter: int = 1000,
    covariance_type: str = "diag",
    seed: int = 42,
) -> EnhancedHMMParams:
    """Train a multi-dimensional Gaussian HMM.

    Args:
        data: (n_samples, n_dimensions) array of observations.
        n_states: Number of hidden states.
        n_iter: EM iterations.
        covariance_type: "diag" or "full".
        seed: Random seed.

    Returns:
        EnhancedHMMParams with trained parameters.
    """
    from hmmlearn.hmm import GaussianHMM

    data = np.asarray(data, dtype=np.float64)
    if data.ndim == 1:
        data = data.reshape(-1, 1)

    n_dims = data.shape[1]

    model = GaussianHMM(
        n_components=n_states,
        covariance_type=covariance_type,
        n_iter=n_iter,
        random_state=seed,
    )
    model.fit(data)

    order = np.argsort(model.means_[:, 0])
    means_sorted = model.means_[order]
    covars_sorted = model.covars_[order]
    transmat_sorted = model.transmat_[order][:, order]
    startprob_sorted = model.startprob_[order]

    labels = ["CALM", "TRENDING", "HIGH_VOL", "CRISIS"]

    emission_means = means_sorted.tolist()
    if covariance_type == "diag":
        emission_covars = covars_sorted.tolist()
    else:
        emission_covars = [c.tolist() for c in covars_sorted]

    params = EnhancedHMMParams(
        n_states=n_states,
        n_dimensions=n_dims,
        state_labels=labels[:n_states],
        transition=transmat_sorted.tolist(),
        initial_probs=startprob_sorted.tolist(),
        emission_means=emission_means,
        emission_covars=emission_covars,
    )

    logger.info(
        "trained %d-state %d-dim HMM: %d samples",
        n_states, n_dims, len(data),
    )
    return params


def export_enhanced_params_json(params: EnhancedHMMParams, path: str | Path) -> None:
    params.to_json(path)
    logger.info("exported enhanced HMM params to %s", path)


def build_multi_dim_observations(
    returns: np.ndarray,
    vols: np.ndarray,
    vol_ratios: np.ndarray,
    spread_pcts: np.ndarray,
    cvd_divergences: np.ndarray,
) -> np.ndarray:
    """Build multi-dimensional observation matrix for HMM training.

    Args:
        returns: Log returns, shape (n,).
        vols: Rolling volatilities, shape (n,).
        vol_ratios: Volume ratios, shape (n,).
        spread_pcts: Spread percentages, shape (n,).
        cvd_divergences: CVD divergence values, shape (n,).

    Returns:
        Observations matrix, shape (n, 5).
    """
    min_len = min(len(returns), len(vols), len(vol_ratios), len(spread_pcts), len(cvd_divergences))
    data = np.column_stack([
        returns[:min_len],
        vols[:min_len],
        vol_ratios[:min_len],
        spread_pcts[:min_len],
        cvd_divergences[:min_len],
    ])
    return data
