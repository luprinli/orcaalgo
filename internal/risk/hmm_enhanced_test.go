package risk

import (
	"math"
	"os"
	"testing"
)

func TestEmissionProbMultiDim_NearMean(t *testing.T) {
	means := [4][5]float64{
		{0.0, 0.01, 0.5, 0.001, 0.0},
	}
	sds := [4][5]float64{
		{0.02, 0.01, 0.3, 0.002, 0.1},
	}
	params := EnhancedHMMParams{EmissionMeans: means, EmissionDiagSDs: sds, Loaded: true}
	obs := [5]float64{0.0, 0.01, 0.5, 0.001, 0.0}

	prob := emissionProbMultiDim(0, obs, params)
	if prob <= 0 {
		t.Errorf("Expected positive likelihood, got %f", prob)
	}
}

func TestEmissionProbMultiDim_FarFromMean(t *testing.T) {
	means := [4][5]float64{
		{0.0, 0.01, 0.5, 0.001, 0.0},
	}
	sds := [4][5]float64{
		{0.02, 0.01, 0.3, 0.002, 0.1},
	}
	params := EnhancedHMMParams{EmissionMeans: means, EmissionDiagSDs: sds, Loaded: true}
	obs := [5]float64{0.06, 0.04, 1.2, 0.006, 0.3}

	probFar := emissionProbMultiDim(0, obs, params)
	probNear := emissionProbMultiDim(0, [5]float64{0, 0.01, 0.5, 0.001, 0}, params)
	if probFar <= 0 {
		t.Errorf("Should not underflow to zero")
	}
	if probFar >= probNear {
		t.Errorf("Far obs should be less likely: far=%e, near=%e", probFar, probNear)
	}
}

func TestEmissionProbMultiDim_InvalidState(t *testing.T) {
	params := EnhancedHMMParams{Loaded: true}
	prob := emissionProbMultiDim(-1, [5]float64{}, params)
	if prob != 0 {
		t.Errorf("Invalid state should return 0, got %f", prob)
	}
	prob = emissionProbMultiDim(4, [5]float64{}, params)
	if prob != 0 {
		t.Errorf("Invalid state should return 0, got %f", prob)
	}
}

func TestEmissionProbMultiDim_ZeroSD(t *testing.T) {
	means := [4][5]float64{{}}
	sds := [4][5]float64{{}}
	params := EnhancedHMMParams{EmissionMeans: means, EmissionDiagSDs: sds, Loaded: true}

	prob := emissionProbMultiDim(0, [5]float64{0, 0, 0, 0, 0}, params)
	if prob <= 0 {
		t.Errorf("Zero SD should be clamped to minimum, got %f", prob)
	}
}

func TestLoadEnhancedHMMParams_InvalidPath(t *testing.T) {
	_, err := LoadEnhancedHMMParams("/nonexistent/path/hmm.json")
	if err == nil {
		t.Error("Non-existent path should return error")
	}
}

func TestLoadEnhancedHMMParams_ValidJSON(t *testing.T) {
	json := `{
		"transition": [[0.85,0.10,0.04,0.01],[0.08,0.80,0.10,0.02],[0.03,0.10,0.80,0.07],[0.01,0.02,0.10,0.87]],
		"initial_probs": [0.7,0.2,0.08,0.02],
		"emission_means": [[0,0.01,0.5,0.001,0],[0,0.02,0.8,0.002,0.1],[0,0.05,1.5,0.005,0.3],[0,0.10,2.5,0.01,0.5]],
		"emission_covars": [[[1,0,0,0,0],[0,1,0,0,0],[0,0,1,0,0],[0,0,0,1,0],[0,0,0,0,1]]],
		"emission_diag_sds": [[0.02,0.005,0.3,0.002,0.1],[0.03,0.008,0.5,0.004,0.2],[0.06,0.015,1.0,0.008,0.4],[0.10,0.03,2.0,0.02,0.8]]
	}`
	tmpFile, err := os.CreateTemp("", "hmm_enhanced_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(json); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	params, err := LoadEnhancedHMMParams(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load valid JSON: %v", err)
	}
	if !params.Loaded {
		t.Error("Params should be marked as loaded")
	}
	if params.Transition[0][0] != 0.85 {
		t.Errorf("Transition[0][0]=%f, expected 0.85", params.Transition[0][0])
	}
	if params.EmissionDiagSDs[3][4] != 0.8 {
		t.Errorf("EmissionDiagSDs[3][4]=%f, expected 0.8", params.EmissionDiagSDs[3][4])
	}
}

func TestHMMTracker_UpdateEnhanced_Basic(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	params := EnhancedHMMParams{
		Transition: [4][4]float64{
			{0.85, 0.10, 0.04, 0.01},
			{0.08, 0.80, 0.10, 0.02},
			{0.03, 0.10, 0.80, 0.07},
			{0.01, 0.02, 0.10, 0.87},
		},
		InitialProbs: [4]float64{0.7, 0.2, 0.08, 0.02},
		EmissionMeans: [4][5]float64{
			{0.0, 0.01, 0.5, 0.001, 0.0},
			{0.001, 0.02, 0.8, 0.002, 0.1},
			{0.002, 0.05, 1.5, 0.005, 0.3},
			{0.005, 0.10, 2.5, 0.01, 0.5},
		},
		EmissionDiagSDs: [4][5]float64{
			{0.02, 0.005, 0.3, 0.002, 0.1},
			{0.03, 0.008, 0.5, 0.004, 0.2},
			{0.06, 0.015, 1.0, 0.008, 0.4},
			{0.10, 0.03, 2.0, 0.02, 0.8},
		},
		Loaded: true,
	}

	prevPrice := int64(100_000_000)
	for i := 0; i < 20; i++ {
		priceRaw := prevPrice + int64(float64(prevPrice)*0.001*float64(i%3-1)/252)
		obs := [5]float64{
			float64(priceRaw-prevPrice) / float64(prevPrice),
			0.01 + float64(i)*0.0001,
			0.5 + float64(i%5)*0.05,
			0.001,
			0.0,
		}
		tracker.UpdateEnhanced(priceRaw, prevPrice, obs, params)
		prevPrice = priceRaw
	}

	if tracker.CurrentState < 0 || tracker.CurrentState > 3 {
		t.Errorf("CurrentState should be 0-3, got %d", tracker.CurrentState)
	}
	if tracker.Confidence < 0 || tracker.Confidence > 1 {
		t.Errorf("Confidence should be in [0,1], got %f", tracker.Confidence)
	}
}

func TestHMMTracker_UpdateEnhanced_NoPrevPrice(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	params := EnhancedHMMParams{Loaded: true}
	tracker.UpdateEnhanced(100_000_000, 0, [5]float64{}, params)
	if tracker.ReturnCount != 0 {
		t.Error("Zero prevPrice should not update tracker")
	}
}

func TestHMMTracker_UpdateEnhanced_NotLoaded(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	params := EnhancedHMMParams{Loaded: false}
	prevPrice := int64(100_000_000)
	priceRaw := int64(100_050_000)
	tracker.UpdateEnhanced(priceRaw, prevPrice, [5]float64{0.0005, 0.01, 0.5, 0.001, 0.0}, params)
	if tracker.ReturnCount != 0 {
		t.Error("Not-loaded params should not update tracker")
	}
}

func TestHMMTracker_UpdateEnhanced_InsufficientData(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	params := EnhancedHMMParams{Loaded: true}
	prevPrice := int64(100_000_000)
	for i := 0; i < 5; i++ {
		priceRaw := int64(100_000_000 + i*10_000)
		tracker.UpdateEnhanced(priceRaw, prevPrice, [5]float64{0.0001, 0.01, 0.5, 0.001, 0.0}, params)
		prevPrice = priceRaw
	}
	if tracker.CurrentState != HMMState(0) {
		t.Errorf("Insufficient data should keep default state (Calm=0), got %d", tracker.CurrentState)
	}
}

func TestHMMTracker_UpdateEnhanced_RegimeSwitch(t *testing.T) {
	tracker := NewHMMTracker(DefaultHMM())
	params := EnhancedHMMParams{
		Transition: [4][4]float64{
			{0.85, 0.10, 0.04, 0.01},
			{0.08, 0.80, 0.10, 0.02},
			{0.03, 0.10, 0.80, 0.07},
			{0.01, 0.02, 0.10, 0.87},
		},
		InitialProbs:    [4]float64{0.7, 0.2, 0.08, 0.02},
		EmissionMeans:   [4][5]float64{{0, 0.01, 0.5, 0.001, 0}, {0, 0.01, 0.5, 0.001, 0}, {0, 0.01, 0.5, 0.001, 0}, {0, 0.01, 0.5, 0.001, 0}},
		EmissionDiagSDs: [4][5]float64{{0.02, 0.005, 0.3, 0.002, 0.1}, {0.02, 0.005, 0.3, 0.002, 0.1}, {0.02, 0.005, 0.3, 0.002, 0.1}, {0.02, 0.005, 0.3, 0.002, 0.1}},
		Loaded:          true,
	}

	prevPrice := int64(100_000_000)
	for i := 0; i < 256; i++ {
		vol := 0.01
		if i > 200 {
			vol = 0.05
		}
		ret := (float64(i%5-2) * 0.001 * vol)
		priceRaw := int64(float64(prevPrice) * (1.0 + ret))
		if priceRaw <= 0 {
			priceRaw = 100_000_000
		}
		obs := [5]float64{ret, vol, 0.5, 0.001, 0.0}
		tracker.UpdateEnhanced(priceRaw, prevPrice, obs, params)
		prevPrice = priceRaw
	}

	if tracker.Confidence < 0 {
		t.Error("Confidence should be non-negative after 256 updates")
	}
}

func TestLoadEnhancedHMMParams_MalformedJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "hmm_enhanced_malformed_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("{invalid json")
	tmpFile.Close()

	_, err = LoadEnhancedHMMParams(tmpFile.Name())
	if err == nil {
		t.Error("Malformed JSON should return error")
	}
}

func TestEmissionProbMultiDim_HighDiscrimination(t *testing.T) {
	params := EnhancedHMMParams{
		EmissionMeans: [4][5]float64{
			{0.0, 0.005, 0.3, 0.001, 0.0},
			{0.001, 0.01, 0.6, 0.002, 0.1},
		},
		EmissionDiagSDs: [4][5]float64{
			{0.01, 0.002, 0.1, 0.001, 0.05},
			{0.01, 0.002, 0.1, 0.001, 0.05},
		},
		Loaded: true,
	}

	calmObs := [5]float64{0.0, 0.005, 0.3, 0.001, 0.0}
	trendObs := [5]float64{0.001, 0.01, 0.6, 0.002, 0.1}

	pCalm_givenCalm := emissionProbMultiDim(0, calmObs, params)
	pTrend_givenCalm := emissionProbMultiDim(0, trendObs, params)
	pTrend_givenTrend := emissionProbMultiDim(1, trendObs, params)
	pCalm_givenTrend := emissionProbMultiDim(1, calmObs, params)

	if pCalm_givenCalm <= pTrend_givenCalm {
		t.Errorf("Calm obs should be more likely under Calm state: calm|calm=%e, calm|trend=%e", pCalm_givenCalm, pTrend_givenCalm)
	}
	if pTrend_givenTrend <= pCalm_givenTrend {
		t.Errorf("Trend obs should be more likely under Trend state: trend|trend=%e, trend|calm=%e", pTrend_givenTrend, pCalm_givenTrend)
	}

	ratio := pCalm_givenCalm / pCalm_givenTrend
	if math.Abs(ratio-1.0) < 0.01 {
		t.Errorf("States should produce discriminative probabilities; calm ratio=%.4f", ratio)
	}
}
