package universe

import "testing"

func TestMapAssetClass(t *testing.T) {
	cases := map[string]string{
		"us_equity": "equity",
		"crypto":    "crypto",
		"forex":     "equity",
		"":          "equity",
	}
	for in, want := range cases {
		if got := mapAssetClass(in); got != want {
			t.Errorf("mapAssetClass(%q) = %q, want %q", in, got, want)
		}
	}
}
