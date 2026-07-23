package strategy

import (
	"math"

	"github.com/cinar/indicator"
)

func ATR(prices []float64, count int, period int) float64 {
	if count < period+1 || period <= 0 {
		return 0
	}
	start := count - period - 1
	if start < 0 {
		start = 0
	}
	sum := 0.0
	n := 0
	for i := start; i < count-1 && i < len(prices)-1; i++ {
		diff := prices[i+1] - prices[i]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func TrueRangeATR(highs, lows, closes []float64, count int, period int) float64 {
	if count < period || period <= 0 || len(highs) == 0 {
		return 0
	}
	end := count
	start := end - period
	if start < 0 {
		start = 0
	}
	_, atrSeries := indicator.Atr(period, highs, lows, closes)
	if len(atrSeries) == 0 {
		return 0
	}
	idx := len(atrSeries) - 1
	if idx >= len(atrSeries) {
		idx = len(atrSeries) - 1
	}
	return atrSeries[idx]
}

func EMA(values []float64, count int, period int) float64 {
	if count < period || period <= 0 || len(values) == 0 {
		return 0
	}
	window := values
	if count < len(values) {
		window = values[:count]
	}
	emaSeries := indicator.Ema(period, window)
	if len(emaSeries) == 0 {
		return 0
	}
	return emaSeries[len(emaSeries)-1]
}

func SMA(values []float64, count int, period int) float64 {
	if count < period || period <= 0 || len(values) == 0 {
		return 0
	}
	window := values
	if count < len(values) {
		window = values[:count]
	}
	smaSeries := indicator.Sma(period, window)
	if len(smaSeries) == 0 {
		return 0
	}
	return smaSeries[len(smaSeries)-1]
}

func Mean(values []float64, count int, lookback int) float64 {
	if count <= 0 || lookback <= 0 || len(values) == 0 {
		return 0
	}
	n := lookback
	if count < n {
		n = count
	}
	start := count - n
	if start < 0 {
		start = 0
	}
	sum := 0.0
	actual := 0
	for i := start; i < count; i++ {
		idx := i % len(values)
		sum += values[idx]
		actual++
	}
	if actual == 0 {
		return 0
	}
	return sum / float64(actual)
}

func StdDev(values []float64, count int, period int) float64 {
	if count < period || period <= 1 || len(values) == 0 {
		return 0
	}
	window := values
	if count < len(values) {
		window = values[:count]
	}
	sdSeries := indicator.Std(period, window)
	if len(sdSeries) == 0 {
		return 0
	}
	return sdSeries[len(sdSeries)-1]
}

func ZScore(value, mean, std float64) float64 {
	if std <= 0 {
		return 0
	}
	return (value - mean) / std
}

func ADX(prices, highs, lows []float64, count int, period int) float64 {
	if count < period*2 || period <= 0 {
		return 0
	}
	trValues := make([]float64, count-1)
	for i := 1; i < count && i < len(prices); i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - prices[i-1])
		lc := math.Abs(lows[i] - prices[i-1])
		trValues[i-1] = math.Max(hl, math.Max(hc, lc))
	}
	upMoves := make([]float64, count-1)
	downMoves := make([]float64, count-1)
	for i := 1; i < count && i < len(prices); i++ {
		up := highs[i] - highs[i-1]
		down := lows[i-1] - lows[i]
		if up > 0 && up > down {
			upMoves[i-1] = up
		}
		if down > 0 && down > up {
			downMoves[i-1] = down
		}
	}
	if len(trValues) == 0 {
		return 0
	}
	atr := trValues[0]
	plusDI := upMoves[0]
	minusDI := downMoves[0]
	alpha := 2.0 / (float64(period) + 1.0)
	for i := 1; i < len(trValues); i++ {
		atr = alpha*trValues[i] + (1.0-alpha)*atr
		plusDI = alpha*upMoves[i] + (1.0-alpha)*plusDI
		minusDI = alpha*downMoves[i] + (1.0-alpha)*minusDI
	}
	if atr <= 0 {
		return 0
	}
	pdi := plusDI / atr * 100
	mdi := minusDI / atr * 100
	var dxSum float64
	if pdi+mdi > 0 {
		dxSum = math.Abs(pdi-mdi) / (pdi + mdi) * 100
	}
	emaDX := dxSum
	for i := 1; i < period && i < len(trValues); i++ {
		atrI := trValues[0]
		pdiI := upMoves[0]
		mdiI := downMoves[0]
		a := 2.0 / (float64(period) + 1.0)
		for j := 1; j <= i; j++ {
			atrI = a*trValues[j] + (1.0-a)*atrI
			pdiI = a*upMoves[j] + (1.0-a)*pdiI
			mdiI = a*downMoves[j] + (1.0-a)*mdiI
		}
		var prevDX float64
		if atrI > 0 && pdiI+mdiI > 0 {
			prevDX = math.Abs(pdiI-mdiI) / (pdiI + mdiI) * 100
		}
		emaDX = a*prevDX + (1.0-a)*emaDX
	}
	return emaDX
}

// RSI returns the Relative Strength Index value for the given price window.
// period defaults to 14. count is the number of valid entries in values.
func RSI(values []float64, count int, period int) float64 {
	if count < period+1 || period <= 0 || len(values) == 0 {
		return 0
	}
	window := values
	if count < len(values) {
		window = values[:count]
	}
	_, rsiSeries := indicator.RsiPeriod(period, window)
	if len(rsiSeries) == 0 {
		return 0
	}
	return rsiSeries[len(rsiSeries)-1]
}

// RSI2 returns the 2-period RSI value — used in Connors-style mean reversion strategies.
// Extremely sensitive: values above 95-98 signal extreme overbought, below 2-5 signal extreme oversold.
func RSI2(values []float64, count int) float64 {
	if count < 5 || len(values) == 0 {
		return 0
	}
	window := values
	if count < len(values) {
		window = values[:count]
	}
	_, rsi2Series := indicator.Rsi2(window)
	if len(rsi2Series) == 0 {
		return 0
	}
	return rsi2Series[len(rsi2Series)-1]
}

// MACD returns the MACD line and signal line values.
// Uses default 12/26/9 periods. count is the number of valid entries in closing.
func MACD(closing []float64, count int) (float64, float64) {
	if count < 26 || len(closing) == 0 {
		return 0, 0
	}
	window := closing
	if count < len(closing) {
		window = closing[:count]
	}
	macdSeries, signalSeries := indicator.Macd(window)
	if len(macdSeries) == 0 || len(signalSeries) == 0 {
		return 0, 0
	}
	return macdSeries[len(macdSeries)-1], signalSeries[len(signalSeries)-1]
}

// BollingerBands returns the upper band, middle band (SMA), and lower band values.
// period defaults to 20 with 2 standard deviations. count is the number of valid entries in closing.
func BollingerBands(closing []float64, count int) (float64, float64, float64) {
	if count < 20 || len(closing) == 0 {
		return 0, 0, 0
	}
	window := closing
	if count < len(closing) {
		window = closing[:count]
	}
	upper, middle, lower := indicator.BollingerBands(window)
	if len(upper) == 0 || len(middle) == 0 || len(lower) == 0 {
		return 0, 0, 0
	}
	return upper[len(upper)-1], middle[len(middle)-1], lower[len(lower)-1]
}

// StochasticOscillator returns the %K and %D values (fast stochastic).
// Uses default 14,3,3 periods. Values above 80 = overbought, below 20 = oversold.
func StochasticOscillator(highs, lows, closes []float64, count int) (float64, float64) {
	if count < 14 || len(highs) < count || len(lows) < count || len(closes) < count {
		return 0, 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	k, d := indicator.StochasticOscillator(h, l, c)
	if len(k) == 0 || len(d) == 0 {
		return 0, 0
	}
	return k[len(k)-1], d[len(d)-1]
}

// IchimokuCloud returns the five Ichimoku components.
// Returns: tenkan, kijun, senkouA, senkouB, chikou.
// Bullish: price > cloud, tenkan > kijun. Bearish: price < cloud, tenkan < kijun.
func IchimokuCloud(highs, lows, closes []float64, count int) (float64, float64, float64, float64, float64) {
	if count < 52 || len(highs) < count || len(lows) < count || len(closes) < count {
		return 0, 0, 0, 0, 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	tenkan, kijun, senkouA, senkouB, chikou := indicator.IchimokuCloud(h, l, c)
	if len(tenkan) == 0 {
		return 0, 0, 0, 0, 0
	}
	return tenkan[len(tenkan)-1], kijun[len(kijun)-1], senkouA[len(senkouA)-1], senkouB[len(senkouB)-1], chikou[len(chikou)-1]
}

// DonchianChannel returns the upper, middle, and lower Donchian channel values.
// period defaults to 20. Upper = max(high, period), Lower = min(low, period).
func DonchianChannel(closing []float64, count int, period int) (float64, float64, float64) {
	if count < period || period <= 0 || len(closing) == 0 {
		return 0, 0, 0
	}
	window := closing
	if count < len(closing) {
		window = closing[:count]
	}
	upper, middle, lower := indicator.DonchianChannel(period, window)
	if len(upper) == 0 {
		return 0, 0, 0
	}
	return upper[len(upper)-1], middle[len(middle)-1], lower[len(lower)-1]
}

// KeltnerChannel returns the upper, middle, and lower Keltner channel values.
// Uses EMA with ATR bands. period defaults to 20.
func KeltnerChannel(highs, lows, closes []float64, count int, period int) (float64, float64, float64) {
	if count < period || period <= 0 || len(highs) < count || len(lows) < count || len(closes) < count {
		return 0, 0, 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	upper, middle, lower := indicator.KeltnerChannel(period, h, l, c)
	if len(upper) == 0 {
		return 0, 0, 0
	}
	return upper[len(upper)-1], middle[len(middle)-1], lower[len(lower)-1]
}

// WilliamsR returns the Williams %R value. Range: -100 to 0.
// Above -20 = overbought, below -80 = oversold. Uses default 14 period.
func WilliamsR(highs, lows, closes []float64, count int) float64 {
	if count < 14 || len(highs) < count || len(lows) < count || len(closes) < count {
		return 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	wr := indicator.WilliamsR(l, h, c)
	if len(wr) == 0 {
		return 0
	}
	return wr[len(wr)-1]
}

// Aroon returns the Aroon Up and Aroon Down values. Range: 0 to 100.
// AroonUp > 70 = strong uptrend. AroonDown > 70 = strong downtrend. Uses default 25 period.
func Aroon(highs, lows []float64, count int) (float64, float64) {
	if count < 25 || len(highs) < count || len(lows) < count {
		return 0, 0
	}
	h := highs[:count]
	l := lows[:count]
	up, down := indicator.Aroon(h, l)
	if len(up) == 0 || len(down) == 0 {
		return 0, 0
	}
	return up[len(up)-1], down[len(down)-1]
}

// MFI returns the Money Flow Index value. Range: 0 to 100.
// Volume-weighted RSI variant. Above 80 = overbought, below 20 = oversold.
func MFI(highs, lows, closes, volumes []float64, count int, period int) float64 {
	if count < period+1 || period <= 0 || len(highs) < count || len(lows) < count || len(closes) < count || len(volumes) < count {
		return 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	v := volumes[:count]
	mfi := indicator.MoneyFlowIndex(period, h, l, c, v)
	if len(mfi) == 0 {
		return 0
	}
	return mfi[len(mfi)-1]
}

// VWAP returns the Volume-Weighted Average Price value. Uses default 14 period.
func VWAP(closes, volumes []float64, count int, period int) float64 {
	if count < period || period <= 0 || len(closes) < count || len(volumes) < count {
		return 0
	}
	c := closes[:count]
	v := volumes[:count]
	vwap := indicator.Vwma(period, c, v)
	if len(vwap) == 0 {
		return 0
	}
	return vwap[len(vwap)-1]
}

// ForceIndex returns the Force Index value. Combines price change and volume.
// Positive = buying pressure. Negative = selling pressure. Uses 13-period EMA smoothing.
func ForceIndex(closes, volumes []float64, count int) float64 {
	if count < 14 || len(closes) < count || len(volumes) < count {
		return 0
	}
	c := closes[:count]
	v := volumes[:count]
	fi := indicator.ForceIndex(13, c, v)
	if len(fi) == 0 {
		return 0
	}
	return fi[len(fi)-1]
}

// ChandelierExit returns long and short Chandelier Exit values using ATR-based stops.
// Long exit = max(high, period) - ATR * multiplier. Short exit = min(low, period) + ATR * multiplier.
func ChandelierExit(highs, lows, closes []float64, count int) (float64, float64) {
	if count < 22 || len(highs) < count || len(lows) < count || len(closes) < count {
		return 0, 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	longExit, shortExit := indicator.ChandelierExit(h, l, c)
	if len(longExit) == 0 || len(shortExit) == 0 {
		return 0, 0
	}
	return longExit[len(longExit)-1], shortExit[len(shortExit)-1]
}

// OBV returns the On-Balance Volume value. Accumulates volume on up days, subtracts on down days.
func OBV(closes, volumes []float64, count int) float64 {
	if count < 2 || len(closes) < count || len(volumes) < count {
		return 0
	}
	c := closes[:count]
	v := volumes[:count]
	obv := indicator.Obv(c, v)
	if len(obv) == 0 {
		return 0
	}
	return obv[len(obv)-1]
}

// CMF returns the Chaikin Money Flow value. Range: -1 to +1.
// Above 0.05 = accumulation. Below -0.05 = distribution.
func CMF(highs, lows, closes, volumes []float64, count int) float64 {
	if count < 20 || len(highs) < count || len(lows) < count || len(closes) < count || len(volumes) < count {
		return 0
	}
	h := highs[:count]
	l := lows[:count]
	c := closes[:count]
	v := volumes[:count]
	cmf := indicator.ChaikinMoneyFlow(h, l, c, v)
	if len(cmf) == 0 {
		return 0
	}
	return cmf[len(cmf)-1]
}
