package model

type FeeModel interface {
	MakerFee(notional float64) float64
	TakerFee(notional float64) float64
}

type FixedFee struct {
	MakerBps float64
	TakerBps float64
}

func (f FixedFee) MakerFee(notional float64) float64 {
	return notional * f.MakerBps / 10000.0
}

func (f FixedFee) TakerFee(notional float64) float64 {
	return notional * f.TakerBps / 10000.0
}

type ZeroFee struct{}

func (z ZeroFee) MakerFee(notional float64) float64  { return 0 }
func (z ZeroFee) TakerFee(notional float64) float64   { return 0 }
