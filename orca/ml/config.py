"""ML subsystem configuration constants.

Shared between Python training and Go inference to prevent train/serve skew.
Must be kept in sync with internal/ml/types.go.
"""

from __future__ import annotations

# ── Feature vector dimensions ────────────────────────────────────────────────
FEATURE_DIM = 21

# Cyclic time encoding uses sin/cos pairs
TIME_FEATURE_PAIRS = 2  # (hour_sin, hour_cos) + (day_sin, day_cos)

# One-hot encoded dimensions
HMM_REGIME_DIM = 4       # HMM alpha vector (4 states)
SIGNAL_TYPE_COUNT = 10    # one-hot for strategy type (max 10 strategies)
ASSET_CLASS_COUNT = 4     # equity, forex, crypto, commodity

# ── Triple-barrier defaults ──────────────────────────────────────────────────
BARRIER_PROFIT_FACTOR = 2.0    # upper = entry * (1 + profit_factor * sigma)
BARRIER_STOP_FACTOR = 1.0      # lower = entry * (1 - stop_factor * sigma)
BARRIER_TIME_HORIZON = 20      # bars before time-based exit
BARRIER_MIN_RETURN = 0.001     # minimum absolute return to count as win/loss

# ── Meta-labeling thresholds ─────────────────────────────────────────────────
META_WIN_THRESHOLD = 0.55      # reject signals with p_win < this
META_EXTREME_LOW = 0.05        # skip ML for signals with confidence < this
META_EXTREME_HIGH = 0.95       # skip ML for signals with confidence > this
META_POSITION_SCALE_CAP = 1.50  # max position multiplier from p_win

# ── Model training hyperparameters ───────────────────────────────────────────
XGB_N_ESTIMATORS = 200
XGB_MAX_DEPTH = 5
XGB_LEARNING_RATE = 0.05
XGB_SUBSAMPLE = 0.8
XGB_COLSAMPLE_BYTREE = 0.8
XGB_SCALE_POS_WEIGHT = 1.5
XGB_EARLY_STOPPING = 20

# ── Purged cross-validation ──────────────────────────────────────────────────
CV_N_SPLITS = 5
CV_EMBARGO_PCT = 0.01          # fraction of samples embargoed after test
CV_TRAIN_YEARS = 2
CV_TEST_MONTHS = 6

# ── Retraining triggers ──────────────────────────────────────────────────────
BRIER_DEGRADATION_THRESHOLD = 1.10   # retrain if brier > training_brier * 1.10
WIN_RATE_DEGRADATION_PP = 0.05      # retrain if win rate drops by 5pp
WIN_RATE_DEGRADATION_DAYS = 5        # must persist for N consecutive days
PSI_DRIFT_THRESHOLD = 0.20           # retrain if PSI > 0.20
PSI_DRIFT_MODERATE = 0.10            # alert if PSI > 0.10
VIX_SHIFT_RATIO = 2.0               # review if 20d VIX avg changes by >2x

# ── Minimum sample requirements ──────────────────────────────────────────────
# Production guard: 100,000 minimum samples for model training. Unit tests
# override via MetaLabelingTrainer(min_samples=...) parameter. Do not lower
# for production without re-running the calibration audit.
MIN_SAMPLES_GLOBAL = 100_000
MIN_SAMPLES_ASSET_CLASS = 50_000
MIN_SAMPLES_PER_SYMBOL = 50_000
MIN_SAMPLES_PER_BIN = 20            # Murphy decomposition minimum per bin

# ── Model quality gates ──────────────────────────────────────────────────────
BRIER_MAX = 0.20                    # meta-labeling: must achieve Brier < 0.20
ROC_AUC_MIN = 0.65                  # meta-labeling: must achieve ROC-AUC > 0.65
REGIME_ACCURACY_MIN = 0.80          # regime classifier: must achieve > 80%

# ── Inference latency budget (microseconds) ──────────────────────────────────
INFERENCE_TIMEOUT_US = 100           # hard cap per single inference
BATCH_INFERENCE_BUDGET_US = 500     # budget for batched inference
FEATURE_COMPUTE_BUDGET_US = 50      # budget for feature computation

# ── Canary deployment ────────────────────────────────────────────────────────
SHADOW_MODE_DAYS = 7                # days in shadow before canary
CANARY_TRAFFIC_PCT = 0.10           # fraction of traffic for canary model
CANARY_EVAL_DAYS = 3                # minimum days before full rollout
ROLLBACK_CONSECUTIVE_DAYS = 3       # consecutive days of underperformance trigger
ROLLBACK_SLA_MINUTES = 5            # max time from detection to rollback

# ── Model registry paths ─────────────────────────────────────────────────────
MODEL_DIR = "models"
REGISTRY_PATH = "models/registry.yaml"

# ── Feature names (must match internal/ml/types.go order) ─────────────────────
FEATURE_NAMES = [
    "ret1",              #  0: 1-bar log return
    "ret5",              #  1: 5-bar log return
    "ret20",             #  2: 20-bar log return
    "volatility20",      #  3: 20-period EWMA volatility
    "atr_ratio",         #  4: ATR / Close
    "rsi14",             #  5: RSI(14)
    "macd_hist",         #  6: MACD histogram
    "adx14",             #  7: ADX(14)
    "bb_percent_b",      #  8: Bollinger %B
    "volume_ratio",      #  9: Volume / 20-period avg volume
    "cvd_divergence",    # 10: CVD divergence flag
    "spread_pct",        # 11: Bid-ask spread / Close
    "hmm_state_0",       # 12: HMM alpha[0]
    "hmm_state_1",       # 13: HMM alpha[1]
    "hmm_state_2",       # 14: HMM alpha[2]
    "hmm_state_3",       # 15: HMM alpha[3]
    "hmm_confidence",    # 16: HMM confidence
    "signal_type",       # 17: Strategy type (one-hot encoded in training)
    "signal_strength",   # 18: Signal conviction (z-score / EMA distance)
    "hour_sin",          # 19: sin(2π * hour / 24)
    "hour_cos",          # 20: cos(2π * hour / 24)
]

# Cyclic time features: index pairs (sin, cos)
TIME_FEATURE_INDICES = {
    "hour": (19, 20),
}
