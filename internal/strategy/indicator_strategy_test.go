package strategy

import "testing"

func TestRSI2(t *testing.T) {
	prices := make([]float64, 50)
	for i := 0; i < 50; i++ {
		prices[i] = 100.0 + float64(i)*0.1
	}
	rsi2 := RSI2(prices, 50)
	if rsi2 < 0 || rsi2 > 100 {
		t.Errorf("RSI2 should be in [0,100], got %.2f", rsi2)
	}
}

func TestStochasticOscillator(t *testing.T) {
	n := 50
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
	}
	k, d := StochasticOscillator(highs, lows, closes, n)
	if k < 0 || k > 100 || d < 0 || d > 100 {
		t.Errorf("Stochastic values should be in [0,100], got k=%.1f d=%.1f", k, d)
	}
}

func TestIchimokuCloud(t *testing.T) {
	n := 100
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
	}
	ten, kij, sa, sb, ch := IchimokuCloud(highs, lows, closes, n)
	if ten <= 0 || kij <= 0 {
		t.Error("Ichimoku tenkan/kijun should be non-zero")
	}
	_ = sa
	_ = sb
	_ = ch
}

func TestDonchianChannel(t *testing.T) {
	closes := make([]float64, 60)
	for i := 0; i < 60; i++ {
		closes[i] = 100.0 + float64(i)*0.1
	}
	upper, middle, lower := DonchianChannel(closes, 60, 20)
	if upper <= 0 || middle <= 0 || lower <= 0 {
		t.Error("Donchian values should be non-zero")
	}
}

func TestKeltnerChannel(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.3
		lows[i] = closes[i] - 0.3
	}
	upper, middle, lower := KeltnerChannel(highs, lows, closes, n, 20)
	if middle <= 0 {
		t.Error("Keltner middle should be non-zero")
	}
	_ = upper
	_ = lower
}

func TestWilliamsR(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.05
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
	}
	wr := WilliamsR(highs, lows, closes, n)
	if wr > 0 || wr < -100 {
		t.Errorf("Williams %%R should be in [-100,0], got %.1f", wr)
	}
}

func TestAroon(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	for i := 0; i < n; i++ {
		highs[i] = 100.0 + float64(i)*0.1
		lows[i] = 99.0 + float64(i)*0.1
	}
	up, down := Aroon(highs, lows, n)
	if up < 0 || up > 100 || down < 0 || down > 100 {
		t.Errorf("Aroon values should be in [0,100], got up=%.1f down=%.1f", up, down)
	}
}

func TestMFI(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.3
		lows[i] = closes[i] - 0.3
		volumes[i] = 1000
	}
	mfi := MFI(highs, lows, closes, volumes, n, 14)
	if mfi < 0 || mfi > 100 {
		t.Errorf("MFI should be in [0,100], got %.1f", mfi)
	}
}

func TestVWAP(t *testing.T) {
	n := 20
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		volumes[i] = 1000
	}
	vwap := VWAP(closes, volumes, n, 14)
	if vwap <= 0 {
		t.Error("VWAP should be non-zero")
	}
}

func TestForceIndex(t *testing.T) {
	n := 20
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		volumes[i] = 1000
	}
	fi := ForceIndex(closes, volumes, n)
	_ = fi
}

func TestChandelierExit(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
	}
	longExit, shortExit := ChandelierExit(highs, lows, closes, n)
	if longExit <= 0 || shortExit <= 0 {
		t.Error("Chandelier Exit values should be non-zero")
	}
}

func TestOBV(t *testing.T) {
	n := 20
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		volumes[i] = 1000
	}
	obv := OBV(closes, volumes, n)
	if obv == 0 {
		t.Error("OBV should accumulate")
	}
}

func TestCMF(t *testing.T) {
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.3
		lows[i] = closes[i] - 0.3
		volumes[i] = 1000
	}
	cmf := CMF(highs, lows, closes, volumes, n)
	if cmf < -1 || cmf > 1 {
		t.Errorf("CMF should be in [-1,1], got %.3f", cmf)
	}
}

func TestRSI2Runner_SignalGeneration(t *testing.T) {
	r := NewRSI2MeanReversionRunner()
	for i := 0; i < 40; i++ {
		r.Evaluate(Candle{Symbol: "T", Close: 100.0 + float64(i)*0.5}, 0)
	}
	sig := r.Evaluate(Candle{Symbol: "T", Close: 100 + 40*0.5}, 0)
	if sig != nil {
		t.Logf("RSI2 signal: %s", sig.Side)
	}
}

func TestDonchianRunner(t *testing.T) {
	r := NewDonchianBreakoutRunner()
	if r.Name() != "donchian_breakout" {
		t.Error("wrong name")
	}
	if r.Type() != "breakout" {
		t.Error("wrong type")
	}
	r.SetParams(map[string]float64{"channel_period": 25})
	if r.ChannelPeriod != 25 {
		t.Error("SetParams failed")
	}
}

func TestKeltnerMACDRunner(t *testing.T) {
	r := NewKeltnerMACDRunner()
	if r.Name() != "keltner_macd" {
		t.Error("wrong name")
	}
	r.SetParams(map[string]float64{"macd_requirement": 0})
	if r.MacdRequirement {
		t.Error("SetParams should set MacdRequirement to false")
	}
}

func TestIchimokuRunner(t *testing.T) {
	r := NewIchimokuRunner()
	if r.Name() != "ichimoku_cloud" {
		t.Error("wrong name")
	}
	if r.CloudConfirm != true || r.UseChandelier != true {
		t.Error("defaults should be true")
	}
}

func TestAllStrategiesRegistered(t *testing.T) {
	reg := GlobalRegistry().All()
	names := make(map[string]bool)
	for _, s := range reg {
		names[s.Name()] = true
	}
	required := []string{
		"ma_crossover", "rsi2_reversion", "donchian_breakout",
		"keltner_macd", "ichimoku_cloud", "trend_following",
		"opening_range_breakout", "grid", "session_scalp", "mean_reversion",
	}
	for _, name := range required {
		if !names[name] {
			t.Errorf("strategy %s not registered", name)
		}
	}
}
