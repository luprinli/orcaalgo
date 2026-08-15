"""Hierarchical model training: global → asset-class → per-symbol.

Multi-level training strategy that learns from all symbols (global model),
then specializes per asset class, and optionally fine-tunes per symbol
when sufficient data is available.
"""

from __future__ import annotations

import logging
from pathlib import Path

from orca.ml.config import (
    MIN_SAMPLES_ASSET_CLASS,
    MIN_SAMPLES_GLOBAL,
    MIN_SAMPLES_PER_SYMBOL,
    REGISTRY_PATH,
)

logger = logging.getLogger("orca.ml.train.hierarchical")

ASSET_CLASS_SYMBOLS: dict[str, list[str]] = {
    "equity": ["SPX500", "NAS100", "US30", "AAPL", "MSFT", "GOOGL", "AMZN", "TSLA"],
    "forex": ["EURUSD", "GBPUSD", "USDJPY", "AUDUSD", "USDCAD"],
    "crypto": ["BTCUSD", "ETHUSD", "SOLUSD"],
    "commodity": ["XAUUSD", "XAGUSD", "CL"],
}


def classify_asset_class(symbol: str) -> str:
    for asset_class, symbols in ASSET_CLASS_SYMBOLS.items():
        if symbol.upper() in [s.upper() for s in symbols]:
            return asset_class
    return "equity"


def get_sample_count(dataset, symbol: str) -> int:
    return sum(1 for s in dataset.samples if s.symbol == symbol)


def filter_by_symbol(dataset, symbol: str):
    filtered = [s for s in dataset.samples if s.symbol == symbol]
    from orca.ml.dataset import FeatureDataset

    return FeatureDataset(
        samples=filtered,
        feature_names=dataset.feature_names,
        metadata={**dataset.metadata, "filter": f"symbol={symbol}"},
    )


def filter_by_asset_class(dataset, asset_class: str):
    symbols_upper = [s.upper() for s in ASSET_CLASS_SYMBOLS.get(asset_class, [])]
    filtered = [s for s in dataset.samples if s.symbol.upper() in symbols_upper]
    from orca.ml.dataset import FeatureDataset

    return FeatureDataset(
        samples=filtered,
        feature_names=dataset.feature_names,
        metadata={**dataset.metadata, "filter": f"asset_class={asset_class}"},
    )


class HierarchicalTrainer:
    def __init__(self, trainer, output_dir: str | Path = REGISTRY_PATH):
        self.trainer = trainer
        self.output_dir = Path(output_dir)

    def train_all(
        self,
        dataset,
        timestamps=None,
        feature_indices=None,
    ) -> dict:
        results: dict = {"global": None, "asset_classes": {}, "per_symbol": {}}

        n_total = len(dataset.samples)
        if n_total >= MIN_SAMPLES_GLOBAL:
            logger.info("training global model on %d samples", n_total)
            result = self.trainer.train(dataset, timestamps, feature_indices)
            results["global"] = result

        asset_classes = {"equity", "forex", "crypto", "commodity"}
        for ac in asset_classes:
            ac_dataset = filter_by_asset_class(dataset, ac)
            if ac_dataset.n_samples >= MIN_SAMPLES_ASSET_CLASS:
                logger.info("training %s model on %d samples", ac, ac_dataset.n_samples)
                result = self.trainer.train(ac_dataset, timestamps, feature_indices)
                results["asset_classes"][ac] = result

        for symbol in ASSET_CLASS_SYMBOLS.get("equity", [])[:5]:
            sym_count = get_sample_count(dataset, symbol)
            if sym_count >= MIN_SAMPLES_PER_SYMBOL:
                sym_dataset = filter_by_symbol(dataset, symbol)
                logger.info("fine-tuning %s on %d samples", symbol, sym_dataset.n_samples)
                result = self.trainer.train(sym_dataset, timestamps, feature_indices)
                results["per_symbol"][symbol] = result

        return results

    def save_all(self, results: dict, version: str) -> dict[str, str]:
        """Save all model levels. Returns dict of {level: path}."""
        paths: dict[str, str] = {}
        self.output_dir.mkdir(parents=True, exist_ok=True)

        if results["global"] is not None and results["global"].passed_gate:
            path = self.output_dir / f"meta_labeling_{version}.json"
            self.trainer.save_model(results["global"], path)
            paths["global"] = str(path)

        for ac, result in results.get("asset_classes", {}).items():
            if result.passed_gate:
                path = self.output_dir / f"meta_labeling_{ac}_{version}.json"
                self.trainer.save_model(result, path)
                paths[f"asset_class/{ac}"] = str(path)

        for symbol, result in results.get("per_symbol", {}).items():
            if result.passed_gate:
                path = self.output_dir / f"meta_labeling_{symbol}_{version}.json"
                self.trainer.save_model(result, path)
                paths[f"symbol/{symbol}"] = str(path)

        logger.info("saved %d models", len(paths))
        return paths
