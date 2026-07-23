package risk

type GlobalRiskState struct {
	HMMTracker  HMMTracker
	Adversarial AdversarialState
	Halted      bool
	HaltReason  string
}

func NewGlobalRiskState() *GlobalRiskState {
	model := DefaultHMM()
	tracker := NewHMMTracker(model)
	return &GlobalRiskState{
		HMMTracker: *tracker,
	}
}
