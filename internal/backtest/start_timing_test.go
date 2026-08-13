package backtest

import (
	"testing"
	"time"
)

func TestGenerateStartTimingWindows_Basic(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	windows := GenerateStartTimingWindows(start, end, 6, 4)
	if len(windows) == 0 {
		t.Fatal("expected at least one window")
	}
	// First window starts at the start date; later windows are staggered by 4 weeks.
	if !windows[0].StartDate.Equal(start) {
		t.Errorf("first window should start at the global start, got %v", windows[0].StartDate)
	}
	for i := 1; i < len(windows); i++ {
		if !windows[i].StartDate.After(windows[i-1].StartDate) {
			t.Errorf("windows must be staggered ascending, window %d not after %d", i, i-1)
		}
		if !windows[i].EndDate.After(windows[i].StartDate) {
			t.Errorf("window %d end must be after its start", i)
		}
	}
}

func TestGenerateStartTimingWindows_TruncatesAtEnd(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
	windows := GenerateStartTimingWindows(start, end, 6, 4)
	for _, w := range windows {
		if w.EndDate.After(end) {
			t.Errorf("window end %v must not exceed global end %v", w.EndDate, end)
		}
	}
}

func TestGenerateStartTimingWindows_InvalidInputs(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := GenerateStartTimingWindows(start, start, 6, 4); got != nil {
		t.Errorf("degenerate range should return nil, got %v", got)
	}
	if got := GenerateStartTimingWindows(start, start.AddDate(1, 0, 0), 0, 4); got != nil {
		t.Errorf("non-positive horizon should return nil, got %v", got)
	}
}
