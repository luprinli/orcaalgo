package engine

import (
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

	StrategyHash string // content-addressable instance hash of the deployed strategy
}

func (e *LiveEngine) SetMetaLabeler(p ml.Predictor) { e.metaLabeler = p }
func (e *LiveEngine) SetFeatureStore(fs *ml.FeatureStore) { e.featureStore = fs }
func (e *LiveEngine) SetExitOrchestrator(orch *ml.ExitOrchestrator) { e.exitOrch = orch }
func (e *LiveEngine) SetRegimeEnhancer(re *ml.RegimeEnhancer) { e.regimeEnhancer = re }

type SymbolState struct {
	SymbolID   uint32
	OrderBook  *market.OrderBook
	Aggregator *strategy.BarAggregator
	PrevPrice  int64
}

func NewLiveEngine() *LiveEngine {
	return &LiveEngine{
		Symbols:        make(map[uint32]*SymbolState),
		RiskState:      risk.NewGlobalRiskState(),
		openPositions:  make(map[string]*backtest.ActiveStop),
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

	signals := strategy.GlobalRegistry().EvaluateAll(goCandle, regimeInt8)

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
			result := e.metaLabeler.(*ml.SubprocessPredictor).EvaluateSignal(features)
			if !result.Accepted {
				continue // H1: gate — reject low-confidence signals
			}
			sig.PWin = result.PWin // H2: priority field
			// H3: PWin-weighted sizing — scale Kelly by confidence
			if result.PWin > 0 {
				sig.Quantity *= result.PWin / 0.5
			}
		}

		approvedSignals = append(approvedSignals, sig)
	}

	if hasML && len(approvedSignals) > 1 {
		sort.Slice(approvedSignals, func(i, j int) bool {
			return approvedSignals[i].PWin > approvedSignals[j].PWin
		})
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
	price := goCandle.Close
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
				EntryPrice: price,
				Side:       side,
				StopPrice:  stopPrice,
				StopType:   stopType,
				PeakPrice:  price,
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
			if stop.Side == "BUY" && newStop > stop.StopPrice {
				stop.StopPrice = newStop
			}
			if stop.Side == "SELL" && newStop < stop.StopPrice {
				stop.StopPrice = newStop
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
	score, _ := e.regimeEnhancer.Evaluate(hmmAlpha, 20.0, 50, 0.0, 0.5, 12.0)
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

func (e *LiveEngine) SignalOutcome(symbol string, side string, pnl float64) int {
	if pnl > 0 {
		return 1
	}
	return 0
}

func NanoToTime(ns int64) time.Time {
	return time.Unix(0, ns)
}
