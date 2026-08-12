package market

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type EarningsEvent struct {
	Symbol    string
	Date      time.Time
	TimeOfDay string
}

// EarningsCalendar tracks earnings announcement dates per symbol.
// Symbols are keyed by ticker; the wildcard "*" key represents
// fallback dates covering the four quarterly earnings seasons.
type EarningsCalendar struct {
	events map[string][]EarningsEvent
}

func NewEarningsCalendar() *EarningsCalendar {
	ec := &EarningsCalendar{
		events: make(map[string][]EarningsEvent),
	}
	ec.loadFallbackCalendar()
	return ec
}

// IsEarningsDay returns true when the given date matches an earnings
// announcement for the symbol. Falls back to the wildcard calendar if no
// symbol-specific events are registered.
func (ec *EarningsCalendar) IsEarningsDay(symbol string, date time.Time) bool {
	dateKey := date.Truncate(24 * time.Hour)
	events, ok := ec.events[symbol]
	if !ok {
		events, ok = ec.events["*"]
		if !ok {
			return false
		}
	}
	for _, e := range events {
		if e.Date.Truncate(24 * time.Hour).Equal(dateKey) {
			return true
		}
	}
	return false
}

// GetEarningsWindow checks whether the given date falls on the day before,
// the day of, or the day after an earnings announcement for the symbol.
func (ec *EarningsCalendar) GetEarningsWindow(symbol string, date time.Time) (before, during, after bool) {
	dateKey := date.Truncate(24 * time.Hour)
	events, ok := ec.events[symbol]
	if !ok {
		return false, false, false
	}
	for _, e := range events {
		eventDate := e.Date.Truncate(24 * time.Hour)
		if eventDate.Equal(dateKey) {
			during = true
		}
		if dateKey.Equal(eventDate.AddDate(0, 0, -1)) {
			before = true
		}
		if dateKey.Equal(eventDate.AddDate(0, 0, 1)) {
			after = true
		}
	}
	return before, during, after
}

// LoadFromCSV imports earnings events from a CSV file. Expected columns:
// symbol, date (YYYY-MM-DD), and optionally time_of_day (defaults to "BMO").
func (ec *EarningsCalendar) LoadFromCSV(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("earnings: cannot open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("earnings: cannot read header: %w", err)
	}

	symIdx, dateIdx, todIdx := -1, -1, -1
	for i, col := range header {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "symbol":
			symIdx = i
		case "date":
			dateIdx = i
		case "time_of_day":
			todIdx = i
		}
	}
	if symIdx < 0 || dateIdx < 0 {
		return fmt.Errorf("earnings: CSV must have 'symbol' and 'date' columns")
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("earnings: read error: %w", err)
		}
		parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(row[dateIdx]))
		if err != nil {
			continue
		}
		tod := "BMO"
		if todIdx >= 0 && todIdx < len(row) {
			tod = strings.TrimSpace(row[todIdx])
		}
		if tod == "" {
			tod = "BMO"
		}
		sym := strings.TrimSpace(strings.ToUpper(row[symIdx]))
		ec.events[sym] = append(ec.events[sym], EarningsEvent{
			Symbol:    sym,
			Date:      parsedDate,
			TimeOfDay: tod,
		})
	}
	return nil
}

func (ec *EarningsCalendar) loadFallbackCalendar() {
	seasons := []struct {
		month     time.Month
		peakWeeks []int
	}{
		{time.January, []int{3, 4}},
		{time.April, []int{3, 4}},
		{time.July, []int{3, 4}},
		{time.October, []int{3, 4}},
	}

	for year := 2019; year <= 2027; year++ {
		for _, s := range seasons {
			firstDay := time.Date(year, s.month, 1, 0, 0, 0, 0, time.UTC)
			for _, weekNum := range s.peakWeeks {
				dayOffset := (weekNum-1)*7 + int(time.Monday-firstDay.Weekday())
				if dayOffset < 0 {
					dayOffset += 7
				}
				date := firstDay.AddDate(0, 0, dayOffset)
				for day := 0; day < 5; day++ {
					d := date.AddDate(0, 0, day)
					ec.events["*"] = append(ec.events["*"], EarningsEvent{
						Symbol:    "*",
						Date:      d,
						TimeOfDay: "BMO",
					})
				}
			}
		}
	}
}
