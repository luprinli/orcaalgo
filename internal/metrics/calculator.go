package metrics

import (
	"math"
	"sort"
	"time"
)

type Calculator struct {
	riskFreeRate float64
	tradingDays  int
}

func NewCalculator(riskFreeRate float64) *Calculator {
	if riskFreeRate < 0 {
		riskFreeRate = 0
	}
	return &Calculator{
		riskFreeRate: riskFreeRate,
		tradingDays:  252,
	}
}

func (c *Calculator) SetTradingDays(days int) {
	if days > 0 {
		c.tradingDays = days
	}
}

func (c *Calculator) ComputeSnapshot(equity []MetricEquityPoint, trades []TradeSummary) PerformanceSnapshot {
	snap := PerformanceSnapshot{
		Timestamp: time.Now().UTC(),
	}

	if len(equity) == 0 {
		return snap
	}

	last := equity[len(equity)-1]
	snap.Equity = last.Equity
	snap.Balance = last.Balance

	eqValues := make([]float64, len(equity))
	for i, p := range equity {
		eqValues[i] = p.Equity
	}

	dailyReturns := c.dailyReturnsFromEquity(equity)
	snap.NumTrades = len(trades)

	if len(dailyReturns) > 0 {
		returns := make([]float64, len(dailyReturns))
		for i, dr := range dailyReturns {
			returns[i] = dr.ReturnPct / 100.0
		}

		snap.WinRate = c.winRateFromReturns(returns)
		snap.Sharpe = c.Sharpe(returns)
		snap.Sortino = c.Sortino(returns, 0)
		snap.VaR95 = c.VaR(returns, 0.95)
		snap.CVaR95 = c.CVaR(returns, 0.95)

		startEquity := eqValues[0]
		endEquity := eqValues[len(eqValues)-1]
		days := equity[len(equity)-1].Timestamp.Sub(equity[0].Timestamp).Hours() / 24
		if days > 0 && startEquity > 0 {
			snap.CAGR = c.CAGR(startEquity, endEquity, int(days))
		}

		if len(equity) >= 2 {
			yesterdayEq := eqValues[len(eqValues)-2]
			if yesterdayEq > 0 {
				snap.DailyPnL = endEquity - yesterdayEq
				snap.DailyPnLPct = (endEquity - yesterdayEq) / yesterdayEq * 100.0
			}
		}

		ddPct, _, _ := c.maxDrawdownPct(eqValues)
		snap.MaxDrawdownPct = ddPct
		snap.DrawdownPct = c.currentDrawdownPct(eqValues)

		if snap.MaxDrawdownPct != 0 {
			snap.Calmar = snap.CAGR / (-snap.MaxDrawdownPct)
		}
		snap.UlcerIndex = c.ulcerIndexFromEquity(eqValues)
	}

	snap.ProfitFactor = c.profitFactorFromTrades(trades)

	return snap
}

func (c *Calculator) Sharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := average(returns)
	std := stddev(returns, mean)
	if std == 0 {
		return 0
	}
	annualMean := mean * float64(c.tradingDays)
	annualStd := std * math.Sqrt(float64(c.tradingDays))
	rfDaily := c.riskFreeRate / float64(c.tradingDays)
	return (annualMean - (rfDaily * float64(c.tradingDays))) / annualStd
}

func (c *Calculator) Sortino(returns []float64, mar float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := average(returns)
	downStd := 0.0
	numDownObs := 0
	for _, r := range returns {
		d := r - mar
		if d < 0 {
			downStd += d * d
			numDownObs++
		}
	}
	if numDownObs == 0 {
		return 1e9
	}
	downStd = math.Sqrt(downStd / float64(len(returns)))
	if downStd == 0 {
		return 1e9
	}
	annualMean := mean * float64(c.tradingDays)
	annualDown := downStd * math.Sqrt(float64(c.tradingDays))
	rfDaily := c.riskFreeRate / float64(c.tradingDays)
	return (annualMean - (rfDaily * float64(c.tradingDays))) / annualDown
}

func (c *Calculator) Calmar(cagr, maxDD float64) float64 {
	if maxDD >= 0 {
		return 0
	}
	return cagr / (-maxDD)
}

func (c *Calculator) VaR(returns []float64, confidence float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := average(returns)
	std := stddev(returns, mean)
	zScore := normInv(1 - confidence)
	return -(mean + zScore*std) * 100
}

func (c *Calculator) CVaR(returns []float64, confidence float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	varValue := c.VaR(returns, confidence)
	tailReturns := make([]float64, 0)
	for _, r := range returns {
		if r*100 <= -varValue {
			tailReturns = append(tailReturns, r)
		}
	}
	if len(tailReturns) == 0 {
		return varValue
	}
	return -average(tailReturns) * 100
}

func (c *Calculator) CAGR(startEquity, endEquity float64, days int) float64 {
	if days <= 0 || startEquity <= 0 {
		return 0
	}
	years := float64(days) / float64(c.tradingDays)
	totalReturn := endEquity / startEquity
	return (math.Pow(totalReturn, 1.0/years) - 1.0) * 100.0
}

func (c *Calculator) ProfitFactor(trades []TradeSummary) float64 {
	var grossProfit, grossLoss float64
	for _, t := range trades {
		if t.PnL > 0 {
			grossProfit += t.PnL
		} else {
			grossLoss += -t.PnL
		}
	}
	if grossLoss == 0 {
		if grossProfit > 0 {
			return 1e9
		}
		return 1.0
	}
	return grossProfit / grossLoss
}

// ComputeTradeDistribution derives the per-trade distribution of a backtest's
// trades. It is pure over the input and independent of the equity curve, so it
// can be unit-tested without a database. HoldDuration is expected in minutes.
func (c *Calculator) ComputeTradeDistribution(trades []TradeSummary) TradeDistribution {
	d := TradeDistribution{TotalTrades: len(trades)}
	if len(trades) == 0 {
		return d
	}

	pnls := make([]float64, 0, len(trades))
	pnlPcts := make([]float64, 0, len(trades))
	durations := make([]float64, 0, len(trades))
	var wins, losses []float64
	tickers := make(map[string]struct{}, len(trades))

	for _, t := range trades {
		pnls = append(pnls, t.PnL)
		pnlPcts = append(pnlPcts, t.PnLPct)
		if t.HoldDuration > 0 {
			durations = append(durations, t.HoldDuration/60.0)
		}
		tickers[t.Symbol] = struct{}{}
		if t.PnL > 0 {
			d.WinningTrades++
			wins = append(wins, t.PnL)
		} else if t.PnL < 0 {
			d.LosingTrades++
			losses = append(losses, t.PnL)
		}
	}

	d.WinRatePct = float64(d.WinningTrades) / float64(len(trades)) * 100.0
	d.AvgTradePnL = average(pnls)
	d.MedianTradePnL = median(pnls)
	d.AvgTradePnlPct = average(pnlPcts)
	d.MedianTradePnlPct = median(pnlPcts)
	d.BestTrade = maxValue(pnls)
	d.WorstTrade = minValue(pnls)
	d.AvgTradeDurationHours = average(durations)
	d.MedianTradeDurationHrs = median(durations)
	d.AvgWinningPnL = average(wins)
	d.AvgLosingPnL = average(losses)
	d.UniqueTickers = len(tickers)

	return d
}

func (c *Calculator) WinRate(trades []TradeSummary) float64 {
	if len(trades) == 0 {
		return 0
	}
	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(trades)) * 100.0
}

func (c *Calculator) UlcerIndex(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	sumSq := 0.0
	peak := equity[0]
	for _, e := range equity {
		if e > peak {
			peak = e
		}
		if peak > 0 {
			dd := (peak - e) / peak * 100.0
			sumSq += dd * dd
		}
	}
	return math.Sqrt(sumSq / float64(len(equity)))
}

func (c *Calculator) MaxDrawdown(equity []float64) (drawdownPct float64, peakIdx, troughIdx int) {
	if len(equity) < 2 {
		return 0, 0, 0
	}
	peakIdx = 0
	troughIdx = 0
	maxDD := 0.0
	peakVal := equity[0]
	for i, e := range equity {
		if e > peakVal {
			peakVal = e
			peakIdx = i
		}
		dd := (peakVal - e) / peakVal * 100.0
		if dd > maxDD {
			maxDD = dd
			troughIdx = i
		}
	}
	return -maxDD, peakIdx, troughIdx
}

func (c *Calculator) DailyReturnsFromEquity(equity []MetricEquityPoint) []DailyReturn {
	return c.dailyReturnsFromEquity(equity)
}

func (c *Calculator) MonthlyReturns(daily []DailyReturn) []MonthlyReturn {
	monthMap := make(map[[2]int]float64)
	for _, dr := range daily {
		key := [2]int{dr.Date.Year(), int(dr.Date.Month())}
		monthMap[key] += dr.ReturnPct
	}
	result := make([]MonthlyReturn, 0, len(monthMap))
	for key, ret := range monthMap {
		result = append(result, MonthlyReturn{
			Year:      key[0],
			Month:     key[1],
			ReturnPct: ret,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Year != result[j].Year {
			return result[i].Year < result[j].Year
		}
		return result[i].Month < result[j].Month
	})
	return result
}

func (c *Calculator) RollingSharpe(returns []float64, window int) []RollingMetric {
	if len(returns) < window || window < 2 {
		return nil
	}
	result := make([]RollingMetric, 0, len(returns)-window+1)
	for i := window - 1; i < len(returns); i++ {
		windowReturns := returns[i-window+1 : i+1]
		s := c.Sharpe(windowReturns)
		result = append(result, RollingMetric{
			Timestamp: time.Now().UTC(),
			Value:     s,
			Window:    window,
		})
	}
	return result
}

func (c *Calculator) dailyReturnsFromEquity(equity []MetricEquityPoint) []DailyReturn {
	if len(equity) < 2 {
		return nil
	}
	result := make([]DailyReturn, 0)
	for i := 1; i < len(equity); i++ {
		prev := equity[i-1]
		curr := equity[i]
		if prev.Equity > 0 {
			retPct := (curr.Equity - prev.Equity) / prev.Equity * 100.0
			result = append(result, DailyReturn{
				Date:      curr.Timestamp,
				ReturnPct: retPct,
				PnL:       curr.Equity - prev.Equity,
			})
		}
	}
	return result
}

func (c *Calculator) maxDrawdownPct(equity []float64) (drawdownPct float64, peakIdx, troughIdx int) {
	dd, p, t := c.MaxDrawdown(equity)
	return dd, p, t
}

func (c *Calculator) currentDrawdownPct(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}
	peak := equity[0]
	for _, e := range equity {
		if e > peak {
			peak = e
		}
	}
	current := equity[len(equity)-1]
	if peak == 0 {
		return 0
	}
	return -(peak - current) / peak * 100.0
}

func (c *Calculator) winRateFromReturns(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	wins := 0
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(returns)) * 100.0
}

func (c *Calculator) profitFactorFromTrades(trades []TradeSummary) float64 {
	return c.ProfitFactor(trades)
}

func (c *Calculator) ulcerIndexFromEquity(equity []float64) float64 {
	return c.UlcerIndex(equity)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func median(values []float64) float64 {
	filtered := make([]float64, 0, len(values))
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		return 0
	}
	sort.Float64s(filtered)
	mid := len(filtered) / 2
	if len(filtered)%2 == 0 {
		return (filtered[mid-1] + filtered[mid]) / 2.0
	}
	return filtered[mid]
}

func maxValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func stddev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func normInv(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	a := [6]float64{-3.969683028665376e+01, 2.209460984245205e+02,
		-2.759285104469687e+02, 1.383577518672690e+02,
		-3.066479806614716e+01, 2.506628277459239e+00}
	b := [5]float64{-5.447609879822406e+01, 1.615858368580409e+02,
		-1.556989798598866e+02, 6.680131188771972e+01,
		-1.328068155288572e+01}
	c := [6]float64{-7.784894002430293e-03, -3.223964580411365e-01,
		-2.400758277161838e+00, -2.549732539343734e+00,
		4.374664141464968e+00, 2.938163982698783e+00}
	d := [4]float64{7.784695709041462e-03, 3.224671290700398e-01,
		2.445134137142996e+00, 3.754408661907416e+00}

	plow := 0.02425
	phigh := 1 - plow
	var q, r, x float64

	if p < plow {
		q = math.Sqrt(-2 * math.Log(p))
		x = (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	} else if p <= phigh {
		q = p - 0.5
		r = q * q
		x = (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		x = -(((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	}
	return x
}
