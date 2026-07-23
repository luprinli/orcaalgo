package ingest

import (
	"fmt"
	"math"
	"time"
)

type ValidationResult struct {
	Valid       bool
	TotalRows   int
	NullRows    int
	OHLCErrors  int
	Gaps        []time.Time
	GapCount    int
	Sorted      bool
	Message     string
}

func ValidateCandles(candles []CandleData, symbol string, expectedFrequency string) *ValidationResult {
	r := &ValidationResult{TotalRows: len(candles), Valid: true, Sorted: true}

	if len(candles) == 0 {
		r.Valid = false
		r.Message = "no candles provided"
		return r
	}

	for i, c := range candles {
		if math.IsNaN(c.Open) || math.IsNaN(c.High) || math.IsNaN(c.Low) || math.IsNaN(c.Close) || math.IsNaN(c.Volume) {
			r.NullRows++
			continue
		}
		if c.Open <= 0 || c.High <= 0 || c.Low <= 0 || c.Close <= 0 {
			r.NullRows++
			continue
		}

		if c.High < c.Open || c.High < c.Close || c.Low > c.Open || c.Low > c.Close {
			r.OHLCErrors++
		}
		if c.High < c.Low {
			r.OHLCErrors++
		}
		if c.Volume < 0 {
			r.OHLCErrors++
		}

		if i > 0 && c.Time.Before(candles[i-1].Time) {
			r.Sorted = false
		}
	}

	if expectedFrequency == "1d" || expectedFrequency == "daily" {
		maxGapDays := 5
		for i := 1; i < len(candles); i++ {
			delta := candles[i].Time.Sub(candles[i-1].Time)
			gapDays := int(delta.Hours() / 24)
			if gapDays > maxGapDays {
				if !isWeekendOnly(candles[i-1].Time, candles[i].Time, gapDays) {
					r.Gaps = append(r.Gaps, candles[i-1].Time)
				}
			}
		}
	}

	r.GapCount = len(r.Gaps)
	if r.NullRows > 0 || r.OHLCErrors > 0 || r.GapCount > 0 || !r.Sorted {
		r.Valid = false
	}

	var issues []string
	if r.NullRows > 0 {
		issues = append(issues, fmt.Sprintf("%d null rows", r.NullRows))
	}
	if r.OHLCErrors > 0 {
		issues = append(issues, fmt.Sprintf("%d OHLC errors", r.OHLCErrors))
	}
	if r.GapCount > 0 {
		issues = append(issues, fmt.Sprintf("%d gaps", r.GapCount))
	}
	if !r.Sorted {
		issues = append(issues, "unsorted")
	}
	if len(issues) > 0 {
		r.Message = fmt.Sprintf("%s: %v", symbol, issues)
	} else {
		r.Message = fmt.Sprintf("%s: valid (%d rows)", symbol, r.TotalRows)
	}

	return r
}

func isWeekendOnly(start, end time.Time, gapDays int) bool {
	if gapDays <= 3 {
		return true
	}
	return false
}
