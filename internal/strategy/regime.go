package strategy

// Market-regime codes (4-state) passed to Evaluate and consumed by the
// RiskPipeline activation matrix. These are intentionally named constants so a
// raw `regime == 0/1/2/3` literal cannot slip back in (anti-pattern Rule 14).
//
// NOTE: these are distinct from internal/ml.RegimeState, which uses a 6-state
// mapping with different numeric values. Do not mix the two.
const (
	RegimeCalm     int8 = 0
	RegimeTrending int8 = 1
	RegimeHighVol  int8 = 2
	RegimeCrisis   int8 = 3
)

var regimeNames = [4]string{"Calm", "Trending", "HighVol", "Crisis"}

// RegimeName returns the human-readable name for a 4-state regime code, or
// "Unknown" for an out-of-range value.
func RegimeName(regime int8) string {
	if regime >= 0 && int(regime) < len(regimeNames) {
		return regimeNames[regime]
	}
	return "Unknown"
}
