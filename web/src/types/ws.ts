export interface WSEnvelope {
  channel: string
  data: unknown
}

export interface WSRiskData {
  halted: boolean
  reason: string
  connections: number
  regime: number
  confidence: number
  vix: number
  sentiment: number
  daily_loss_used: number
  drawdown_used: number
  daily_limit_pct: number
  max_dd_pct: number
  consistency_multiplier: number
  tick_count: number
  balance: number
  equity: number
  daily_pnl_pct: number
}

export interface WSPerformanceData {
  timestamp: string
  equity: number
  balance: number
  daily_pnl: number
  daily_pnl_pct: number
  drawdown_pct: number
  max_drawdown_pct: number
  sharpe: number
  sortino: number
  win_rate: number
  profit_factor: number
  num_trades: number
}

export interface WSPositionData {
  positions: Array<{
    symbol: string
    side: string
    quantity: number
    average_entry_price: number
    unrealized_pnl: number
    last_updated: string
  }>
}

export interface WSOrderData {
  orders: Array<{
    order_id: string
    symbol: string
    side: string
    quantity: number
    state: string
    filled_quantity: number
  }>
}

export interface WSFillData {
  account_id: string
  broker_order_id: string
  symbol: string
  side: string
  filled_qty: number
  avg_fill_price: number
  status: string
  strategy_id: string
}

export interface WSTickData {
  symbol: string
  price: number
  volume: number
  side: string
  time: string
}

export interface WSCVDData {
  bar: {
    time: string
    open: number
    high: number
    low: number
    close: number
    buy_volume: number
    sell_volume: number
    delta: number
    num_trades: number
  }
}

export interface WSDivergenceData {
  type: string
  confidence: number
  time: string
}

export interface WSPnLHistoryData {
  daily_pnl_pct: number
  cumulative_pnl: number
  equity: number
  time: string
}

export interface WSBacktestProgressData {
  batch_id: string
  progress: number
  completed: number
  total: number
  status: string
}

export interface WSIndicatorUpdateData {
  indicator: string
  symbol: string
  timeframe: string
  timestamp_ms: number
  values: Record<string, number>
}
