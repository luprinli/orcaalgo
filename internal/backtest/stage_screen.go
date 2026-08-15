package backtest

// StageOneResult holds the outcome of a cheap daily-screen backtest used to
// determine whether a (strategy, symbol) pair is viable before running the
// expensive intraday + optimization deep pass.
type StageOneResult struct {
	Strategy  string  `json:"strategy"`
	Symbol    string  `json:"symbol"`
	NumTrades int     `json:"num_trades"`
	Sharpe    float64 `json:"sharpe"`
	Viable    bool    `json:"viable"`
}

// ScreenStageOne runs a cheap daily pass on every unique (strategy, symbol)
// pair and returns only the viable pairs (NumTrades > 0). Non-viable pairs
// are returned separately so the deep-stage can skip their intraday combos.
//
// This is the two-stage funnel described in the execution framework (§3.3):
//
//	Stage 1 — Broad screen on 1d, default params, to eliminate non-viable pairs
//	Stage 2 — Deep run on intraday tfs + optimization for survivors only
func ScreenStageOne(db Database, config MatrixBacktestConfig) (viable []StageOneResult, skipped []StageOneResult, _ error) {
	type pair struct{ strategy, symbol string }
	seen := make(map[string]struct{})
	var pairs []pair
	for _, s := range config.StrategyIDs {
		for _, sym := range config.Symbols {
			k := s + "|" + sym
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			pairs = append(pairs, pair{s, sym})
		}
	}

	for _, p := range pairs {
		btCfg := BacktestConfig{
			StrategyID:     p.strategy,
			Symbols:        []string{p.symbol},
			StartDate:      config.StartDate,
			EndDate:        config.EndDate,
			InitialCapital: config.InitialCapital,
			Timeframe:      "1d",
			DataSource:     config.DataSource,
		}
		engine := NewEngine(db)
		result, err := engine.Run(nil, btCfg)
		if err != nil || result.NumTrades == 0 {
			skipped = append(skipped, StageOneResult{
				Strategy: p.strategy, Symbol: p.symbol,
				Viable: false,
			})
			continue
		}
		viable = append(viable, StageOneResult{
			Strategy:  p.strategy,
			Symbol:    p.symbol,
			NumTrades: result.NumTrades,
			Sharpe:    result.SharpeRatio,
			Viable:    true,
		})
	}
	return viable, skipped, nil
}

// FilterIntradayCombos returns only the combos whose (strategy, symbol) pair
// is in the viable set. This is called to rebuild the cartesian product after
// stage-one screening.
func FilterIntradayCombos(combos []batchTuple, viable []StageOneResult) []batchTuple {
	v := make(map[string]bool, len(viable))
	for _, sr := range viable {
		v[sr.Strategy+"|"+sr.Symbol] = true
	}
	var filtered []batchTuple
	for _, c := range combos {
		if v[c.Strategy+"|"+c.Symbol] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// StageTwoConfig wraps the matrix config to carry the upstream screening
// results and the deep-stage timeframes.
type StageTwoConfig struct {
	MatrixBacktestConfig
	Skipped  int `json:"skipped"`
	Survived int `json:"survived"`
}
