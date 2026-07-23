package hash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gkrPath(t *testing.T, name string) string {
	t.Helper()
	paths := []string{
		filepath.Join("..", "..", "configs", "strategies", name+".gkr.yaml"),
		filepath.Join("configs", "strategies", name+".gkr.yaml"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skipf("GKR config file %s.gkr.yaml not found — skipping subprocess test", name)
	return ""
}

func TestComputeInstanceHash_ValidFile(t *testing.T) {
	path := gkrPath(t, "intraday_mr")
	h, err := ComputeInstanceHash(path)
	if err != nil {
		t.Fatalf("ComputeInstanceHash: %v", err)
	}
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("expected sha256: prefix, got %s", h)
	}
	if len(h) != 71 {
		t.Errorf("expected 71-char hash, got %d: %s", len(h), h)
	}
}

func TestComputeInstanceHash_Deterministic(t *testing.T) {
	path := gkrPath(t, "intraday_mr")
	h1, err := ComputeInstanceHash(path)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	h2, err := ComputeInstanceHash(path)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if h1 != h2 {
		t.Errorf("hash not deterministic: %s vs %s", h1, h2)
	}
}

func TestVerifyInstanceHash_Match(t *testing.T) {
	path := gkrPath(t, "intraday_mr")
	actual, err := ComputeInstanceHash(path)
	if err != nil {
		t.Fatalf("ComputeInstanceHash: %v", err)
	}
	if err := VerifyInstanceHash(path, actual); err != nil {
		t.Errorf("expected match, got: %v", err)
	}
}

func TestVerifyInstanceHash_Mismatch(t *testing.T) {
	path := gkrPath(t, "intraday_mr")
	if err := VerifyInstanceHash(path, "sha256:0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected mismatch error, got nil")
	}
}
