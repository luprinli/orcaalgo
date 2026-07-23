package scheduler

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
)

func TestScheduler_Registration(t *testing.T) {
	vault := &risk.EnvVault{}
	tg := monitor.NewTelegramBot()
	s := NewScheduler(vault, tg)

	if len(s.jobs) != 0 {
		t.Error("expected empty jobs list initially")
	}

	s.RegisterKeyRotationJob()
	if len(s.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(s.jobs))
	}

	s.RegisterDailyHealthJob()
	if len(s.jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(s.jobs))
	}

	if s.jobs[0].Name != "key_rotation_check" {
		t.Errorf("expected 'key_rotation_check', got %q", s.jobs[0].Name)
	}
	if s.jobs[1].Name != "daily_health" {
		t.Errorf("expected 'daily_health', got %q", s.jobs[1].Name)
	}
}
