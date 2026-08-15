package backtest

import (
	"math"
	"strconv"
)

type MetricFunc func(equityCurve []EquityPoint, trades []Trade) float64

type MetricDef struct {
	Name        string
	Compute     MetricFunc
	Formatter   func(float64) string
	Group       string
	Description string
}

var registeredMetrics = make(map[string]MetricDef)

func init() {
	RegisterMetric(MetricDef{
		Name: "sharpe_ratio", Group: "Risk", Description: "Annualized Sharpe ratio",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeSharpe(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) },
	})
	RegisterMetric(MetricDef{
		Name: "sortino_ratio", Group: "Risk", Description: "Annualized Sortino ratio",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeSortino(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) },
	})
	RegisterMetric(MetricDef{
		Name: "max_drawdown_pct", Group: "Risk", Description: "Maximum peak-to-trough drawdown",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeMaxDrawdown(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" },
	})
	RegisterMetric(MetricDef{
		Name: "win_rate_pct", Group: "Performance", Description: "Percentage of winning trades",
		Compute:   func(_ []EquityPoint, trades []Trade) float64 { return computeWinRate(trades) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" },
	})
	RegisterMetric(MetricDef{
		Name: "profit_factor", Group: "Performance", Description: "Gross profit / gross loss ratio",
		Compute:   func(_ []EquityPoint, trades []Trade) float64 { return computeProfitFactor(trades) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) },
	})
	RegisterMetric(MetricDef{
		Name: "total_return_pct", Group: "Performance", Description: "Total return percentage",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeTotalReturn(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" },
	})
	RegisterMetric(MetricDef{
		Name: "num_trades", Group: "Performance", Description: "Total number of trades",
		Compute:   func(_ []EquityPoint, trades []Trade) float64 { return float64(len(trades)) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) },
	})
	RegisterMetric(MetricDef{
		Name: "max_drawdown_duration_days", Group: "Risk", Description: "Max drawdown duration in trading days",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeMaxDrawdownDuration(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) + " days" },
	})
	RegisterMetric(MetricDef{
		Name: "cagr_pct", Group: "Performance", Description: "Compound Annual Growth Rate",
		Compute:   func(eq []EquityPoint, _ []Trade) float64 { return computeCAGR(eq) },
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" },
	})
	RegisterMetric(MetricDef{
		Name: "trading_volume", Group: "Performance", Description: "Total trading volume",
		Compute: func(_ []EquityPoint, trades []Trade) float64 {
			var vol float64
			for _, t := range trades {
				vol += t.Quantity
			}
			return vol
		},
		Formatter: func(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) },
	})
}

func RegisterMetric(def MetricDef) {
	registeredMetrics[def.Name] = def
}

func AllMetrics() map[string]MetricDef {
	return registeredMetrics
}

func ComputeAllMetrics(equityCurve []EquityPoint, trades []Trade) map[string]float64 {
	result := make(map[string]float64)
	for name, def := range registeredMetrics {
		result[name] = def.Compute(equityCurve, trades)
	}
	return result
}

func computeSharpe(equity []EquityPoint) float64 {
	returns := equityToDailyReturns(equity)
	if len(returns) < 2 {
		return 0
	}
	mean := mean(returns)
	std := stdDev(returns)
	if std == 0 {
		return 0
	}
	return mean / std * math.Sqrt(252)
}

func CalculateMtmSharpe(equity []EquityPoint, barsPerDay float64) float64 {
	return calculateMtmSharpe(equity, barsPerDay)
}

func calculateMtmSharpe(equity []EquityPoint, barsPerDay float64) float64 {
	return computeSharpe(equity)
}

func computeSortino(equity []EquityPoint) float64 {
	returns := equityToDailyReturns(equity)
	if len(returns) < 2 {
		return 0
	}
	mean := mean(returns)
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
	downside := math.Sqrt(sumSq / float64(count))
	downsideFloor := math.Abs(mean) * 0.10
	if downside < downsideFloor {
		return 0
	}
	sortino := mean / downside * math.Sqrt(252)
	if sortino > 20 {
		return 0
	}
	return sortino
}

func computeMaxDrawdown(equity []EquityPoint) float64 {
	if len(equity) == 0 {
		return 0
	}
	peak := equity[0].Value
	maxDD := 0.0
	for _, pt := range equity {
		if pt.Value > peak {
			peak = pt.Value
		}
		dd := (peak - pt.Value) / peak * 100.0
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

func computeWinRate(trades []Trade) float64 {
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

func computeProfitFactor(trades []Trade) float64 {
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
			return math.Inf(1)
		}
		return 0
	}
	pf := grossProfit / grossLoss
	if pf > 100 {
		return math.Inf(1)
	}
	return pf
}

func computeTotalReturn(equity []EquityPoint) float64 {
	if len(equity) < 2 {
		return 0
	}
	initial := equity[0].Value
	final := equity[len(equity)-1].Value
	if initial == 0 {
		return 0
	}
	return (final - initial) / initial * 100.0
}

func computeMaxDrawdownDuration(equity []EquityPoint) float64 {
	return float64(calculateMaxDrawdownDuration(equity))
}

func computeCAGR(equity []EquityPoint) float64 {
	if len(equity) < 2 {
		return 0
	}
	initial := equity[0].Value
	final := equity[len(equity)-1].Value
	if initial <= 0 || final <= 0 {
		return 0
	}
	days := equity[len(equity)-1].Time.Sub(equity[0].Time).Hours() / 24.0
	if days < 1 {
		return 0
	}
	years := days / 365.25
	ratio := final / initial
	return (math.Pow(ratio, 1.0/years) - 1.0) * 100.0
}

func equityToDailyReturns(equity []EquityPoint) []float64 {
	if len(equity) < 2 {
		return nil
	}
	type dayRange struct{ first, last float64 }
	dayMap := make(map[string]dayRange)
	var orderedDays []string
	for _, pt := range equity {
		key := pt.Time.Format("2006-01-02")
		if dr, exists := dayMap[key]; !exists {
			orderedDays = append(orderedDays, key)
			dayMap[key] = dayRange{first: pt.Value, last: pt.Value}
		} else {
			dr.last = pt.Value
			dayMap[key] = dr
		}
	}
	if len(orderedDays) < 2 {
		return nil
	}
	returns := make([]float64, len(orderedDays)-1)
	for i := 1; i < len(orderedDays); i++ {
		prev := dayMap[orderedDays[i-1]].last
		curr := dayMap[orderedDays[i]].last
		if prev > 0 {
			returns[i-1] = (curr - prev) / prev
		}
	}
	return returns
}

func equityToReturns(equity []EquityPoint) []float64 {
	if len(equity) < 2 {
		return nil
	}
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1].Value > 0 {
			returns[i-1] = (equity[i].Value - equity[i-1].Value) / equity[i-1].Value
		}
	}
	return returns
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := mean(values)
	var sumSq float64
	for _, v := range values {
		sumSq += (v - m) * (v - m)
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}
