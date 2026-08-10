package backtest

import "math"

type VIXAccelerationDetector struct {
	history   []vixPoint
	threshold float64
	window    int
}

type vixPoint struct {
	value    float64
	smoothed bool
}

func NewVIXAccelerationDetector(threshold float64) *VIXAccelerationDetector {
	return &VIXAccelerationDetector{
		threshold: threshold,
		window:    2,
	}
}

func (v *VIXAccelerationDetector) Feed(rawVIX, smoothVIX float64) float64 {
	point := vixPoint{value: rawVIX, smoothed: false}
	v.history = append(v.history, point)

	if len(v.history) > v.window+1 {
		v.history = v.history[len(v.history)-v.window-1:]
	}

	if len(v.history) >= 2 {
		delta := math.Abs(v.history[len(v.history)-1].value - v.history[0].value)
		if delta > v.threshold {
			return rawVIX
		}
	}
	return smoothVIX
}

func (v *VIXAccelerationDetector) IsSpike(rawVIX, smoothVIX float64) bool {
	v.Feed(rawVIX, smoothVIX)
	if len(v.history) >= 2 {
		delta := math.Abs(v.history[len(v.history)-1].value - v.history[0].value)
		return delta > v.threshold
	}
	return false
}

func (v *VIXAccelerationDetector) Reset() {
	v.history = nil
}
