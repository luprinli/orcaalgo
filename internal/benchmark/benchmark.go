package benchmark

// Shared market-based benchmark filter evaluation (Go side). The math runs in
// the Python `orca benchmark-filter` subprocess (HP #1: canonical sizing math
// stays in Python). This package is the single Go entry point used by both the
// HTTP API and the matrix runner.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Verdict is the Go mirror of the Python BenchmarkVerdict.as_dict() output.
type Verdict struct {
	Passed               bool    `json:"passed"`
	Kind                 string  `json:"kind"`
	DeflatedActiveSharpe float64 `json:"deflated_active_sharpe"`
	NTrials              int     `json:"n_trials"`
	Metrics              struct {
		InformationRatio *float64 `json:"information_ratio"`
		AlphaAnnualized  *float64 `json:"alpha_annualized"`
		Beta             *float64 `json:"beta"`
	} `json:"metrics"`
}

// Evaluate runs the benchmark filter over aligned per-period decimal return
// series and returns the verdict. kind is one of the BenchmarkSpec kinds;
// symbol is the benchmark ticker (or benchmark_series name for risk_free);
// hurdle is the annualized excess-return floor for risk_free.
func Evaluate(ctx context.Context, strategy, benchmark []float64, kind, symbol string, hurdle float64, nTrials int) (*Verdict, error) {
	spec := map[string]interface{}{"kind": kind, "ticker": symbol}
	if kind == "risk_free" {
		spec["risk_free_hurdle"] = hurdle
	}
	payload, err := json.Marshal(map[string]interface{}{
		"strategy":  strategy,
		"benchmark": benchmark,
		"spec":      spec,
		"n_trials":  nTrials,
	})
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, argv := range [][]string{
		{"orca", "benchmark-filter"},
		{"python", "-m", "orca.cli", "benchmark-filter"},
	} {
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Stdin = bytes.NewReader(payload)
		out, err := cmd.Output()
		if err != nil {
			lastErr = err
			if len(bytes.TrimSpace(out)) == 0 {
				continue
			}
		}
		var verdict Verdict
		if err := json.Unmarshal(bytes.TrimSpace(out), &verdict); err != nil {
			lastErr = err
			continue
		}
		if verdict.Kind == "" && !verdict.Passed {
			// A non-passing verdict with no kind may be an error payload; skip
			// only when the verdict clearly parsed (has Kind or Passed==true).
			if verdict.NTrials == 0 && verdict.DeflatedActiveSharpe == 0 {
				continue
			}
		}
		return &verdict, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("benchmark filter subprocess unavailable")
	}
	return nil, lastErr
}

// SpecHash returns a deterministic SHA-256 of the canonical benchmark spec JSON
// (HP #3: the benchmark choice is hashed, not tuned post-hoc).
func SpecHash(kind, symbol string) string {
	b, _ := json.Marshal(map[string]string{"kind": kind, "ticker": symbol})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
