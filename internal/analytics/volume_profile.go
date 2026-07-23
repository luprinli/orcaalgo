package analytics

import (
	"math"
	"sort"
	"time"
)

type VolumeLevel struct {
	Price      float64
	BuyVolume  float64
	SellVolume float64
	TotalVolume float64
	Delta      float64
}

type VolumeProfile struct {
	Levels     []VolumeLevel
	POC        float64
	VAH        float64
	VAL        float64
	VARange    float64
	TotalVolume float64
	RangeStart time.Time
	RangeEnd   time.Time
}

func ComputeVolumeProfile(prices []float64, volumes []float64, sides []uint8, numBins int) *VolumeProfile {
	if len(prices) == 0 || numBins <= 0 {
		return &VolumeProfile{}
	}

	minPrice := prices[0]
	maxPrice := prices[0]
	for _, p := range prices {
		if p < minPrice {
			minPrice = p
		}
		if p > maxPrice {
			maxPrice = p
		}
	}

	if minPrice == maxPrice {
		maxPrice = minPrice * 1.01
	}

	binSize := (maxPrice - minPrice) / float64(numBins)
	if binSize == 0 {
		binSize = 0.01
	}

	levels := make([]VolumeLevel, numBins)
	for i := range levels {
		levels[i].Price = minPrice + float64(i)*binSize + binSize/2
	}

	if len(volumes) == len(prices) && len(sides) == len(prices) {
		for i, p := range prices {
			bin := int((p - minPrice) / binSize)
			if bin < 0 {
				bin = 0
			}
			if bin >= numBins {
				bin = numBins - 1
			}
			levels[bin].TotalVolume += volumes[i]
			if sides[i] == 1 {
				levels[bin].BuyVolume += volumes[i]
			} else {
				levels[bin].SellVolume += volumes[i]
			}
			levels[bin].Delta = levels[bin].BuyVolume - levels[bin].SellVolume
		}
	} else {
		for i, p := range prices {
			bin := int((p - minPrice) / binSize)
			if bin < 0 {
				bin = 0
			}
			if bin >= numBins {
				bin = numBins - 1
			}
			v := 1.0
			if i < len(volumes) {
				v = volumes[i]
			}
			levels[bin].TotalVolume += v
		}
	}

	vp := &VolumeProfile{Levels: levels}

	var maxVol float64
	for _, l := range levels {
		vp.TotalVolume += l.TotalVolume
		if l.TotalVolume > maxVol {
			maxVol = l.TotalVolume
			vp.POC = l.Price
		}
	}

	vp.VAH, vp.VAL = computeValueArea(levels, 0.70, maxVol)
	vp.VARange = vp.VAH - vp.VAL

	return vp
}

func computeValueArea(levels []VolumeLevel, areaPct float64, maxVol float64) (float64, float64) {
	if len(levels) == 0 {
		return 0, 0
	}

	var pocIdx int
	for i, l := range levels {
		if l.TotalVolume >= maxVol {
			pocIdx = i
			break
		}
	}
	_ = pocIdx

	type indexedLevel struct {
		idx   int
		level VolumeLevel
	}
	var sorted []indexedLevel
	for i, l := range levels {
		sorted = append(sorted, indexedLevel{i, l})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].level.TotalVolume > sorted[j].level.TotalVolume
	})

	var totalVol float64
	for _, l := range levels {
		totalVol += l.TotalVolume
	}
	targetVol := totalVol * areaPct

	var accumulatedVol float64
	var minIdx, maxIdx int = len(levels), 0

	for _, sl := range sorted {
		if accumulatedVol >= targetVol {
			break
		}
		accumulatedVol += sl.level.TotalVolume
		if sl.idx < minIdx {
			minIdx = sl.idx
		}
		if sl.idx > maxIdx {
			maxIdx = sl.idx
		}
	}

	if minIdx == len(levels) {
		minIdx = 0
	}
	if maxIdx < minIdx {
		maxIdx = minIdx
	}

	return levels[maxIdx].Price, levels[minIdx].Price
}

type ValueAreaLevels struct {
	POC float64
	VAH float64
	VAL float64
}

func ComputePOC(prices []float64, volumes []float64, numBins int) float64 {
	vp := ComputeVolumeProfile(prices, volumes, nil, numBins)
	return vp.POC
}

func ComputeValueArea(prices []float64, volumes []float64, areaPct float64, numBins int) ValueAreaLevels {
	vp := ComputeVolumeProfile(prices, volumes, nil, numBins)
	return ValueAreaLevels{POC: vp.POC, VAH: vp.VAH, VAL: vp.VAL}
}

func CalculateVWAP(prices []float64, volumes []float64) float64 {
	if len(prices) == 0 || len(volumes) == 0 {
		return 0
	}
	var totalValue, totalVolume float64
	for i, p := range prices {
		v := 1.0
		if i < len(volumes) {
			v = volumes[i]
		}
		totalValue += p * v
		totalVolume += v
	}
	if totalVolume == 0 {
		return 0
	}
	return totalValue / totalVolume
}

func CalculateStandardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sum, mean float64
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))
	var variance float64
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}
