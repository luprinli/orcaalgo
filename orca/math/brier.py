from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class BinStats:
    bin_start: float
    bin_end: float
    count: int
    mean_prediction: float
    hit_rate: float


@dataclass(frozen=True)
class MurphyResult:
    brier: float
    reliability: float
    resolution: float
    uncertainty: float
    bin_stats: list[BinStats]


def brier_score(predictions: list[float], outcomes: list[int]) -> float:
    if len(predictions) != len(outcomes):
        raise ValueError("Predictions and outcomes must have same length")
    if len(predictions) == 0:
        raise ValueError("Empty input")
    n = len(predictions)
    return sum((p - o) ** 2 for p, o in zip(predictions, outcomes)) / n


def murphy_decomposition(predictions: list[float], outcomes: list[int], n_bins: int = 10) -> MurphyResult:
    n = len(predictions)
    if n == 0:
        raise ValueError("Empty input")

    brier = brier_score(predictions, outcomes)
    base_rate = sum(outcomes) / n

    bin_stats: list[BinStats] = []
    reliability = 0.0
    resolution = 0.0

    for b in range(n_bins):
        low = b / n_bins
        high = (b + 1) / n_bins
        bin_preds = []
        bin_outs = []
        for p, o in zip(predictions, outcomes):
            if low <= p < high or (b == n_bins - 1 and p == high):
                bin_preds.append(p)
                bin_outs.append(o)

        count = len(bin_preds)
        if count == 0:
            bin_stats.append(BinStats(bin_start=low, bin_end=high, count=0, mean_prediction=0.0, hit_rate=0.0))
            continue

        mean_p = sum(bin_preds) / count
        hit_r = sum(bin_outs) / count
        bin_stats.append(BinStats(bin_start=low, bin_end=high, count=count, mean_prediction=mean_p, hit_rate=hit_r))

        reliability += (count / n) * (mean_p - hit_r) ** 2
        resolution += (count / n) * (hit_r - base_rate) ** 2

    uncertainty = base_rate * (1 - base_rate)

    return MurphyResult(
        brier=brier,
        reliability=reliability,
        resolution=resolution,
        uncertainty=uncertainty,
        bin_stats=bin_stats,
    )
