package strategy

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

const (
	MaxBars1M  = 256
	MaxBars5M  = 256
	MaxBars15M = 128
	MaxBars1H  = 64
)

type Bar struct {
	Timestamp int64
	Open      int64
	High      int64
	Low       int64
	Close     int64
	Volume    uint64
	VWAP      int64
	Count     int
}

type BarAggregator struct {
	SymbolID   uint32
	Bars1M     [MaxBars1M]Bar
	Bars5M     [MaxBars5M]Bar
	Bars15M    [MaxBars15M]Bar
	Bars1H     [MaxBars1H]Bar
	Count1M    int
	Count5M    int
	Count15M   int
	Count1H    int
	Current1M  Bar
	Current5M  Bar
	Current15M Bar
	Current1H  Bar
}

const PriceScaleBar = int64(100_000)

func NewBarAggregator(symbolID uint32) *BarAggregator {
	return &BarAggregator{SymbolID: symbolID}
}

func (agg *BarAggregator) AddTick(priceRaw, volumeRaw uint64, timestamp int64) {
	price := int64(priceRaw)
	volume := volumeRaw
	ts := timestamp

	min1m := ts / 60_000_000_000
	min5m := ts / 300_000_000_000
	min15m := ts / 900_000_000_000
	min1h := ts / 3_600_000_000_000

	rollBar(&agg.Current1M, agg.Bars1M[:], &agg.Count1M, MaxBars1M, min1m, price, volume)
	rollBar(&agg.Current5M, agg.Bars5M[:], &agg.Count5M, MaxBars5M, min5m, price, volume)
	rollBar(&agg.Current15M, agg.Bars15M[:], &agg.Count15M, MaxBars15M, min15m, price, volume)
	rollBar(&agg.Current1H, agg.Bars1H[:], &agg.Count1H, MaxBars1H, min1h, price, volume)
}

func rollBar(current *Bar, bars []Bar, count *int, maxBars int, periodTs int64, price int64, volume uint64) {
	if current.Timestamp != periodTs {
		if current.Timestamp != 0 {
			if *count < maxBars {
				bars[*count] = *current
			} else {
				for i := 0; i < maxBars-1; i++ {
					bars[i] = bars[i+1]
				}
				bars[maxBars-1] = *current
			}
			*count++
			if *count > maxBars {
				*count = maxBars
			}
		}
		*current = Bar{
			Timestamp: periodTs,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
			Count:     1,
		}
	} else {
		if price > current.High {
			current.High = price
		}
		if price < current.Low {
			current.Low = price
		}
		current.Close = price
		current.Volume += volume
		current.Count++
	}
}

func (agg *BarAggregator) GetBars(timeframe string, count int) []Bar {
	switch timeframe {
	case "1m":
		n := minInt(agg.Count1M, count)
		return agg.Bars1M[:n]
	case "5m":
		n := minInt(agg.Count5M, count)
		return agg.Bars5M[:n]
	case "15m":
		n := minInt(agg.Count15M, count)
		return agg.Bars15M[:n]
	case "1h":
		n := minInt(agg.Count1H, count)
		return agg.Bars1H[:n]
	}
	return nil
}

func (agg *BarAggregator) GetLatestBar(timeframe string) Bar {
	switch timeframe {
	case "1m":
		if agg.Count1M > 0 {
			return agg.Bars1M[agg.Count1M-1]
		}
	case "5m":
		if agg.Count5M > 0 {
			return agg.Bars5M[agg.Count5M-1]
		}
	case "15m":
		if agg.Count15M > 0 {
			return agg.Bars15M[agg.Count15M-1]
		}
	case "1h":
		if agg.Count1H > 0 {
			return agg.Bars1H[agg.Count1H-1]
		}
	}
	return Bar{}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BarToCandle(b Bar) Candle {
	scale := float64(PriceScaleBar)
	return Candle{
		Time:   time.Unix(0, b.Timestamp),
		Open:   types.PriceFromFloat(float64(b.Open) / scale),
		High:   types.PriceFromFloat(float64(b.High) / scale),
		Low:    types.PriceFromFloat(float64(b.Low) / scale),
		Close:  types.PriceFromFloat(float64(b.Close) / scale),
		Volume: float64(b.Volume),
	}
}

func BarsToCandles(bars []Bar) []Candle {
	candles := make([]Candle, len(bars))
	for i, b := range bars {
		candles[i] = BarToCandle(b)
	}
	return candles
}
