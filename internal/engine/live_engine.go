package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

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

	PrevTickNS int64
	TickCount   uint64
	SignalCount uint64

	metaLabeler    ml.Predictor      // H1–H3: meta-labeler for signal gating + priority
	featureStore   *ml.FeatureStore  // H1: feature computation for ML inference
	exitOrch       *ml.ExitOrchestrator // H5: ML dynamic stop adjustment
	regimeEnhancer *ml.RegimeEnhancer   // H4: regime-adaptive sizing
	regimeDailyLossPct float64          // H4: cached daily loss limit
	openPositions  map[string]*backtest.ActiveStop  // H5: symbol → active stop

	lastVIX         float64 // live VIX reading for regime classifier
	lastSentiment   float64 // live sentiment score (0-100)
	lastCVD         float64 // live cumulative volume delta trend
	lastVolStructure float64 // ratio of short-term to long-term realized volatility

	StrategyHash  string  // content-addressable instance hash of the deployed strategy
	KellyFraction float64 // fractional Kelly multiplier (0.25 default)

	pipeline  *risk.RiskPipeline             // shared signal-audit pipeline (optional)
	multiPool *risk.MultiAccountCapitalPool   // per-account capital pools (optional)

	accountRegistries map[string]*strategy.Registry // per-account isolated strategy instances
	defaultRegistry   *strategy.Registry            // fallback for single-account (created from factories)

	slippageModel       backtest.SlippageModel // adaptive slippage model calibrated from observed fills
	slippageSampleCount int
}

func (e *LiveEngine) SetMetaLabeler(p ml.Predictor) { e.metaLabeler = p }
func (e *LiveEngine) SetFeatureStore(fs *ml.FeatureStore) { e.featureStore = fs }
func (e *LiveEngine) SetExitOrchestrator(orch *ml.ExitOrchestrator) { e.exitOrch = orch }
func (e *LiveEngine) SetRegimeEnhancer(re *ml.RegimeEnhancer) { e.regimeEnhancer = re }

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
		accountRegistries: make(map[string]*strategy.Registry),
	}
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

	reg := e.getRegistryForAccount(accountID)
	for _, runner := range reg.All() {
		if receiver, ok := runner.(strategy.VIXReceiver); ok {
			receiver.SetVIX(e.lastVIX)
		}
	}

	signals := e.getRegistryForAccount(accountID).EvaluateAll(goCandle, regimeInt8)

	var approvedSignals []*strategy.Signal
	hasML := e.metaLabeler != nil && e.metaLabeler.IsHealthy()

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
			if sp, ok := e.metaLabeler.(*ml.SubprocessPredictor); ok {
				result = sp.EvaluateSignal(features)
			} else {
				pWin, err := e.metaLabeler.Predict(features)
				if err != nil || pWin <= 0 {
					continue
				}
				result = ml.MetaLabelingResult{
					PWin:     pWin,
					Accepted: pWin >= 0.55,
				}
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
			result := e.pipeline.ProcessSignal(context.Background(), risk.ProcessSignalRequest{
				StrategyID:       "live",
				Symbol:           sig.Symbol,
				Side:             sig.Side,
				Price:            goCandle.Close.Float64(),
				Confidence:       pWin,
				BaseSize:         sig.Quantity,
				ExistingPosition: 0,
				RunningCapital:   e.regimeDailyLossPct * 10000,
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

	// Track new positions from approved signals
	for _, sig := range *approvedSignals {
		if sig.Side == "BUY" || sig.Side == "SELL" {
			stopType := backtest.StopLossATR
			side := sig.Side
			stopPrice := price * 0.98
			if side == "SELL" {
				stopPrice = price * 1.02
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

	// Also update the per-account capital pool if configured.
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

func NanoToTime(ns int64) time.Time {
	return time.Unix(0, ns)
}
