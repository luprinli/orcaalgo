package backtest

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/hash"
	"github.com/lee-econ/orca-core/internal/market"
	"github.com/lee-econ/orca-core/internal/ml"
	"github.com/lee-econ/orca-core/internal/model"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/propfirm"
	"github.com/lee-econ/orca-core/internal/risk"
	strategy "github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
	"golang.org/x/sync/errgroup"
)

type Candle = strategy.Candle
type Signal = strategy.Signal

type UniverseSnapshot struct {
	Date    time.Time
	Symbols []string
}

const (
	minTradesForMetrics  = 5   // minimum trades to compute Sharpe/Sortino
	minTradesForReliable = 20  // minimum trades for statistically meaningful metrics
)

type BacktestConfig struct {
	StrategyID            string
	Symbols               []string
	StartDate             time.Time
	EndDate               time.Time
	InitialCapital        float64
	RegimeFilter          *int8
	SlippageModel         SlippageModel
	CommissionBps         float64
	BrokerFee             broker.BrokerageFeeConfig
	PropFirmEnabled       bool
	StopLoss              *StopLossConfig
	TakeProfit            *TakeProfitConfig
	DataSource            string
	UseUniverseSnapshots  bool
	UniverseConfigID      string
	Timeframe             string
	ApplyGate             bool
	GateProfile           string
	UseSeasonalityOverlay bool
	StrategyParams        map[string]float64
	FixedSeed             int64
	SizingPercent         float64
	KellyFraction         float64
	EngineVersion         string `json:"engine_version,omitempty"`
	StrategyHash          string `json:"strategy_hash,omitempty"`
	GKRPath               string `json:"gkr_path,omitempty"`
	EnablePrefetch        bool   `json:"enable_prefetch,omitempty"`
	WarmUpBars            int               `json:"warmup_bars,omitempty"`
	SecondarySymbols      map[string]string `json:"secondary_symbols,omitempty"` // primary → secondary for pairs trading
	EarningsCalendar      *market.EarningsCalendar `json:"-"`
	SkipEarningsDays      bool                     `json:"skip_earnings_days,omitempty"`
	AdjustmentProvider    market.AdjustmentProvider `json:"-"`
}

type MatrixBacktestConfig struct {
	StrategyIDs       []string  `json:"strategy_ids"`
	Symbols           []string  `json:"symbols"`
	Timeframes        []string  `json:"timeframes"`
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	InitialCapital    float64   `json:"initial_capital"`
	DataSource        string    `json:"data_source"`
	GateProfile       string    `json:"gate_profile"`
	PropFirmEnabled   bool      `json:"propfirm_enabled"`
	SizingPercent     float64   `json:"sizing_percent"`
	KellyFraction     float64   `json:"kelly_fraction"`
	SkipLightOptimize bool      `json:"skip_light_optimize"`
	WirePipeline      bool      `json:"wire_pipeline"`
}

type ComboResult struct {
	RunID              string              `json:"run_id"`
	Symbol             string              `json:"symbol"`
	StrategyID         string              `json:"strategy_id"`
	Timeframe          string              `json:"timeframe"`
	SharpeRatio        float64             `json:"sharpe_ratio"`
	SortinoRatio       float64             `json:"sortino_ratio"`
	MaxDrawdown        float64             `json:"max_drawdown"`
	MaxDrawdownDur     int                 `json:"max_drawdown_duration"`
	TotalReturn        float64             `json:"total_return"`
	WinRate            float64             `json:"win_rate"`
	ProfitFactor       float64             `json:"profit_factor"`
	AvgTrade           float64             `json:"avg_trade"`
	AvgWin             float64             `json:"avg_win"`
	AvgLoss            float64             `json:"avg_loss"`
	NumTrades          int                 `json:"num_trades"`
	NumWins            int                 `json:"num_wins"`
	NumLosses          int                 `json:"num_losses"`
	AvgMAE             float64             `json:"avg_mae"`
	AvgMFE             float64             `json:"avg_mfe"`
	Error              string              `json:"error,omitempty"`
	Warnings           []string            `json:"warnings,omitempty"`
	GatePassed         *bool               `json:"gate_passed,omitempty"`
	AdverseSelectRate  float64             `json:"adverse_selection_rate,omitempty"`
	BestParams         map[string]float64  `json:"best_params,omitempty"`
	Optimized          bool                `json:"optimized"`
	StrategyParams     map[string]float64  `json:"strategy_params,omitempty"`
	EquityCurve        []EquityPoint       `json:"equity_curve,omitempty"`
	Trades             []Trade             `json:"trades,omitempty"`
	LongTrades         int                 `json:"long_trades"`
	ShortTrades        int                 `json:"short_trades"`
	LongWinRate        float64             `json:"long_win_rate"`
	ShortWinRate       float64             `json:"short_win_rate"`
	LongGrossPnL       float64             `json:"long_gross_pnl"`
	ShortGrossPnL      float64             `json:"short_gross_pnl"`
	LongPF             float64             `json:"long_profit_factor"`
	ShortPF            float64             `json:"short_profit_factor"`
	ZeroPnLTrades      int                 `json:"zero_pnl_trades"`
	ExpectedPF         float64             `json:"expected_pf"`
	RewardRiskRatio    float64             `json:"reward_risk_ratio"`
	DailyVolatility    float64             `json:"daily_volatility"`
	TrainPct           float64             `json:"train_pct"`
	MtmSharpeRatio     float64             `json:"mtm_sharpe_ratio,omitempty"`
	MtmMaxDrawdown     float64             `json:"mtm_max_drawdown,omitempty"`
	MLFeatureEnabled   bool                `json:"ml_feature_enabled,omitempty"`
	TotalFees          float64             `json:"total_fees,omitempty"`
	AvgSlippageBps     float64             `json:"avg_slippage_bps,omitempty"`
	CalmarRatio        float64             `json:"calmar_ratio,omitempty"`
	CandleCount        int                 `json:"candle_count,omitempty"`
	FirstCandleTime    time.Time           `json:"first_candle_time,omitempty"`
	LastCandleTime     time.Time           `json:"last_candle_time,omitempty"`
	DeclaredBarsPerDay float64             `json:"declared_bars_per_day,omitempty"`
	EffectiveBarsPerDay float64            `json:"effective_bars_per_day,omitempty"`
}

type MatrixResult struct {
	RunID   string              `json:"run_id"`
	Combos  int                 `json:"total_combos"`
	Results []ComboResult       `json:"results"`
	Config  MatrixBacktestConfig `json:"config"`
}

type Trade struct {
	Symbol           string
	Side             string
	Quantity         float64
	EntryPrice       types.Price
	ExitPrice        types.Price
	EntryTime        time.Time
	ExitTime         time.Time
	PnL              float64
	PnLPct           float64
	HMMRegime        int8
	StrategyID       string
	StopPrice        types.Price
	TakePrice        types.Price
	ExitReason       string
	EndOfData        bool
	BrokerFee        float64
	SlippageMidBps   float64
	SlippageLastBps  float64
	AdverseSelection bool
	MAE              float64
	MFE              float64
	lowestSinceEntry float64
	highestSinceEntry float64
}

type LongShortBreakdown struct {
	LongTrades    int     `json:"long_trades"`
	ShortTrades   int     `json:"short_trades"`
	LongWins      int     `json:"long_wins"`
	ShortWins     int     `json:"short_wins"`
	LongWinRate   float64 `json:"long_win_rate"`
	ShortWinRate  float64 `json:"short_win_rate"`
	LongGrossPnL  float64 `json:"long_gross_pnl"`
	ShortGrossPnL float64 `json:"short_gross_pnl"`
	LongAvgPnL    float64 `json:"long_avg_pnl"`
	ShortAvgPnL   float64 `json:"short_avg_pnl"`
	LongPF        float64 `json:"long_profit_factor"`
	ShortPF       float64 `json:"short_profit_factor"`
	LongAvgMAE    float64 `json:"long_avg_mae"`
	ShortAvgMAE   float64 `json:"short_avg_mae"`
	LongAvgMFE    float64 `json:"long_avg_mfe"`
	ShortAvgMFE   float64 `json:"short_avg_mfe"`
	DirectionalBias float64 `json:"directional_bias"`
}

type BacktestResult struct {
	Config         BacktestConfig
	Trades         []Trade
	SharpeRatio    float64
	SortinoRatio   float64
	MaxDrawdown    float64
	MaxDrawdownDuration int
	TotalReturn    float64
	TotalReturnPct float64
	WinRate        float64
	ProfitFactor   float64
	AvgTrade       float64
	AvgWin         float64
	AvgLoss        float64
	NumTrades      int
	NumWins        int
	NumLosses      int
	AdverseSelectionRate float64
	AvgMAE        float64
	AvgMFE        float64
	RegimeStats    []RegimeStat
	EquityCurve    []EquityPoint
	MtmEquity      []EquityPoint `json:"mtm_equity,omitempty"`
	MtmSharpeRatio float64       `json:"mtm_sharpe_ratio,omitempty"`
	MtmMaxDrawdown float64       `json:"mtm_max_drawdown,omitempty"`
	DailyReturns   []DailyReturn
	TemporalBreakdown TemporalBreakdown
	ComplianceReport     *ComplianceReport
	CompletedAt    time.Time
	RegimeLogError string          `json:"regime_log_error,omitempty"`
	Warnings       []string        `json:"warnings,omitempty"`
	MetricGateStatus *MultiMetricVerdict `json:"metric_gate_status,omitempty"`
	StrategyParams   map[string]float64   `json:"strategy_params,omitempty"`
	CalmarRatio      float64              `json:"calmar_ratio,omitempty"`
	TrainPct         float64
	SignalDiag     SignalDiag          `json:"signal_diag,omitempty"`
	EngineVersion  string          `json:"engine_version,omitempty"`
	StrategyHash   string          `json:"strategy_hash,omitempty"`
	SchemaVersion  int             `json:"schema_version,omitempty"`
	LongShort      LongShortBreakdown `json:"long_short,omitempty"`
	MLFeatureEnabled bool             `json:"ml_feature_enabled,omitempty"`
	TotalFees        float64             `json:"total_fees,omitempty"`
	AvgSlippageBps   float64             `json:"avg_slippage_bps,omitempty"`
	CandleCount      int                 `json:"candle_count,omitempty"`
	FirstCandleTime  time.Time           `json:"first_candle_time,omitempty"`
	LastCandleTime   time.Time           `json:"last_candle_time,omitempty"`
	EffectiveBarsPerDay float64          `json:"effective_bars_per_day,omitempty"`
	DeclaredBarsPerDay  float64          `json:"declared_bars_per_day,omitempty"`
}

type TemporalBreakdown struct {
	Yearly  []PeriodStat `json:"yearly"`
	Monthly []PeriodStat `json:"monthly"`
	Weekly  []PeriodStat `json:"weekly"`
	Daily   []PeriodStat `json:"daily"`
}

type PeriodStat struct {
	Period       string  `json:"period"`
	NetPnL       float64 `json:"net_pnl"`
	NumTrades    int     `json:"num_trades"`
	WinRate      float64 `json:"win_rate"`
	GrossProfit  float64 `json:"gross_profit"`
	GrossLoss    float64 `json:"gross_loss"`
	Commission   float64 `json:"commission"`
	BrokerFees   float64 `json:"broker_fees"`
}

type RegimeStat struct {
	Regime       int8    `json:"regime"`
	Label        string  `json:"label"`
	NumTrades    int     `json:"num_trades"`
	WinRate      float64 `json:"win_rate"`
	TotalReturn  float64 `json:"total_return"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	ProfitFactor float64 `json:"profit_factor"`
}

type SignalDiag struct {
	CandlesSeen       int `json:"candles_seen"`
	SignalAttempts    int `json:"signal_attempts"`
	CapitalZero       int `json:"capital_zero"`
	VolHalted         int `json:"vol_halted"`
	RateLimited       int `json:"rate_limited"`
	BaseSizeZero      int `json:"base_size_zero"`
	StrategyNil       int `json:"strategy_nil"`
	ExitSignalZeroQty int `json:"exit_signal_zero_qty"`
	QuantityTooSmall  int `json:"quantity_too_small"`
	ExposureBlocked   int `json:"exposure_blocked"`
	PipelineRejected  int `json:"pipeline_rejected"`
	SignalsPassed     int `json:"signals_passed"`
	TradesOpened      int `json:"trades_opened"`
	FillRejected      int `json:"fill_rejected"`
	MLRejected        int `json:"ml_rejected"`
}

type EquityPoint struct {
	Time   time.Time `json:"time"`
	Value  float64   `json:"value"`
	Regime int8      `json:"regime"`
}

type DailyReturn struct {
	Date   time.Time `json:"date"`
	Return float64   `json:"return"`
}

type Engine struct {
	db             Database
	fillSim        *FillSimulator
	feeModel       model.FeeModel
	latencyModel   model.LatencyModel
	recorder       model.Recorder
	stratBySymbol  map[string]strategy.Strategy
	ftmo           *PropFirmEnforcer
	orderLimiter   *risk.OrderRateLimiter
	volHalt        *risk.VolatilityHalt
	exposure       *risk.ExposureTracker
	kellyMult      float64
	positionSizer  *risk.PositionSizer
	signalDiag     SignalDiag
	metaLabeler    ml.Predictor
	batchInferrer  *ml.BatchInferrer
	metaCfg        ml.MetaLabelerConfig
	regimeEnhancer *ml.RegimeEnhancer
	exitOrch       *ml.ExitOrchestrator
	pipeline       *risk.RiskPipeline
	featureStore   *ml.FeatureStore
}

type Database interface {
	LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error)
	LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]Candle, error)
	LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error)
	LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe string, source string) ([][]Candle, error)
	LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]Candle, error)
	LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error)
	LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error)
	LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]SentimentLog, error)
	CountCandles(ctx context.Context) (int64, error)
	CountSyntheticCandles(ctx context.Context) (int64, error)
	CountRegimeLogs(ctx context.Context) (int64, error)
	LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]UniverseSnapshot, error)
}

type RegimeLog struct {
	Time       time.Time
	HMMState   int8
	Confidence float64
	Symbol     string
}

type VIXLog struct {
	Time      time.Time
	VIXValue  float64
	VIXChange float64
}

type SentimentLog struct {
	Time  time.Time
	Score int
	Label string
}

func NewEngine(db Database) *Engine {
	return NewEngineBuilder(db).Build()
}

// SetMetaLabeler configures the ML meta-labeling subsystem.
func (e *Engine) SetMetaLabeler(predictor ml.Predictor) error {
	e.metaLabeler = predictor
	var err error
	e.batchInferrer = ml.NewBatchInferrer(predictor, e.metaCfg)
	return err
}

// SetMetaLabelerConfig updates the meta-labeling configuration.
func (e *Engine) SetMetaLabelerConfig(cfg ml.MetaLabelerConfig) {
	e.metaCfg = cfg
	if e.batchInferrer != nil {
		e.batchInferrer = ml.NewBatchInferrer(e.metaLabeler, cfg)
	}
}

// SetRegimeEnhancer configures the ML regime enhancement subsystem.
func (e *Engine) SetRegimeEnhancer(enhancer *ml.RegimeEnhancer) {
	e.regimeEnhancer = enhancer
}

// SetExitOrchestrator configures the ML exit optimization subsystem.
func (e *Engine) SetExitOrchestrator(orch *ml.ExitOrchestrator) {
	e.exitOrch = orch
}

// SetRiskPipeline injects a shared signal-and-fill audit pipeline. When set,
// ProcessSignal becomes the primary path in generateSignal (inline checks become
// the fallback), and ReconcileFillWithoutPropFirm is called on every trade close.
func (e *Engine) SetRiskPipeline(p *risk.RiskPipeline) {
	e.pipeline = p
}

// SetFeatureStore injects a feature store for ML feature computation.
// When set, generateSignal will compute 21-dim feature vectors from candle data
// and pass them to the batch inferrer instead of nil features.
func (e *Engine) SetFeatureStore(fs *ml.FeatureStore) {
	e.featureStore = fs
}

// WirePipeline creates a SignalGateImpl from the engine's existing volHalt,
// positionSizer, and exposure components, then sets up the canonical
// RiskPipeline. The pipeline becomes the primary path in generateSignal.
// Call this after any SetMetaLabeler / SetRegimeEnhancer calls.
func (e *Engine) WirePipeline() {
	signalGate := risk.NewSignalGateImpl(e.volHalt, e.positionSizer, e.exposure, nil)
	signalGate.SetBacktestMode(true)

	var propFirmGate risk.PropFirmGate
	if e.ftmo != nil {
		propFirmGate = e.ftmo
	}

	e.pipeline = &risk.RiskPipeline{
		SignalGate:   signalGate,
		Capital:      nil,
		PropFirm:     propFirmGate,
		KellyMult:    e.kellyMult,
		RegimeMatrix: risk.NewRegimeActivationMatrix(),
	}
}

// GetDB returns the underlying database handle, enabling callers that need the raw Database
// interface (e.g., RunLightOptimize) to use an already-configured Engine.
func (e *Engine) GetDB() Database {
	return e.db
}

func NewEngineWithFixedSeed(db Database, seed int64) *Engine {
	return NewEngineBuilder(db).WithSeed(seed).Build()
}

func NewEngineWithSlippage(db Database, model SlippageModel) *Engine {
	return NewEngineBuilder(db).WithSlippage(model).Build()
}

func NewEngineWithStrategy(db Database, sr strategy.Strategy) *Engine {
	return NewEngineBuilder(db).WithStrategy(sr).Build()
}

func NewEngineWithSlippageAndStrategy(db Database, model SlippageModel, sr strategy.Strategy) *Engine {
	return NewEngineBuilder(db).WithSlippage(model).WithStrategy(sr).Build()
}

func (e *Engine) Run(ctx context.Context, config BacktestConfig) (result *BacktestResult, err error) {
	e.resetStrategies()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("engine panic recovered: %v", r)
			result = nil
		}
	}()

	var (
		regimeLogs       []RegimeLog
		vixLogs          []VIXLog
		sentimentLogs    []SentimentLog
		candlesBySymbol  [][]Candle
		candleErr        error
	)

	if config.EnablePrefetch {
		var g errgroup.Group

		g.Go(func() error {
			var err error
			regimeLogs, err = e.db.LoadRegimeLogs(ctx, config.StartDate, config.EndDate)
			if err != nil { regimeLogs = nil }
			return nil
		})
		g.Go(func() error {
			var err error
			vixLogs, err = e.db.LoadVIXLogs(ctx, config.StartDate, config.EndDate)
			if err != nil { vixLogs = nil }
			return nil
		})
		g.Go(func() error {
			var err error
			sentimentLogs, err = e.db.LoadSentimentLogs(ctx, config.StartDate, config.EndDate)
			if err != nil { sentimentLogs = nil }
			return nil
		})
		g.Go(func() error {
			var err error
			if config.Timeframe != "" && config.Timeframe != "1d" {
				candlesBySymbol, err = e.db.LoadCandlesTFFiltered(ctx, config.Symbols, config.StartDate, config.EndDate, config.Timeframe, config.DataSource)
			} else {
				candlesBySymbol, err = e.db.LoadCandlesFiltered(ctx, config.Symbols, config.StartDate, config.EndDate, config.DataSource)
			}
			candleErr = err
			return nil
		})
		_ = g.Wait()
	} else {
		regimeLogs, _ = e.db.LoadRegimeLogs(ctx, config.StartDate, config.EndDate)
		vixLogs, _ = e.db.LoadVIXLogs(ctx, config.StartDate, config.EndDate)
		sentimentLogs, _ = e.db.LoadSentimentLogs(ctx, config.StartDate, config.EndDate)
		if config.Timeframe != "" && config.Timeframe != "1d" {
			candlesBySymbol, candleErr = e.db.LoadCandlesTFFiltered(ctx, config.Symbols, config.StartDate, config.EndDate, config.Timeframe, config.DataSource)
		} else {
			candlesBySymbol, candleErr = e.db.LoadCandlesFiltered(ctx, config.Symbols, config.StartDate, config.EndDate, config.DataSource)
		}
	}

	if candleErr != nil {
		return nil, candleErr
	}

	result = &BacktestResult{Config: config, SchemaVersion: 1}

	if config.EngineVersion == "" {
		config.EngineVersion = "dev"
	}
	result.EngineVersion = config.EngineVersion
	result.StrategyHash = config.StrategyHash

	var universeSnapshots map[time.Time][]string
	if config.UseUniverseSnapshots {
		snaps, snapErr := e.db.LoadUniverseSnapshots(ctx, config.StartDate, config.EndDate)
		if snapErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("universe_snapshots: failed to load (%v), backtest running unfiltered", snapErr))
		} else if len(snaps) > 0 {
			universeSnapshots = make(map[time.Time][]string, len(snaps))
			for _, snap := range snaps {
				dateKey := snap.Date.Truncate(24 * time.Hour)
				universeSnapshots[dateKey] = snap.Symbols
			}
		}
	}

	allCandles := mergeCandlesByTime(candlesBySymbol)
	result.CandleCount = len(allCandles)
	if len(allCandles) > 0 {
		result.FirstCandleTime = allCandles[0].Time
		result.LastCandleTime = allCandles[len(allCandles)-1].Time
	}

	if config.CommissionBps <= 0 {
		config.CommissionBps = 5.0
	}
	if !config.BrokerFee.Enabled && config.BrokerFee.PerTradeFixed == 0 {
		config.BrokerFee = broker.DefaultBrokerageFee()
	}
	if config.StrategyHash == "" && config.GKRPath != "" {
		if h, err := hash.ComputeInstanceHash(config.GKRPath); err == nil {
			config.StrategyHash = h
		}
	}
	if config.FixedSeed != 0 {
		e.fillSim = NewFillSimulatorWithSeed(e.fillSim.model, config.FixedSeed)
	}

	if regimeLogs == nil {
		result.Warnings = append(result.Warnings, "regime_logs: failed to load, all candles treated as regime 0 (Calm)")
	}
	capital := config.InitialCapital
	peakCapital := config.InitialCapital
	equity := []EquityPoint{}
	mtmEquity := []EquityPoint{}
	trades := []Trade{}

	e.ftmo = nil
	if config.PropFirmEnabled {
		e.ftmo = DefaultPropFirmEnforcer(config.InitialCapital)
	}

	// Allow the optimizer to tune the fractional Kelly multiplier per run.
	// Hard cap at 0.25 per HP #6: fractional Kelly is mandatory in both backtest and live paths.
	if config.KellyFraction > 0 {
		e.kellyMult = config.KellyFraction
	}
	if e.kellyMult > 0.25 {
		e.kellyMult = 0.25
	}

	openTrades := make(map[string]*Trade)
	activeStops := make(map[string]*ActiveStop)
	pendingAS := make(map[string]*Trade)
	var lastDay string
	var atrWindow []Candle
	var hasHighVIX bool
	var hasVIXSpike bool
	var prevVIX float64
	atrPeriod := 14
	if config.StopLoss != nil && config.StopLoss.ATRPeriod > 0 {
		atrPeriod = config.StopLoss.ATRPeriod
	}

	declaredBarsPerDay := barsPerDayFromTimeframe(config.Timeframe)
	barsPerDay := declaredBarsPerDay

	for i := range allCandles {
		candle := allCandles[i]
		e.signalDiag.CandlesSeen++

		if universeSnapshots != nil {
			dateKey := candle.Time.Truncate(24 * time.Hour)
			if activeSymbols, ok := universeSnapshots[dateKey]; ok {
				found := false
				for _, sym := range activeSymbols {
					if sym == candle.Symbol {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
		}

		atrWindow = append(atrWindow, candle)
		if len(atrWindow) > atrPeriod*3 {
			atrWindow = atrWindow[len(atrWindow)-atrPeriod*3:]
		}

		currentDay := candle.Time.Format("2006-01-02")
		if currentDay != lastDay && e.ftmo != nil {
			e.ftmo.OnNewDay()
			lastDay = currentDay
		}

		if e.ftmo != nil && config.PropFirmEnabled && e.ftmo.NewsTradingRestricted {
			if !e.ftmo.CheckNewsTrading(candle.Time) {
				continue
			}
		}

		regime := getRegimeAt(candle.Time, regimeLogs)
		vix := getVIXAt(candle.Time, vixLogs)
		sentiment := getSentimentAt(candle.Time, sentimentLogs)
		if vix > 25.0 {
			hasHighVIX = true
		}
		if prevVIX > 0 && vix > 0 {
			vixDelta := math.Abs(vix - prevVIX)
			if vixDelta > 3.0 {
				hasVIXSpike = true
			}
		}
		prevVIX = vix
		e.positionSizer.UpdateMarketState(vix, sentiment, regime)

		sr := e.getRunnerForSymbolAndStrategy(candle.Symbol, config.StrategyID, config)
		if receiver, ok := sr.(strategy.VIXReceiver); ok {
			receiver.SetVIX(vix)
		}
		if atrReceiver, ok := sr.(strategy.ATRReceiver); ok {
			atrVal := ComputeATR(atrWindow, atrPeriod)
			if config.EarningsCalendar != nil && config.EarningsCalendar.IsEarningsDay(candle.Symbol, candle.Time) {
				atrVal *= 1.2
				if config.SkipEarningsDays {
					continue
				}
				log.Printf("backtest: earnings day for %s on %s — ATR adjusted x1.2", candle.Symbol, candle.Time.Format("2006-01-02"))
			}
			atrReceiver.SetATR(atrVal)
		}
		if pendingTrade, ok := pendingAS[candle.Symbol]; ok {
			if pendingTrade.Side == "BUY" && candle.Close.Float64() < pendingTrade.EntryPrice.Float64() {
				pendingTrade.AdverseSelection = true
			}
			if pendingTrade.Side == "SELL" && candle.Close.Float64() > pendingTrade.EntryPrice.Float64() {
				pendingTrade.AdverseSelection = true
			}
			delete(pendingAS, candle.Symbol)
		}
		if e.ftmo != nil {
			e.ftmo.CurrentRegime = regime
		}

		if config.RegimeFilter != nil && regime != *config.RegimeFilter {
			continue
		}

		if e.ftmo != nil && e.ftmo.IsHalted() {
			continue
		}

		for sym, ot := range openTrades {
			if stop, ok := activeStops[sym]; ok && stop.StopType == StopLossTrail {
				UpdateTrailingStop(stop, candle)
			}

			// ML dynamic exit: adjust stop based on exit urgency model
			if e.exitOrch != nil && e.exitOrch.IsHealthy() {
				if stop, ok := activeStops[sym]; ok && stop.StopPrice.Float64() > 0 {
					exitCtx := ml.ExitContext{
						EntryPrice:     stop.EntryPrice,
						CurrentPrice:   candle.Close,
						CurrentStop:    stop.StopPrice,
						HighSinceEntry: stop.PeakPrice,
						LowSinceEntry:  candle.Low,
						BarsSinceEntry: 10,
						ATR:            stop.ATRValue,
						VolAtEntry:     0.01,
						VolCurrent:     0.01,
						HMMState:       int(regime),
						ADX:            25.0,
						Hour:           float64(candle.Time.Hour()),
					}
					_, mult := e.exitOrch.Evaluate(exitCtx)
					if mult > 0 && stop.EntryPrice.Float64() > 0 {
						if stop.Side == "BUY" {
							dynamicStop := candle.Close.Float64() - mult*stop.ATRValue
							if dynamicStop < stop.EntryPrice.Float64()*0.80 {
								dynamicStop = stop.EntryPrice.Float64() * 0.80
							}
							if dynamicStop > stop.StopPrice.Float64() && stop.StopType == StopLossTrail {
							} else {
								stop.StopPrice = types.PriceFromFloat(dynamicStop)
							}
						} else {
							dynamicStop := candle.Close.Float64() + mult*stop.ATRValue
							if dynamicStop > stop.EntryPrice.Float64()*1.20 {
								dynamicStop = stop.EntryPrice.Float64() * 1.20
							}
							if dynamicStop < stop.StopPrice.Float64() && stop.StopType == StopLossTrail {
							} else {
								stop.StopPrice = types.PriceFromFloat(dynamicStop)
							}
						}
					}
				}
			}

			exitReason := ""
			exitPrice := candle.Close.Float64() * candle.AdjustmentFactor
			fillQty := ot.Quantity
			shouldExit := false

			if stop, ok := activeStops[sym]; ok {
				if stopHit, sp := CheckStopHit(candle, stop); stopHit {
					exitPrice = sp
					exitReason = "stop_loss"
					shouldExit = true
				} else if tpHit, tp := CheckTakeProfitHit(candle, stop); tpHit {
					exitPrice = tp
					exitReason = "take_profit"
					shouldExit = true
				}
			}

			low := candle.Low.Float64()
			high := candle.High.Float64()
			if low < ot.lowestSinceEntry {
				ot.lowestSinceEntry = low
			}
			if high > ot.highestSinceEntry {
				ot.highestSinceEntry = high
			}

			if !shouldExit {
				reverseSignal := e.generateSignalForExit(candle, regime, config, capital)
				if reverseSignal != nil && reverseSignal.Symbol == sym {
					exitReason = "signal_reverse"
					shouldExit = true
				}
			}

			if shouldExit {
				midPrice := (candle.High.Float64() + candle.Low.Float64()) / 2.0
				simulatedExit := e.fillSim.SimulateFillWithTCA(uint32(len(trades)+1), ot.Symbol, ot.EntryPrice.Float64(), fillQty, invertSide(ot.Side), exitPrice, candle.Time, midPrice, candle.Close.Float64()*candle.AdjustmentFactor, candle.Volume)
				if simulatedExit.FillPrice.Float64() > 0 {
					exitPrice = simulatedExit.FillPrice.Float64()
				}
				if simulatedExit.FillQuantity > 0 {
					fillQty = simulatedExit.FillQuantity
				}

				commission := ot.EntryPrice.Float64() * fillQty * config.CommissionBps / 10000.0 * 2
				brokerFee := config.BrokerFee.CalculateFee(fillQty, ot.EntryPrice.Float64()) +
					config.BrokerFee.CalculateFee(fillQty, exitPrice)

			if ot.Side == "BUY" {
				ot.PnL = (exitPrice-ot.EntryPrice.Float64())*fillQty - commission - brokerFee
			} else {
				ot.PnL = (ot.EntryPrice.Float64()-exitPrice)*fillQty - commission - brokerFee
			}
			safe, clamped := risk.SanitizeTradePnL(ot.PnL, fillQty, ot.EntryPrice.Float64(), config.InitialCapital)
			ot.PnL = safe
			if clamped {
				exitReason = "pnl_clamped"
			}
				ot.PnLPct = ot.PnL / config.InitialCapital * 100
				ot.ExitPrice = types.PriceFromFloat(exitPrice)
				ot.ExitTime = candle.Time
				ot.Quantity = fillQty
				ot.HMMRegime = regime
				ot.ExitReason = exitReason
				ot.BrokerFee = brokerFee

				entry := ot.EntryPrice.Float64()
				if entry > 0 {
					if ot.Side == "BUY" {
						ot.MAE = (entry - ot.lowestSinceEntry) / entry * 100.0
						ot.MFE = (ot.highestSinceEntry - entry) / entry * 100.0
					} else {
						ot.MAE = (ot.highestSinceEntry - entry) / entry * 100.0
						ot.MFE = (entry - ot.lowestSinceEntry) / entry * 100.0
					}
				}

				capital += ot.PnL
			maxCapital := config.InitialCapital * 100
			if capital > maxCapital {
				capital = maxCapital
				result.Warnings = append(result.Warnings, fmt.Sprintf("capital clamped at %.0fx initial (overflow guard)", maxCapital/config.InitialCapital))
			}
			if capital <= 0 {
				capital = 0
			}
			if e.ftmo != nil {
				e.ftmo.OnFill(ot.PnL, 0)
			}
			if e.pipeline != nil {
				e.pipeline.ReconcileFillWithoutPropFirm(ot.StrategyID, ot.Symbol, ot.Side, ot.PnL, ot.Quantity, ot.ExitPrice.Float64())
			}
				trades = append(trades, *ot)
				delete(openTrades, sym)
				delete(activeStops, sym)
			}
		}

		if _, alreadyOpen := openTrades[candle.Symbol]; !alreadyOpen {
			// MaxDD guard (E7): stop entering new positions at 80% drawdown.
			if peakCapital > 0 {
				currentDD := (peakCapital - capital) / peakCapital
				if currentDD > 0.80 {
					e.signalDiag.SignalAttempts++
					continue
				}
			}
			isSecondary := false
			if len(config.SecondarySymbols) > 0 {
				for primary := range config.SecondarySymbols {
					if secondary := config.SecondarySymbols[primary]; secondary == candle.Symbol {
						if sr := e.getRunnerForSymbolAndStrategy(primary, config.StrategyID, config); sr != nil {
							if receiver, ok := sr.(strategy.SecondaryPriceReceiver); ok {
								receiver.PushSecondaryPrice(candle.Close)
							}
						}
						isSecondary = true
						break
					}
				}
			}
			if isSecondary {
				e.signalDiag.SignalAttempts++
				continue
			}
			e.signalDiag.SignalAttempts++
			if i < config.WarmUpBars {
				sr := e.getRunnerForSymbolAndStrategy(candle.Symbol, config.StrategyID, config)
				if sr != nil {
					sr.Evaluate(candle, regime)
				}
				continue
			}
			signal := e.generateSignal(candle, regime, config, capital)
			if signal != nil {
			e.signalDiag.TradesOpened++
			midPrice := (candle.High.Float64() + candle.Low.Float64()) / 2.0
			simulatedEntry := e.fillSim.SimulateFillWithTCA(uint32(len(trades)+1), candle.Symbol, candle.Close.Float64()*candle.AdjustmentFactor, signal.Quantity, signal.Side, candle.Close.Float64()*candle.AdjustmentFactor, candle.Time, midPrice, candle.Close.Float64()*candle.AdjustmentFactor, candle.Volume)
			entryPrice := simulatedEntry.FillPrice.Float64()
			entryQty := simulatedEntry.FillQuantity
			entrySlippageMid := simulatedEntry.SlippageMidBps
			entrySlippageLast := simulatedEntry.SlippageLastBps
			if entryPrice <= 0 {
				entryPrice = candle.Close.Float64()
			}
			if entryQty <= 0 {
				entryQty = signal.Quantity
			}
			if entryQty <= 0 {
				e.signalDiag.FillRejected++
				entryQty = 0
			}

			atrVal := ComputeATR(atrWindow, atrPeriod)
			stopPrice := 0.0
			takePrice := 0.0
			var stop *ActiveStop

			if config.StopLoss != nil && config.StopLoss.Type != StopLossNone {
				stopPrice = CalculateStopPrice(entryPrice, signal.Side, config.StopLoss, atrVal, candle.High.Float64())
				if config.TakeProfit != nil && config.TakeProfit.Type != TakeProfitNone {
					takePrice = CalculateTakeProfitPrice(entryPrice, signal.Side, config.TakeProfit, stopPrice, atrVal)
				}
				stop = &ActiveStop{
					TradeID:    len(trades) + 1,
					EntryPrice: types.PriceFromFloat(entryPrice),
					Side:       signal.Side,
					StopPrice:  types.PriceFromFloat(stopPrice),
					TakePrice:  types.PriceFromFloat(takePrice),
					PeakPrice:  types.PriceFromFloat(entryPrice),
					ATRValue:   atrVal,
					StopType:   config.StopLoss.Type,
				}
				if config.TakeProfit != nil {
					stop.TakeType = config.TakeProfit.Type
				}
				activeStops[candle.Symbol] = stop
			}

			newTrade := &Trade{
				Symbol:           candle.Symbol,
				Side:             signal.Side,
				Quantity:         entryQty,
				EntryPrice:       types.PriceFromFloat(entryPrice),
				EntryTime:        candle.Time,
				HMMRegime:        regime,
				StrategyID:       config.StrategyID,
				StopPrice:        types.PriceFromFloat(stopPrice),
				TakePrice:        types.PriceFromFloat(takePrice),
				SlippageMidBps:   entrySlippageMid,
				SlippageLastBps:  entrySlippageLast,
				lowestSinceEntry:  entryPrice,
				highestSinceEntry: entryPrice,
			}
			openTrades[candle.Symbol] = newTrade
			pendingAS[candle.Symbol] = newTrade
			}
		}

		if i > 0 && allCandles[i-1].Close > 0 && candle.Close > 0 {
			ret := (candle.Close.Float64() - allCandles[i-1].Close.Float64()) / allCandles[i-1].Close.Float64()
			e.volHalt.UpdateReturn(ret)
		}

		equity = append(equity, EquityPoint{
			Time:   candle.Time,
			Value:  capital,
			Regime: regime,
		})

		var unrealizedPnL float64
		for _, ot := range openTrades {
			if ot.Side == "BUY" {
				unrealizedPnL += (candle.Close.Float64() - ot.EntryPrice.Float64()) * ot.Quantity
			} else {
				unrealizedPnL += (ot.EntryPrice.Float64() - candle.Close.Float64()) * ot.Quantity
			}
		}
		mtmEquity = append(mtmEquity, EquityPoint{
			Time:   candle.Time,
			Value:  capital + unrealizedPnL,
			Regime: regime,
		})

		if e.recorder != nil {
			midPrice := (candle.High.Float64() + candle.Low.Float64()) / 2.0
			var position, volume, value float64
			for _, ot := range openTrades {
				position += ot.Quantity
				volume += ot.Quantity
				value += ot.EntryPrice.Float64() * ot.Quantity
			}
			e.recorder.Record(&model.TradingState{
				Timestamp:     candle.Time,
				Balance:       capital,
				Position:      position,
				MidPrice:      uint64(midPrice * 100000),
				TradingVolume: volume,
				TradingValue:  value,
				NumTrades:     int64(len(trades)),
			}, nil)
		}

		if capital > peakCapital {
			peakCapital = capital
		}
	}

	if len(allCandles) == 0 {
		result.Trades = trades
		result.EquityCurve = equity
		result.MtmEquity = mtmEquity
		result.NumTrades = 0
		result.TotalReturnPct = 0
		result.TotalReturn = 0
		result.CompletedAt = time.Now()
		result.Warnings = append(result.Warnings, "no candle data available for requested symbol/timeframe combination")
		return result, nil
	}

	lastCandle := allCandles[len(allCandles)-1]
	for _, ot := range openTrades {
		exitPrice := lastCandle.Close.Float64() * lastCandle.AdjustmentFactor
		exitReason := "end_of_data"
		if stop, ok := activeStops[ot.Symbol]; ok {
			if stopHit, sp := CheckStopHit(lastCandle, stop); stopHit {
				exitPrice = sp
				exitReason = "stop_loss"
			} else if tpHit, tp := CheckTakeProfitHit(lastCandle, stop); tpHit {
				exitPrice = tp
				exitReason = "take_profit"
			}
		}
		commission := ot.EntryPrice.Float64() * ot.Quantity * config.CommissionBps / 10000.0 * 2
		brokerFee := config.BrokerFee.CalculateFee(ot.Quantity, ot.EntryPrice.Float64()) +
			config.BrokerFee.CalculateFee(ot.Quantity, exitPrice)
	if ot.Side == "BUY" {
		ot.PnL = (exitPrice-ot.EntryPrice.Float64())*ot.Quantity - commission - brokerFee
	} else {
		ot.PnL = (ot.EntryPrice.Float64()-exitPrice)*ot.Quantity - commission - brokerFee
	}
	safe, clamped := risk.SanitizeTradePnL(ot.PnL, ot.Quantity, ot.EntryPrice.Float64(), config.InitialCapital)
	ot.PnL = safe
	if clamped {
		exitReason = "pnl_clamped"
	}
		ot.PnLPct = ot.PnL / config.InitialCapital * 100
		ot.ExitPrice = types.PriceFromFloat(exitPrice)
		ot.ExitTime = lastCandle.Time
		ot.ExitReason = exitReason
		ot.EndOfData = true
		ot.BrokerFee = brokerFee

		entry := ot.EntryPrice.Float64()
		if entry > 0 {
			if ot.Side == "BUY" {
				ot.MAE = (entry - ot.lowestSinceEntry) / entry * 100.0
				ot.MFE = (ot.highestSinceEntry - entry) / entry * 100.0
			} else {
				ot.MAE = (ot.highestSinceEntry - entry) / entry * 100.0
				ot.MFE = (entry - ot.lowestSinceEntry) / entry * 100.0
			}
		}

		capital += ot.PnL
		if capital > config.InitialCapital*100 {
			capital = config.InitialCapital * 100
		}
		if capital <= 0 {
			capital = 0
		}
		if e.pipeline != nil {
			e.pipeline.ReconcileFillWithoutPropFirm(ot.StrategyID, ot.Symbol, ot.Side, ot.PnL, ot.Quantity, exitPrice)
		}
		trades = append(trades, *ot)
	}

	result.Trades = trades
	result.EquityCurve = equity
	result.MtmEquity = mtmEquity
	result.NumTrades = len(trades)

	var sumMAE, sumMFE, totalFees, totalSlippage float64
	var slippageCount int
	for _, t := range trades {
		sumMAE += t.MAE
		sumMFE += t.MFE
		totalFees += t.BrokerFee
		if t.SlippageMidBps > 0 {
			totalSlippage += t.SlippageMidBps
			slippageCount++
		}
	}
	if len(trades) > 0 {
		result.AvgMAE = sumMAE / float64(len(trades))
		result.AvgMFE = sumMFE / float64(len(trades))
		result.TotalFees = totalFees
	}
	if slippageCount > 0 {
		result.AvgSlippageBps = totalSlippage / float64(slippageCount)
	}

	result.TotalReturnPct = (capital - config.InitialCapital) / config.InitialCapital * 100
	if math.IsNaN(capital) || math.IsInf(capital, 0) {
		result.TotalReturn = 0
		result.TotalReturnPct = 0
		result.Warnings = append(result.Warnings, "metrics: NaN/Inf capital — clamped to zero")
	}
	result.TotalReturn = capital - config.InitialCapital
	result.CompletedAt = time.Now()

	if e.ftmo != nil {
		report := e.ftmo.Summary()
		result.ComplianceReport = &report
		if !report.Passed {
			monitor.RecordPropfirmBreach(report.HaltReason)
		}
	}

	if hasHighVIX {
		result.Warnings = append(result.Warnings, "VIX exceeded 25 during backtest period: regime detection accuracy may be reduced vs live (VIX modulation not applied to pre-computed regime logs)")
	}
	if hasVIXSpike {
		result.Warnings = append(result.Warnings, "VIX spike detected (daily change > 3 points): volatility regime transition may cause whipsaw losses")
	}

	barsPerDay = effectiveBarsPerDay(allCandles, declaredBarsPerDay)
	result.DeclaredBarsPerDay = declaredBarsPerDay
	result.EffectiveBarsPerDay = barsPerDay
	if declaredBarsPerDay > 0 && math.Abs(barsPerDay-declaredBarsPerDay) > declaredBarsPerDay*0.01 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("barsPerDay corrected: declared=%.1f, effective=%.1f (timeframe label may not match data resolution)",
				declaredBarsPerDay, barsPerDay))
	}

	diag := preflightDataQuality(allCandles)
	if diag != "" {
		result.Warnings = append(result.Warnings, diag)
	}

	if config.DataSource == "synthetic" && (config.StrategyID == "breakout" || config.StrategyID == "opening_range_breakout" || config.StrategyID == "session_scalp" || config.StrategyID == "scalp") {
		result.Warnings = append(result.Warnings, "strategy requires intraday microstructure patterns absent from synthetic data — use real market data for reliable evaluation")
	}

	if result.NumTrades > 0 {
		result.WinRate = calculateWinRate(trades)
		if result.NumTrades >= minTradesForMetrics {
			result.SharpeRatio = calculateSharpe(equity, barsPerDay)
			monitor.SetStrategySharpe(config.StrategyID, "backtest", result.SharpeRatio)
		}
		if result.NumTrades >= minTradesForReliable {
			result.SortinoRatio = calculateSortino(equity, barsPerDay)
			result.MaxDrawdown = calculateMaxDrawdown(equity)
			result.MaxDrawdownDuration = calculateMaxDrawdownDuration(equity)
			result.CalmarRatio = calculateCalmar(equity, barsPerDay)
			result.AvgTrade = result.TotalReturn / float64(result.NumTrades)
			result.AvgWin, result.AvgLoss = calculateAvgWinLoss(trades)
			result.NumWins, result.NumLosses = countWinsLosses(trades)
		}
		if len(mtmEquity) >= 2 && result.NumTrades >= minTradesForMetrics {
			result.MtmSharpeRatio = calculateMtmSharpe(mtmEquity, barsPerDay)
		}
		if result.NumTrades >= minTradesForReliable {
			result.MtmMaxDrawdown = calculateMaxDrawdown(mtmEquity)
		}
		result.ProfitFactor = calculateProfitFactor(trades)
		result.AdverseSelectionRate = calculateAdverseSelectionRate(trades)
		result.DailyReturns = calculateDailyReturns(equity)
		result.RegimeStats = calculateRegimeStats(trades)
		result.TemporalBreakdown = calculateTemporalBreakdown(trades)
		result.LongShort = calculateLongShortBreakdown(trades)
	}

	// Gate evaluation MUST run after the metrics above are computed — otherwise it
	// sees zeroed Sharpe/ProfitFactor and every strategy fails the gate.
	if config.ApplyGate {
		gateProfile := config.GateProfile
		if gateProfile == "" {
			gateProfile = "default"
		}
		var std MultiMetricStandard
		switch gateProfile {
		// Legacy strictness names.
		case "lenient":
			std = LenientMultiMetricStandard()
		case "strict":
			std = StrictMultiMetricStandard()
		// Deployment-profile taxonomy (matches the frontend Gate dropdown and
		// AGENTS.md §8.4 profile gating): research → paper → pretrade → production,
		// increasing in strictness. Previously these were unrecognized and silently
		// fell through to the strict-ish default, so nothing ever passed the gate.
		case "research":
			std = LenientMultiMetricStandard()
		case "paper":
			std = DefaultMultiMetricStandard()
		case "pretrade":
			std = StrictMultiMetricStandard()
		case "production_guarded", "production":
			std = StrictMultiMetricStandard()
		default:
			std = DefaultMultiMetricStandard()
		}
		verdict := EvaluateBacktestMultiMetric(result, std)
		result.MetricGateStatus = &verdict
	}

	if len(config.StrategyParams) > 0 {
		result.StrategyParams = make(map[string]float64, len(config.StrategyParams))
		for k, v := range config.StrategyParams {
			result.StrategyParams[k] = v
		}
	}

	result.SignalDiag = e.signalDiag
	result.MLFeatureEnabled = e.featureStore != nil

	return result, nil
}

func (e *Engine) generateSignal(candle Candle, regime int8, config BacktestConfig, runningCapital float64) *Signal {
	sr := e.getRunnerForSymbolAndStrategy(candle.Symbol, config.StrategyID, config)
	var raw *strategy.Signal
	func() {
		defer func() {
			if r := recover(); r != nil {
				raw = nil
				e.signalDiag.StrategyNil++
			}
		}()
		raw = sr.Evaluate(candle, regime)
	}()
	if raw == nil {
		e.signalDiag.StrategyNil++
		return nil
	}

	if raw.Quantity == 0 {
		e.signalDiag.ExitSignalZeroQty++
		return &Signal{
			Symbol:   candle.Symbol,
			Side:     raw.Side,
			Quantity: 0,
		}
	}

	confidence := 1.0
	if e.batchInferrer != nil && e.metaLabeler != nil && e.metaLabeler.IsHealthy() {
		hmmAlpha := [4]float64{}
		if regime >= 0 && regime < 4 {
			hmmAlpha[regime] = 1.0
		}

		var metaResult ml.MetaLabelingResult
		if e.featureStore != nil {
			e.featureStore.Push(candle)
			features, err := e.featureStore.Compute(candle.Time, hmmAlpha, confidence, 1, 0.75, 0.0, 0.001)
			if err == nil && features != nil && features.Validate() {
				metaResult = e.batchInferrer.Evaluate(features.ToSlice(), raw.PWin)
			} else {
				metaResult = e.batchInferrer.Evaluate(nil, raw.PWin)
			}
		} else {
			metaResult = e.batchInferrer.Evaluate(nil, raw.PWin)
		}

		if !metaResult.Accepted {
			e.signalDiag.MLRejected++
			return nil
		}
		confidence = metaResult.PWin
	}

	seasonalityMult := 1.0
	if config.UseSeasonalityOverlay {
		szn := NewSeasonalityOverlay()
		seasonalityMult = szn.Multiplier(candle.Time)
	}

	if e.regimeEnhancer != nil && e.regimeEnhancer.IsHealthy() {
		hmmVec := [4]float64{}
		if regime >= 0 && regime < 4 {
			hmmVec[regime] = 1.0
		}
		score, _ := e.regimeEnhancer.Evaluate(
			hmmVec, 0, 0, 0, 0.5, float64(candle.Time.Hour()),
		)
		e.positionSizer.SetRegimeScore(score)
	}

	sp := config.SizingPercent
	if sp <= 0 {
		sp = 0.02
	}
	baseSizeCapital := runningCapital * sp * seasonalityMult
	baseSize := baseSizeCapital / candle.Close.Float64()

	if e.pipeline != nil {
		e.pipeline.CurrentRegime = regime
		pipeResult := e.pipeline.ProcessSignal(context.Background(), risk.ProcessSignalRequest{
			StrategyID:       config.StrategyID,
			Symbol:           candle.Symbol,
			Side:             raw.Side,
			Price:            candle.Close.Float64(),
			Confidence:       confidence,
			BaseSize:         baseSize,
			ExistingPosition: 0,
			RunningCapital:   runningCapital,
		})
		if !pipeResult.Approved {
			e.signalDiag.PipelineRejected++
			return nil
		}
		e.signalDiag.SignalsPassed++
		return &Signal{
			Symbol:   candle.Symbol,
			Side:     raw.Side,
			Quantity: pipeResult.Size,
		}
	}

	return e.generateSignalInlineFallback(candle, regime, config, runningCapital, raw, confidence, seasonalityMult)
}

func (e *Engine) generateSignalInlineFallback(
	candle Candle, regime int8, config BacktestConfig,
	runningCapital float64, raw *strategy.Signal,
	confidence, seasonalityMult float64,
) *Signal {
	if runningCapital <= 0 {
		e.signalDiag.CapitalZero++
		return nil
	}

	if e.volHalt != nil && e.volHalt.IsHalted() {
		e.signalDiag.VolHalted++
		return nil
	}

	e.volHalt.UpdateReturn(0)
	e.exposure.SetEquity(runningCapital)

	c := runningCapital
	baseSize := c * 0.02 * seasonalityMult
	if config.SizingPercent > 0 {
		baseSize = c * config.SizingPercent * seasonalityMult
	}
	if e.ftmo != nil {
		baseSize *= e.ftmo.GetRegimeMultiplier()
	} else {
		switch regime {
		case 0:
			baseSize *= 1.5
		case 2:
			baseSize *= 0.5
		case 3:
			baseSize = 0
		}
	}
	if baseSize <= 0 {
		e.signalDiag.BaseSizeZero++
		return nil
	}

	baseSize *= e.kellyMult
	baseSize *= math.Min(confidence*1.5, 1.0)

	positionPct := baseSize / c
	if positionPct > 0.03 {
		positionPct = 0.03
	}
	quantity := (c * positionPct) / candle.Close.Float64()
	if e.ftmo != nil && config.PropFirmEnabled {
		quantity = e.ftmo.GetPositionSize(quantity)
	}

	quantity = e.positionSizer.ComputeSizeUncapped(1.0, quantity, 0)
	if quantity < 0.001 {
		e.signalDiag.QuantityTooSmall++
		return nil
	}

	notional := quantity * candle.Close.Float64()
	if ok, _ := e.exposure.CheckOrder(candle.Symbol, raw.Side, notional); !ok {
		e.signalDiag.ExposureBlocked++
		return nil
	}

	e.signalDiag.SignalsPassed++
	return &Signal{
		Symbol:   candle.Symbol,
		Side:     raw.Side,
		Quantity: quantity,
	}
}

func (e *Engine) generateSignalForExit(candle Candle, regime int8, config BacktestConfig, runningCapital float64) *Signal {
	sr := e.getRunnerForSymbolAndStrategy(candle.Symbol, config.StrategyID, config)
	var raw *strategy.Signal
	func() {
		defer func() { recover() }()
		raw = sr.Evaluate(candle, regime)
	}()
	if raw == nil {
		return nil
	}
	return &Signal{
		Symbol:   candle.Symbol,
		Side:     raw.Side,
		Quantity: 0,
	}
}

func barsPerDayFromTimeframe(timeframe string) float64 {
	switch timeframe {
	case "1m":
		return 390.0
	case "5m", "5min":
		return 78.0
	case "15m", "15min":
		return 26.0
	case "30m", "30min":
		return 13.0
	case "1h", "60m", "hourly":
		return 6.5
	case "4h":
		return 1.625
	case "1d", "daily", "":
		return 1.0
	default:
		return 1.0
	}
}

// effectiveBarsPerDay computes the actual bars-per-trading-day from the candle
// data, guarding against timeframe label mismatches (e.g. when synthetic daily
// data is labeled as "5m"). Falls back to declared barsPerDay if data is empty.
func effectiveBarsPerDay(candles []Candle, declaredBarsPerDay float64) float64 {
	if len(candles) == 0 {
		return declaredBarsPerDay
	}
	n := 0
	for _, c := range candles {
		if c.Time.Weekday() != time.Saturday && c.Time.Weekday() != time.Sunday {
			n++
		}
	}
	if n == 0 {
		return declaredBarsPerDay
	}
	tradingDays := 0
	seen := make(map[string]bool)
	for _, c := range candles {
		key := c.Time.Format("2006-01-02")
		if !seen[key] {
			seen[key] = true
			if c.Time.Weekday() != time.Saturday && c.Time.Weekday() != time.Sunday {
				tradingDays++
			}
		}
	}
	if tradingDays == 0 {
		return declaredBarsPerDay
	}
	effective := float64(n) / float64(tradingDays)
	tolerance := declaredBarsPerDay * 0.3
	if declaredBarsPerDay > 0 && math.Abs(effective-declaredBarsPerDay) > tolerance {
		return effective
	}
	return declaredBarsPerDay
}

func (e *Engine) getRunnerForStrategy(strategyID string, config BacktestConfig) strategy.Strategy {
	// Create a fresh runner instance per engine. Backtest matrices run many
	// engines concurrently; sharing a singleton runner (with mutable per-run
	// state such as GridRunner.openPositions) causes concurrent map writes.
	runner := strategy.GlobalRegistry().Create(strategyID)
	if runner != nil {
		if len(config.StrategyParams) > 0 {
			runner.SetParams(config.StrategyParams)
		}
		return runner
	}
	sr := strategy.NewMeanReversionRunner(20, 1.5, 0.1, 60)
	if len(config.StrategyParams) > 0 {
		sr.SetParams(config.StrategyParams)
	}
	return sr
}

func ParamDefsForStrategy(strategyID string) []strategy.ParamDef {
	runner := strategy.GlobalRegistry().Get(strategyID)
	if runner != nil {
		return runner.ParamDefs()
	}
	return strategy.NewMeanReversionRunner(20, 1.5, 0.1, 60).ParamDefs()
}

func (e *Engine) getRunnerForSymbolAndStrategy(symbol string, strategyID string, config BacktestConfig) strategy.Strategy {
	key := symbol + ":" + strategyID
	if sr, ok := e.stratBySymbol[key]; ok {
		return sr
	}
	sr := e.getRunnerForStrategy(strategyID, config)
	e.stratBySymbol[key] = sr
	return sr
}

func (e *Engine) resetStrategies() {
	e.stratBySymbol = make(map[string]strategy.Strategy)
}

func mergeCandlesByTime(candlesBySymbol [][]Candle) []Candle {
	var all []Candle
	for _, candles := range candlesBySymbol {
		all = append(all, candles...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Time.Before(all[j].Time)
	})
	return all
}

func getRegimeAt(t time.Time, logs []RegimeLog) int8 {
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Time.Before(t) || logs[i].Time.Equal(t) {
			return logs[i].HMMState
		}
	}
	return 0
}

func getVIXAt(t time.Time, logs []VIXLog) float64 {
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Time.Before(t) || logs[i].Time.Equal(t) {
			return logs[i].VIXValue
		}
	}
	return 0
}

func getSentimentAt(t time.Time, logs []SentimentLog) int {
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Time.Before(t) || logs[i].Time.Equal(t) {
			return logs[i].Score
		}
	}
	return 50
}

func calculateWinRate(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(trades)) * 100
}

func calculateSharpe(equity []EquityPoint, barsPerDay float64) float64 {
	if len(equity) < 2 {
		return 0
	}

	dayMap := make(map[string]float64)
	dayOrder := make([]string, 0)
	for _, e := range equity {
		dayKey := e.Time.Format("2006-01-02")
		if _, exists := dayMap[dayKey]; !exists {
			dayOrder = append(dayOrder, dayKey)
		}
		dayMap[dayKey] = e.Value
	}

	if len(dayOrder) < 2 {
		return 0
	}

	returns := make([]float64, 0, len(dayOrder)-1)
	for i := 1; i < len(dayOrder); i++ {
		prev := dayMap[dayOrder[i-1]]
		curr := dayMap[dayOrder[i]]
		if prev > 0 {
			returns = append(returns, (curr-prev)/prev)
		}
	}

	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	if variance == 0 {
		return 0
	}
	stdDev := math.Sqrt(variance)
	if stdDev < 1e-6 {
		return 0
	}
	return mean / stdDev * math.Sqrt(252.0)
}

func calculateSortino(equity []EquityPoint, barsPerDay float64) float64 {
	if len(equity) < 2 {
		return 0
	}

	dayMap := make(map[string]float64)
	dayOrder := make([]string, 0)
	for _, e := range equity {
		dayKey := e.Time.Format("2006-01-02")
		if _, exists := dayMap[dayKey]; !exists {
			dayOrder = append(dayOrder, dayKey)
		}
		dayMap[dayKey] = e.Value
	}

	if len(dayOrder) < 2 {
		return 0
	}

	returns := make([]float64, 0, len(dayOrder)-1)
	for i := 1; i < len(dayOrder); i++ {
		prev := dayMap[dayOrder[i-1]]
		curr := dayMap[dayOrder[i]]
		if prev > 0 {
			returns = append(returns, (curr-prev)/prev)
		}
	}

	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	var sumSq float64
	var count int
	for _, r := range returns {
		if r < 0 {
			sumSq += r * r
			count++
		}
	}
	if count == 0 || sumSq == 0 {
		return 0
	}
	if math.IsNaN(mean) || math.IsNaN(sumSq) || math.IsInf(mean, 0) || math.IsInf(sumSq, 0) {
		return 0
	}
	downStdDev := math.Sqrt(sumSq / float64(count))
	if downStdDev < 1e-6 {
		return 0
	}
	result := mean / downStdDev * math.Sqrt(252.0)
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0
	}
	return result
}

func calculateCalmar(equity []EquityPoint, _ float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	cagr := computeCAGR(equity)
	maxDD := computeMaxDrawdown(equity)
	if maxDD <= 0 {
		return 0
	}
	return cagr / maxDD
}

func calculateMaxDrawdown(equity []EquityPoint) float64 {
	peak := 0.0
	maxDD := 0.0
	for _, p := range equity {
		if p.Value > peak {
			peak = p.Value
		}
		dd := (peak - p.Value) / peak * 100
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

func calculateMaxDrawdownDuration(equity []EquityPoint) int {
	peak := 0.0
	currentDuration := 0
	maxDuration := 0
	inDrawdown := false
	for _, p := range equity {
		if p.Value >= peak {
			peak = p.Value
			if inDrawdown {
				if currentDuration > maxDuration {
					maxDuration = currentDuration
				}
				currentDuration = 0
				inDrawdown = false
			}
		} else {
			inDrawdown = true
			currentDuration++
		}
	}
	if currentDuration > maxDuration {
		maxDuration = currentDuration
	}
	return maxDuration
}

func calculateProfitFactor(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	grossWin := 0.0
	grossLoss := 0.0
	for _, t := range trades {
		if t.PnL > 0 {
			grossWin += t.PnL
		} else {
			grossLoss += -t.PnL
		}
	}
	if grossLoss == 0 {
		return 0
	}
	if math.IsNaN(grossWin) || math.IsNaN(grossLoss) || math.IsInf(grossWin, 0) || math.IsInf(grossLoss, 0) {
		return 0
	}
	return grossWin / grossLoss
}

func calculateAvgWinLoss(trades []Trade) (float64, float64) {
	var wins, losses float64
	var winCount, lossCount int
	for _, t := range trades {
		if t.PnL > 0 {
			wins += t.PnL
			winCount++
		} else {
			losses += -t.PnL
			lossCount++
		}
	}
	avgWin := 0.0
	avgLoss := 0.0
	if winCount > 0 {
		avgWin = wins / float64(winCount)
	}
	if lossCount > 0 {
		avgLoss = losses / float64(lossCount)
	}
	return avgWin, avgLoss
}

func countWinsLosses(trades []Trade) (int, int) {
	wins, losses := 0, 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		} else if t.PnL < 0 {
			losses++
		}
	}
	return wins, losses
}

func calculateLongShortBreakdown(trades []Trade) LongShortBreakdown {
	var b LongShortBreakdown
	var longWins, shortWins int
	var longLoss float64
	var shortLoss float64
	var longMAE, shortMAE float64
	var longMFE, shortMFE float64

	for _, t := range trades {
		if math.IsNaN(t.PnL) || math.IsInf(t.PnL, 0) || math.IsNaN(t.MAE) || math.IsInf(t.MAE, 0) || math.IsNaN(t.MFE) || math.IsInf(t.MFE, 0) {
			continue
		}
		if t.Side == "BUY" {
			b.LongTrades++
			b.LongGrossPnL += t.PnL
			longMAE += t.MAE
			longMFE += t.MFE
			if t.PnL > 0 {
				longWins++
			} else {
				longLoss += -t.PnL
			}
		} else {
			b.ShortTrades++
			b.ShortGrossPnL += t.PnL
			shortMAE += t.MAE
			shortMFE += t.MFE
			if t.PnL > 0 {
				shortWins++
			} else {
				shortLoss += -t.PnL
			}
		}
	}

	b.LongWins = longWins
	b.ShortWins = shortWins

	if b.LongTrades > 0 {
		b.LongWinRate = float64(longWins) / float64(b.LongTrades) * 100.0
		b.LongAvgPnL = b.LongGrossPnL / float64(b.LongTrades)
		b.LongAvgMAE = longMAE / float64(b.LongTrades)
		b.LongAvgMFE = longMFE / float64(b.LongTrades)
	}
	if b.ShortTrades > 0 {
		b.ShortWinRate = float64(shortWins) / float64(b.ShortTrades) * 100.0
		b.ShortAvgPnL = b.ShortGrossPnL / float64(b.ShortTrades)
		b.ShortAvgMAE = shortMAE / float64(b.ShortTrades)
		b.ShortAvgMFE = shortMFE / float64(b.ShortTrades)
	}

	if longLoss > 0 {
		b.LongPF = (b.LongGrossPnL + longLoss) / longLoss
	} else if b.LongGrossPnL > 0 {
		b.LongPF = b.LongGrossPnL
	}
	if shortLoss > 0 {
		b.ShortPF = (b.ShortGrossPnL + shortLoss) / shortLoss
	} else if b.ShortGrossPnL > 0 {
		b.ShortPF = b.ShortGrossPnL
	}

	totalAbsPnL := math.Abs(b.LongGrossPnL) + math.Abs(b.ShortGrossPnL)
	if totalAbsPnL > 0 {
		b.DirectionalBias = (b.LongGrossPnL - b.ShortGrossPnL) / totalAbsPnL
	}

	return b
}

func calculateDailyReturns(equity []EquityPoint) []DailyReturn {
	if len(equity) < 2 {
		return nil
	}
	eodMap := make(map[string]float64)
	for _, pt := range equity {
		dateKey := pt.Time.Truncate(24 * time.Hour).Format("2006-01-02")
		eodMap[dateKey] = pt.Value
	}

	var dates []string
	for d := range eodMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	var result []DailyReturn
	for i := 1; i < len(dates); i++ {
		prevValue := eodMap[dates[i-1]]
		currValue := eodMap[dates[i]]
		date, _ := time.Parse("2006-01-02", dates[i])
		dr := DailyReturn{Date: date}
		if prevValue > 0 {
			dr.Return = (currValue - prevValue) / prevValue * 100
		}
		result = append(result, dr)
	}
	return result
}

type MultiBacktestConfig struct {
	StrategyIDs          []string
	Symbols              []string
	StartDate            time.Time
	EndDate              time.Time
	InitialCapital       float64
	ProfileID            string
	SlippageModel        SlippageModel
	CommissionBps        float64
	UseUniverseSnapshots bool
	UniverseConfigID     string
}

type MultiBacktestResult struct {
	Config          MultiBacktestConfig
	TotalReturn     float64
	TotalReturnPct  float64
	SharpeRatio     float64
	MaxDrawdown     float64
	WinRate         float64
	NumTrades       int
	EquityCurve     []EquityPoint
	MtmSharpeRatio  float64       `json:"mtm_sharpe_ratio,omitempty"`
	MtmMaxDrawdown  float64       `json:"mtm_max_drawdown,omitempty"`
	StrategyMetrics map[string]*StrategyBacktestMetric
	ComplianceReport      *ComplianceReport
}

type StrategyBacktestMetric struct {
	StrategyID  string
	NumTrades   int
	WinRate     float64
	TotalReturn float64
	SharpeRatio float64
	MaxDrawdown float64
}

type EngineMulti struct {
	db           Database
	fillSim      *FillSimulator
	feeModel     model.FeeModel
	latencyModel model.LatencyModel
	recorder     model.Recorder
	registry     *strategy.Registry
	poolSim      *CapitalPoolSim
	slippageModel SlippageModel
	commissionBps float64
}

func NewEngineMulti(db Database, reg *strategy.Registry) *EngineMulti {
	return &EngineMulti{
		db:            db,
		fillSim:       NewFillSimulator(DefaultEquitySlippage()),
		registry:      reg,
		slippageModel: DefaultEquitySlippage(),
	}
}

func NewEngineMultiWithSlippage(db Database, reg *strategy.Registry, model SlippageModel, commissionBps float64) *EngineMulti {
	return &EngineMulti{
		db:            db,
		fillSim:       NewFillSimulator(model),
		registry:      reg,
		slippageModel: model,
		commissionBps: commissionBps,
	}
}

func (e *EngineMulti) RunMulti(ctx context.Context, config MultiBacktestConfig) (*MultiBacktestResult, error) {
	profile := propfirm.DefaultFTMOProfile()
	if config.ProfileID != "" {
		_ = config.ProfileID
	}

	regimeLogs, err := e.db.LoadRegimeLogs(ctx, config.StartDate, config.EndDate)
	if err != nil {
		regimeLogs = nil
	}

	candlesBySymbol, err := e.db.LoadCandles(ctx, config.Symbols, config.StartDate, config.EndDate)
	if err != nil {
		return nil, err
	}

	var universeSnapshots map[time.Time][]string
	if config.UseUniverseSnapshots {
		snaps, snapErr := e.db.LoadUniverseSnapshots(ctx, config.StartDate, config.EndDate)
		if snapErr == nil && len(snaps) > 0 {
			universeSnapshots = make(map[time.Time][]string, len(snaps))
			for _, snap := range snaps {
				dateKey := snap.Date.Truncate(24 * time.Hour)
				universeSnapshots[dateKey] = snap.Symbols
			}
		}
	}

	allCandles := mergeCandlesByTime(candlesBySymbol)

	e.poolSim = NewCapitalPoolSim(profile, config.InitialCapital)
	for _, sid := range config.StrategyIDs {
		runner := e.registry.Create(sid)
		if runner != nil {
			e.poolSim.AddStrategy(sid, &runnerAdapter{runner: runner})
		}
	}

	result := &MultiBacktestResult{
		Config:          config,
		StrategyMetrics: make(map[string]*StrategyBacktestMetric),
	}

	capital := config.InitialCapital
	peakCapital := config.InitialCapital
	equity := []EquityPoint{}
	mtmMultiEquity := []EquityPoint{}
	trades := []Trade{}

	type openPosition struct {
		Trade      Trade
		StrategyID string
	}
	openPositions := make(map[string]*openPosition)

	var lastDay string
	for i := range allCandles {
		candle := allCandles[i]

		if universeSnapshots != nil {
			dateKey := candle.Time.Truncate(24 * time.Hour)
			if activeSymbols, ok := universeSnapshots[dateKey]; ok {
				found := false
				for _, sym := range activeSymbols {
					if sym == candle.Symbol {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
		}

		currentDay := candle.Time.Format("2006-01-02")
		if currentDay != lastDay {
			e.poolSim.ResetDaily()
			lastDay = currentDay
		}

		regime := getRegimeAt(candle.Time, regimeLogs)

		if e.poolSim.Halted() {
			continue
		}

		signals := e.poolSim.EvaluateAll(candle, regime)

		for sid, sig := range signals {
			if _, hasOpen := openPositions[sid]; hasOpen {
				continue
			}
			if sig == nil {
				continue
			}
			openPositions[sid] = &openPosition{
				Trade: Trade{
					Symbol:     sig.Symbol,
					Side:       sig.Side,
					Quantity:   sig.Quantity,
					EntryPrice: types.PriceFromFloat(candle.Close.Float64() * candle.AdjustmentFactor),
					EntryTime:  candle.Time,
					StrategyID: sid,
				},
				StrategyID: sid,
			}
		}

		for sid, op := range openPositions {
			exitPrice := candle.Close.Float64() * candle.AdjustmentFactor
			commission := op.Trade.EntryPrice.Float64() * op.Trade.Quantity * config.CommissionBps / 10000.0 * 2

			var pnl float64
			if op.Trade.Side == "BUY" {
				pnl = (exitPrice-op.Trade.EntryPrice.Float64())*op.Trade.Quantity - commission
			} else {
				pnl = (op.Trade.EntryPrice.Float64()-exitPrice)*op.Trade.Quantity - commission
			}

			pnlPct := 0.0
			if capital > 0 {
				pnlPct = pnl / capital * 100
			}
			op.Trade.PnL = pnl
			op.Trade.PnLPct = pnlPct
			op.Trade.ExitPrice = types.PriceFromFloat(exitPrice)
			op.Trade.ExitTime = candle.Time
			op.Trade.HMMRegime = regime

			capital += pnl
			e.poolSim.RecordFill(sid, op.Trade.Symbol, op.Trade.Side, pnl, op.Trade.Quantity)
			trades = append(trades, op.Trade)
			delete(openPositions, sid)
		}

		equity = append(equity, EquityPoint{
			Time:   candle.Time,
			Value:  capital,
			Regime: regime,
		})

		var multiUnrealizedPnL float64
		for _, op := range openPositions {
			if op.Trade.Side == "BUY" {
				multiUnrealizedPnL += (candle.Close.Float64() - op.Trade.EntryPrice.Float64()) * op.Trade.Quantity
			} else {
				multiUnrealizedPnL += (op.Trade.EntryPrice.Float64() - candle.Close.Float64()) * op.Trade.Quantity
			}
		}
		mtmMultiEquity = append(mtmMultiEquity, EquityPoint{
			Time:   candle.Time,
			Value:  capital + multiUnrealizedPnL,
			Regime: regime,
		})

		if capital > peakCapital {
			peakCapital = capital
		}
	}

	if len(allCandles) > 0 {
		for sid, op := range openPositions {
			lastCandle := allCandles[len(allCandles)-1]
			commission := op.Trade.EntryPrice.Float64() * op.Trade.Quantity * config.CommissionBps / 10000.0 * 2
			var pnl float64
			if op.Trade.Side == "BUY" {
				pnl = (lastCandle.Close.Float64()-op.Trade.EntryPrice.Float64())*op.Trade.Quantity - commission
			} else {
				pnl = (op.Trade.EntryPrice.Float64()-lastCandle.Close.Float64())*op.Trade.Quantity - commission
			}
			op.Trade.PnL = pnl
			op.Trade.ExitPrice = lastCandle.Close
			op.Trade.ExitTime = lastCandle.Time
			capital += pnl
			trades = append(trades, op.Trade)
			delete(openPositions, sid)
		}
	}

	result.EquityCurve = equity
	result.NumTrades = len(trades)
	result.TotalReturnPct = (capital - config.InitialCapital) / config.InitialCapital * 100
	if math.IsNaN(capital) || math.IsInf(capital, 0) {
		result.TotalReturn = 0
		result.TotalReturnPct = 0
	}
	result.TotalReturn = capital - config.InitialCapital

	if result.NumTrades > 0 {
		result.WinRate = calculateWinRate(trades)
		result.SharpeRatio = calculateSharpe(equity, 1.0)
		result.MaxDrawdown = calculateMaxDrawdown(equity)
	}

	if len(mtmMultiEquity) >= 2 {
		result.MtmSharpeRatio = calculateMtmSharpe(mtmMultiEquity, 1.0)
		result.MtmMaxDrawdown = calculateMaxDrawdown(mtmMultiEquity)
	}

	for _, sid := range config.StrategyIDs {
		var stratTrades []Trade
		for _, t := range trades {
			if t.StrategyID == sid {
				stratTrades = append(stratTrades, t)
			}
		}
		metric := &StrategyBacktestMetric{
			StrategyID: sid,
			NumTrades:  len(stratTrades),
		}
		if len(stratTrades) > 0 {
			metric.WinRate = calculateWinRate(stratTrades)
			metric.TotalReturn = 0
			for _, t := range stratTrades {
				metric.TotalReturn += t.PnL
			}
		}
		result.StrategyMetrics[sid] = metric
	}

	if e.poolSim.Halted() {
		result.ComplianceReport = &ComplianceReport{
			Passed:     false,
			HaltReason: e.poolSim.HaltReason(),
		}
	} else {
		result.ComplianceReport = &ComplianceReport{Passed: true}
	}

	return result, nil
}

type runnerAdapter struct {
	runner strategy.Runner
}

func (a *runnerAdapter) Name() string {
	return a.runner.Name()
}

func (a *runnerAdapter) Type() string {
	return a.runner.Type()
}

func (a *runnerAdapter) Evaluate(candle Candle, regime int8) *Signal {
	sc := strategy.Candle{
		Time:   candle.Time,
		Open:   candle.Open,
		High:   candle.High,
		Low:    candle.Low,
		Close:  candle.Close,
		Volume: candle.Volume,
		Symbol: candle.Symbol,
	}
	sig := a.runner.Evaluate(sc, regime)
	if sig == nil {
		return nil
	}
	return &Signal{
		Symbol:   sig.Symbol,
		Side:     sig.Side,
		Quantity: sig.Quantity,
	}
}

func (a *runnerAdapter) Reset() {
	a.runner.Reset()
}

func invertSide(side string) string {
	if side == "BUY" {
		return "SELL"
	}
	return "BUY"
}

func calculateRegimeStats(trades []Trade) []RegimeStat {
	regimeMap := make(map[int8]*RegimeStat)
	labels := map[int8]string{0: "Calm", 1: "Trending", 2: "High Vol", 3: "Crisis"}

	for _, t := range trades {
		rs, ok := regimeMap[t.HMMRegime]
		if !ok {
			rs = &RegimeStat{
				Regime: t.HMMRegime,
				Label:  labels[t.HMMRegime],
			}
			regimeMap[t.HMMRegime] = rs
		}
		rs.NumTrades++
		rs.TotalReturn += t.PnL
	}

	for _, rs := range regimeMap {
		if rs.NumTrades > 0 {
			rs.TotalReturn /= float64(rs.NumTrades)
		}
		wins := 0
		for _, t := range trades {
			if t.HMMRegime == rs.Regime && t.PnL > 0 {
				wins++
			}
		}
		if rs.NumTrades > 0 {
			rs.WinRate = float64(wins) / float64(rs.NumTrades) * 100
		}
	}

	var result []RegimeStat
	for _, rs := range regimeMap {
		result = append(result, *rs)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Regime < result[j].Regime
	})
	return result
}

func calculateTemporalBreakdown(trades []Trade) TemporalBreakdown {
	tb := TemporalBreakdown{}
	if len(trades) == 0 {
		return tb
	}

	yearly := make(map[string]*PeriodStat)
	monthly := make(map[string]*PeriodStat)
	weekly := make(map[string]*PeriodStat)
	daily := make(map[string]*PeriodStat)

	for _, t := range trades {
		y := t.ExitTime.Format("2006")
		m := t.ExitTime.Format("2006-01")
		w := isoWeek(t.ExitTime)
		d := t.ExitTime.Format("2006-01-02")

		addToPeriod(yearly, y, t)
		addToPeriod(monthly, m, t)
		addToPeriod(weekly, w, t)
		addToPeriod(daily, d, t)
	}

	tb.Yearly = sortedPeriods(yearly)
	tb.Monthly = sortedPeriods(monthly)
	tb.Weekly = sortedPeriods(weekly)
	tb.Daily = sortedPeriods(daily)

	return tb
}

func isoWeek(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func calculateAdverseSelectionRate(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	count := 0
	for _, t := range trades {
		if t.AdverseSelection {
			count++
		}
	}
	return float64(count) / float64(len(trades)) * 100.0
}

func addToPeriod(m map[string]*PeriodStat, key string, trade Trade) {
	ps, ok := m[key]
	if !ok {
		ps = &PeriodStat{Period: key}
		m[key] = ps
	}
	ps.NumTrades++
	ps.NetPnL += trade.PnL
	ps.BrokerFees += trade.BrokerFee
	ps.Commission += trade.EntryPrice.Float64() * trade.Quantity * 0.005 / 10000.0 * 2
	if trade.PnL > 0 {
		ps.GrossProfit += trade.PnL
	}
	if trade.PnL < 0 {
		ps.GrossLoss += -trade.PnL
	}
	if ps.NumTrades > 0 && trade.PnL > 0 {
		ps.WinRate = ps.WinRate*float64(ps.NumTrades-1)+100.0
		ps.WinRate /= float64(ps.NumTrades)
	} else if ps.NumTrades > 0 {
		ps.WinRate = ps.WinRate * float64(ps.NumTrades-1) / float64(ps.NumTrades)
	}
}

func sortedPeriods(m map[string]*PeriodStat) []PeriodStat {
	result := make([]PeriodStat, 0, len(m))
	for _, ps := range m {
		result = append(result, *ps)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Period < result[j].Period
	})
	return result
}

// preflightDataQuality performs a fast diagnostic on candle data and returns a
// warning string if the data fails basic quality gates. Checks:
//  1. Sufficient data points (> 20)
//  2. Non-zero volatility
//  3. Lag-1 autocorrelation sign (positive → trending, negative → mean-reverting, zero → random walk)
//  4. Variance ratio at lag 5 (vs random walk expectation)
func preflightDataQuality(candles []Candle) string {
	if len(candles) < 20 {
		return fmt.Sprintf("data_quality: only %d candles (minimum 20 required for meaningful backtest)", len(candles))
	}
	var closes []float64
	for _, c := range candles {
		closes = append(closes, c.Close.Float64())
	}
	n := len(closes)
	returns := make([]float64, n-1)
	for i := 1; i < n; i++ {
		if closes[i-1] > 0 {
			returns[i-1] = (closes[i] - closes[i-1]) / closes[i-1]
		}
	}
	var sum, sumSq float64
	for _, r := range returns {
		sum += r
		sumSq += r * r
	}
	meanRet := sum / float64(len(returns))
	variance := sumSq/float64(len(returns)) - meanRet*meanRet
	if variance <= 0 {
		return "data_quality: zero price variance (all candles identical — data may be stale or missing)"
	}
	if variance < 1e-12 {
		return "data_quality: near-zero price variance (insufficient volatility for strategy evaluation)"
	}
	// Lag-1 autocorrelation
	if len(returns) >= 10 {
		var cov, denom float64
		for i := 1; i < len(returns); i++ {
			cov += (returns[i] - meanRet) * (returns[i-1] - meanRet)
			denom += (returns[i] - meanRet) * (returns[i] - meanRet)
		}
		if denom > 0 {
			ac1 := cov / denom
			if math.Abs(ac1) < 0.01 {
				return "data_quality: returns show no serial correlation (lag-1 autocorrelation ≈ 0) — trend and mean-reversion strategies may not trigger as expected; verify data is not a driftless random walk"
			}
		}
	}
	// Variance ratio at lag 5: VR = Var(k-step return) / (k * Var(1-step return))
	// Under random walk null, VR ≈ 1. VR > 1 → trending, VR < 1 → mean-reverting.
	if len(returns) >= 25 {
		var var1, var5 float64
		mean1 := meanRet
		for _, r := range returns {
			var1 += (r - mean1) * (r - mean1)
		}
		var1 /= float64(len(returns))
		if var1 > 0 {
			for i := 0; i+5 < len(returns); i++ {
				r5 := (closes[i+5] - closes[i]) / closes[i]
				var5 += (r5) * (r5)
			}
			m := float64(len(returns) - 5)
			if m > 0 {
				var5 /= m
				vr := var5 / (5.0 * var1)
				if vr < 0.4 {
					return fmt.Sprintf("data_quality: variance ratio VR(5)=%.2f indicates strong mean reversion — grid/mean-reversion strategies may overfit if not OOS-validated", vr)
				}
				if vr > 2.5 {
					return fmt.Sprintf("data_quality: variance ratio VR(5)=%.2f indicates strong momentum — trend strategies may overfit if not walk-forward validated", vr)
				}
			}
		}
	}
	if len(closes) > 0 && closes[0] > 0 {
		totalRet := (closes[len(closes)-1] - closes[0]) / closes[0]
		if math.IsInf(totalRet, 0) || math.IsNaN(totalRet) {
			return "data_quality: price path contains NaN or Inf values — data corruption detected"
		}
	}
	// Regime diversity check: warn if returns cluster in a single volatility bucket
	if len(returns) >= 60 {
		absReturns := make([]float64, len(returns))
		copy(absReturns, returns)
		for i := range absReturns {
			if absReturns[i] < 0 {
				absReturns[i] = -absReturns[i]
			}
		}
		sort.Float64s(absReturns)
		p25 := absReturns[len(absReturns)*25/100]
		p75 := absReturns[len(absReturns)*75/100]
		p95 := absReturns[len(absReturns)*95/100]
		// Check if 90%+ of returns are in the "calm" bucket (below the 75th percentile equivalent of a volatile asset)
		// Heuristic: if p95 / (p75 + 1e-8) < 2.0, there's no meaningful tail — single-regime data
		if p95 > 0 && p75 > 0 && p95/p75 < 2.0 {
			return "data_quality: returns show no tail dispersion (p95/p75 ratio < 2) — data may span only a single market regime; regime-specific strategy evaluation may be unreliable"
		}
		_ = p25
	}
	return ""
}
