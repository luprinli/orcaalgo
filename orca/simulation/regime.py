"""Regime-aware synthetic data generation pipeline.

Provides regime sequence generation via Markov chains, regime-conditioned
parameter mapping, and batched generation with halt/resume progress tracking.

Core components:
  - RegimeSequenceGenerator: Markov chain regime sequence generator
  - RegimeParams: per-regime calibrated parameters
  - regime_params_for_state(): map regime ID to model parameters
  - DEFAULT_TRANSITION_MATRIX: 4-state transition probabilities
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

import numpy as np

# 4-state regime model
REGIME_CALM = 0
REGIME_TRENDING = 1
REGIME_HIGH_VOL = 2
REGIME_CRISIS = 3

REGIME_NAMES: dict[int, str] = {
    REGIME_CALM: "Calm",
    REGIME_TRENDING: "Trending",
    REGIME_HIGH_VOL: "HighVol",
    REGIME_CRISIS: "Crisis",
}

# Average duration in trading days per regime
DEFAULT_AVG_DURATION: dict[int, int] = {
    REGIME_CALM: 60,
    REGIME_TRENDING: 40,
    REGIME_HIGH_VOL: 20,
    REGIME_CRISIS: 8,
}

# Transition matrix: rows = from_regime, cols = to_regime
# CANONICAL — must match internal/risk/hmm.go DefaultHMM() and orca/train/hmm.py _generate_synthetic_returns()
DEFAULT_TRANSITION_MATRIX: np.ndarray = np.array(
    [
        [0.85, 0.10, 0.04, 0.01],  # Calm
        [0.08, 0.80, 0.10, 0.02],  # Trending
        [0.03, 0.10, 0.80, 0.07],  # HighVol
        [0.01, 0.02, 0.10, 0.87],  # Crisis
    ],
    dtype=np.float64,
)

# Per-regime parameter presets (can be overridden by calibration)
DEFAULT_REGIME_PARAMS: dict[int, dict[str, Any]] = {
    REGIME_CALM: {
        "mu": 0.0002,
        "sigma": 0.008,
        "jump_intensity": 0.0,
        "jump_mean": 0.0,
        "jump_std": 0.0,
        "volume_mult": 1.0,
        "spread_mult": 1.0,
        "correlation_mult": 1.0,
        "fill_prob": 1.0,
        "trend_bias": 0.0,
    },
    REGIME_TRENDING: {
        "mu": 0.0008,
        "sigma": 0.014,
        "jump_intensity": 0.02,
        "jump_mean": 0.0,
        "jump_std": 0.01,
        "volume_mult": 1.1,
        "spread_mult": 1.0,
        "correlation_mult": 1.0,
        "fill_prob": 0.98,
        "trend_bias": 0.6,
    },
    REGIME_HIGH_VOL: {
        "mu": -0.0003,
        "sigma": 0.028,
        "jump_intensity": 0.08,
        "jump_mean": -0.005,
        "jump_std": 0.025,
        "volume_mult": 1.3,
        "spread_mult": 2.0,
        "correlation_mult": 1.5,
        "fill_prob": 0.90,
        "trend_bias": -0.3,
    },
    REGIME_CRISIS: {
        "mu": -0.0015,
        "sigma": 0.065,
        "jump_intensity": 0.25,
        "jump_mean": -0.015,
        "jump_std": 0.050,
        "volume_mult": 0.5,
        "spread_mult": 3.0,
        "correlation_mult": 2.0,
        "fill_prob": 0.70,
        "trend_bias": -0.8,
    },
}


@dataclass(frozen=True)
class RegimeParams:
    """Parameters for a single regime state."""

    regime_id: int
    name: str
    mu: float
    sigma: float
    jump_intensity: float
    jump_mean: float
    jump_std: float
    volume_mult: float
    spread_mult: float
    correlation_mult: float
    fill_prob: float
    trend_bias: float

    @property
    def annualized_vol(self) -> float:
        return self.sigma * np.sqrt(252)

    def to_dict(self) -> dict[str, Any]:
        return {
            "regime_id": self.regime_id,
            "name": self.name,
            "mu": self.mu,
            "sigma": self.sigma,
            "jump_intensity": self.jump_intensity,
            "jump_mean": self.jump_mean,
            "jump_std": self.jump_std,
            "volume_mult": self.volume_mult,
            "spread_mult": self.spread_mult,
            "correlation_mult": self.correlation_mult,
            "fill_prob": self.fill_prob,
            "trend_bias": self.trend_bias,
            "annualized_vol": self.annualized_vol,
        }


def regime_params_for_state(
    regime_id: int,
    overrides: dict[int, dict[str, Any]] | None = None,
) -> RegimeParams:
    """Get calibrated or default regime parameters."""
    if overrides and regime_id in overrides:
        params = {**DEFAULT_REGIME_PARAMS[regime_id], **overrides[regime_id]}
    else:
        params = DEFAULT_REGIME_PARAMS[regime_id]
    return RegimeParams(
        regime_id=regime_id,
        name=REGIME_NAMES.get(regime_id, "Unknown"),
        mu=params["mu"],
        sigma=params["sigma"],
        jump_intensity=params["jump_intensity"],
        jump_mean=params["jump_mean"],
        jump_std=params["jump_std"],
        volume_mult=params["volume_mult"],
        spread_mult=params["spread_mult"],
        correlation_mult=params["correlation_mult"],
        fill_prob=params["fill_prob"],
        trend_bias=params["trend_bias"],
    )


class RegimeSequenceGenerator:
    """Generate regime label sequences using a Markov chain.

    Usage:
        gen = RegimeSequenceGenerator(transition_matrix=..., seed=42)
        labels = gen.generate_sequence(n_days=1260)  # ~5 years
        gen.save_sequence(labels, Path("regime_labels.csv"))
    """

    def __init__(
        self,
        transition_matrix: np.ndarray | None = None,
        initial_regime: int = REGIME_CALM,
        seed: int | None = None,
    ):
        if transition_matrix is not None:
            self.transition_matrix = transition_matrix.copy()
        else:
            self.transition_matrix = DEFAULT_TRANSITION_MATRIX.copy()
        self._validate_transition()
        self.initial_regime = initial_regime
        self.rng = np.random.default_rng(seed)

    def _validate_transition(self) -> None:
        tm = self.transition_matrix
        if tm.shape != (4, 4):
            raise ValueError(f"Transition matrix must be 4x4, got {tm.shape}")
        for i in range(4):
            row_sum = tm[i].sum()
            if abs(row_sum - 1.0) > 0.01:
                tm[i] = tm[i] / row_sum

    def generate_sequence(
        self, n_days: int, start_date: datetime | None = None
    ) -> tuple[np.ndarray, np.ndarray]:
        """Generate a regime label sequence of length n_days.

        Returns:
            (regime_labels, timestamps) as numpy arrays.
            regime_labels: int array of length n_days, values 0-3.
            timestamps: datetime array if start_date provided, else float days.
        """
        labels = np.zeros(n_days, dtype=np.int32)
        current = self.initial_regime
        labels[0] = current

        for t in range(1, n_days):
            probs = self.transition_matrix[current]
            probs = probs / probs.sum()
            current = int(self.rng.choice(4, p=probs))
            labels[t] = current

        if start_date is not None:
            timestamps = np.array([start_date + timedelta(days=i) for i in range(n_days)])
        else:
            timestamps = np.arange(n_days, dtype=np.float64)

        return labels, timestamps

    def get_avg_durations(self, labels: np.ndarray) -> dict[int, float]:
        """Compute average regime duration from a sequence."""
        durations: dict[int, list[int]] = {0: [], 1: [], 2: [], 3: []}
        if len(labels) == 0:
            return {k: 0.0 for k in durations}
        current = labels[0]
        count = 1
        for i in range(1, len(labels)):
            if labels[i] == current:
                count += 1
            else:
                durations[int(current)].append(count)
                current = int(labels[i])
                count = 1
        durations[int(current)].append(count)
        return {k: float(np.mean(v)) if v else 0.0 for k, v in durations.items()}

    def to_dict(self) -> dict[str, Any]:
        return {
            "transition_matrix": self.transition_matrix.tolist(),
            "initial_regime": self.initial_regime,
            "avg_durations": DEFAULT_AVG_DURATION,
        }


class RegimeBatchState:
    """Tracks progress of a batch regime-aware generation job.

    Provides halt/resume capability via a flag file and progress reporting.
    """

    def __init__(self, generation_id: str, total_days: int):
        self.generation_id = generation_id
        self.total_days = total_days
        self.completed_days = 0
        self.current_regime = REGIME_CALM
        self.started_at: datetime | None = None
        self.last_update: datetime | None = None
        self.halt_file: Path | None = None
        self._halted = False

    def start(self, halt_dir: Path | None = None) -> None:
        self.started_at = datetime.now()
        self.last_update = self.started_at
        if halt_dir:
            halt_dir.mkdir(parents=True, exist_ok=True)
            self.halt_file = halt_dir / f".halt_{self.generation_id}"

    def check_halt(self) -> bool:
        if self.halt_file and self.halt_file.exists():
            self._halted = True
            self.halt_file.unlink(missing_ok=True)
            return True
        return False

    @property
    def is_halted(self) -> bool:
        return self._halted

    @property
    def progress_pct(self) -> float:
        if self.total_days == 0:
            return 100.0
        return min(100.0, self.completed_days / self.total_days * 100)

    @property
    def elapsed_seconds(self) -> float:
        if self.started_at is None:
            return 0.0
        return (datetime.now() - self.started_at).total_seconds()

    @property
    def eta_seconds(self) -> float:
        if self.completed_days == 0 or self.started_at is None:
            return float("inf")
        rate = self.completed_days / max(1.0, self.elapsed_seconds)
        remaining = self.total_days - self.completed_days
        return remaining / max(0.0001, rate)

    def advance(self, days: int = 1, regime: int = REGIME_CALM) -> None:
        self.completed_days += days
        self.current_regime = regime
        self.last_update = datetime.now()

    def progress_dict(self) -> dict[str, Any]:
        return {
            "generation_id": self.generation_id,
            "progress_pct": round(self.progress_pct, 1),
            "completed_days": self.completed_days,
            "total_days": self.total_days,
            "current_regime": REGIME_NAMES.get(self.current_regime, "Unknown"),
            "elapsed_s": round(self.elapsed_seconds, 1),
            "eta_s": round(self.eta_seconds, 1) if self.eta_seconds < 1e9 else None,
            "halted": self._halted,
            "started_at": self.started_at.isoformat() if self.started_at else None,
        }
