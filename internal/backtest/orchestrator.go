package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/lee-econ/orca-core/internal/propfirm"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

type OrchestratorConfig struct {
	Strategies             []OrchestratorStrategy
	StartDate              time.Time
	EndDate                time.Time
	InitialCapital         float64
	RebalanceBars          int
	KellyFraction          float64
	MaxPositionPct         float64
	AllowFractional        bool
	EnableCorrelationBrake bool
	CorrelationThreshold   float64
	FrictionModel          string
	CommissionBps          float64
}

type OrchestratorStrategy struct {
	StrategyID string
	Symbol     string
	Timeframe  string
}

type orchestratorEngine struct {
	strategyID    string
	symbol        string
	timeframe     string
	runner        strategy.Strategy
	pipeline      *risk.RiskPipeline
	positionSizer *risk.PositionSizer
	volHalt       *risk.VolatilityHalt
	exposure      *risk.ExposureTracker
	kellyMult     float64

	currentPosition float64
	entryPrice      float64 // float for PnL tracking; order entry price is types.Price
	side            string
}

type Orchestrator struct {
	db                Database
	engines           []*orchestratorEngine
	enginesByID       map[string]*orchestratorEngine
	enginesBySymbolTF map[string][]*orchestratorEngine
	pool              *CapitalPoolSim
	regimeMatrix      *risk.RegimeActivationMatrix
	scheduler         *RebalanceScheduler
	correlation       *CorrelationTracker
	vixDetector       *VIXAccelerationDetector
	reevaluator       *StrategyReevaluator
	registry          *strategy.Registry
	fillSim           *FillSimulator
	config            OrchestratorConfig
	orderIDSeq        uint32
}

func NewOrchestrator(db Database, cfg OrchestratorConfig) (*Orchestrator, error) {
	registry := strategy.GlobalRegistry()
	regimeMatrix := risk.NewRegimeActivationMatrix()

	var fillSim *FillSimulator
	if cfg.FrictionModel == "realistic" {
		fillSim = NewFillSimulator(RealisticEquitySlippage())
	} else {
		fillSim = NewFillSimulator(DefaultEquitySlippage())
	}

	if cfg.RebalanceBars <= 0 {
		cfg.RebalanceBars = 20
	}
	if cfg.CorrelationThreshold <= 0 {
		cfg.CorrelationThreshold = 0.6
	}
	if cfg.KellyFraction <= 0 {
		cfg.KellyFraction = 0.25
	}
	if cfg.MaxPositionPct <= 0.001 || cfg.MaxPositionPct > 0.20 {
		cfg.MaxPositionPct = 0.02
	}
	if cfg.CommissionBps <= 0 {
		cfg.CommissionBps = 2.0
	}

	return &Orchestrator{
		db:                db,
		enginesByID:       make(map[string]*orchestratorEngine),
		enginesBySymbolTF: make(map[string][]*orchestratorEngine),
		regimeMatrix:      regimeMatrix,
		scheduler:         NewRebalanceScheduler(cfg.RebalanceBars, regimeMatrix),
		correlation:       NewCorrelationTracker(30, cfg.CorrelationThreshold),
		vixDetector:       NewVIXAccelerationDetector(5.0),
		registry:          registry,
		fillSim:           fillSim,
		config:            cfg,
	}, nil
}

func (o *Orchestrator) engineID(sym, tf, sid string) string {
	return fmt.Sprintf("%s:%s:%s", sym, tf, sid)
}

func (o *Orchestrator) AddStrategy(sym, tf, sid string) error {
	key := o.engineID(sym, tf, sid)
	if _, exists := o.enginesByID[key]; exists {
		return fmt.Errorf("duplicate engine: %s", key)
	}

	runner := o.registry.Create(sid)
	if runner == nil {
		return fmt.Errorf("unknown strategy: %s", sid)
	}

	volHalt := risk.NewVolatilityHalt(3.0)
	positionSizer := risk.NewPositionSizer(nil)
	exposure := risk.NewExposureTracker(5.0, 0.25)

	signalGate := risk.NewSignalGateImpl(volHalt, positionSizer, exposure, nil)
	signalGate.SetBacktestMode(true)

	pipeline := &risk.RiskPipeline{
		SignalGate:   signalGate,
		Capital:      nil,
		PropFirm:     nil,
		KellyMult:    o.config.KellyFraction,
		RegimeMatrix: o.regimeMatrix,
	}

	eng := &orchestratorEngine{
		strategyID:    sid,
		symbol:        sym,
		timeframe:     tf,
		runner:        runner,
		pipeline:      pipeline,
		positionSizer: positionSizer,
		volHalt:       volHalt,
		exposure:      exposure,
		kellyMult:     o.config.KellyFraction,
	}

	o.engines = append(o.engines, eng)
	o.enginesByID[key] = eng

	symTFKey := fmt.Sprintf("%s:%s", sym, tf)
	o.enginesBySymbolTF[symTFKey] = append(o.enginesBySymbolTF[symTFKey], eng)

	return nil
}

type OrchestrationRunResult struct {
	PoolEquity          []EquityPoint         `json:"pool_equity"`
	PoolSharpe          float64               `json:"pool_sharpe"`
	PoolSortino         float64               `json:"pool_sortino"`
	PoolMaxDD           float64               `json:"pool_maxdd"`
	PoolReturnPct       float64               `json:"pool_return_pct"`
	RebalanceCosts      float64               `json:"rebalance_costs"`
	Trades              []Trade               `json:"trades"`
	StrategyPnL         map[string]float64    `json:"strategy_pnl"`
	ActiveCount         []int                 `json:"active_count"`
	AllocationHistory   []OrchAllocationPoint `json:"allocation_history"`
	CorrelationBreaches []BreachEvent         `json:"correlation_breaches"`
}

type OrchAllocationPoint struct {
	BarTime    time.Time `json:"bar_time"`
	StrategyID string    `json:"strategy_id"`
	Weight     float64   `json:"weight"`
	Capital    float64   `json:"capital"`
	IsActive   bool      `json:"is_active"`
}

func (o *Orchestrator) Run(ctx context.Context) (*OrchestrationRunResult, error) {
	profile := propfirm.DefaultFTMOProfile()

	regimeLogs, err := o.db.LoadRegimeLogs(ctx, o.config.StartDate, o.config.EndDate)
	if err != nil {
		regimeLogs = nil
	}
	regimeTimeIndex := make(map[int64]int8)
	if regimeLogs != nil {
		for _, r := range regimeLogs {
			regimeTimeIndex[r.Time.Unix()] = r.HMMState
		}
	}

	vixLogs, err := o.db.LoadVIXLogs(ctx, o.config.StartDate, o.config.EndDate)
	if err != nil {
		vixLogs = nil
	}

	type symTF struct{ symbol, tf string }
	var pairs []symTF
	seen := make(map[symTF]bool)
	for _, eng := range o.engines {
		p := symTF{eng.symbol, eng.timeframe}
		if !seen[p] {
			seen[p] = true
			pairs = append(pairs, p)
		}
	}

	var candleSets [][]Candle
	for _, p := range pairs {
		candles, err := o.db.LoadCandlesTF(ctx, []string{p.symbol}, o.config.StartDate, o.config.EndDate, p.tf)
		if err == nil && len(candles) > 0 {
			candleSets = append(candleSets, candles[0])
		}
	}

	allCandles := mergeCandlesByTime(candleSets)

	o.pool = NewCapitalPoolSim(profile, o.config.InitialCapital)
	for _, eng := range o.engines {
		eng.pipeline.Capital = o.pool
	}

	capital := o.config.InitialCapital
	peakCapital := o.config.InitialCapital
	equity := make([]EquityPoint, 0)
	trades := make([]Trade, 0)
	allocationHistory := make([]OrchAllocationPoint, 0)
	correlationBreaches := make([]BreachEvent, 0)
	strategyPnL := make(map[string]float64)
	activeCounts := make([]int, 0)

	type openPosition struct {
		Trade      Trade
		StrategyID string
	}
	openPositions := make(map[string]*openPosition)

	currentWeights := make(map[string]float64)
	strategyTrades := make(map[string][]Trade)

	var lastDay string
	for i := range allCandles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		candle := allCandles[i]

		currentDay := candle.Time.Format("2006-01-02")
		if currentDay != lastDay {
			o.pool.ResetDaily()
			lastDay = currentDay
		}

		regime := int8(0)
		if st, ok := regimeTimeIndex[candle.Time.Unix()]; ok {
			regime = st
		}

		var currentVIX float64
		for i := len(vixLogs) - 1; i >= 0; i-- {
			if vixLogs[i].Time.Before(candle.Time) || vixLogs[i].Time.Equal(candle.Time) {
				currentVIX = vixLogs[i].VIXValue
				break
			}
		}

		effectiveVIX := o.vixDetector.Feed(currentVIX, currentVIX)

		if o.pool.Halted() {
			continue
		}

		if o.scheduler.IsFullRebalanceDue() {
			var activeResults []EligibilityResult
			for _, eng := range o.engines {
				eid := o.engineID(eng.symbol, eng.timeframe, eng.strategyID)
				_, hasOpen := openPositions[eid]
				result := o.scheduler.EvaluateEligibility(eng.strategyID, regime, !hasOpen, eng.kellyMult)
				if result.Eligible {
					activeResults = append(activeResults, result)
				}
			}
			currentWeights = o.scheduler.ComputeWeights(activeResults)

			rbCost := o.executeRebalance(currentWeights, capital, candle)
			capital -= rbCost

			for _, eng := range o.engines {
				w := o.scheduler.ActiveWeight(currentWeights, eng.strategyID)
				allocationHistory = append(allocationHistory, OrchAllocationPoint{
					BarTime:    candle.Time,
					StrategyID: eng.strategyID,
					Weight:     w,
					Capital:    w * capital,
					IsActive:   w > 0,
				})
			}
		}

		_, breaches := o.correlation.CheckCorrelations()
		if o.config.EnableCorrelationBrake {
			correlationBreaches = append(correlationBreaches, breaches...)
		}

		for _, eng := range o.engines {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			eid := o.engineID(eng.symbol, eng.timeframe, eng.strategyID)
			if candle.Symbol != eng.symbol {
				continue
			}
			if _, hasOpen := openPositions[eid]; hasOpen {
				continue
			}

			if o.regimeMatrix.ParticipationForRegime(eng.strategyID, regime) <= 0 {
				continue
			}

			w := o.scheduler.ActiveWeight(currentWeights, eng.strategyID)
			kelly := o.scheduler.EvaluateEligibility(eng.strategyID, regime, true, eng.kellyMult).Kelly
			if kelly <= 0 {
				kelly = eng.kellyMult
			}

			if vixReceiver, ok := eng.runner.(strategy.VIXReceiver); ok {
				vixReceiver.SetVIX(effectiveVIX)
			}

			raw := eng.runner.Evaluate(candle, regime)
			if raw == nil || raw.Quantity == 0 {
				continue
			}

			sizingPct := o.config.MaxPositionPct * kelly
			if w > 0 {
				sizingPct = sizingPct * w
			} else {
				sizingPct = sizingPct * 0.5
			}
			price := candle.Close.Float64()
			baseSize := capital * sizingPct / price
			entrySide := raw.Side

			eng.pipeline.CurrentRegime = regime
			pipeResult := eng.pipeline.ProcessSignal(ctx, risk.ProcessSignalRequest{
				StrategyID:       eng.strategyID,
				Symbol:           candle.Symbol,
				Side:             entrySide,
				Price:            price,
				Confidence:       raw.PWin,
				BaseSize:         baseSize,
				ExistingPosition: eng.currentPosition,
				RunningCapital:   capital,
			})

			fillQty := baseSize
			if pipeResult.Approved && pipeResult.Size > 0 {
				fillQty = pipeResult.Size
			}
			minQty := 1.0
			if o.config.AllowFractional {
				minQty = 0.01
			}
			if fillQty < minQty {
				continue
			}

			o.orderIDSeq++
			orderID := o.orderIDSeq
			fillResult := o.fillSim.SimulateFillWithTCA(
				orderID, candle.Symbol, price, fillQty, entrySide,
				price, candle.Time, price, price, candle.Volume,
			)

			fp := fillResult.FillPrice.Float64()
			if fp <= 0 {
				fp = price
			}
			fq := fillResult.FillQuantity
			if fq <= 0 {
				fq = fillQty
			}

			eng.currentPosition = fq
			eng.entryPrice = fp
			eng.side = entrySide

			openPositions[eid] = &openPosition{
				Trade: Trade{
					Symbol:     candle.Symbol,
					Side:       entrySide,
					Quantity:   fq,
					EntryPrice: types.PriceFromFloat(fp),
					EntryTime:  candle.Time,
					StrategyID: eng.strategyID,
				},
				StrategyID: eng.strategyID,
			}
		}

		for eid, op := range openPositions {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			eng := o.enginesByID[eid]
			if eng == nil {
				delete(openPositions, eid)
				continue
			}

			price := candle.Close.Float64()

			o.orderIDSeq++
			orderID := o.orderIDSeq
			fillResult := o.fillSim.SimulateFillWithTCA(
				orderID, op.Trade.Symbol, price, op.Trade.Quantity,
				op.Trade.Side, price, candle.Time, price, price, candle.Volume,
			)

			fp := fillResult.FillPrice.Float64()
			if fp <= 0 {
				fp = price
			}

			var pnl float64
			if op.Trade.Side == "BUY" {
				pnl = (fp - op.Trade.EntryPrice.Float64()) * op.Trade.Quantity
			} else {
				pnl = (op.Trade.EntryPrice.Float64() - fp) * op.Trade.Quantity
			}

			commission := op.Trade.EntryPrice.Float64() * op.Trade.Quantity * (o.config.CommissionBps / 10000.0)
			pnl -= commission

			pnlPct := 0.0
			if capital > 0 {
				pnlPct = pnl / capital * 100
			}

			safePnl, _ := risk.SanitizeTradePnL(pnl, capital, op.Trade.Quantity, op.Trade.EntryPrice.Float64())

			op.Trade.PnL = safePnl
			op.Trade.PnLPct = pnlPct
			op.Trade.ExitPrice = types.PriceFromFloat(fp)
			op.Trade.ExitTime = candle.Time
			op.Trade.HMMRegime = regime
			op.Trade.SlippageMidBps = fillResult.SlippageMidBps
			op.Trade.SlippageLastBps = fillResult.SlippageLastBps

			capital += safePnl
			o.pool.RecordFill(eid, op.Trade.Symbol, op.Trade.Side, safePnl, op.Trade.Quantity)
			trades = append(trades, op.Trade)
			strategyPnL[eng.strategyID] += safePnl
			strategyTrades[eng.strategyID] = append(strategyTrades[eng.strategyID], op.Trade)
			o.scheduler.RecordSharpe(eng.strategyID, strategyTrades[eng.strategyID])
			delete(openPositions, eid)

			eng.currentPosition = 0
			eng.entryPrice = 0
			eng.side = ""
		}

		equity = append(equity, EquityPoint{
			Time:   candle.Time,
			Value:  capital,
			Regime: regime,
		})

		if capital > peakCapital {
			peakCapital = capital
		}

		for _, eng := range o.engines {
			o.correlation.RecordEquity(eng.strategyID, capital+strategyPnL[eng.strategyID])
		}

		activeCount := 0
		for _, eng := range o.engines {
			if o.scheduler.ActiveWeight(currentWeights, eng.strategyID) > 0 {
				activeCount++
			}
		}
		activeCounts = append(activeCounts, activeCount)
	}

	for eid, op := range openPositions {
		if len(allCandles) > 0 {
			lastCandle := allCandles[len(allCandles)-1]
			eng := o.enginesByID[eid]
			sid := ""
			if eng != nil {
				sid = eng.strategyID
			} else {
				sid = eid
			}
			fp := lastCandle.Close.Float64()

			var pnl float64
			if op.Trade.Side == "BUY" {
				pnl = (fp - op.Trade.EntryPrice.Float64()) * op.Trade.Quantity
			} else {
				pnl = (op.Trade.EntryPrice.Float64() - fp) * op.Trade.Quantity
			}
			safePnl, _ := risk.SanitizeTradePnL(pnl, capital, op.Trade.Quantity, op.Trade.EntryPrice.Float64())
			op.Trade.PnL = safePnl
			op.Trade.ExitPrice = types.PriceFromFloat(fp)
			op.Trade.ExitTime = lastCandle.Time
			capital += safePnl
			strategyPnL[sid] += safePnl
			trades = append(trades, op.Trade)
			delete(openPositions, eid)
		}
	}

	result := &OrchestrationRunResult{
		PoolEquity:          equity,
		Trades:              trades,
		StrategyPnL:         strategyPnL,
		ActiveCount:         activeCounts,
		AllocationHistory:   allocationHistory,
		CorrelationBreaches: correlationBreaches,
		PoolReturnPct:       (capital - o.config.InitialCapital) / o.config.InitialCapital * 100,
	}

	if len(equity) > 1 {
		result.PoolSharpe = calculateSharpe(equity, 1.0)
		result.PoolSortino = calculateSortino(equity, 1.0)
		result.PoolMaxDD = calculateMaxDrawdown(equity)
	}

	return result, nil
}

func (o *Orchestrator) executeRebalance(weights map[string]float64, capital float64, candle Candle) float64 {
	var totalCost float64
	if len(weights) == 0 {
		return 0
	}
	for sid, w := range weights {
		eng, ok := o.enginesByID[sid]
		if !ok {
			continue
		}
		targetCapital := w * capital
		if eng.currentPosition > 0 && eng.entryPrice > 0 {
			currentNotional := eng.currentPosition * eng.entryPrice
			delta := targetCapital - currentNotional
			if math.Abs(delta) < 1.0 {
				continue
			}
			cost := math.Abs(delta) * 0.0002
			totalCost += cost
		}
	}
	if math.IsNaN(totalCost) || math.IsInf(totalCost, 0) {
		return 0
	}
	return totalCost
}

func (r *OrchestrationRunResult) ToJSON() json.RawMessage {
	data, _ := json.Marshal(r)
	return data
}

type EnrichedOrchResult struct {
	PoolEquity          []EquityPoint            `json:"pool_equity"`
	Trades              []Trade                  `json:"trades"`
	DailyReturns        []DailyReturn            `json:"daily_returns"`
	MonthlyReturns      []MonthlyReturn          `json:"monthly_returns"`
	StrategyPnL         map[string]float64       `json:"strategy_pnl"`
	AllocationHistory   []OrchAllocationPoint    `json:"allocation_history"`
	CorrelationBreaches []BreachEvent            `json:"correlation_breaches"`
	ActiveCount         []int                    `json:"active_count"`
	PerStrategyStats    map[string]StrategyStats `json:"per_strategy_stats"`
	WinRate             float64                  `json:"win_rate"`
	ProfitFactor        float64                  `json:"profit_factor"`
	NumTrades           int                      `json:"num_trades"`
	NumWins             int                      `json:"num_wins"`
	NumLosses           int                      `json:"num_losses"`
	MonteCarlo          *MCResult                `json:"monte_carlo,omitempty"`
}

type StrategyStats struct {
	NumTrades    int     `json:"num_trades"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	TotalPnL     float64 `json:"total_pnl"`
}

type MonthlyReturn struct {
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	ReturnPct float64 `json:"return_pct"`
}

func EnrichResultJSON(result *OrchestrationRunResult) *EnrichedOrchResult {
	enriched := &EnrichedOrchResult{
		PoolEquity:          result.PoolEquity,
		Trades:              result.Trades,
		StrategyPnL:         result.StrategyPnL,
		AllocationHistory:   result.AllocationHistory,
		CorrelationBreaches: result.CorrelationBreaches,
		ActiveCount:         result.ActiveCount,
		PerStrategyStats:    make(map[string]StrategyStats),
	}

	enriched.DailyReturns = computeDailyReturnsFromEquity(result.PoolEquity)
	enriched.MonthlyReturns = computeMonthlyReturnsFromEquity(result.PoolEquity)
	enriched.NumTrades = len(result.Trades)

	var wins, losses int
	var grossProfit, grossLoss float64
	for _, t := range result.Trades {
		if t.PnL > 0 {
			wins++
			grossProfit += t.PnL
		} else if t.PnL < 0 {
			losses++
			grossLoss += -t.PnL
		}
	}
	enriched.NumWins = wins
	enriched.NumLosses = losses
	if wins+losses > 0 {
		enriched.WinRate = float64(wins) / float64(wins+losses)
	}
	if grossLoss > 0 {
		enriched.ProfitFactor = grossProfit / grossLoss
	}

	for sid, trades := range groupTradesByStrategy(result.Trades) {
		st := computeStrategyTradeStats(trades)
		st.TotalPnL = result.StrategyPnL[sid]
		enriched.PerStrategyStats[sid] = st
	}

	dailyReturnValues := extractDailyReturnValues(enriched.DailyReturns)
	if len(dailyReturnValues) >= 2 {
		enriched.MonteCarlo = RunPoolMonteCarlo(dailyReturnValues, 500)
	}

	return enriched
}

func computeDailyReturnsFromEquity(equity []EquityPoint) []DailyReturn {
	if len(equity) < 2 {
		return nil
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
		return nil
	}
	returns := make([]DailyReturn, 0, len(dayOrder)-1)
	for i := 1; i < len(dayOrder); i++ {
		prev := dayMap[dayOrder[i-1]]
		curr := dayMap[dayOrder[i]]
		if prev > 0 {
			returns = append(returns, DailyReturn{
				Date:   time.Time{},
				Return: (curr - prev) / prev,
			})
		}
	}
	return returns
}

func computeMonthlyReturnsFromEquity(equity []EquityPoint) []MonthlyReturn {
	if len(equity) < 2 {
		return nil
	}
	first := equity[0].Time
	last := equity[len(equity)-1].Time
	monthlyPnL := make(map[string]float64)
	for i := 1; i < len(equity); i++ {
		prev := equity[i-1].Value
		curr := equity[i].Value
		key := equity[i].Time.Format("2006-01")
		monthlyPnL[key] += curr - prev
	}

	monthEquity := equity[0].Value
	var returns []MonthlyReturn
	cursor := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, first.Location())
	endCursor := time.Date(last.Year(), last.Month(), 1, 0, 0, 0, 0, last.Location())
	for !cursor.After(endCursor) {
		key := cursor.Format("2006-01")
		pnl := monthlyPnL[key]
		startEq := monthEquity
		monthEquity += pnl
		if startEq > 0 {
			returns = append(returns, MonthlyReturn{
				Year:      cursor.Year(),
				Month:     int(cursor.Month()),
				ReturnPct: pnl / startEq * 100,
			})
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return returns
}

func groupTradesByStrategy(trades []Trade) map[string][]Trade {
	grouped := make(map[string][]Trade)
	for _, t := range trades {
		grouped[t.StrategyID] = append(grouped[t.StrategyID], t)
	}
	return grouped
}

func computeStrategyTradeStats(trades []Trade) StrategyStats {
	s := StrategyStats{NumTrades: len(trades)}
	if len(trades) == 0 {
		return s
	}
	var wins int
	var grossProfit, grossLoss float64
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
			grossProfit += t.PnL
		} else if t.PnL < 0 {
			grossLoss += -t.PnL
		}
	}
	if len(trades) > 0 {
		s.WinRate = float64(wins) / float64(len(trades))
	}
	if grossLoss > 0 {
		s.ProfitFactor = grossProfit / grossLoss
	}
	return s
}

func RunPoolMonteCarlo(dailyReturns []float64, iterations int) *MCResult {
	if iterations <= 0 {
		iterations = 500
	}
	if len(dailyReturns) < 2 {
		empty := &MCResult{
			Config:     MCConfig{Iterations: iterations, BarsPerSim: len(dailyReturns)},
			Iterations: []MCIterationResult{},
		}
		empty.Summary = computeMCSummary(empty.Iterations, empty.Config)
		return empty
	}

	results := make([]MCIterationResult, iterations)
	for i := 0; i < iterations; i++ {
		localRng := rand.New(rand.NewPCG(uint64(i), uint64(time.Now().UnixNano())))
		path := bootstrapBlockPath(dailyReturns, len(dailyReturns), localRng, 7)
		pnlPct, maxDD := computePathMetrics(path)
		results[i] = MCIterationResult{
			PnlPct:   pnlPct,
			MaxDDPct: maxDD,
		}
	}

	cfg := MCConfig{Iterations: iterations, BarsPerSim: len(dailyReturns)}
	return newMCResult(cfg, results)
}

func extractDailyReturnValues(dr []DailyReturn) []float64 {
	if len(dr) == 0 {
		return nil
	}
	vals := make([]float64, len(dr))
	for i, d := range dr {
		vals[i] = d.Return
	}
	return vals
}
