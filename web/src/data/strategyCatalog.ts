export interface StrategyCatalogEntry {
  typeKey: string
  displayName: string
  inEngine: boolean
  description?: string
  regimes?: string[]
}

export const STRATEGY_CATALOG: StrategyCatalogEntry[] = [
  { typeKey: 'grid_trading', displayName: 'Grid Trading', inEngine: true, regimes: ['Calm'], description: 'Multi-level grid with take-profit and stop-loss. Disabled by default.' },
  { typeKey: 'vol_grid', displayName: 'Vol-Adjusted Grid', inEngine: true, regimes: ['Calm'], description: 'Grid with dynamic spacing based on ATR/VIX.' },
  { typeKey: 'trend_following', displayName: 'Trend Following', inEngine: true, regimes: ['Trending'], description: 'EMA crossover with ADX confirmation, CHOP filter, trailing stop.' },
  { typeKey: 'session_scalp', displayName: 'Session Scalp', inEngine: true, regimes: ['Calm', 'Trending', 'HighVol'], description: 'Session-range breakout scalp, volume-confirmed, max trades/day limit.' },
  { typeKey: 'intraday_mr', displayName: 'Intraday Mean Reversion', inEngine: true, regimes: ['Calm'], description: 'Z-score mean reversion with ATR-normalized exit.' },
  { typeKey: 'vwap_mr', displayName: 'VWAP Mean Reversion', inEngine: true, regimes: ['Calm'], description: 'Volume-weighted average price as mean reference.' },
  { typeKey: 'opening_range_breakout', displayName: 'ORB (5m)', inEngine: true, regimes: ['Calm', 'Trending', 'HighVol'], description: '5-minute opening range breakout with ATR stops.' },
  { typeKey: 'orb_15m', displayName: 'ORB (15m)', inEngine: true, regimes: ['Calm', 'Trending', 'HighVol'], description: '15-minute opening range breakout, fewer false signals.' },
  { typeKey: 'pairs_trading', displayName: 'Pairs Trading', inEngine: true, regimes: ['Calm', 'HighVol'], description: 'Cointegration spread trading with cached hedge ratio.' },
  { typeKey: 'volatility_harvesting', displayName: 'Volatility Harvesting', inEngine: true, regimes: ['HighVol'], description: 'VIX-gated mean reversion, vol spike fade, tight stops.' },
  { typeKey: 'dragon_trend', displayName: 'Dragon Capital Trend', inEngine: true, regimes: ['Trending', 'HighVol'], description: 'Multi-EMA ribbon (8,21,50,200) with proportional sizing.' },
  { typeKey: 'volume_scalp', displayName: 'Volume-Weighted Scalp', inEngine: true, regimes: ['Calm', 'Trending'], description: 'Volume-confirmed scalp (V > avg * 2), session range breakout.' },
  { typeKey: 'vix_futures_carry', displayName: 'VIX Futures Carry', inEngine: true, regimes: ['HighVol'], description: 'Contango proxy via spot VIX, fade mean-reversion.' },
  { typeKey: 'ma_crossover', displayName: 'MA Crossover', inEngine: true, regimes: ['All'], description: 'Moving average crossover with RSI/MACD filters.' },
  { typeKey: 'rsi2_reversion', displayName: 'RSI(2) Reversion', inEngine: true, regimes: ['All'], description: 'RSI(2) mean reversion with trend filter.' },
  { typeKey: 'donchian_breakout', displayName: 'Donchian Breakout', inEngine: true, regimes: ['All'], description: 'Donchian channel breakout with entry buffer.' },
  { typeKey: 'keltner_macd', displayName: 'Keltner MACD', inEngine: true, regimes: ['All'], description: 'Keltner channel with MACD confirmation.' },
  { typeKey: 'ichimoku_cloud', displayName: 'Ichimoku Cloud', inEngine: true, regimes: ['All'], description: 'Ichimoku cloud strategy.' },
  { typeKey: 'momentum_12_1', displayName: 'Time-Series Momentum (12-1)', inEngine: true, regimes: ['Calm', 'Trending'], description: 'Long-only absolute momentum with trend filter and ATR trailing stop.' },
  { typeKey: 'fx_carry', displayName: 'FX Carry', inEngine: true, regimes: ['Calm', 'HighVol'], description: 'Interest-rate carry with trend filter and trailing stop.' },
  { typeKey: 'session_momentum', displayName: 'Session Momentum', inEngine: true, regimes: ['Calm', 'Trending', 'HighVol'], description: 'Intraday session-open drift with volume confirmation.' },
]
