package metrics

import "time"

type PerformanceSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	Equity         float64   `json:"equity"`
	Balance        float64   `json:"balance"`
	DailyPnL       float64   `json:"daily_pnl"`
	DailyPnLPct    float64   `json:"daily_pnl_pct"`
	DrawdownPct    float64   `json:"drawdown_pct"`
	MaxDrawdownPct float64   `json:"max_drawdown_pct"`
	CAGR           float64   `json:"cagr"`
	Sharpe         float64   `json:"sharpe"`
	Sortino        float64   `json:"sortino"`
	Calmar         float64   `json:"calmar"`
	WinRate        float64   `json:"win_rate"`
	ProfitFactor   float64   `json:"profit_factor"`
	VaR95          float64   `json:"var_95"`
	CVaR95         float64   `json:"cvar_95"`
	UlcerIndex     float64   `json:"ulcer_index"`
	NumTrades      int       `json:"num_trades"`
	CommissionBps  float64   `json:"commission_bps,omitempty"`
	TotalCommission float64  `json:"total_commission,omitempty"`
}

type MetricEquityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Equity    float64   `json:"equity"`
	Balance   float64   `json:"balance"`
	Drawdown  float64   `json:"drawdown"`
}

type TradeSummary struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Quantity     float64   `json:"quantity"`
	EntryPrice   float64   `json:"entry_price"`
	ExitPrice    float64   `json:"exit_price"`
	PnL          float64   `json:"pnl"`
	PnLPct       float64   `json:"pnl_pct"`
	EntryTime    time.Time `json:"entry_time"`
	ExitTime     time.Time `json:"exit_time"`
	HoldDuration float64   `json:"hold_duration"`
	MAE          float64   `json:"mae"`
	MFE          float64   `json:"mfe"`
	StrategyID   string    `json:"strategy_id"`
	ExitReason   string    `json:"exit_reason"`
	Commission   float64   `json:"commission"`
}

type DailyReturn struct {
	Date      time.Time `json:"date"`
	ReturnPct float64   `json:"return_pct"`
	PnL       float64   `json:"pnl"`
}

type MonthlyReturn struct {
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	ReturnPct float64 `json:"return_pct"`
}

type RollingMetric struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Window    int       `json:"window"`
}

type OptimizationFootprint struct {
	DeflatedSharpe      float64 `json:"deflated_sharpe"`
	ConventionalSharpe  float64 `json:"conventional_sharpe"`
	GridPasses          int     `json:"grid_passes"`
	BayesianIterations  int     `json:"bayesian_iterations"`
	WalkForwardWindows  int     `json:"walk_forward_windows"`
	PassedWindows       int     `json:"passed_windows"`
	IVS                 float64 `json:"ivs"`
	OOSAverageSharpe    float64 `json:"oos_average_sharpe"`
	SharpeDegradation   float64 `json:"sharpe_degradation"`
	BestParamsJSON      string  `json:"best_params_json"`
}
