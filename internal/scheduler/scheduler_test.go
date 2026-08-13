package scheduler

import (
	"context"
	"testing"
)

func TestListJobs_And_RunJobNow(t *testing.T) {
	s := NewScheduler(nil, nil)
	ran := false
	s.jobs = append(s.jobs, Job{
		Name:     "test_job",
		Schedule: "0 0 * * *",
		Run: func(ctx context.Context) error {
			ran = true
			return nil
		},
	})

	jobs := s.ListJobs()
	if len(jobs) != 1 || jobs[0].Name != "test_job" {
		t.Fatalf("expected one test_job, got %+v", jobs)
	}

	if err := s.RunJobNow(context.Background(), "test_job"); err != nil {
		t.Fatalf("RunJobNow returned error: %v", err)
	}
	if !ran {
		t.Error("job did not run")
	}

	// Last-run status should now be populated.
	jobs = s.ListJobs()
	if jobs[0].LastRun.IsZero() {
		t.Error("last run should be recorded")
	}
}

func TestRunJobNow_UnknownJob(t *testing.T) {
	s := NewScheduler(nil, nil)
	if err := s.RunJobNow(context.Background(), "does_not_exist"); err == nil {
		t.Error("expected error for unknown job")
	}
}
