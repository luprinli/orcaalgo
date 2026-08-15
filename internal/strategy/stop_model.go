package strategy

import "github.com/lee-econ/orca-core/internal/types"

// This file is the single home for stop/target/trailing price arithmetic, so
// the price-vs-distance distinction is structural (Rule 3). `distance` is
// ALWAYS relative (float64); the returned value is ALWAYS absolute
// (types.Price). A trailing-stop bug (`peak - StopLoss-price` vs
// `peak - distance`) is impossible when callers use these helpers.

// StopPrice returns the absolute stop for a side at `distance` from `entry`.
func StopPrice(entry types.Price, distance float64, side string) types.Price {
	if side == "SELL" {
		return types.PriceFromFloat(entry.Float64() + distance)
	}
	return types.PriceFromFloat(entry.Float64() - distance)
}

// TargetPrice returns the absolute take-profit for a side at `distance` from `entry`.
func TargetPrice(entry types.Price, distance float64, side string) types.Price {
	if side == "SELL" {
		return types.PriceFromFloat(entry.Float64() - distance)
	}
	return types.PriceFromFloat(entry.Float64() + distance)
}

// TrailingStop returns the trailing stop for a side at `distance` from `peak`.
func TrailingStop(peak types.Price, distance float64, side string) types.Price {
	if side == "SELL" {
		return types.PriceFromFloat(peak.Float64() + distance)
	}
	return types.PriceFromFloat(peak.Float64() - distance)
}
