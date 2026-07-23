"""VectorBT implementations of OrcaAlgo strategy types.

PARAM NAMES ARE LOCKED to match orca/optimize/indicator_factory.py.
DO NOT RENAME PARAMETERS without updating indicator_factory.py first.

Each strategy function returns a pd.Series:
    1  = buy/entry signal
   -1  = sell/exit signal
    0  = hold (no signal)
"""

import numpy as np
import pandas as pd

# ── Soft import for vectorbt (allows operation without the library installed) ──
# We verify the imported module actually exposes the indicator API we need.
# This guards against a same-named test package (e.g. tests/vectorbt/) shadowing
# the real library on sys.path, which would otherwise set HAS_VBT=True on an
# empty module and raise AttributeError at call time.
try:
    import vectorbt as vbt

    HAS_VBT = all(hasattr(vbt, attr) for attr in ("RSI", "ADX", "ATR"))
    if not HAS_VBT:
        vbt = None  # type: ignore[assignment]
except ImportError:
    vbt = None  # type: ignore[assignment]
    HAS_VBT = False


# ═══════════════════════════════════════════════════════════════════════
# Strategy 1: Intraday Mean Reversion (RSI-based)
# ═══════════════════════════════════════════════════════════════════════

def intraday_mr(
    close: pd.Series,
    rsi_period: int = 20,
    entry_threshold: float = 30.0,
    exit_threshold: float = 50.0,
) -> pd.Series:
    """RSI-based mean reversion: buy when RSI < entry_threshold, sell > exit_threshold.

    Param names match indicator_factory.py: rsi_period, entry_threshold, exit_threshold.
    """
    if HAS_VBT:
        rsi = vbt.RSI.run(close, window=rsi_period).rsi
    else:
        rsi_values = _compute_rsi_numpy(close.values, rsi_period)
        rsi = pd.Series(rsi_values, index=close.index)

    entries = rsi < entry_threshold
    exits = rsi > exit_threshold
    return entries.astype(int) - exits.astype(int)


# ═══════════════════════════════════════════════════════════════════════
# Strategy 2: Trend Following (EMA crossover + ADX filter)
# ═══════════════════════════════════════════════════════════════════════

def trend_following(
    close: pd.Series,
    high: pd.Series,
    low: pd.Series,
    ema_fast: int = 20,
    ema_slow: int = 50,
    adx_threshold: float = 22.0,
) -> pd.Series:
    """EMA crossover with ADX filter.

    Param names match indicator_factory.py: ema_fast, ema_slow, adx_threshold.
    """
    ema_f = close.ewm(span=ema_fast, adjust=False).mean()
    ema_s = close.ewm(span=ema_slow, adjust=False).mean()

    if HAS_VBT:
        adx_series = vbt.ADX.run(high, low, close, window=14).adx
    else:
        adx_values = _compute_adx_numpy(high.values, low.values, close.values, period=14)
        adx_series = pd.Series(adx_values, index=close.index)

    entries = (ema_f > ema_s) & (adx_series > adx_threshold)
    exits = ema_f < ema_s
    return entries.astype(int) - exits.astype(int)


# ═══════════════════════════════════════════════════════════════════════
# Strategy 3: Opening Range Breakout
# ═══════════════════════════════════════════════════════════════════════

def opening_range_breakout(
    open_price: pd.Series,
    high: pd.Series,
    low: pd.Series,
    close: pd.Series,
    range_minutes: int = 5,
    atr_mult: float = 2.0,
    volume_mult: float = 1.5,
) -> pd.Series:
    """Opening range breakout: first N bars define the range, break above/below triggers.

    Param names match indicator_factory.py: range_minutes, atr_mult, volume_mult.
    """
    n = len(close)
    signals = np.zeros(n, dtype=int)
    if n < range_minutes + 2:
        return pd.Series(signals, index=close.index)

    if HAS_VBT:
        atr = vbt.ATR.run(high, low, close, window=14).atr.values
    else:
        atr = _compute_atr_numpy(high.values, low.values, close.values, period=14)

    day_start = 0
    for i in range(range_minutes, n):
        if i >= day_start + range_minutes:
            range_high = high.iloc[day_start : day_start + range_minutes].max()
            range_low = low.iloc[day_start : day_start + range_minutes].min()
            atr_val = max(atr[i], 1e-10)

            if close.iloc[i] > range_high + atr_val * atr_mult:
                signals[i] = 1
                day_start = n
            elif close.iloc[i] < range_low - atr_val * atr_mult:
                signals[i] = -1
                day_start = n
        if signals[i] != 0:
            break

    return pd.Series(signals, index=close.index)


# ═══════════════════════════════════════════════════════════════════════
# Strategy 4: Grid Trading
# ═══════════════════════════════════════════════════════════════════════

def grid_trading(
    close: pd.Series,
    grid_levels: int = 5,
    grid_spacing_pct: float = 1.0,
    max_open: int = 10,
) -> pd.Series:
    """Grid trading: place buy/sell orders at equally spaced price levels.

    Param names match indicator_factory.py: grid_levels, grid_spacing_pct, max_open.
    """
    n = len(close)
    signals = np.zeros(n, dtype=int)
    if n < 2:
        return pd.Series(signals, index=close.index)

    mid = close.iloc[0]
    spacing = mid * grid_spacing_pct / 100.0

    for i in range(1, n):
        price = close.iloc[i]
        levels_above = int((price - mid) / max(spacing, 1e-10))
        levels_below = int((mid - price) / max(spacing, 1e-10))

        if levels_above > 0 and levels_above <= grid_levels:
            signals[i] = -1
        elif levels_below > 0 and levels_below <= grid_levels:
            signals[i] = 1

    return pd.Series(signals, index=close.index)


# ═══════════════════════════════════════════════════════════════════════
# Strategy 5: Session Scalp (stub)
# ═══════════════════════════════════════════════════════════════════════

def session_scalp(
    close: pd.Series,
    high: pd.Series,
    low: pd.Series,
    volume: pd.Series,
    lookback: int = 10,
    profit_target: float = 0.5,
) -> pd.Series:
    """Session-based scalping — stub implementation.

    Param names match expected conventions; full implementation deferred to
    match Go runner in internal/strategy/.
    """
    n = len(close)
    signals = np.zeros(n, dtype=int)
    if n < lookback + 1:
        return pd.Series(signals, index=close.index)

    momentum = close.diff(lookback)
    entries = momentum > 0
    exits = momentum < 0
    return entries.astype(int) - exits.astype(int)


# ═══════════════════════════════════════════════════════════════════════
# Pure numpy fallback implementations (no vectorbt dependency)
# ═══════════════════════════════════════════════════════════════════════

def _compute_rsi_numpy(prices: np.ndarray, period: int = 14) -> np.ndarray:
    """Wilder's smoothed RSI — identical to sweeper._compute_rsi()."""
    n = len(prices)
    if n < period + 1:
        return np.full(n, np.nan)

    deltas = np.diff(prices)
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)

    avg_gain = np.full(n, np.nan)
    avg_loss = np.full(n, np.nan)
    avg_gain[period] = np.mean(gains[:period])
    avg_loss[period] = np.mean(losses[:period])

    for i in range(period + 1, n):
        avg_gain[i] = (avg_gain[i - 1] * (period - 1) + gains[i - 1]) / period
        avg_loss[i] = (avg_loss[i - 1] * (period - 1) + losses[i - 1]) / period

    with np.errstate(divide="ignore", invalid="ignore"):
        rs = np.divide(avg_gain, avg_loss, out=np.full_like(avg_gain, np.nan), where=avg_loss != 0)
    return 100.0 - (100.0 / (1.0 + rs))


def _compute_atr_numpy(
    high: np.ndarray,
    low: np.ndarray,
    close: np.ndarray,
    period: int = 14,
) -> np.ndarray:
    """True-range ATR — Wilder's smoothed."""
    n = len(close)
    if n < 2:
        return np.full(n, 0.0)

    prev_close = np.roll(close, 1)
    prev_close[0] = close[0]
    tr = np.maximum(
        high - low,
        np.maximum(
            np.abs(high - prev_close),
            np.abs(low - prev_close),
        ),
    )

    atr = np.full(n, np.nan)
    atr[period] = np.mean(tr[1:period])
    for i in range(period + 1, n):
        atr[i] = (atr[i - 1] * (period - 1) + tr[i]) / period

    return np.nan_to_num(atr, nan=0.0)


def _compute_adx_numpy(
    high: np.ndarray,
    low: np.ndarray,
    close: np.ndarray,
    period: int = 14,
) -> np.ndarray:
    """ADX computation using true range and directional movement.

    Simplification: uses smoothed +DM/-DM over period.
    For production accuracy, reference go-trader ADX implementation.
    """
    n = len(close)
    if n < period + 1:
        return np.full(n, 25.0)

    prev_high = np.roll(high, 1)
    prev_low = np.roll(low, 1)
    prev_high[0] = high[0]
    prev_low[0] = low[0]

    up_move = high - prev_high
    down_move = prev_low - low

    plus_dm = np.where((up_move > down_move) & (up_move > 0), up_move, 0.0)
    minus_dm = np.where((down_move > up_move) & (down_move > 0), down_move, 0.0)

    tr = np.maximum(
        high - low,
        np.maximum(
            np.abs(high - prev_high),
            np.abs(low - prev_low),
        ),
    )

    atr = _smooth(tr, period)
    plus_di = 100.0 * _smooth(plus_dm, period) / np.maximum(atr, 1e-10)
    minus_di = 100.0 * _smooth(minus_dm, period) / np.maximum(atr, 1e-10)

    with np.errstate(divide="ignore", invalid="ignore"):
        dx = 100.0 * np.abs(plus_di - minus_di) / np.maximum(plus_di + minus_di, 1e-10)

    adx = _smooth(np.nan_to_num(dx, nan=0.0), period)
    return np.nan_to_num(adx, nan=25.0)


def _smooth(series: np.ndarray, period: int) -> np.ndarray:
    """Wilder's smoothing: EMA with alpha = 1/period."""
    n = len(series)
    result = np.full(n, np.nan)
    result[period] = np.mean(series[1:period + 1])
    for i in range(period + 1, n):
        result[i] = (result[i - 1] * (period - 1) + series[i]) / period
    return result


# ═══════════════════════════════════════════════════════════════════════
# Strategy registry — keys MUST match indicator_factory.STRATEGY_INDICATORS
# ═══════════════════════════════════════════════════════════════════════

STRATEGY_MAP: dict[str, callable] = {
    "intraday_mr": intraday_mr,
    "trend_following": trend_following,
    "opening_range_breakout": opening_range_breakout,
    "grid_trading": grid_trading,
    "session_scalp": session_scalp,
}
