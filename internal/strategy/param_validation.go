package strategy

import "fmt"

// ValidateParams returns hard correctness violations for a strategy's params:
// values outside their declared ParamDef range, and inconsistent relationships
// (entry_z <= exit_z, kelly_fraction outside (0,1], regime_w_* outside [0,1]).
// It does not judge profitability. An empty slice means the params are
// internally consistent. This is the guard invoked by tests and (in a later
// phase) the shared SetParams wrapper + the light optimizer.
func ValidateParams(strategyName string, params map[string]float64) []string {
	var violations []string
	if len(params) == 0 {
		return violations
	}

	defByName := make(map[string]ParamDef)
	if s := GlobalRegistry().Get(strategyName); s != nil {
		for _, d := range s.ParamDefs() {
			defByName[d.Name] = d
		}
	}
	for name, v := range params {
		def, ok := defByName[name]
		if !ok {
			continue
		}
		if v < def.Min || v > def.Max {
			violations = append(violations, fmt.Sprintf("%s=%v out of range [%v, %v]", name, v, def.Min, def.Max))
		}
	}

	if ez, ok1 := params["entry_z"]; ok1 {
		if xz, ok2 := params["exit_z"]; ok2 && ez <= xz {
			violations = append(violations, "entry_z must be greater than exit_z")
		}
	}
	if k, ok := params["kelly_fraction"]; ok && (k <= 0 || k > 1) {
		violations = append(violations, fmt.Sprintf("kelly_fraction=%v must be in (0, 1]", k))
	}
	for _, name := range []string{"regime_w_calm", "regime_w_trending", "regime_w_highvol", "regime_w_crisis"} {
		if v, ok := params[name]; ok && (v < 0 || v > 1) {
			violations = append(violations, fmt.Sprintf("%s=%v must be in [0, 1]", name, v))
		}
	}

	return violations
}

// WarnParams returns non-fatal advisories for params that are legal but
// suspicious. Currently: inverted reward:risk (take_profit < stop_loss).
func WarnParams(strategyName string, params map[string]float64) []string {
	_ = strategyName
	var warnings []string
	if tp, ok := params["take_profit_pct"]; ok {
		if sl, ok2 := params["stop_loss_pct"]; ok2 && tp < sl {
			warnings = append(warnings, "take_profit_pct < stop_loss_pct: inverted reward:risk")
		}
	}
	return warnings
}
