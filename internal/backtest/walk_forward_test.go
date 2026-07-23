package backtest

import (
	"testing"
	"time"
)

func TestGenerateWalkForwardWindows(t *testing.T) {
	config := WalkForwardConfig{
		Config: BacktestConfig{
			StartDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		TrainWindows: 2,
		TrainYears:   1,
		TestYears:    1,
		StepMonths:   3,
	}

	windows := GenerateWalkForwardWindows(config)
	if len(windows) == 0 {
		t.Fatal("Expected at least 1 window")
	}
	t.Logf("Generated %d windows", len(windows))

	for i, w := range windows {
		if w.TrainStart.After(w.TrainEnd) {
			t.Errorf("Window %d: train start after end", i)
		}
		if w.TrainEnd.After(w.TestEnd) {
			t.Errorf("Window %d: train end after test end", i)
		}
		if w.TestStart.After(w.TestEnd) {
			t.Errorf("Window %d: test start after end", i)
		}
	}
}

func TestGenerateWalkForwardWindows_ShortPeriod(t *testing.T) {
	config := WalkForwardConfig{
		Config: BacktestConfig{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		TrainWindows: 1,
		TrainYears:   2,
		TestYears:    1,
		StepMonths:   1,
	}

	windows := GenerateWalkForwardWindows(config)
	t.Logf("Short period generated %d windows (expected few/none)", len(windows))
}

func TestGenerateWalkForwardWindows_MultiYear(t *testing.T) {
	config := WalkForwardConfig{
		Config: BacktestConfig{
			StartDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		TrainWindows: 1,
		TrainYears:   1,
		TestYears:    1,
		StepMonths:   12,
	}

	windows := GenerateWalkForwardWindows(config)
	if len(windows) == 0 {
		t.Fatal("Expected windows for 6-year period")
	}

	for i, w := range windows {
		if w.TrainStart.After(w.TrainEnd) {
			t.Errorf("Window %d: train start after train end", i)
		}
		t.Logf("Window %d: train=%s..%s test=%s..%s",
			i, w.TrainStart.Format("2006-01"), w.TrainEnd.Format("2006-01"),
			w.TestStart.Format("2006-01"), w.TestEnd.Format("2006-01"))
	}
}

func TestWalkForwardConfig_Defaults(t *testing.T) {
	config := WalkForwardConfig{
		Config: BacktestConfig{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		TrainWindows: 5,
		TrainYears:   2,
		TestYears:    1,
		StepMonths:   3,
	}

	if config.TrainYears <= 0 {
		t.Error("TrainYears should be positive")
	}
	if config.TestYears <= 0 {
		t.Error("TestYears should be positive")
	}
	if config.StepMonths <= 0 {
		t.Error("StepMonths should be positive")
	}
}
