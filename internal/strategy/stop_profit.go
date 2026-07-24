package strategy

import "github.com/lee-econ/orca-core/internal/types"

type StopLossChecker struct{}

func (StopLossChecker) IsStopLossHit(price, stopLoss types.Price, side string) bool {
	if side == "BUY" {
		return price.Compare(stopLoss) <= 0
	}
	return price.Compare(stopLoss) >= 0
}

type TakeProfitChecker struct{}

func (TakeProfitChecker) IsTakeProfitHit(price, takeProfit types.Price, side string) bool {
	if side == "BUY" {
		return price.Compare(takeProfit) >= 0
	}
	return price.Compare(takeProfit) <= 0
}
