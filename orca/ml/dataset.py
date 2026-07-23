"""Feature dataset builder for offline model training.

Loads trade and candle data (via Go backend subprocess or from parquet files),
computes features using the canonical feature specification, applies triple-barrier
labels, and exports training-ready datasets.
"""

from __future__ import annotations

import json
import logging
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from pathlib import Path

import numpy as np

from orca.ml.barriers import (
    BarrierConfig,
    BarrierResult,
    batch_triple_barrier_labels,
    label_to_binary,
)
from orca.ml.config import (
    FEATURE_NAMES,
    MIN_SAMPLES_GLOBAL,
)
from orca.ml.feature_selection import validate_feature_vector
from orca.ml.features import (
    compute_full_feature_vector,
)

logger = logging.getLogger("orca.ml.dataset")


@dataclass(frozen=True)
class TrainingSample:
    """A single labeled training sample."""
    symbol: str
    timestamp: datetime
    feature_vector: np.ndarray  # shape (21,)
    label: int                  # binary: 1 = win, 0 = loss
    label_detail: BarrierResult | None = None
    strategy_type: str = ""
    regime: int = 0

    def to_dict(self) -> dict:
        d = asdict(self)
        d["feature_vector"] = self.feature_vector.tolist()
        if self.label_detail:
            d["label_detail"] = asdict(self.label_detail)
        d["timestamp"] = self.timestamp.isoformat()
        return d


@dataclass
class FeatureDataset:
    """Container for a complete training dataset."""
    samples: list[TrainingSample] = field(default_factory=list)
    feature_names: list[str] = field(default_factory=lambda: FEATURE_NAMES.copy())
    metadata: dict = field(default_factory=dict)

    @property
    def n_samples(self) -> int:
        return len(self.samples)

    @property
    def win_rate(self) -> float:
        if not self.samples:
            return 0.0
        wins = sum(1 for s in self.samples if s.label == 1)
        return wins / len(self.samples)

    def to_numpy(self) -> tuple[np.ndarray, np.ndarray]:
        """Return (X, y) as numpy arrays."""
        X = np.array([s.feature_vector for s in self.samples], dtype=np.float64)
        y = np.array([s.label for s in self.samples], dtype=np.int32)
        return X, y

    def to_feature_dict(self) -> dict[str, np.ndarray]:
        """Return features as a dict of named columns."""
        X, y = self.to_numpy()
        d = {name: X[:, i] for i, name in enumerate(self.feature_names)}
        d["target"] = y
        return d

    def validate(self) -> tuple[bool, list[str]]:
        """Validate the dataset for training readiness.

        Returns:
            (is_valid, issues) — issues is empty if valid.
        """
        issues: list[str] = []

        if self.n_samples < MIN_SAMPLES_GLOBAL:
            issues.append(f"insufficient samples: {self.n_samples} < {MIN_SAMPLES_GLOBAL}")

        for i, sample in enumerate(self.samples):
            if not validate_feature_vector(sample.feature_vector):
                issues.append(f"invalid feature vector at sample {i}")

        if self.n_samples > 0:
            pos = sum(1 for s in self.samples if s.label == 1)
            neg = len(self.samples) - pos
            if pos == 0 or neg == 0:
                issues.append("single-class labels — need both positive and negative samples")

        if issues:
            logger.warning("dataset validation failed: %s", issues)
            return False, issues

        return True, []


def build_dataset_from_trade_logs(
    trades: list[dict],
    candle_data: dict[str, dict[str, np.ndarray]],
    hmm_data: dict[str, list] | None = None,
    barrier_config: BarrierConfig | None = None,
    min_bars: int = 40,
) -> FeatureDataset:
    """Build a training dataset from trade logs and candle data.

    Args:
        trades: List of trade dicts with keys: symbol, side, entry_price, placed_at,
                pnl, entry_time (ISO string or datetime).
        candle_data: Dict of {symbol: {"open": arr, "high": arr, "low": arr,
                      "close": arr, "volume": arr, "timestamp": [...]}}.
        hmm_data: Optional dict of {symbol: [(timestamp, alpha, confidence), ...]}.
        barrier_config: Triple-barrier config.
        min_bars: Minimum bars needed after entry for labeling.

    Returns:
        FeatureDataset ready for training.
    """
    dataset = FeatureDataset(
        metadata={
            "created_at": datetime.now(UTC).isoformat(),
            "n_trades_loaded": len(trades),
            "barrier_config": asdict(barrier_config or BarrierConfig()),
        }
    )

    skipped_no_data = 0
    skipped_no_candle = 0
    skipped_insufficient_bars = 0

    for trade in trades:
        symbol = trade.get("symbol", "")
        if symbol not in candle_data:
            skipped_no_data += 1
            continue

        symbol_data = candle_data[symbol]
        closes = symbol_data.get("close")
        if closes is None or len(closes) < min_bars:
            skipped_no_candle += 1
            continue

        entry_time_str = trade.get("entry_time") or trade.get("placed_at", "")
        try:
            if isinstance(entry_time_str, str):
                entry_time = datetime.fromisoformat(entry_time_str.replace("Z", "+00:00"))
            elif isinstance(entry_time_str, datetime):
                entry_time = entry_time_str
            else:
                skipped_no_candle += 1
                continue
        except (ValueError, TypeError):
            skipped_no_candle += 1
            continue

        # Find entry index in candle timestamps
        timestamps = symbol_data.get("timestamp", [])
        entry_idx = -1
        for i, ts in enumerate(timestamps):
            ts_dt = (
                ts if isinstance(ts, datetime)
                else datetime.fromisoformat(str(ts).replace("Z", "+00:00"))
            )
            if ts_dt >= entry_time:
                entry_idx = i
                break

        if entry_idx < 0 or entry_idx + barrier_config.time_horizon + 1 >= len(closes):
            skipped_insufficient_bars += 1
            continue

        # Triple-barrier label from price path
        prices = closes
        entry_price = closes[entry_idx]
        if entry_price <= 0:
            skipped_insufficient_bars += 1
            continue

        barrier_result = None
        try:
            barrier_result = next(iter(batch_triple_barrier_labels(
                np.array([entry_price]),
                np.array([entry_idx]),
                prices,
                barrier_config or BarrierConfig(),
            )))
        except Exception:
            skipped_insufficient_bars += 1
            continue

        # Compute features at entry bar
        try:
            fv = compute_full_feature_vector(
                closes=closes[:entry_idx + 1],
                highs=symbol_data.get("high", closes)[:entry_idx + 1],
                lows=symbol_data.get("low", closes)[:entry_idx + 1],
                volumes=symbol_data.get("volume", np.ones(entry_idx + 1))[:entry_idx + 1],
                ts=entry_time,
                signal_type=0,
                signal_strength=0.0,
            )
        except (ValueError, IndexError):
            skipped_insufficient_bars += 1
            continue

        binary_label = label_to_binary(barrier_result.label)
        regime = trade.get("HMMRegime", trade.get("hmm_regime", 0))

        sample = TrainingSample(
            symbol=symbol,
            timestamp=entry_time,
            feature_vector=fv,
            label=binary_label,
            label_detail=barrier_result,
            strategy_type=trade.get("StrategyID", trade.get("strategy_id", "")),
            regime=regime,
        )
        dataset.samples.append(sample)

    dataset.metadata["skipped"] = {
        "no_data": skipped_no_data,
        "no_candle": skipped_no_candle,
        "insufficient_bars": skipped_insufficient_bars,
    }
    dataset.metadata["n_samples"] = dataset.n_samples

    logger.info(
        "dataset built: %d samples, %d skipped (win_rate=%.2f)",
        dataset.n_samples,
        skipped_no_data + skipped_no_candle + skipped_insufficient_bars,
        dataset.win_rate,
    )

    return dataset


def split_temporal(
    dataset: FeatureDataset,
    train_ratio: float = 0.70,
    val_ratio: float = 0.15,
) -> tuple[FeatureDataset, FeatureDataset, FeatureDataset]:
    """Split dataset temporally (not randomly) into train/val/test.

    Maintains temporal ordering — train is earliest, test is latest.

    Returns:
        (train_dataset, val_dataset, test_dataset)
    """
    dataset.samples.sort(key=lambda s: s.timestamp)
    n = len(dataset.samples)

    train_end = int(n * train_ratio)
    val_end = int(n * (train_ratio + val_ratio))

    train = FeatureDataset(
        samples=dataset.samples[:train_end],
        feature_names=dataset.feature_names,
        metadata={**dataset.metadata, "split": "train"},
    )
    val = FeatureDataset(
        samples=dataset.samples[train_end:val_end],
        feature_names=dataset.feature_names,
        metadata={**dataset.metadata, "split": "val"},
    )
    test = FeatureDataset(
        samples=dataset.samples[val_end:],
        feature_names=dataset.feature_names,
        metadata={**dataset.metadata, "split": "test"},
    )

    logger.info(
        "temporal split: train=%d val=%d test=%d",
        train.n_samples, val.n_samples, test.n_samples,
    )
    return train, val, test


def save_dataset(dataset: FeatureDataset, path: str | Path) -> None:
    """Save a dataset to a JSON file."""
    path = Path(path)
    data = {
        "samples": [s.to_dict() for s in dataset.samples],
        "feature_names": dataset.feature_names,
        "metadata": dataset.metadata,
    }
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w") as f:
        json.dump(data, f, default=str)
    logger.info("dataset saved to %s (%d samples)", path, dataset.n_samples)


def load_dataset(path: str | Path) -> FeatureDataset:
    """Load a dataset from a JSON file."""
    path = Path(path)
    with open(path) as f:
        data = json.load(f)

    samples = []
    for s in data.get("samples", []):
        sample = TrainingSample(
            symbol=s["symbol"],
            timestamp=datetime.fromisoformat(s["timestamp"]),
            feature_vector=np.array(s["feature_vector"], dtype=np.float64),
            label=s["label"],
            label_detail=None,
            strategy_type=s.get("strategy_type", ""),
            regime=s.get("regime", 0),
        )
        samples.append(sample)

    return FeatureDataset(
        samples=samples,
        feature_names=data.get("feature_names", FEATURE_NAMES),
        metadata=data.get("metadata", {}),
    )
