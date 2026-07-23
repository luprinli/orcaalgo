package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ComputeEWMAVolatility delegates EWMA computation to the Python canonical
// implementation in orca/sizing/volatility.py. This satisfies Hard
// Prohibition #1: Do not reimplement canonical math functions in Go.
func ComputeEWMAVolatility(ctx context.Context, returns []float64, span int) (float64, error) {
	input, err := json.Marshal(returns)
	if err != nil {
		return 0, fmt.Errorf("ewma: marshal returns: %w", err)
	}
	cmd := exec.CommandContext(ctx, "python", "-c", fmt.Sprintf(
		"import sys, json, numpy as np; from orca.sizing.volatility import ewma_volatility; print(json.dumps(ewma_volatility(np.array(json.loads(sys.argv[1])), span=%d)))", span),
		string(input),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ewma: python subprocess: %w: %s", err, string(output))
	}
	var result float64
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("ewma: unmarshal result: %w", err)
	}
	return result, nil
}
