"""VectorBT — Stage 1 (Broad Screening) of the 5-stage OrcaAlgo optimisation pipeline.

Purely additive integration — nothing in orca/optimize/ or internal/ is modified.
VectorBT handles high-throughput parameter discovery (1000+ combos, multi-symbol,
multi-timeframe grid/random search) and exports narrowed candidate ranges to GKR YAML.
The Go engine handles Stages 2–5 (refined search, walk-forward, multi-metric gate,
live deployment).

Modules:
    data        — Dual-backend data connector (TimescaleDB + file)
    strategies  — VectorBT strategy function wrappers with numpy fallback
    optimize    — Broad-screen parameter sweep (vectorbt GridSearch → native sweeper fallback)
    validate    — Cross-engine validation bridge (VectorBT vs Go engine)
    to_gkr      — GKR YAML export matching orca/optimize/exporter.py v1 schema
"""

__all__ = [
    "data",
    "optimize",
    "strategies",
    "to_gkr",
    "validate",
]
