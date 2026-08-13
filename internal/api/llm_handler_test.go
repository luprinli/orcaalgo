package api

import (
	"strings"
	"testing"
)

func TestMaskSuffix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"sk-1234567890abcd", "••••abcd"},
		{"abcd", "••••"},
		{"a", "••••"},
		{"", "••••"},
	}
	for _, c := range cases {
		if got := maskSuffix(c.in); got != c.want {
			t.Errorf("maskSuffix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMaskSuffix_DoesNotLeakFullKey(t *testing.T) {
	const key = "sk-super-secret-key-that-must-not-leak"
	masked := maskSuffix(key)
	if masked == key {
		t.Errorf("masked value equals the full key: %q", masked)
	}
	if strings.Contains(masked, "secret") || strings.Contains(masked, "super") {
		t.Errorf("masked value leaks the key body: %q", masked)
	}
	if masked != "••••leak" {
		t.Errorf("masked value should be only suffix: %q", masked)
	}
}
