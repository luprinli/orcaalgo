export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  roles: string[]
}

export interface Strategy {
  id: string
  name: string
  type: string
  enabled: boolean
  parameters?: Record<string, unknown>
  created_at?: string
}

export interface CreateStrategyRequest {
  name: string
  type: string
  parameters?: Record<string, unknown>
  enabled?: boolean
}

export interface UpdateStrategyRequest {
  name?: string
  type?: string
  parameters?: Record<string, unknown>
  enabled?: boolean
}

export interface DeployStrategyRequest {
  strategy_name: string
  backtest_id: string
  account_id?: string
  capital_allocation_pct?: number
}

export interface DeployStrategyResponse {
  deployed: boolean
  strategy_name: string
  backtest_id: string
  account_id: string
  capital_allocation_pct: number
}

export interface PreflightCheck {
  name: string
  status: 'pass' | 'warn' | 'fail'
  message: string
}

export interface PreflightResponse {
  passed: boolean
  passed_count: number
  warned_count: number
  failed_count: number
  checks: PreflightCheck[]
}

export interface StrategyValidationRequest {
  name?: string
  type?: string
  parameters?: Record<string, unknown>
  yaml?: string
  json?: string
}

export interface StrategyValidationResponse {
  valid: boolean
  errors: string[]
  diagnostics?: unknown[]
}

export interface BacktestRequest {
  mode?: string
  strategy_id?: string
  strategy_ids?: string[]
  symbols: string[]
  timeframes?: string[]
  start_date: string
  end_date: string
  capital?: number
  gate_profile?: string
  strategy_params?: Record<string, number>
  data_source?: string
  propfirm_enabled?: boolean
  optimize?: boolean
  sizing_percent?: number
}

export interface ComboResult {
  batch_id?: string
  run_id?: string
  symbol: string
  strategy_id: string
  timeframe: string
  sharpe_ratio: number
  sortino_ratio: number
  max_drawdown: number
  total_return: number
  win_rate: number
  profit_factor: number
  avg_trade: number
  avg_win: number
  avg_loss: number
  num_trades: number
  num_wins?: number
  num_losses?: number
  error?: string
  gate_passed?: boolean
  adverse_selection_rate?: number
  optimized?: boolean
  best_params?: Record<string, number>
  strategy_params?: Record<string, number>
  long_trades?: number
  short_trades?: number
  long_win_rate?: number
  short_win_rate?: number
  long_gross_pnl?: number
  short_gross_pnl?: number
  long_profit_factor?: number
  short_profit_factor?: number
  zero_pnl_trades?: number
  expected_pf?: number
  reward_risk_ratio?: number
  daily_volatility?: number
  max_drawdown_duration?: number
  warnings?: string[]
  avg_mae?: number
  avg_mfe?: number
  equity_curve?: EquityPoint[]
  trades?: TradeSummary[]
  total_fees?: number
  avg_slippage_bps?: number
  calmar_ratio?: number
  candle_count?: number
  gross_return_pct?: number
  data_source?: string
  engine_version?: string
  wf_is_sharpe?: number
  wf_oos_sharpe?: number
  first_candle_time?: string
  last_candle_time?: string
  declared_bars_per_day?: number
  effective_bars_per_day?: number
  mtm_sharpe_ratio?: number
  mtm_max_drawdown?: number
}

export interface MatrixResultsResponse {
  summary: {
    total_combos: number
    passed: number
    failed: number
    total_trades: number
    best_sharpe: number
    best_strategy: string
    best_symbol: string
    status: string
    message?: string
    // Telemetry (execution-framework §5.2)
    batch_run_id?: string
    completed?: number
    running?: number
    skipped?: number
    phase?: string
    cancelled?: boolean
    percent?: number
    throughput_per_min?: number
    eta_seconds?: number
    current?: { strategy: string; symbol: string; timeframe: string } | null
    chunk?: { index: number; total: number }
    seq?: number
  }
  results: ComboResult[]
  batch_id: string
  status: string
  completed?: number
  total?: number
  failed?: number
  seq?: number
}

export interface SystemHealth {
  heap_inuse_mb: number
  heap_budget_mb: number
  num_goroutine: number
  num_cpu: number
  matrix_workers: number
  db_pool_in_use: number
  db_pool_max: number
  near_capacity: boolean
}

export interface BacktestResponse {
  id?: string
  sharpe_ratio: number
  sortino_ratio: number
  max_drawdown: number
  win_rate: number
  num_trades: number
  profit_factor: number
  total_return: number
  equity_curve?: EquityPoint[]
  error?: string
}

export interface BacktestMetrics {
  sharpe_ratio: number
  sortino_ratio: number
  max_drawdown_pct: number
  win_rate_pct: number
  profit_factor: number
  total_return_pct: number
  num_trades: number
  trading_volume: number
  strategy_name: string
  pass_probability: number
  calmar: number
  var_95: number
  cvar_95: number
  ulcer_index: number
  cagr: number
  balance: number
  equity: number
  warnings?: string[]
  commission_bps?: number
  total_commission?: number
}

export interface EquityPoint {
  time: string
  value: number
  regime: number
}

export interface TradeSummary {
  id: string
  symbol: string
  side: string
  quantity: number
  entry_price: number
  exit_price: number
  pnl: number
  pnl_pct: number
  entry_time: string
  exit_time: string
  hold_duration: number
  mae: number
  mfe: number
  strategy_id: string
  exit_reason: string
  commission: number
  hmm_regime: number
  stop_price: number
  take_price: number
  slippage_mid_bps: number
  slippage_last_bps: number
  adverse_selection: boolean
}

export interface TradeChange {
  timestamp: string
  field: string
  from?: string
  to?: string
  reason?: string
}

export interface TradeDetail extends TradeSummary {
  changes: TradeChange[]
  lowest_price: number
  highest_price: number
}

export interface TradeDistribution {
  total_trades: number
  winning_trades: number
  losing_trades: number
  win_rate_pct: number
  avg_trade_pnl: number
  median_trade_pnl: number
  avg_trade_pnl_pct: number
  median_trade_pnl_pct: number
  best_trade: number
  worst_trade: number
  avg_trade_duration_hours: number
  median_trade_duration_hours: number
  avg_winning_pnl: number
  avg_losing_pnl: number
  unique_tickers: number
}

export interface LLMKey {
  id: number
  user_id: string
  provider: string
  base_url: string
  model: string
  masked_suffix: string
  created_at?: string
  updated_at?: string
}

export interface DailyReturn {
  date: string
  return_pct: number
  pnl: number
}

export interface MonthlyReturn {
  year: number
  month: number
  return_pct: number
}

export interface RollingMetric {
  timestamp: string
  value: number
  window: number
}

export interface RegimeStat {
  regime: number
  label: string
  num_trades: number
  win_rate: number
  total_return: number
  max_drawdown: number
  profit_factor: number
}

export interface WalkForwardWindow {
  window?: number
  train_start?: string
  test_start?: string
  test_end?: string
  in_sample_sharpe?: number
  out_sample_sharpe?: number
  oos_win_rate?: number
  oos_return_pct?: number
  oos_profit_factor?: number
  oos_trades?: number
  passed_compliance?: boolean
}

export interface WalkForwardResponse {
  windows: WalkForwardWindow[]
  passed_windows: number
  total_windows: number
  oos_avg_sharpe?: number
  sharpe_degradation?: number
  overall_sharpe: number
  overall_win_rate: number
  message?: string
}

export interface OptimizationFootprint {
  deflated_sharpe: number
  conventional_sharpe: number
  grid_passes: number
  bayesian_iterations: number
  walk_forward_windows: number
  passed_windows: number
  ivs: number
  oos_average_sharpe: number
  sharpe_degradation: number
  best_params_json: string
}

export interface LiveComparisonResponse {
  backtest_equity: EquityPoint[]
  live_equity: EquityPoint[]
  metrics: {
    cumulative_slippage_bps: number
    fill_rate_ratio: number
    max_equity_divergence_pct: number
  }
}

export interface RiskStatus {
  halted: boolean
  reason: string
  last_trigger: string
  balance: number
  equity: number
  daily_pnl_pct: number
  daily_loss_used: number
  drawdown_used: number
  daily_limit_pct: number
  max_dd_pct: number
  consistency_multiplier: number
}

export interface PlaceOrderRequest {
  account_id?: string
  symbol: string
  side: 'BUY' | 'SELL'
  type: 'MARKET' | 'LIMIT' | 'STOP' | 'STOP_LIMIT'
  quantity: number
  limitPrice?: number
  stopPrice?: number
  timeInForce?: string
  strategy_id?: string
}

export interface Order {
  order_id: string
  symbol: string
  side: string
  quantity: number
  price: number | null
  order_type: string
  state: string
  filled_quantity: number
  created_at: string
  updated_at: string | null
}

export interface Position {
  symbol: string
  side: string
  quantity: number
  average_entry_price: number
  unrealized_pnl: number
  last_updated: string
}

export interface Account {
  id: string
  label: string
  firm: string
  broker_type: string
  environment: string
  masked_key: string
  type: string
  is_default: boolean
  halted: boolean
  balance: number
  equity: number
  daily_pnl_pct: number
  buying_power?: number
}

export interface CreateAccountRequest {
  id?: string
  name: string
  broker_type: string
  prop_firm_profile_id?: string
  is_default?: boolean
  environment?: string
  api_key?: string
  api_secret?: string
}

export interface PropFirmProfile {
  id: string
  name: string
  max_daily_loss_pct: number
  max_drawdown_pct: number
  drawdown_type: string
  max_position_pct: number
  max_open_positions: number
  max_trades_per_day: number
  consistency_enabled: boolean
  consistency_threshold_pct?: number
  profit_target_pct_phase1: number
  profit_target_pct_phase2: number
  min_trading_days: number
}

export interface PropFirmState {
  profile_id: string
  starting_balance: number
  peak_balance: number
  daily_pnl: number
  daily_pnl_pct: number
  cumulative_pnl: number
  consistency_mult: number
  trading_days: number
  current_phase: number
  phase_target_met: boolean
  violated: boolean
  violation_reason: string
}

export interface Candle {
  time: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface CandleResponse {
  symbol: string
  range: string
  candles: Candle[]
  warning?: string
}

export interface UniverseEntry {
  ticker: string
  active: boolean
  added_at: string
}

export interface LiveMetrics {
  timestamp: string
  equity: number
  balance: number
  daily_pnl: number
  daily_pnl_pct: number
  sharpe: number
  sortino: number
  drawdown_pct: number
  max_drawdown_pct: number
  win_rate: number
  profit_factor: number
  num_trades: number
}

export interface BacktestHistoryEntry {
  id: string
  run_type: string
  started_at: string
  completed_at: string | null
  status: string
  strategy_ids: string[]
  symbols: string[]
}

export interface AppSettings {
  general?: Record<string, unknown>
  risk?: Record<string, unknown>
  notifications?: Record<string, unknown>
}

export interface AuditLogEntry {
  id: string
  user_id: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  created_at: string
}

export interface ParamDef {
  name: string
  type: 'continuous' | 'integer' | 'categorical'
  default: number
  min: number
  max: number
  step: number
  group: string
  description: string
}

export interface ErrorLogEntry {
  id: string
  severity: string
  component: string
  message: string
  stack: string
  created_at: string
}

export interface CalibrationBinStats {
  bin_start: number
  bin_end: number
  count: number
  mean_prediction: number
  hit_rate: number
}

export interface CalibrationSegmentReport {
  name: string
  n: number
  brier: number
  reliability: number
  resolution: number
  uncertainty: number
  bin_stats: CalibrationBinStats[]
  needs_calibration: boolean
}

export interface CalibrationReportResponse {
  overall: CalibrationSegmentReport
  segments: Record<string, CalibrationSegmentReport>
  generated_at: string
  error?: string
}

export interface AttributionSliceStats {
  n: number
  wins: number
  hit_rate: number
  hit_rate_ci_low: number
  hit_rate_ci_high: number
  total_pnl: number
  total_cost: number
  roi: number
}

export interface AttributionReportResponse {
  overall: AttributionSliceStats
  by_side: Record<string, AttributionSliceStats>
  by_price_bucket: Record<string, AttributionSliceStats>
  by_edge_bucket: Record<string, AttributionSliceStats>
  generated_at: string
  error?: string
}

export interface DataValidateCheck {
  name: string
  status: string
  message: string
  symbol?: string
  timeframe?: string
}

export interface DataValidateResponse {
  passed: boolean
  passed_count: number
  warned_count: number
  failed_count: number
  checks: DataValidateCheck[]
}

export interface IndicatorParamDef {
  name: string
  type: 'int' | 'float' | 'string'
  default: number | string
  min?: number
  max?: number
  step?: number
  options?: string[]
  description?: string
}

export interface IndicatorPlotOptions {
  color: string
  lineWidth?: number
  precision?: number
  minMove?: number
}

export interface IndicatorOutputMeta {
  name: string
  type: 'line' | 'histogram'
  plotOptions?: IndicatorPlotOptions
}

export interface IndicatorSpec {
  id: string
  name: string
  description: string
  overlay: boolean
  parameters: IndicatorParamDef[]
  outputs: IndicatorOutputMeta[]
  warmup: number
}

export interface IndicatorPoint {
  time: number
  values: Record<string, number>
}

export interface IndicatorResult {
  id: string
  name: string
  overlay: boolean
  outputs: IndicatorOutputMeta[]
  data: IndicatorPoint[]
}

export interface IndicatorComputeRequest {
  parameters: Record<string, number | string>
  candles: Candle[]
}

export interface IndicatorComputeResponse {
  indicator: IndicatorResult
  metadata: IndicatorSpec
}

export interface IndicatorWithData {
  _id: string
  spec: IndicatorSpec
  result: IndicatorResult | null
  parameters: Record<string, number | string>
  paneIndex: number
  dataVersion: number
  loading: boolean
  error: string | null
}

export interface SimulateGenerateRequest {
  symbols: number
  days: number
  regimes: {
    calm: number
    trending: number
    highVol: number
    crisis: number
  }
  signals?: {
    trend?: boolean
    meanReversion?: boolean
    breakout?: boolean
    strength?: number
  }
}

export interface SimulateGenerateResponse {
  progress?: number
  download_url?: string
  status: string
  error?: string
}

export interface SimulateCalibrateRequest {
  symbol: string
  timeframe: string
  start_date: string
  end_date: string
}

export interface SimulateCalibrateResponse {
  transition_matrix?: number[][]
  state_means?: Record<string, number[]>
  regimes?: string[]
  error?: string
}

export interface SimulateValidateResponse {
  regime_persistence?: {
    passed: boolean
    score: number
  }
  coverage?: {
    passed: boolean
    score: number
  }
  signal_quality?: {
    passed: boolean
    score: number
  }
  overall_passed?: boolean
  error?: string
}

export interface ParamVersion {
  id: string
  strategy_id: string
  version_tag: string
  params: Record<string, number>
  in_sample_start?: string
  in_sample_end?: string
  oos_sharpe?: number
  oos_max_dd?: number
  oos_return_pct?: number
  objective_score?: number
  is_active: boolean
  created_at: string
}

export interface OrchestrationRun {
  id: string
  created_at: string
  completed_at?: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
  start_date: string
  end_date: string
  initial_capital: number
  strategy_ids: string[]
  symbol_tf_pairs: string[]
  pool_sharpe?: number
  pool_sortino?: number
  pool_maxdd?: number
  pool_return_pct?: number
  rebalance_costs?: number
  result_json?: OrchestrationRunResult
}

export interface OrchestrationRunResult {
  pool_equity?: EquityPoint[]
  pool_sharpe: number
  pool_sortino: number
  pool_maxdd: number
  pool_return_pct: number
  rebalance_costs: number
  trades?: TradeSummary[]
  strategy_pnl?: Record<string, number>
  active_count?: number[]
  allocation_history?: AllocationEntry[]
  correlation_breaches?: BreachEvent[]
  daily_returns?: DailyReturn[]
  monthly_returns?: MonthlyReturn[]
  monte_carlo?: MCOrchResult
  per_strategy_stats?: Record<string, StrategyStats>
  win_rate?: number
  profit_factor?: number
  num_trades?: number
  num_wins?: number
  num_losses?: number
}

export interface MCOrchResult {
  config: { iterations: number; bars_per_sim: number }
  iterations: Array<{ pnl_pct: number; max_dd_pct: number }>
  summary: MCOrchSummary
  pass_probability: number
}

export interface MCOrchSummary {
  num_simulations: number
  num_days: number
  avg_pnl_pct: number
  median_pnl_pct: number
  p5_pnl_pct: number
  p10_pnl_pct: number
  avg_max_dd_pct: number
  median_max_dd_pct: number
  p95_max_dd_pct: number
  bust_probability: number
}

export interface StrategyStats {
  num_trades: number
  win_rate: number
  profit_factor: number
  total_pnl: number
}
export interface OrchestrationStrategy {
  strategy_id: string
  symbol: string
  timeframe: string
}

export interface OrchestrationSubmitRequest {
  strategies: OrchestrationStrategy[]
  start_date: string
  end_date: string
  initial_capital?: number
  rebalance_bars?: number
  kelly_fraction?: number
  max_position_pct?: number
  allow_fractional?: boolean
  enable_correlation_brake?: boolean
  correlation_threshold?: number
  friction_model?: string
}

export interface AllocationEntry {
  bar_time: string
  strategy_id: string
  weight: number
  allocated_capital: number
  position_size?: number
  is_active: boolean
}

export interface StrategyStatus {
  strategy_id: string
  status: 'active' | 'inactive' | 'standby' | 'violated' | 'validated'
  allocation_pct: number
  trailing_sharpe?: number
  trailing_sortino?: number
  trailing_maxdd?: number
  last_signal_at?: string
  active_since?: string
  demoted_at?: string
  demotion_reason?: string
  orchestration_run_id?: string
  last_evaluated: string
  updated_at: string
}

export interface BreachEvent {
  strategy_a: string
  strategy_b: string
  correlation: number
  action: 'brake_applied' | 'brake_released' | 'brake_applied_velocity'
}
