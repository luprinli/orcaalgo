package strategy

import "testing"

// TestAllRegistryDefaultsValid guards the invariant that every registered
// strategy's default params pass the validator (Rule 4). A default that falls
// outside its ParamDef range — or an inverted relationship like entry_z <=
// exit_z — fails here rather than silently trading broken config in a matrix.
func TestAllRegistryDefaultsValid(t *testing.T) {
	for _, s := range GlobalRegistry().All() {
		if s == nil {
			continue
		}
		if v := ValidateParams(s.Name(), s.Params()); len(v) > 0 {
			t.Errorf("%s default params invalid: %v", s.Name(), v)
		}
	}
}

func TestValidateParams_RejectsOutOfRange(t *testing.T) {
	v := ValidateParams("intraday_mr", map[string]float64{"lookback": 999})
	if len(v) == 0 {
		t.Error("expected out-of-range violation for lookback=999")
	}
}

func TestValidateParams_RejectsInvertedZ(t *testing.T) {
	v := ValidateParams("intraday_mr", map[string]float64{"entry_z": 0.5, "exit_z": 2.0})
	if len(v) == 0 {
		t.Error("expected entry_z <= exit_z violation")
	}
}

func TestValidateParams_RejectsBadKelly(t *testing.T) {
	v := ValidateParams("trend_following", map[string]float64{"kelly_fraction": 1.5})
	if len(v) == 0 {
		t.Error("expected kelly_fraction out-of-range violation")
	}
}

func TestValidateParams_EmptyParamsOK(t *testing.T) {
	if v := ValidateParams("intraday_mr", nil); len(v) != 0 {
		t.Errorf("expected no violations for nil params, got %v", v)
	}
}

func TestWarnParams_RejectsInvertedRR(t *testing.T) {
	w := WarnParams("grid_trading", map[string]float64{"take_profit_pct": 0.5, "stop_loss_pct": 1.5})
	if len(w) == 0 {
		t.Error("expected inverted reward:risk warning")
	}
}

func TestRegimeName(t *testing.T) {
	cases := map[int8]string{
		RegimeCalm:     "Calm",
		RegimeTrending: "Trending",
		RegimeHighVol:  "HighVol",
		RegimeCrisis:   "Crisis",
		99:             "Unknown",
	}
	for r, want := range cases {
		if got := RegimeName(r); got != want {
			t.Errorf("RegimeName(%d) = %q, want %q", r, got, want)
		}
	}
}
