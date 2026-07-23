package backtest

func FilterByRegime(result *BacktestResult, regime int8) *BacktestResult {
	filtered := &BacktestResult{
		Config: result.Config,
	}

	trades := []Trade{}
	for _, t := range result.Trades {
		if t.HMMRegime == regime {
			trades = append(trades, t)
		}
	}

	equity := []EquityPoint{}
	for _, e := range result.EquityCurve {
		if e.Regime == regime {
			equity = append(equity, e)
		}
	}

	filtered.Trades = trades
	filtered.EquityCurve = equity
	filtered.NumTrades = len(trades)

	if filtered.NumTrades > 0 {
		filtered.WinRate = calculateWinRate(trades)
		filtered.SharpeRatio = calculateSharpe(equity, 1.0)
		filtered.MaxDrawdown = calculateMaxDrawdown(equity)
	}

	return filtered
}

func computeRegimeStats(result *BacktestResult) []RegimeStat {
	regimes := make(map[int8]*RegimeStat)

	for _, t := range result.Trades {
		rs, ok := regimes[t.HMMRegime]
		if !ok {
			rs = &RegimeStat{Regime: t.HMMRegime}
			switch t.HMMRegime {
			case 0:
				rs.Label = "Calm"
			case 1:
				rs.Label = "Trending"
			case 2:
				rs.Label = "High Vol"
			case 3:
				rs.Label = "Crisis"
			default:
				rs.Label = "Unknown"
			}
			regimes[t.HMMRegime] = rs
		}
		rs.NumTrades++
		if t.PnL > 0 {
			rs.WinRate++
		}
		rs.TotalReturn += t.PnL
	}

	stats := []RegimeStat{}
	for _, rs := range regimes {
		if rs.NumTrades > 0 {
			rs.WinRate = rs.WinRate / float64(rs.NumTrades) * 100.0
		}
		stats = append(stats, *rs)
	}
	return stats
}
