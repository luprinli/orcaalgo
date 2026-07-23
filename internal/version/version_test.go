package version

import "testing"

func TestDefaults(t *testing.T) {
	// Without -ldflags injection the defaults must be the documented placeholders.
	if Commit != "dev" {
		t.Errorf("default Commit = %q, want dev", Commit)
	}
	if BuildTime != "unknown" {
		t.Errorf("default BuildTime = %q, want unknown", BuildTime)
	}
}

func TestEngineAndBuildAccessors(t *testing.T) {
	if Engine() != Commit {
		t.Errorf("Engine() = %q, want %q (Commit)", Engine(), Commit)
	}
	if Build() != BuildTime {
		t.Errorf("Build() = %q, want %q (BuildTime)", Build(), BuildTime)
	}

	// Simulate an ldflags injection and confirm the accessors reflect it.
	origCommit, origBuild := Commit, BuildTime
	t.Cleanup(func() { Commit, BuildTime = origCommit, origBuild })

	Commit = "3e32c04051ef22e3318cfe94ead0b04048c80be8"
	BuildTime = "2026-07-08T00:00:00Z"
	if Engine() != Commit {
		t.Errorf("Engine() did not reflect injected Commit: got %q", Engine())
	}
	if Build() != BuildTime {
		t.Errorf("Build() did not reflect injected BuildTime: got %q", Build())
	}
}
