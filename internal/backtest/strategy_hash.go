package backtest

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// HashGKRFile computes the canonical instance_hash_v2 for a GKR strategy config by
// invoking the Python `orca hash` command — the single source of truth for strategy
// hashing (docs/backtest_live_parity_audit_report.md R5). It is BEST-EFFORT: it
// returns "" (never an error into the hot path) when the file or the orca toolchain
// is unavailable, so a backtest is never failed merely because hashing could not run.
// Prefers the `orca` console script and falls back to `python -m orca.cli`.
func HashGKRFile(ctx context.Context, gkrPath string) string {
	if gkrPath == "" {
		return ""
	}
	if _, err := os.Stat(gkrPath); err != nil {
		return ""
	}
	for _, args := range [][]string{
		{"orca", "hash", gkrPath},
		{"python", "-m", "orca.cli", "hash", gkrPath},
	} {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		if h := strings.TrimSpace(string(out)); strings.HasPrefix(h, "sha256:") {
			return h
		}
	}
	return ""
}
