package risk

import "testing"

func TestHMMDefaultModel(t *testing.T) {
	model := DefaultHMM()
	if !model.Loaded {
		t.Error("default HMM model should be loaded")
	}
	if model.Transition[0][0] != 0.85 {
		t.Errorf("expected self-transition 0.85, got %f", model.Transition[0][0])
	}
}

func TestHMMTrackerUpdate(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())

	for i := 0; i < 20; i++ {
		tracker.Update(100000, 100100)
	}

	state, conf := tracker.GetRegime()
	if conf <= 0 {
		t.Error("confidence should be > 0 after updates")
	}
	_ = state
}

func TestHMMTrackerInsufficientData(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	tracker.Update(100000, 99900)

	state, conf := tracker.GetRegime()
	if conf != 0 {
		t.Error("confidence should be 0 with insufficient data")
	}
	if state != HMMCalm {
		t.Error("default state should be Calm")
	}
}

func TestHMMTrackerNoPrice(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	tracker.Update(100000, 0)
	state, conf := tracker.GetRegime()
	if conf != 0 {
		t.Error("should not update with zero prev price")
	}
	_ = state
}

func TestHMMLoadCalibrated(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	transition := [4][4]float64{
		{0.90, 0.05, 0.04, 0.01},
		{0.10, 0.80, 0.08, 0.02},
		{0.05, 0.10, 0.78, 0.07},
		{0.02, 0.03, 0.10, 0.85},
	}
	probs := [4]float64{0.60, 0.25, 0.10, 0.05}
	means := [4]float64{0.0001, 0.0004, -0.0002, -0.0010}
	sds := [4]float64{0.004, 0.010, 0.020, 0.050}

	tracker.LoadCalibratedParams(transition, probs, means, sds)

	if !tracker.Model.Loaded {
		t.Error("model should remain loaded")
	}
	if tracker.Model.Transition[0][0] != 0.90 {
		t.Error("transition matrix not updated")
	}
}

func TestVIXModulateSDs(t *testing.T) {
	sds := [4]float64{0.005, 0.012, 0.025, 0.060}

	high := VIXModulateSDs(sds, 35)
	if high[0] != sds[0]*1.5 {
		t.Error("VIX > 30 should multiply all SDs by 1.5")
	}

	mid := VIXModulateSDs(sds, 27)
	if mid[3] != sds[3]*1.3 {
		t.Error("VIX > 25 should multiply crisis SD by 1.3")
	}

	low := VIXModulateSDs(sds, 10)
	if low[0] != sds[0]*0.75 {
		t.Error("VIX < 12 should multiply all SDs by 0.75")
	}
}

func TestHMMStateEnum(t *testing.T) {
	if HMMCalm != 0 || HMMTrending != 1 || HMMHighVol != 2 || HMMCrisis != 3 {
		t.Error("HMM state enum values changed")
	}
}
