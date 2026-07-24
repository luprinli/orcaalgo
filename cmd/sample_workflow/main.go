// Package sample demonstrates a complete runnable workflow from raw price data
// to trading signals using the unified backtest/live engine with multiple
// indicator-based strategies.
//
// Build: go build -o sample_workflow ./cmd/sample_workflow/
// Run:   ./sample_workflow
package main

import (
	"fmt"
	"time"

	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func main() {
	fmt.Println("=== OrcaAlgo Sample Workflow ===")
	fmt.Println("Raw price data → indicators → strategy evaluation → buy/sell signals")
	fmt.Println()

	// -- STEP 1: Generate synthetic price data (simulating daily candles) --
	prices := generateSyntheticData(200)
	fmt.Printf("Generated %d synthetic daily candles\n", len(prices))

	// -- STEP 2: Print the first 10 raw candles --
	fmt.Println("\n--- Raw Price Data (first 10 candles) ---")
	fmt.Println("Time                 Open     High     Low      Close    Volume")
	for i := 0; i < 10 && i < len(prices); i++ {
		c := prices[i]
		fmt.Printf("%s  %7.2f  %7.2f  %7.2f  %7.2f  %7.0f\n",
			c.Time.Format("2006-01-02"),
			c.Open.Float64(), c.High.Float64(), c.Low.Float64(), c.Close.Float64(), c.Volume)
	}

	// -- STEP 3: Compute indicators on the price history --
	closing := make([]float64, len(prices))
	for i := range prices {
		closing[i] = prices[i].Close.Float64()
	}

	fast := make([]float64, len(closing))
	slow := make([]float64, len(closing))
	rsiValues := make([]float64, len(closing))
	macdLines := make([]float64, len(closing))
	macdSignal := make([]float64, len(closing))
	bbUpper := make([]float64, len(closing))
	bbMiddle := make([]float64, len(closing))
	bbLower := make([]float64, len(closing))

	for i := 0; i < len(closing); i++ {
		count := i + 1
		fast[i] = strategy.EMA(closing, count, 9)
		slow[i] = strategy.EMA(closing, count, 21)
		rsiValues[i] = strategy.RSI(closing, count, 14)
		macdLines[i], macdSignal[i] = strategy.MACD(closing, count)
		bbUpper[i], bbMiddle[i], bbLower[i] = strategy.BollingerBands(closing, count)
	}

	// -- STEP 4: Print indicators at a mid-point snapshot --
	idx := 150
	fmt.Printf("\n--- Indicator Snapshot at candle %d ---\n", idx)
	fmt.Printf("Price:        %.2f\n", closing[idx])
	fmt.Printf("EMA(9):       %.2f\n", fast[idx])
	fmt.Printf("EMA(21):      %.2f\n", slow[idx])
	fmt.Printf("RSI(14):      %.2f\n", rsiValues[idx])
	fmt.Printf("MACD Line:    %.4f\n", macdLines[idx])
	fmt.Printf("MACD Signal:  %.4f\n", macdSignal[idx])
	fmt.Printf("BB Upper:     %.2f\n", bbUpper[idx])
	fmt.Printf("BB Middle:    %.2f\n", bbMiddle[idx])
	fmt.Printf("BB Lower:     %.2f\n", bbLower[idx])

	// -- STEP 5: Run the MACrossover strategy through the unified engine --
	fmt.Println("\n--- Running MACrossover Strategy ---")
	runner := strategy.NewMACrossoverRunner()
	var signals []*strategy.Signal
	for i := range prices {
		sig := runner.Evaluate(prices[i], 0)
		if sig != nil {
			signals = append(signals, sig)
		}
	}

	// -- STEP 6: Show generated signals --
	fmt.Printf("Generated %d signals across %d candles\n", len(signals), len(prices))
	fmt.Println("\n--- Trade Signals ---")
	fmt.Println("Candle  Time                 Side   Price")
	buyCount := 0
	sellCount := 0
	for _, sig := range signals {
		if sig.Side == "BUY" {
			buyCount++
		} else {
			sellCount++
		}
	}
	for i, sig := range signals {
		if i < 20 {
			fmt.Printf("%-7d  %s    %-5s  %.2f\n", i+1, prices[i].Time.Format("2006-01-02"), sig.Side, prices[i].Close.Float64())
		}
	}
	if len(signals) > 20 {
		fmt.Printf("... (%d more signals, %d BUY, %d SELL)\n", len(signals)-20, buyCount, sellCount)
	}
	fmt.Printf("\nSummary: %d BUY signals, %d SELL signals\n", buyCount, sellCount)

	// -- STEP 7: Show crossovers visually --
	fmt.Println("\n--- EMA Crossover Points (last 30 candles) ---")
	for i := len(closing) - 30; i < len(closing); i++ {
		arrow := " "
		if i > 0 && fast[i-1] <= slow[i-1] && fast[i] > slow[i] {
			arrow = "▲" // bullish crossover
		} else if i > 0 && fast[i-1] >= slow[i-1] && fast[i] < slow[i] {
			arrow = "▼" // bearish crossover
		}
		fmt.Printf("%s  Close: %7.2f  EMA9: %7.2f  EMA21: %7.2f  RSI: %5.1f  %s\n",
			prices[i].Time.Format("2006-01-02"), closing[i], fast[i], slow[i], rsiValues[i], arrow)
	}

	fmt.Println("\n=== Workflow Complete ===")
}

// generateSyntheticData creates 200 candles with a trend + noise pattern.
func generateSyntheticData(n int) []strategy.Candle {
	candles := make([]strategy.Candle, n)
	base := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	price := 100.0

	for i := 0; i < n; i++ {
		if i > 0 {
			trend := 0.05
			if i > 80 && i < 120 {
				trend = -0.10
			}
			if i > 160 {
				trend = 0.15
			}
			noise := (float64(i%7) - 3.0) * 0.3
			price += trend + noise
			if price < 80 {
				price = 80
			}
		}

		high := price + float64(i%3)*0.5
		low := price - float64((i+1)%4)*0.4
		open := low + (high-low)*0.4

		candles[i] = strategy.Candle{
			Time:   base.AddDate(0, 0, i),
			Open:   types.PriceFromFloat(open),
			High:   types.PriceFromFloat(high),
			Low:    types.PriceFromFloat(low),
			Close:  types.PriceFromFloat(price),
			Volume: 1000 + float64(i%50)*100,
			Symbol: "SAMPLE",
		}
	}
	return candles
}
