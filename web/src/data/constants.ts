export type SortField = 'sharpe' | 'sortino' | 'max_dd' | 'return' | 'win_rate' | 'profit_factor' | 'trades' | 'calmar' | 'total_fees' | 'slippage' | 'candles'

export const GATE_PROFILES = ['none', 'default', 'lenient', 'strict']

export const DATA_SOURCES = ['stooq', 'yahoo']

export const ALL_STRATEGIES: string[] = [
  'grid_trading',
  'trend_following',
  'session_scalp',
  'intraday_mr',
  'vwap_mr',
  'opening_range_breakout',
  'orb_15m',
  'pairs_trading',
  'volatility_harvesting',
  'dragon_trend',
  'volume_scalp',
  'vix_futures_carry',
  'ma_crossover',
  'rsi2_reversion',
  'donchian_breakout',
  'keltner_macd',
  'ichimoku_cloud',
  'vol_grid',
]

export const STRATEGY_DISPLAY: Record<string, string> = {
  grid_trading: 'Grid Trading',
  trend_following: 'Trend Following',
  session_scalp: 'Session Scalp',
  intraday_mr: 'Intraday Mean Reversion',
  vwap_mr: 'VWAP Mean Reversion',
  opening_range_breakout: 'Opening Range Breakout (5m)',
  orb_15m: '15-Minute ORB',
  pairs_trading: 'Pairs Trading',
  volatility_harvesting: 'Volatility Harvesting',
  dragon_trend: 'Dragon Capital Trend',
  volume_scalp: 'Volume-Weighted Scalp',
  vix_futures_carry: 'VIX Futures Carry',
  ma_crossover: 'MA Crossover',
  rsi2_reversion: 'RSI(2) Reversion',
  donchian_breakout: 'Donchian Breakout',
  keltner_macd: 'Keltner MACD',
  ichimoku_cloud: 'Ichimoku Cloud',
  vol_grid: 'Vol-Adjusted Grid',
}

// Canonical 18-symbol prop-firm universe (mirrors configs/universe.json).
// These are fallback values only: BacktestHub fetches strategies and symbols
// dynamically from the API and uses these when the API is unavailable.
export const SYMBOL_OPTIONS = [
  'SPY', 'QQQ', 'AAPL', 'MSFT', 'NVDA', 'TSLA', 'IWM', 'GLD', 'TLT',
  'EURUSD', 'GBPUSD', 'USDJPY', 'AUDUSD', 'USDCAD', 'BTC-USD', 'ETH-USD', '^_US', '^DAX',
]

// Friendly display names for opaque/hyphenated universe tickers. Equity ETFs
// and stocks fall back to their raw ticker, which is already human-readable.
export const SYMBOL_DISPLAY: Record<string, string> = {
  '^_US': 'S&P 500',
  '^DAX': 'DAX',
  'BTC-USD': 'Bitcoin',
  'ETH-USD': 'Ethereum',
  EURUSD: 'EUR/USD',
  GBPUSD: 'GBP/USD',
  USDJPY: 'USD/JPY',
  AUDUSD: 'AUD/USD',
  USDCAD: 'USD/CAD',
}

// Backtest timeframes supported by the engine (mirrors configs/universe.json
// `timeframes`). Charting uses its own timeframe list in useCandleAggregation.
export const TIMEFRAME_OPTIONS = ['5m', '15m', '30m', '1h', '4h', '1d']

export const POLL_INTERVALS = {
  MONITOR_REFRESH: 10_000,
  ORCH_MATRIX_POLL: 3_000,
} as const
