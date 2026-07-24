package backtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/hash"
	"github.com/lee-econ/orca-core/internal/ml"
	"github.com/lee-econ/orca-core/internal/model"
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
}

type MatrixBacktestConfig struct {
	StrategyIDs      []string `json:"strategy_ids"`
	Symbols          []string `json:"symbols"`
	Timeframes       []string `json:"timeframes"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	InitialCapital   float64   `json:"initial_capital"`
	DataSource       string    `json:"data_source"`
	GateProfile      string    `json:"gate_profile"`
	PropFirmEnabled  bool      `json:"propfirm_enabled"`
	SizingPercent    float64   `json:"sizing_percent"`
	KellyFraction    float64   `json:"kelly_fraction"`
}

type ComboResult struct {
	RunID            string             `json:"run_id"`
	Symbol           string             `json:"symbol"`
	StrategyID       string             `json:"strategy_id"`
	Timeframe        string             `json:"timeframe"`
	SharpeRatio      float64            `json:"sharpe_ratio"`
	SortinoRatio     float64            `json:"sortino_ratio"`
	MaxDrawdown      float64            `json:"max_drawdown"`
	TotalReturn      float64            `json:"total_return"`
	WinRate          float64            `json:"win_rate"`
	ProfitFactor     float64            `json:"profit_factor"`
	AvgTrade         float64            `json:"avg_trade"`
	AvgWin           float64            `json:"avg_win"`
	AvgLoss          float64            `json:"avg_loss"`
	NumTrades        int                `json:"num_trades"`
	Error            string             `json:"error,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	GatePassed       *bool              `json:"gate_passed,omitempty"`
	AdverseSelectRate float64           `json:"adverse_selection_rate,omitempty"`
	BestParams       map[string]float64 `json:"best_params,omitempty"`
	Optimized        bool               `json:"optimized"`
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
	BrokerFee        float64
	SlippageMidBps   float64
	SlippageLastBps  float64
	AdverseSelection bool
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
	RegimeStats    []RegimeStat
	EquityCurve    []EquityPoint
	DailyReturns   []DailyReturn
	TemporalBreakdown TemporalBreakdown
	ComplianceReport     *ComplianceReport
	CompletedAt    time.Time
	RegimeLogError string          `json:"regime_log_error,omitempty"`
	Warnings       []string        `json:"warnings,omitempty"`
	MetricGateStatus *MultiMetricVerdict `json:"metric_gate_status,omitempty"`
	StrategyParams map[string]float64   `json:"strategy_params,omitempty"`
	SignalDiag     SignalDiag          `json:"signal_diag,omitempty"`
	EngineVersion  string          `json:"engine_version,omitempty"`
	StrategyHash   string          `json:"strategy_hash,omitempty"`
	SchemaVersion  int             `json:"schema_version,omitempty"`
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
	fillModel      model.FillModel
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
	return &Engine{
		db:             db,
		fillSim:        NewFillSimulator(DefaultEquitySlippage()),
		stratBySymbol:  make(map[string]strategy.Strategy),
		orderLimiter:   risk.NewOrderRateLimiter(10),
		volHalt:        risk.NewVolatilityHalt(3.0),
		exposure:       risk.NewExposureTracker(5.0, 0.25),
		kellyMult:      0.25,
		positionSizer:  risk.NewPositionSizer(nil),
		metaCfg:        ml.DefaultMetaLabelerConfig(),
	}
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

func NewEngineWithFixedSeed(db Database, seed int64) *Engine {
	model := DefaultEquitySlippage()
	e := &Engine{
		db:             db,
		fillSim:        NewFillSimulatorWithSeed(model, seed),
		stratBySymbol:  make(map[string]strategy.Strategy),
		orderLimiter:   risk.NewOrderRateLimiter(10),
		volHalt:        risk.NewVolatilityHalt(3.0),
		exposure:       risk.NewExposureTracker(5.0, 0.25),
		kellyMult:      0.25,
		positionSizer:  risk.NewPositionSizer(nil),
	}
	return e
}

func NewEngineWithSlippage(db Database, model SlippageModel) *Engine {
	return &Engine{
		db:             db,
		fillSim:        NewFillSimulator(model),
		stratBySymbol:  make(map[string]strategy.Strategy),
		orderLimiter:   risk.NewOrderRateLimiter(10),
		volHalt:        risk.NewVolatilityHalt(3.0),
		exposure:       risk.NewExposureTracker(5.0, 0.25),
		kellyMult:      0.25,
		positionSizer:  risk.NewPositionSizer(nil),
	}
}

func NewEngineWithStrategy(db Database, sr strategy.Strategy) *Engine {
	e := &Engine{
		db:             db,
		fillSim:        NewFillSimulator(DefaultEquitySlippage()),
		stratBySymbol:  make(map[string]strategy.Strategy),
		orderLimiter:   risk.NewOrderRateLimiter(10),
		volHalt:        risk.NewVolatilityHalt(3.0),
		exposure:       risk.NewExposureTracker(5.0, 0.25),
		kellyMult:      0.25,
		positionSizer:  risk.NewPositionSizer(nil),
	}
	e.stratBySymbol["default"] = sr
	return e
}

func NewEngineWithSlippageAndStrategy(db Database, model SlippageModel, sr strategy.Strategy) *Engine {
	e := &Engine{
		db:             db,
		fillSim:        NewFillSimulator(model),
		stratBySymbol:  make(map[string]strategy.Strategy),
		orderLimiter:   risk.NewOrderRateLimiter(10),
		volHalt:        risk.NewVolatilityHalt(3.0),
		exposure:       risk.NewExposureTracker(5.0, 0.25),
		kellyMult:      0.25,
		positionSizer:  risk.NewPositionSizer(nil),
	}
	e.stratBySymbol["default"] = sr
	return e
}

func (e *Engine) Run(ctx context.Context, config BacktestConfig) (*BacktestResult, error) {
	e.resetStrategies()

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

	result := &BacktestResult{Config: config, SchemaVersion: 1}

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
	trades := []Trade{}

	e.ftmo = nil
	if config.PropFirmEnabled {
		e.ftmo = DefaultPropFirmEnforcer(config.InitialCapital)
	}

	// Allow the optimizer to tune the fractional Kelly multiplier per run.
	if config.KellyFraction > 0 {
		e.kellyMult = config.KellyFraction
	}

	openTrades := make(map[string]*Trade)
	activeStops := make(map[string]*ActiveStop)
	pendingAS := make(map[string]*Trade)
	var lastDay string
	var atrWindow []Candle
	var hasHighVIX bool
	atrPeriod := 14
	if config.StopLoss != nil && config.StopLoss.ATRPeriod > 0 {
		atrPeriod = config.StopLoss.ATRPeriod
	}

	barsPerDay := barsPerDayFromTimeframe(config.Timeframe)

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
		e.positionSizer.UpdateMarketState(vix, sentiment, regime)
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

		if e.ftmo != nil && e.ftmo.IsHalted {
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
							stop.StopPrice = types.PriceFromFloat(dynamicStop)
						}
					}
				}
			}

			exitReason := ""
			exitPrice := candle.Close.Float64()
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

			if !shouldExit {
				reverseSignal := e.generateSignalForExit(candle, regime, config, capital)
				if reverseSignal != nil && reverseSignal.Symbol == sym {
					exitReason = "signal_reverse"
					shouldExit = true
				}
			}

			if shouldExit {
				midPrice := (candle.High.Float64() + candle.Low.Float64()) / 2.0
				simulatedExit := e.fillSim.SimulateFillWithTCA(uint32(len(trades)+1), ot.Symbol, ot.EntryPrice.Float64(), fillQty, invertSide(ot.Side), exitPrice, candle.Time, midPrice, candle.Close.Float64())
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
				ot.PnLPct = ot.PnL / config.InitialCapital * 100
				ot.ExitPrice = types.PriceFromFloat(exitPrice)
				ot.ExitTime = candle.Time
				ot.Quantity = fillQty
				ot.HMMRegime = regime
				ot.ExitReason = exitReason
				ot.BrokerFee = brokerFee

				capital += ot.PnL
			if capital <= 0 {
				capital = 0
			}
				if e.ftmo != nil {
					e.ftmo.OnFill(ot.PnL)
				}
				trades = append(trades, *ot)
				delete(openTrades, sym)
				delete(activeStops, sym)
			}
		}

		if _, alreadyOpen := openTrades[candle.Symbol]; !alreadyOpen {
			e.signalDiag.SignalAttempts++
			signal := e.generateSignal(candle, regime, config, capital)
			if signal != nil {
			e.signalDiag.TradesOpened++
			midPrice := (candle.High.Float64() + candle.Low.Float64()) / 2.0
			simulatedEntry := e.fillSim.SimulateFillWithTCA(uint32(len(trades)+1), candle.Symbol, candle.Close.Float64(), signal.Quantity, signal.Side, candle.Close.Float64(), candle.Time, midPrice, candle.Close.Float64())
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
		result.NumTrades = 0
		result.TotalReturnPct = 0
		result.TotalReturn = 0
		result.CompletedAt = time.Now()
		result.Warnings = append(result.Warnings, "no candle data available for requested symbol/timeframe combination")
		return result, nil
	}

	lastCandle := allCandles[len(allCandles)-1]
	for _, ot := range openTrades {
		exitPrice := lastCandle.Close.Float64()
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
		ot.PnLPct = ot.PnL / config.InitialCapital * 100
		ot.ExitPrice = types.PriceFromFloat(exitPrice)
		ot.ExitTime = lastCandle.Time
		ot.ExitReason = exitReason
		ot.BrokerFee = brokerFee
		capital += ot.PnL
		if capital <= 0 {
			capital = 0
		}
		trades = append(trades, *ot)
	}

	result.Trades = trades
	result.EquityCurve = equity
	result.NumTrades = len(trades)
	result.TotalReturnPct = (capital - config.InitialCapital) / config.InitialCapital * 100
	result.TotalReturn = capital - config.InitialCapital
	result.CompletedAt = time.Now()

	if e.ftmo != nil {
		report := e.ftmo.Summary()
		result.ComplianceReport = &report
	}

	if hasHighVIX {
		result.Warnings = append(result.Warnings, "VIX exceeded 25 during backtest period: regime detection accuracy may be reduced vs live (VIX modulation not applied to pre-computed regime logs)")
	}

	if result.NumTrades > 0 {
		result.WinRate = calculateWinRate(trades)
		if result.NumTrades >= 5 {
			result.SharpeRatio = calculateSharpe(equity, barsPerDay)
			result.SortinoRatio = calculateSortino(equity, barsPerDay)
		}
		result.MaxDrawdown = calculateMaxDrawdown(equity)
		result.MaxDrawdownDuration = calculateMaxDrawdownDuration(equity)
		result.ProfitFactor = calculateProfitFactor(trades)
		result.AvgTrade = result.TotalReturn / float64(result.NumTrades)
		result.AvgWin, result.AvgLoss = calculateAvgWinLoss(trades)
		result.NumWins, result.NumLosses = countWinsLosses(trades)
		result.AdverseSelectionRate = calculateAdverseSelectionRate(trades)
		result.DailyReturns = calculateDailyReturns(equity)
		result.RegimeStats = calculateRegimeStats(trades)
		result.TemporalBreakdown = calculateTemporalBreakdown(trades)
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

	return result, nil
}

func (e *Engine) generateSignal(candle Candle, regime int8, config BacktestConfig, runningCapital float64) *Signal {
	if runningCapital <= 0 {
		e.signalDiag.CapitalZero++
		return nil
	}

	if e.volHalt != nil && e.volHalt.IsHalted() {
		e.signalDiag.VolHalted++
		return nil
	}

	// NOTE: the wall-clock OrderRateLimiter is intentionally NOT applied in
	// backtests. It is a live-execution safety control (throttling real broker
	// API calls at N orders/sec of wall-clock time). A backtest compresses years
	// of simulated time into milliseconds of real time, so a wall-clock limiter
	// would reject ~99% of otherwise-valid signals and starve every run of
	// trades. Rate/throughput realism in backtests is modeled by the fill
	// simulator and capital pool, not by wall-clock throttling.

	e.volHalt.UpdateReturn(0)
	e.exposure.SetEquity(runningCapital)




	c := runningCapital
	sp := config.SizingPercent
	if sp <= 0 {
		sp = 0.02
	}
	baseSize := c * sp
	if e.ftmo != nil {
		baseSize = c * e.ftmo.GetRegimeMultiplier() * sp
	} else {
		switch regime {
		case 0:
			baseSize = c * sp * 1.5 // calm: 150% of base
		case 1:
			baseSize = c * sp       // normal: 100% of base
		case 2:
			baseSize = c * sp * 0.5 // volatile: 50% of base
		case 3:
			baseSize = 0            // crash: no trading
		}
	}
	if baseSize <= 0 {
		e.signalDiag.BaseSizeZero++
		return nil
	}

	if config.UseSeasonalityOverlay {
		szn := NewSeasonalityOverlay()
		baseSize *= szn.Multiplier(candle.Time)
	}
	baseSize *= e.kellyMult

	sr := e.getRunnerForSymbolAndStrategy(candle.Symbol, config.StrategyID, config)
	raw := sr.Evaluate(candle, regime)
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

	// Meta-labeling ML gate: predict p_win and suppress low-confidence signals
	if e.batchInferrer != nil && e.metaLabeler != nil && e.metaLabeler.IsHealthy() {
		metaResult := e.batchInferrer.Evaluate(nil, raw.PWin)
		if !metaResult.Accepted {
			e.signalDiag.MLRejected++
			return nil
		}
		baseSize *= math.Min(metaResult.PWin*1.5, 1.0)
	}

	positionPct := baseSize / c
	if positionPct > 0.03 {
		positionPct = 0.03
	}
	quantity := (c * positionPct) / candle.Close.Float64()
	if e.ftmo != nil && config.PropFirmEnabled {
		quantity = e.ftmo.GetPositionSize(quantity)
	}

	// ML regime enhancement: replace step-function regime multiplier with
	// continuous regime score if the enhancer is available.
	if e.regimeEnhancer != nil && e.regimeEnhancer.IsHealthy() {
		score, _ := e.regimeEnhancer.Evaluate(
			[4]float64{}, 0, 0, 0, 0.5, float64(candle.Time.Hour()),
		)
		e.positionSizer.SetRegimeScore(score)
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
	raw := sr.Evaluate(candle, regime)
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
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1].Value > 0 {
			returns[i-1] = (equity[i].Value - equity[i-1].Value) / equity[i-1].Value
		}
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
	if stdDev == 0 {
		return 0
	}
	if barsPerDay <= 0 {
		barsPerDay = 1.0
	}
	annualFactor := math.Sqrt(252.0 * barsPerDay)
	return mean / stdDev * annualFactor
}

func calculateSortino(equity []EquityPoint, barsPerDay float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1].Value > 0 {
			returns[i-1] = (equity[i].Value - equity[i-1].Value) / equity[i-1].Value
		}
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	downVariance := 0.0
	for _, r := range returns {
		if r < 0 {
			downVariance += r * r
		}
	}
	downVariance /= float64(len(returns))
	if downVariance == 0 {
		return 0
	}
	downStdDev := math.Sqrt(downVariance)
	if barsPerDay <= 0 {
		barsPerDay = 1.0
	}
	annualFactor := math.Sqrt(252.0 * barsPerDay)
	return mean / downStdDev * annualFactor
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
		if grossWin > 0 {
			return 999.0
		}
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
	fillModel    model.FillModel
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

		if e.poolSim.Halted {
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
					EntryPrice: candle.Close,
					EntryTime:  candle.Time,
					StrategyID: sid,
				},
				StrategyID: sid,
			}
		}

		for sid, op := range openPositions {
			exitPrice := candle.Close.Float64()
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
	result.TotalReturn = capital - config.InitialCapital

	if result.NumTrades > 0 {
		result.WinRate = calculateWinRate(trades)
		result.SharpeRatio = calculateSharpe(equity, 1.0)
		result.MaxDrawdown = calculateMaxDrawdown(equity)
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

	if e.poolSim.Halted {
		result.ComplianceReport = &ComplianceReport{
			Passed:     false,
			HaltReason: e.poolSim.HaltReason,
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
		yw, _ := t.ExitTime.ISOWeek()
		_ = yw
		_ = w

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
