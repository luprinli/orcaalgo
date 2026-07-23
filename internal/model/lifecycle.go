package model

type LifecycleStatus string

const (
	LifecycleDraft      LifecycleStatus = "draft"
	LifecycleBacktested LifecycleStatus = "backtested"
	LifecycleReady      LifecycleStatus = "ready"
	LifecycleLive       LifecycleStatus = "live"
	LifecycleHalted     LifecycleStatus = "halted"
	LifecycleArchived   LifecycleStatus = "archived"
)

var ValidTransitions = map[LifecycleStatus][]LifecycleStatus{
	LifecycleDraft:      {LifecycleBacktested, LifecycleArchived},
	LifecycleBacktested: {LifecycleReady, LifecycleDraft, LifecycleArchived},
	LifecycleReady:      {LifecycleLive, LifecycleDraft, LifecycleArchived},
	LifecycleLive:       {LifecycleHalted, LifecycleReady, LifecycleArchived},
	LifecycleHalted:     {LifecycleLive, LifecycleArchived},
	LifecycleArchived:   {},
}

func (s LifecycleStatus) CanTransitionTo(target LifecycleStatus) bool {
	allowed, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

func (s LifecycleStatus) Color() string {
	switch s {
	case LifecycleDraft:
		return "#8b949e"
	case LifecycleBacktested:
		return "#58a6ff"
	case LifecycleReady:
		return "#d29922"
	case LifecycleLive:
		return "#3fb950"
	case LifecycleHalted:
		return "#da3633"
	case LifecycleArchived:
		return "#484f58"
	}
	return "#8b949e"
}

func (s LifecycleStatus) Label() string {
	switch s {
	case LifecycleDraft:
		return "Draft"
	case LifecycleBacktested:
		return "Backtested"
	case LifecycleReady:
		return "Ready for Live"
	case LifecycleLive:
		return "Live"
	case LifecycleHalted:
		return "Halted"
	case LifecycleArchived:
		return "Archived"
	}
	return "Unknown"
}
