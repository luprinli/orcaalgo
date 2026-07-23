package api

import (
	"math"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
)

const (
	priceMatchToleranceBps = 5.0
	timeMatchTolerance     = 1 * time.Second
)

func MatchReconciliation(internal []db.TradeExecution, brokerFills []broker.TradeFill) ReconciliationResult {
	result := ReconciliationResult{
		InternalCount: len(internal),
	}

	if len(brokerFills) == 0 {
		result.Extra = len(internal)
		result.Matched = 0
		result.Missing = 0
		if len(internal) > 0 {
			result.Date = internal[0].ExecutedAt.Format("2006-01-02")
		}
		return result
	}

	matchedBroker := make([]bool, len(brokerFills))

	for _, exec := range internal {
		matched := -1
		execPrice := exec.Price
		execTime := exec.ExecutedAt

		for j, fill := range brokerFills {
			if matchedBroker[j] {
				continue
			}
			if !symbolsMatch(exec.Symbol, fill.Symbol) {
				continue
			}
			if !sidesMatch(exec.Side, string(fill.Side)) {
				continue
			}
			if !quantitiesMatch(exec.Quantity, fill.Quantity) {
				continue
			}
			if !timesClose(execTime, fill.FillTime) {
				continue
			}

			fillPrice := fill.FillPrice.Float64()
			diffBps := math.Abs(execPrice-fillPrice) / math.Max(math.Abs(execPrice), math.Abs(fillPrice)) * 10000
			if diffBps <= priceMatchToleranceBps {
				matched = j
				result.Matched++
				break
			}

			result.PriceDiscrepancies++
			result.Discrepancies = append(result.Discrepancies, DiscrepancyDetail{
				OrderID:    exec.ID,
				Internal:   execPrice,
				Broker:     fillPrice,
				DiffBps:    diffBps,
			})
			matched = j
			break
		}

		if matched >= 0 {
			matchedBroker[matched] = true
		} else {
			result.Extra++
		}
	}

	for i, matched := range matchedBroker {
		if !matched {
			result.Missing++
			_ = i
		}
	}
	if len(internal) > 0 && result.Date == "" {
		result.Date = internal[0].ExecutedAt.Format("2006-01-02")
	}

	return result
}

func symbolsMatch(internal, broker string) bool {
	if internal == "" || broker == "" {
		return true
	}
	return internal == broker
}

func sidesMatch(internal, broker string) bool {
	return internal == broker || internal == "" || broker == ""
}

func quantitiesMatch(internal, broker float64) bool {
	diff := math.Abs(internal - broker)
	return diff < 0.01
}

func timesClose(internal, broker time.Time) bool {
	if internal.IsZero() || broker.IsZero() {
		return true
	}
	diff := internal.Sub(broker)
	if diff < 0 {
		diff = -diff
	}
	return diff <= timeMatchTolerance
}

var _ = timeMatchTolerance
