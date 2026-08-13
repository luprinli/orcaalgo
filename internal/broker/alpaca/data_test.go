package alpaca

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	if !parseTime("").IsZero() {
		t.Error("empty string should parse to zero time")
	}
	if got := parseTime("2026-08-13T20:00:00Z"); got.Year() != 2026 || got.Month() != time.August || got.Day() != 13 {
		t.Errorf("parseTime = %v, want 2026-08-13", got)
	}
	if !parseTime("not-a-time").IsZero() {
		t.Error("invalid string should parse to zero time")
	}
}
