package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/hash"
	"github.com/lee-econ/orca-core/internal/market"
	"github.com/lee-econ/orca-core/internal/ml"
	"github.com/lee-econ/orca-core/internal/model"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

type LiveEngine struct {
	mu sync.RWMutex

	Symbols   map[uint32]*SymbolState
	RiskState *risk.GlobalRiskState
	Recorder  model.Recorder

	Running bool
	Halted  bool

	PrevTickNS  int64
	TickCount   uint64
	SignalCount uint64

	metaLabeler        ml.Predictor
	batchInferrer      *ml.BatchInferrer
	metaCfg            ml.MetaLabelerConfig
	featureStore       *ml.FeatureStore
	exitOrch           *ml.ExitOrchestrator
	regimeEnhancer     *ml.RegimeEnhancer
	regimeDailyLossPct float64
	openPositions      map[string]*backtest.ActiveStop

	lastVIX          float64
	lastSentiment    float64
	lastCVD          float64
	lastVolStructure float64
	lastDay          string

	StrategyHash   string
	KellyFraction  float64
	runningCapital float64

	pipeline  *risk.RiskPipeline
	multiPool *risk.MultiAccountCapitalPool

	accountRegistries map[string]*strategy.Registry
	defaultRegistry   *strategy.Registry

	slippageModel       backtest.SlippageModel
	slippageSampleCount int

	warmUpCount int
	warmUpTicks map[uint32]int
	stopLossCfg *backtest.StopLossConfig
}

func (e *LiveEngine) SetMetaLabeler(p ml.Predictor) {
	e.metaLabeler = p
	e.batchInferrer = ml.NewBatchInferrer(p, e.metaCfg)
}
func (e *LiveEngine) SetMetaLabelerConfig(cfg ml.MetaLabelerConfig) {
	e.metaCfg = cfg
	if e.metaLabeler != nil {
		e.batchInferrer = ml.NewBatchInferrer(e.metaLabeler, cfg)
	}
}
func (e *LiveEngine) SetFeatureStore(fs *ml.FeatureStore)           { e.featureStore = fs }
func (e *LiveEngine) SetExitOrchestrator(orch *ml.ExitOrchestrator) { e.exitOrch = orch }
func (e *LiveEngine) SetRegimeEnhancer(re *ml.RegimeEnhancer)       { e.regimeEnhancer = re }

// SetRiskPipeline injects the shared signal-audit pipeline. When set, every
// approved signal in ProcessTick runs through ProcessSignal, and reconcile
// calls update the pipeline's capital and prop-firm state.
func (e *LiveEngine) SetRiskPipeline(p *risk.RiskPipeline) {
	e.pipeline = p
}

// SetMultiAccountPool injects per-account capital pools for live multi-account
// deployments.
func (e *LiveEngine) SetMultiAccountPool(mp *risk.MultiAccountCapitalPool) {
	e.multiPool = mp
}

// RegisterAccountStrategies creates an isolated strategy registry for the given
// accountID, populated with factory-created instances and per-account parameters.
// Each account gets its own independent strategy instances with private state
// (indicator buffers, open positions, rolling windows). If params is non-empty,
// each strategy receives its account-specific tuning via SetParams.
func (e *LiveEngine) RegisterAccountStrategies(accountID string, params map[string]map[string]float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	reg := strategy.NewRegistry()
	global := strategy.GlobalRegistry()

	for name, factory := range global.Factories() {
		instance := factory()
		if instance == nil {
			continue
		}
		if accountParams, ok := params[name]; ok {
			instance.SetParams(accountParams)
			// Backtest/live parity: apply the optimized per-regime exit multipliers
			// (stop_mult_highvol/crisis, profit_mult_trending) exactly as the
			// backtest engine does in getRunnerForStrategy.
			if rc, ok := instance.(interface{ SetRegimeExitParams(map[string]float64) }); ok {
				rc.SetRegimeExitParams(accountParams)
			}
			// Backtest/live parity: apply the optimized per-regime participation
			// weights (regime_w_*) to the shared pipeline matrix so a promoted
			// strategy trades the same regimes live as it did in backtest.
			if e.pipeline != nil && e.pipeline.RegimeMatrix != nil {
				risk.ApplyRegimeParticipation(e.pipeline.RegimeMatrix, name, accountParams)
			}
		}
		reg.Register(instance)
	}

	e.accountRegistries[accountID] = reg
}

// getRegistryForAccount returns the per-account registry if one exists,
// falling back to the shared default registry (created lazily from global
// factories) for single-account deployments.
func (e *LiveEngine) getRegistryForAccount(accountID string) *strategy.Registry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if reg, ok := e.accountRegistries[accountID]; ok {
		return reg
	}
	if e.defaultRegistry == nil {
		// Lazily create the default registry on first access.
		reg := strategy.NewRegistry()
		global := strategy.GlobalRegistry()
		for _, factory := range global.Factories() {
			instance := factory()
			if instance != nil {
				reg.Register(instance)
			}
		}
		e.defaultRegistry = reg
	}
	return e.defaultRegistry
}

type SymbolState struct {
	SymbolID   uint32
	OrderBook  *market.OrderBook
	Aggregator *strategy.BarAggregator
	PrevPrice  int64
}

func NewLiveEngine() *LiveEngine {
	return &LiveEngine{
		Symbols:           make(map[uint32]*SymbolState),
		RiskState:         risk.NewGlobalRiskState(),
		openPositions:     make(map[string]*backtest.ActiveStop),
		KellyFraction:     0.25,
		runningCapital:    100000.0,
		accountRegistries: make(map[string]*strategy.Registry),
		warmUpCount:       20,
		warmUpTicks:       make(map[uint32]int),
		stopLossCfg: &backtest.StopLossConfig{
			Type:          backtest.StopLossATR,
			ATRPeriod:     14,
			ATRMultiplier: 2.0,
		},
	}
}

func (e *LiveEngine) SetRunningCapital(capital float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.runningCapital = capital
}

func (e *LiveEngine) GetRunningCapital() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.runningCapital
}

func (e *LiveEngine) SetWarmUpBars(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.warmUpCount = n
}

func (e *LiveEngine) SetStopLossConfig(cfg *backtest.StopLossConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopLossCfg = cfg
}

// VerifyStrategyHash hard-fails if the engine has no strategy hash (unverified
// deployment) or if the given expected hash does not match. Empty StrategyHash
// means the engine was deployed without content-addressable verification —
// which is fatal for live operation.
func (e *LiveEngine) VerifyStrategyHash(gkrPath, expected string) error {
	if expected == "" {
		return fmt.Errorf("live engine: strategy hash is empty — deployment must be content-verified")
	}
	if e.StrategyHash != "" && e.StrategyHash != expected {
		return fmt.Errorf("live engine: strategy hash mismatch — deployed=%s expected=%s", e.StrategyHash, expected)
	}
	actual, err := hash.ComputeInstanceHash(gkrPath)
	if err != nil {
		return fmt.Errorf("live engine: hash verification failed: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("live engine: strategy hash mismatch — computed=%s expected=%s", actual, expected)
	}
	e.StrategyHash = actual
	return nil
}

func (e *LiveEngine) GetOrCreateSymbol(symbolID uint32) *SymbolState {
	e.mu.Lock()
	defer e.mu.Unlock()

	if s, ok := e.Symbols[symbolID]; ok {
		return s
	}
	s := &SymbolState{
		SymbolID:   symbolID,
		OrderBook:  market.NewOrderBook(symbolID),
		Aggregator: strategy.NewBarAggregator(symbolID),
	}
	e.Symbols[symbolID] = s
	return s
}

func (e *LiveEngine) ProcessTick(symbolID uint32, priceRaw, volumeRaw uint64, timestampNS int64) []*strategy.Signal {
	return e.ProcessTickForAccount("default", symbolID, priceRaw, volumeRaw, timestampNS)
}

func (e *LiveEngine) ProcessTickForAccount(accountID string, symbolID uint32, priceRaw, volumeRaw uint64, timestampNS int64) []*strategy.Signal {
	if e.Halted {
		return nil
	}

	s := e.GetOrCreateSymbol(symbolID)

	if s.PrevPrice > 0 {
		priceI64 := int64(priceRaw)
		e.RiskState.HMMTracker.Update(priceI64, s.PrevPrice)
	}
	s.PrevPrice = int64(priceRaw)

	s.Aggregator.AddTick(priceRaw, volumeRaw, timestampNS)

	regime := e.RiskState.HMMTracker.CurrentState
	regimeInt8 := int8(regime)

	candle := s.Aggregator.GetLatestBar("1m")
	goCandle := strategy.BarToCandle(candle)

	currentDay := time.Unix(0, timestampNS).UTC().Format("2006-01-02")
	if e.lastDay != "" && currentDay != e.lastDay {
		if e.pipeline != nil && e.pipeline.Capital != nil {
			if pm, ok := e.pipeline.Capital.(interface{ ResetDaily() }); ok {
				pm.ResetDaily()
			}
		}
		if e.multiPool != nil {
			e.multiPool.ResetAllDaily()
		}
	}
	e.lastDay = currentDay

	if e.pipeline != nil {
		if sg, ok := e.pipeline.SignalGate.(*risk.SignalGateImpl); ok {
			sg.SetVIX(e.lastVIX)
			sg.SetRegime(regimeInt8)
		}
	}

	reg := e.getRegistryForAccount(accountID)
	for _, runner := range reg.All() {
		if receiver, ok := runner.(strategy.VIXReceiver); ok {
			receiver.SetVIX(e.lastVIX)
		}
	}

	e.warmUpTicks[symbolID]++
	if e.warmUpTicks[symbolID] < e.warmUpCount {
		for _, runner := range reg.All() {
			runner.Evaluate(goCandle, regimeInt8)
		}
		return nil
	}

	signals := e.getRegistryForAccount(accountID).EvaluateAll(goCandle, regimeInt8)

	var approvedSignals []*strategy.Signal
	hasML := e.batchInferrer != nil || (e.metaLabeler != nil && e.metaLabeler.IsHealthy())

	for _, sig := range signals {
		if sig == nil {
			continue
		}

		advResult := risk.CheckAdversarial(
			&e.RiskState.Adversarial,
			uint64(sig.Quantity),
			symbolID,
			timestampNS,
		)
		if advResult.TriggerKill {
			e.Halted = true
			e.RiskState.Halted = true
			e.RiskState.HaltReason = advResult.Reason
			return nil
		}
		if advResult.Reject {
			continue
		}

		if hasML {
			features := e.computeFeatures(symbolID)
			var result ml.MetaLabelingResult
			if e.batchInferrer != nil {
				result = e.batchInferrer.Evaluate(features, sig.PWin)
			} else {
				result = ml.MetaLabelingResult{Accepted: true, PWin: sig.PWin, Reason: "no_batch_inferrer"}
			}
			if !result.Accepted {
				continue
			}
			sig.PWin = result.PWin
		}

		approvedSignals = append(approvedSignals, sig)
	}

	if hasML && len(approvedSignals) > 1 {
		sort.Slice(approvedSignals, func(i, j int) bool {
			return approvedSignals[i].PWin > approvedSignals[j].PWin
		})
	}

	if e.pipeline != nil {
		filtered := approvedSignals[:0]
		e.pipeline.CurrentRegime = regimeInt8
		for _, sig := range approvedSignals {
			pWin := sig.PWin
			if pWin <= 0 {
				pWin = 0.5
			}
			capital := e.GetRunningCapital()
			stratID := sig.StrategyID
			if stratID == "" {
				stratID = "live"
			}
			result := e.pipeline.ProcessSignal(context.Background(), risk.ProcessSignalRequest{
				StrategyID:       stratID,
				Symbol:           sig.Symbol,
				Side:             sig.Side,
				Price:            goCandle.Close.Float64(),
				Confidence:       pWin,
				BaseSize:         sig.Quantity,
				ExistingPosition: 0,
				RunningCapital:   capital,
			})
			if result.Approved {
				sig.Quantity = result.Size
				filtered = append(filtered, sig)
			}
		}
		approvedSignals = filtered
	} else {
		kelly := e.KellyFraction
		if kelly <= 0 {
			kelly = 0.25
		}
		for _, sig := range approvedSignals {
			pWin := sig.PWin
			if pWin <= 0 {
				pWin = 0.5
			}
			sig.Quantity *= math.Min(pWin*1.5, 1.0)
			sig.Quantity *= kelly
		}
	}

	e.CheckOpenStops(symbolID, s, goCandle, &approvedSignals)

	if e.TickCount%5000 == 0 {
		e.UpdateRegimeRiskLimit()
	}

	e.TickCount++
	e.SignalCount += uint64(len(approvedSignals))
	e.PrevTickNS = timestampNS
	return approvedSignals
}

func (e *LiveEngine) Halt(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Halted = true
	e.RiskState.Halted = true
	e.RiskState.HaltReason = reason
	// Clear all open positions on halt
	e.openPositions = make(map[string]*backtest.ActiveStop)
}

func (e *LiveEngine) CheckOpenStops(symbolID uint32, s *SymbolState, goCandle strategy.Candle, approvedSignals *[]*strategy.Signal) {
	price := goCandle.Close.Float64()
	if price <= 0 {
		return
	}

	atrVal := 0.0
	cfg := e.stopLossCfg
	if cfg != nil && cfg.Type == backtest.StopLossATR && cfg.ATRPeriod > 0 {
		bars := s.Aggregator.GetBars("1m", cfg.ATRPeriod+1)
		if len(bars) >= cfg.ATRPeriod+1 {
			atrVal = backtest.ComputeATR(strategy.BarsToCandles(bars), cfg.ATRPeriod)
		}
	}

	// Track new positions from approved signals
	for _, sig := range *approvedSignals {
		if sig.Side == "BUY" || sig.Side == "SELL" {
			side := sig.Side
			var stopPrice float64
			var stopType backtest.StopLossType
			if cfg != nil {
				stopPrice = backtest.CalculateStopPrice(price, side, cfg, atrVal, goCandle.High.Float64())
				stopType = cfg.Type
			}
			if stopPrice <= 0 {
				stopPrice = price * 0.98
				if side == "SELL" {
					stopPrice = price * 1.02
				}
				stopType = backtest.StopLossFixed
			}
			e.openPositions[sig.Symbol] = &backtest.ActiveStop{
				EntryPrice: types.PriceFromFloat(price),
				Side:       side,
				StopPrice:  types.PriceFromFloat(stopPrice),
				StopType:   stopType,
				PeakPrice:  types.PriceFromFloat(price),
			}
		}
	}

	// Check existing positions for stop hits and compute ML dynamic stops
	for symbol, stop := range e.openPositions {
		bCandle := backtest.Candle{
			High: goCandle.High, Low: goCandle.Low,
			Close: goCandle.Close, Open: goCandle.Open,
		}

		// H5: ML dynamic stop adjustment
		if e.exitOrch != nil && e.exitOrch.IsHealthy() {
			ctx := ml.ExitContext{
				EntryPrice:     stop.EntryPrice,
				CurrentPrice:   goCandle.Close,
				CurrentStop:    stop.StopPrice,
				BarsSinceEntry: 1,
				ATR:            1.0,
			}
			newStop := e.exitOrch.ComputeNewStop(stop.Side, stop.EntryPrice, goCandle.Close, 1.0, ctx)
			// Only ratchet: tighten stops, never widen
			if stop.Side == "BUY" && newStop > stop.StopPrice.Float64() {
				stop.StopPrice = types.PriceFromFloat(newStop)
			}
			if stop.Side == "SELL" && newStop < stop.StopPrice.Float64() {
				stop.StopPrice = types.PriceFromFloat(newStop)
			}
		}

		// Check stop hit
		if hit, exitPrice := backtest.CheckStopHit(bCandle, stop); hit {
			flip := "SELL"
			if stop.Side == "SELL" {
				flip = "BUY"
			}
			*approvedSignals = append(*approvedSignals, &strategy.Signal{
				Symbol:   symbol,
				Side:     flip,
				Quantity: 0,
			})
			delete(e.openPositions, symbol)
			_ = exitPrice
		}
	}
}

func (e *LiveEngine) computeFeatures(symbolID uint32) []float32 {
	if e.featureStore == nil {
		return nil
	}
	s := e.GetOrCreateSymbol(symbolID)
	candle := s.Aggregator.GetLatestBar("1m")
	goCandle := strategy.BarToCandle(candle)
	e.featureStore.Push(goCandle)

	ts := time.Unix(0, e.PrevTickNS)
	hmmAlpha := [4]float64{
		e.RiskState.HMMTracker.Alpha[0],
		e.RiskState.HMMTracker.Alpha[1],
		e.RiskState.HMMTracker.Alpha[2],
		e.RiskState.HMMTracker.Alpha[3],
	}
	confidence := e.RiskState.HMMTracker.Confidence

	fv, err := e.featureStore.Compute(ts, hmmAlpha, confidence, 1, 0.75, 0.0, 0.001)
	if err != nil {
		return nil
	}
	if !fv.Validate() {
		return nil
	}
	return fv.ToSlice()
}

func (e *LiveEngine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Halted = false
	e.RiskState.Halted = false
	e.RiskState.HaltReason = ""
}

func (e *LiveEngine) UpdateRegimeRiskLimit() {
	if e.regimeEnhancer == nil || !e.regimeEnhancer.IsHealthy() {
		return
	}
	hmmAlpha := [4]float64{
		e.RiskState.HMMTracker.Alpha[0],
		e.RiskState.HMMTracker.Alpha[1],
		e.RiskState.HMMTracker.Alpha[2],
		e.RiskState.HMMTracker.Alpha[3],
	}
	vix := e.lastVIX
	sentiment := int(e.lastSentiment)
	cvd := e.lastCVD
	volStructure := e.lastVolStructure
	hour := float64(time.Now().Hour())
	score, _ := e.regimeEnhancer.Evaluate(hmmAlpha, vix, sentiment, cvd, volStructure, hour)
	switch {
	case score > 0.8:
		e.regimeDailyLossPct = 3.0
	case score > 0.5:
		e.regimeDailyLossPct = 2.0
	case score > 0.3:
		e.regimeDailyLossPct = 1.0
	default:
		e.regimeDailyLossPct = 0.5
	}
}

func (e *LiveEngine) UpdateRegimeInputs(vix, sentiment, cvd, volStructure float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastVIX = vix
	e.lastSentiment = sentiment
	e.lastCVD = cvd
	e.lastVolStructure = volStructure
}

func (e *LiveEngine) SignalOutcome(symbol string, side string, pnl float64) int {
	if e.pipeline != nil {
		price := 0.0
		if s, ok := e.openPositions[symbol]; ok {
			price = s.EntryPrice.Float64()
		}
		e.pipeline.ReconcileFill("global", symbol, side, pnl, 1.0, price)
	}
	if pnl > 0 {
		return 1
	}
	return 0
}

// ReconcileLiveFill notifies the pipeline of a completed trade fill with full
// data. Used when the caller has access to strategy ID, quantity, and fill price.
func (e *LiveEngine) ReconcileLiveFill(strategyID, symbol, side string, pnl, quantity, price float64) {
	if e.pipeline == nil {
		return
	}
	e.pipeline.ReconcileFill(strategyID, symbol, side, pnl, quantity, price)

	if expectedPrice, ok := e.openPositions[symbol]; ok {
		e.RecordSlippageObservation(symbol, expectedPrice.EntryPrice.Float64(), price)
	}

	if e.multiPool != nil {
		// The multiPool records fills by account; for single-engine deployments
		// we use the pool associated with the running engine's first account.
		for _, aid := range e.multiPool.AccountIDs() {
			e.multiPool.RecordFill(aid, strategyID, symbol, side, pnl, quantity)
		}
	}
}

// SetSlippageModel sets the initial or replacement slippage model for adaptive
// calibration. Call before starting the engine.
func (e *LiveEngine) SetSlippageModel(m backtest.SlippageModel) {
	e.slippageModel = m
	e.slippageSampleCount = 0
}

// GetSlippageModel returns the current slippage model after calibration.
func (e *LiveEngine) GetSlippageModel() backtest.SlippageModel {
	return e.slippageModel
}

// RecordSlippageObservation feeds an observed fill (expected price vs actual
// fill price) into the adaptive slippage calibration pipeline. After 10+
// observations, the model is recalibrated using CalibrateSlippageModel.
func (e *LiveEngine) RecordSlippageObservation(symbol string, expectedPrice, actualPrice float64) {
	if expectedPrice <= 0 || actualPrice <= 0 {
		return
	}
	observedBps := (actualPrice - expectedPrice) / expectedPrice * 10000.0
	if observedBps < 0 {
		observedBps = -observedBps
	}
	e.slippageSampleCount++
	e.slippageModel = backtest.CalibrateSlippageModel(e.slippageModel, observedBps, e.slippageSampleCount)
}

// PersistFeatureStore persists the ML feature store to the database for
// state preservation across engine restarts. Call during graceful shutdown.
func (e *LiveEngine) PersistFeatureStore(ctx context.Context, pool *pgxpool.Pool) {
	if e.featureStore == nil || pool == nil {
		return
	}
	e.featureStore.Persist(ctx, pool, "global", "latest")
}

// LoadFeatureStore restores the ML feature store from the database on
// engine startup. Call after NewLiveEngine() and before starting processing.
func (e *LiveEngine) LoadFeatureStore(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	restored, _, err := ml.LoadFeatureStore(ctx, pool, "global")
	if err != nil {
		slog.Warn("feature store load failed, starting fresh", "error", err, "component", "live")
		return
	}
	e.featureStore = restored
}

func NanoToTime(ns int64) time.Time {
	return time.Unix(0, ns)
}
