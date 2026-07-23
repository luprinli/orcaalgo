"""Orca ML - ML pipelines for signal meta-labeling, regime classification, and exit optimization.

Data flows:
  Trade execution records (TimescaleDB via orca-fetch subprocess)
      → Feature dataset builder (dataset.py)
          → Triple-barrier labels (barriers.py)
              → Purged walk-forward CV (purge_cv.py)
                  → Model training (train/)
                      → ONNX export → internal/ml/ (Go inference)

Feature computation is specified in features.py — shared semantic between
Python (training) and Go (real-time inference) to prevent train/serve skew.
"""

__version__ = "0.1.0"

__all__ = [
    "barriers",
    "config",
    "dataset",
    "drift_detection",
    "feature_selection",
    "features",
    "purge_cv",
]
