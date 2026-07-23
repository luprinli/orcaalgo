"""Configuration and tunables for the OrcaAlgo matrix regression audit suite.

All values can be overridden via environment variables so the suite runs
unchanged in CI, locally, or against a remote host.
"""

from __future__ import annotations

import os

# ── Endpoints / auth ────────────────────────────────────────────────────────
API_BASE = os.environ.get("ORCA_API_BASE", "http://localhost:8080/api/v1")
ADMIN_USER = os.environ.get("ORCA_ADMIN_USER", "admin")
ADMIN_PASSWORD = os.environ.get(
    "ORCA_ADMIN_PASSWORD", "dev-admin-password-do-not-use-in-production"
)

# ── Data selection ──────────────────────────────────────────────────────────
# Synthetic-backed symbols are discovered dynamically from /symbols
# (exchange == "SYNTHETIC"); this list is only a fallback and a preference order.
PREFERRED_SYMBOLS = ["USDEUR", "BTC.V", "XAUUSD", "DAX", "ETH.V"]
FALLBACK_SYMBOLS = ["USDEUR", "BTC.V"]
DATA_SOURCE = os.environ.get("ORCA_AUDIT_DATA_SOURCE", "synthetic")

# Date window for single-backtest checks. One year is fast (~1-2s/run) and still
# produces trades for every strategy family.
START_DATE = os.environ.get("ORCA_AUDIT_START", "2023-01-01")
END_DATE = os.environ.get("ORCA_AUDIT_END", "2024-01-01")
CAPITAL = float(os.environ.get("ORCA_AUDIT_CAPITAL", "100000"))

# Strategy families used by the checks.
MEAN_REVERSION_STRATEGIES = ["intraday_mr", "pairs_trading", "volatility_harvesting"]
TREND_STRATEGIES = ["trend_following", "donchian_breakout"]
GRID_STRATEGY = "grid_trading"
LIGHT_STRATEGY = "intraday_mr"        # few trades → fast serialization
HEAVY_STRATEGY = "grid_trading"       # many trades → concurrency stress

# ── Timeouts / budgets (seconds) ────────────────────────────────────────────
REQUEST_TIMEOUT = int(os.environ.get("ORCA_AUDIT_REQ_TIMEOUT", "120"))
BACKTEST_TIMEOUT = int(os.environ.get("ORCA_AUDIT_BT_TIMEOUT", "180"))
MATRIX_POLL_TIMEOUT = int(os.environ.get("ORCA_AUDIT_MATRIX_TIMEOUT", "180"))
MATRIX_POLL_INTERVAL = float(os.environ.get("ORCA_AUDIT_MATRIX_INTERVAL", "1.5"))
OPTIMIZE_TIMEOUT = int(os.environ.get("ORCA_AUDIT_OPT_TIMEOUT", "240"))
CONNECT_RETRIES = int(os.environ.get("ORCA_AUDIT_RETRIES", "2"))

# ── Assertion tolerances (encode today's lessons: RNG-aware, non-flat) ──────
# The matrix engine seeds its fill simulator from the wall clock, so numeric
# results vary run-to-run. These thresholds are deliberately loose to be robust
# to that non-determinism while still catching the flat/zero regressions.
MIN_DISTINCT_EQUITY = 2          # >1 means equity actually moved (not flat)
FLAT_RETURN_EPS = 1e-9           # |return| must exceed this to be "non-flat"
SIZING_SCALE_MIN_RATIO = 2.0     # |ret(high sizing)| must exceed |ret(low)| * this
SIZING_LOW = 0.01
SIZING_HIGH = 0.05
MAX_ALLOWED_RATE_LIMITED = 0     # backtests must NOT wall-clock rate-limit

# Universal optimization parameters that must appear in every search space.
UNIVERSAL_OPT_PARAMS = ["sizing_percent", "kelly_fraction"]
