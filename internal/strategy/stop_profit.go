package strategy

type StopLossChecker struct{}

func (StopLossChecker) IsStopLossHit(price, stopLoss float64, side string) bool {
	if side == "BUY" {
		return price <= stopLoss
	}
	return price >= stopLoss
}

type TakeProfitChecker struct{}

func (TakeProfitChecker) IsTakeProfitHit(price, takeProfit float64, side string) bool {
	if side == "BUY" {
		return price >= takeProfit
	}
	return price <= takeProfit
}
