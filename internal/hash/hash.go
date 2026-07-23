package hash

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ComputeInstanceHash shells out to the Python CLI to compute the instance hash
// of a .gkr.yaml strategy file. Returns the sha256:<hex> string on success.
// Tries 'orca' first, then falls back to 'python -m orca.cli'.
func ComputeInstanceHash(gkrPath string) (string, error) {
	var lastErr error
	for _, args := range [][]string{
		{"orca", "hash", "--instance", gkrPath},
		{"python", "-m", "orca.cli", "hash", "--instance", gkrPath},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String()), nil
		} else {
			lastErr = fmt.Errorf("%s: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
		}
	}
	return "", lastErr
}

// VerifyInstanceHash checks that the strategy at gkrPath produces the expected
// instance hash. Returns nil on match, or an error describing the mismatch.
func VerifyInstanceHash(gkrPath, expected string) error {
	actual, err := ComputeInstanceHash(gkrPath)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("strategy hash mismatch for %s: computed=%s expected=%s", gkrPath, actual, expected)
	}
	return nil
}
