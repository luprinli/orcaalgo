// Package version exposes build provenance as a single source of truth for the
// whole binary. Commit and BuildTime are injected at link time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/lee-econ/orca-core/internal/version.Commit=$(git rev-parse HEAD) \
//	                    -X github.com/lee-econ/orca-core/internal/version.BuildTime=$(date -u +%FT%TZ)"
//
// When built without linker injection they retain the "dev"/"unknown" defaults.
// Every engine-version accessor (CLIs, API server, live engine) must read through
// this package so provenance cannot drift between components
// (see docs/backtest_live_parity_audit_report.md R4).
package version

var (
	// Commit is the git commit SHA the binary was built from ("dev" if not injected).
	Commit = "dev"
	// BuildTime is the UTC build timestamp ("unknown" if not injected).
	BuildTime = "unknown"
)

// Engine returns the engine build identifier (git commit SHA), or "dev" when the
// linker did not inject a value. This is the ONLY engine-version accessor.
func Engine() string { return Commit }

// Build returns the UTC build timestamp, or "unknown" when not injected.
func Build() string { return BuildTime }
