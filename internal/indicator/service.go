package indicator

import (
	"fmt"
	"math"

	"github.com/cinar/indicator"
)

type Candle struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type IndicatorPoint struct {
	Time   int64              `json:"time"`
	Values map[string]float64 `json:"values"`
}

type IndicatorResult struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Overlay  bool              `json:"overlay"`
	Outputs  []OutputMeta      `json:"outputs"`
	Data     []IndicatorPoint  `json:"data"`
}

func extractSource(candles []Candle, source string) []float64 {
	values := make([]float64, len(candles))
	for i, c := range candles {
		switch source {
		case "open":
			values[i] = c.Open
		case "high":
			values[i] = c.High
		case "low":
			values[i] = c.Low
		default:
			values[i] = c.Close
		}
	}
	return values
}

func ComputeSMA(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	period := getIntParam(params, "period", 20)
	source := getStringParam(params, "source", "close")
	values := extractSource(candles, source)
	series := indicator.Sma(period, values)

	spec, _ := Get("sma")
	result := &IndicatorResult{
		ID: "sma", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	for i, v := range series {
		t := candles[i].Time
		if i < period-1 {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time:   t,
			Values: map[string]float64{"sma": v},
		})
	}
	return result, nil
}

func ComputeEMA(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	period := getIntParam(params, "period", 20)
	source := getStringParam(params, "source", "close")
	values := extractSource(candles, source)
	series := indicator.Ema(period, values)

	spec, _ := Get("ema")
	result := &IndicatorResult{
		ID: "ema", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	for i, v := range series {
		t := candles[i].Time
		if i < period-1 {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time:   t,
			Values: map[string]float64{"ema": v},
		})
	}
	return result, nil
}

func ComputeRSI(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	period := getIntParam(params, "period", 14)
	source := getStringParam(params, "source", "close")
	values := extractSource(candles, source)
	_, rsiSeries := indicator.RsiPeriod(period, values)

	spec, _ := Get("rsi")
	result := &IndicatorResult{
		ID: "rsi", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	for i, v := range rsiSeries {
		t := candles[i].Time
		if v == 0 && i < period+10 {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time:   t,
			Values: map[string]float64{"rsi": v},
		})
	}
	return result, nil
}

func ComputeMACD(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	fast := getIntParam(params, "fast", 12)
	slow := getIntParam(params, "slow", 26)
	signalPeriod := getIntParam(params, "signal", 9)
	source := getStringParam(params, "source", "close")
	values := extractSource(candles, source)

	emaFast := indicator.Ema(fast, values)
	emaSlow := indicator.Ema(slow, values)

	spec, _ := Get("macd")
	result := &IndicatorResult{
		ID: "macd", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	macdValues := make([]float64, len(values))
	for i := range values {
		if i < slow-1 {
			macdValues[i] = math.NaN()
		} else {
			macdValues[i] = emaFast[i] - emaSlow[i]
		}
	}

	signalSeries := indicator.Ema(signalPeriod, macdValues)

	for i := range macdValues {
		if math.IsNaN(macdValues[i]) {
			continue
		}
		t := candles[i].Time
		hist := macdValues[i] - signalSeries[i]
		if math.IsNaN(hist) {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time: t,
			Values: map[string]float64{
				"macd":   macdValues[i],
				"signal": signalSeries[i],
				"hist":   hist,
			},
		})
	}
	return result, nil
}

func ComputeBBands(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	period := getIntParam(params, "period", 20)
	stdDev := getFloatParam(params, "std_dev", 2.0)
	source := getStringParam(params, "source", "close")
	values := extractSource(candles, source)

	upper, middle, lower := indicator.BollingerBands(values)

	spec, _ := Get("bbands")
	result := &IndicatorResult{
		ID: "bbands", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	adjustedUpper := make([]float64, len(upper))
	adjustedLower := make([]float64, len(lower))
	for i := range upper {
		width := (upper[i] - middle[i])
		adjustedUpper[i] = middle[i] + width*(stdDev/2.0)
		adjustedLower[i] = middle[i] - width*(stdDev/2.0)
	}

	for i := range upper {
		t := candles[i].Time
		if i < period-1 {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time: t,
			Values: map[string]float64{
				"upper": adjustedUpper[i],
				"mid":   middle[i],
				"lower": adjustedLower[i],
			},
		})
	}
	return result, nil
}

func ComputeATR(candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	period := getIntParam(params, "period", 14)

	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	_, atrSeries := indicator.Atr(period, highs, lows, closes)

	spec, _ := Get("atr")
	result := &IndicatorResult{
		ID: "atr", Name: spec.Name, Overlay: spec.Overlay, Outputs: spec.Outputs,
	}

	for i, v := range atrSeries {
		t := candles[i].Time
		if v == 0 && i < period+5 {
			continue
		}
		result.Data = append(result.Data, IndicatorPoint{
			Time:   t,
			Values: map[string]float64{"atr": v},
		})
	}
	return result, nil
}

var computeFuncs = map[string]func([]Candle, map[string]interface{}) (*IndicatorResult, error){
	"sma":    ComputeSMA,
	"ema":    ComputeEMA,
	"rsi":    ComputeRSI,
	"macd":   ComputeMACD,
	"bbands": ComputeBBands,
	"atr":    ComputeATR,
}

func Compute(indicatorID string, candles []Candle, params map[string]interface{}) (*IndicatorResult, error) {
	fn, ok := computeFuncs[indicatorID]
	if !ok {
		return nil, fmt.Errorf("unknown indicator: %s", indicatorID)
	}
	return fn(candles, params)
}

func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return defaultVal
	}
}

func getFloatParam(params map[string]interface{}, key string, defaultVal float64) float64 {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return defaultVal
	}
}

func getStringParam(params map[string]interface{}, key string, defaultVal string) string {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}
