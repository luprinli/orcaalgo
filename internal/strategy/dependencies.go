package strategy

import "github.com/lee-econ/orca-core/internal/types"

type VIXReceiver interface {
	SetVIX(vix float64)
}

type ATRReceiver interface {
	SetATR(atr float64)
}

type SecondaryPriceReceiver interface {
	PushSecondaryPrice(price types.Price)
}
