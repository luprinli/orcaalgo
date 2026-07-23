package backtest

import "time"

type SeasonalityOverlay struct {
	TurnOfMonthBoost       float64
	TurnOfMonthDaysBefore  int
	TurnOfMonthDaysAfter   int
	JanuaryBoost           float64
	SeptemberReduction     float64
	NovemberBoost          float64
	DecemberBoost          float64
	HolidayCalAdjust       bool
}

func NewSeasonalityOverlay() *SeasonalityOverlay {
	return &SeasonalityOverlay{
		TurnOfMonthBoost:       1.5,
		TurnOfMonthDaysBefore:  2,
		TurnOfMonthDaysAfter:   4,
		JanuaryBoost:           2.0,
		SeptemberReduction:     0.5,
		NovemberBoost:          1.25,
		DecemberBoost:          1.25,
		HolidayCalAdjust:       true,
	}
}

var usHolidays = map[string]bool{
	"2024-01-01": true, "2024-01-15": true, "2024-02-19": true,
	"2024-03-29": true, "2024-05-27": true, "2024-06-19": true,
	"2024-07-04": true, "2024-09-02": true, "2024-11-28": true,
	"2024-12-25": true,
	"2025-01-01": true, "2025-01-20": true, "2025-02-17": true,
	"2025-04-18": true, "2025-05-26": true, "2025-06-19": true,
	"2025-07-04": true, "2025-09-01": true, "2025-11-27": true,
	"2025-12-25": true,
}

func (s *SeasonalityOverlay) Multiplier(t time.Time) float64 {
	dateKey := t.Format("2006-01-02")

	if s.HolidayCalAdjust {
		if usHolidays[dateKey] {
			return 1.0
		}
		prev := t.AddDate(0, 0, -1).Format("2006-01-02")
		next := t.AddDate(0, 0, 1).Format("2006-01-02")
		if usHolidays[prev] || usHolidays[next] {
			return 0.75
		}
	}

	month := t.Month()
	day := t.Day()

	if month == time.January {
		return s.JanuaryBoost
	}
	if month == time.September {
		return s.SeptemberReduction
	}
	if month == time.November || month == time.December {
		return s.NovemberBoost
	}

	if day >= 27 || day <= 4 {
		return s.TurnOfMonthBoost
	}

	return 1.0
}
