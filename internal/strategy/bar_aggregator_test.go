package strategy

import "testing"

func TestBarAggregator_AddTick(t *testing.T) {
	agg := NewBarAggregator(1)
	tickNS := int64(60_000_000_000)

	agg.AddTick(100_000, 100, tickNS)
	if agg.Current1M.Open != 100_000 {
		t.Errorf("expected open 100_000, got %d", agg.Current1M.Open)
	}

	agg.AddTick(101_000, 50, tickNS+1)
	if agg.Current1M.High != 101_000 {
		t.Errorf("expected high 101_000, got %d", agg.Current1M.High)
	}

	nextMin := tickNS + 60_000_000_000
	agg.AddTick(102_000, 75, nextMin)
	if agg.Count1M != 1 {
		t.Errorf("expected 1 completed bar, got %d", agg.Count1M)
	}
	if agg.Bars1M[0].Close != 101_000 {
		t.Errorf("expected close 101_000 from completed bar, got %d", agg.Bars1M[0].Close)
	}
}

func TestBarAggregator_MultipleTimeframes(t *testing.T) {
	agg := NewBarAggregator(1)
	baseNS := int64(60_000_000_000)

	for i := int64(0); i < 10; i++ {
		agg.AddTick(uint64(100_000+i*100), 100, baseNS+i)
	}

	if agg.Current1M.Timestamp == 0 {
		t.Error("1m bar should have timestamp")
	}
}

func TestBarAggregator_GetBars(t *testing.T) {
	agg := NewBarAggregator(1)
	baseNS := int64(60_000_000_000)

	for m := int64(0); m < 5; m++ {
		ts := baseNS + m*60_000_000_000
		for i := int64(0); i < 3; i++ {
			agg.AddTick(uint64(100_000+m*1000+i*10), 100, ts+i)
		}
	}

	bars := agg.GetBars("1m", 10)
	if len(bars) == 0 {
		t.Error("should return bars")
	}
	if bars[0].Volume == 0 {
		t.Error("volume should be non-zero")
	}
}

func TestBarAggregator_GetLatestBar(t *testing.T) {
	agg := NewBarAggregator(1)
	agg.AddTick(100_000, 100, 60_000_000_000)
	agg.AddTick(101_000, 50, 60_000_000_000+1)
	agg.AddTick(102_000, 75, 120_000_000_000)

	latest := agg.GetLatestBar("1m")
	if latest.Close != 101_000 {
		t.Errorf("expected close 101_000 from completed bar, got %d", latest.Close)
	}
}

func TestBarAggregator_EmptyGetBars(t *testing.T) {
	agg := NewBarAggregator(1)
	bars := agg.GetBars("1m", 10)
	if len(bars) != 0 {
		t.Errorf("should return empty for no bars, got %d bars", len(bars))
	}
}

func TestBarToCandle(t *testing.T) {
	bar := Bar{
		Timestamp: 60_000_000_000,
		Open:      100_000,
		High:      101_000,
		Low:       99_000,
		Close:     100_000,
		Volume:    1000,
	}
	candle := BarToCandle(bar)
	if candle.Open <= 0 || candle.High <= 0 {
		t.Error("candle prices should be > 0")
	}
}

func TestBarsToCandles(t *testing.T) {
	bars := []Bar{{Timestamp: 60_000_000_000, Open: 100_000, High: 101_000, Low: 99_000, Close: 100_000, Volume: 100}}
	candles := BarsToCandles(bars)
	if len(candles) != 1 {
		t.Errorf("expected 1 candle, got %d", len(candles))
	}
}
