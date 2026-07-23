package backtest

import (
	"time"

	"github.com/lee-econ/orca-core/internal/risk"
)

// RegisterMLWithKillSwitch wires the engine's ML components to the kill-switch.
// When the kill-switch fires, all ML models are disabled and the system falls
// back to pure rule-based logic.
//
// NOTE: ML component fields (metaLabeler, regimeEnhancer, exitOrch) were removed
// from Engine during the architecture pivot. When those fields are restored, the
// body of this function should be updated to call Disable() on each component.
func (e *Engine) RegisterMLWithKillSwitch(ks *risk.KillSwitch) {
	ks.OnTrigger(func(reason string, ts time.Time) {
		// TODO: when metaLabeler, regimeEnhancer, exitOrch fields are restored to Engine:
		//   if e.metaLabeler != nil { e.metaLabeler.Disable() }
		//   if e.regimeEnhancer != nil { e.regimeEnhancer.Disable() }
		//   if e.exitOrch != nil { e.exitOrch.Disable() }
	})
}
