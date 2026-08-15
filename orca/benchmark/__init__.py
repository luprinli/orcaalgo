"""Market-based benchmark filter (mandatory promotion gate).

Pure-Python math (HP #1) for determining whether a strategy's returns are
economically meaningful relative to a declared benchmark.
"""

from __future__ import annotations

from orca.benchmark.filter import BenchmarkVerdict, apply_benchmark_filter
from orca.benchmark.metrics import compute_benchmark_metrics
from orca.benchmark.spec import BenchmarkSpec, BenchmarkThresholds

__all__ = [
    "BenchmarkSpec",
    "BenchmarkThresholds",
    "BenchmarkVerdict",
    "apply_benchmark_filter",
    "compute_benchmark_metrics",
]
