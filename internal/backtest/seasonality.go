package backtest

import (
	"fmt"
	"time"
)

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

var usHolidays map[string]bool

func init() {
	usHolidays = make(map[string]bool)
	for year := 2024; year <= 2030; year++ {
		addFixedHolidays(year)
		addFloatingHolidays(year)
	}
}

func addFixedHolidays(year int) {
	usHolidays[fmt.Sprintf("%04d-01-01", year)] = true
	usHolidays[fmt.Sprintf("%04d-07-04", year)] = true
	usHolidays[fmt.Sprintf("%04d-12-25", year)] = true
}

func addFloatingHolidays(year int) {
	usHolidays[fmt.Sprintf("%04d-%02d-%02d", year, 1, nthWeekday(year, time.January, time.Monday, 3))] = true   // MLK
	usHolidays[fmt.Sprintf("%04d-%02d-%02d", year, 2, nthWeekday(year, time.February, time.Monday, 3))] = true   // Presidents
	usHolidays[fmt.Sprintf("%04d-%02d-%02d", year, 5, lastWeekday(year, time.May, time.Monday))] = true         // Memorial
	usHolidays[fmt.Sprintf("%04d-%02d-%02d", year, 9, nthWeekday(year, time.September, time.Monday, 1))] = true // Labor
	usHolidays[fmt.Sprintf("%04d-%02d-%02d", year, 11, nthWeekday(year, time.November, time.Thursday, 4))] = true // Thanksgiving

	juneteenth := fmt.Sprintf("%04d-06-19", year)
	usHolidays[juneteenth] = true
	if year < 2026 {
		usHolidays[juneteenth] = (year >= 2021)
	} else {
		usHolidays[juneteenth] = true
	}

	easter := calculateEaster(year)
	usHolidays[easter] = true

	goodFriday := calculateGoodFriday(year)
	usHolidays[goodFriday] = true
}

func nthWeekday(year int, month time.Month, day time.Weekday, n int) int {
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(day) - int(t.Weekday()) + 7) % 7
	dayOfMonth := 1 + offset + (n-1)*7
	return dayOfMonth
}

func lastWeekday(year int, month time.Month, day time.Weekday) int {
	t := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)
	lastDay := t.Day()
	offset := int(t.Weekday()) - int(day)
	if offset < 0 {
		offset += 7
	}
	return lastDay - offset
}

func calculateEaster(year int) string {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := (h + l - 7*m + 114) % 31 + 1
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func calculateGoodFriday(year int) string {
	easter := calculateEaster(year)
	t, _ := time.Parse("2006-01-02", easter)
	gf := t.AddDate(0, 0, -2)
	return gf.Format("2006-01-02")
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
