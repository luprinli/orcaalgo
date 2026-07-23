// Package ml provides machine learning primitives: feature computation,
// model inference (ONNX runtime), and model lifecycle management.
//
// Feature indices and names MUST match orca/ml/config.py to prevent
// train/serve skew.
package ml

import "time"

// FeatureDim is the dimension of the feature vector (21 features).
const FeatureDim = 21

// FeatureNames matches orca/ml/config.py FEATURE_NAMES exactly.
var FeatureNames = [FeatureDim]string{
	"ret1",            //  0: 1-bar log return
	"ret5",            //  1: 5-bar log return
	"ret20",           //  2: 20-bar log return
	"volatility20",    //  3: 20-period EWMA volatility
	"atr_ratio",       //  4: ATR / Close
	"rsi14",           //  5: RSI(14)
	"macd_hist",       //  6: MACD histogram
	"adx14",           //  7: ADX(14)
	"bb_percent_b",    //  8: Bollinger %B
	"volume_ratio",    //  9: Volume / 20-period avg volume
	"cvd_divergence",  // 10: CVD divergence flag
	"spread_pct",      // 11: Bid-ask spread / Close
	"hmm_state_0",     // 12: HMM alpha[0]
	"hmm_state_1",     // 13: HMM alpha[1]
	"hmm_state_2",     // 14: HMM alpha[2]
	"hmm_state_3",     // 15: HMM alpha[3]
	"hmm_confidence",  // 16: HMM confidence
	"signal_type",     // 17: Strategy type (0-9)
	"signal_strength", // 18: Signal conviction
	"hour_sin",        // 19: sin(2π * hour / 24)
	"hour_cos",        // 20: cos(2π * hour / 24)
}

// FeatureVector is a 21-dimensional numeric feature vector for ML inference.
// Values are float32 for ONNX runtime compatibility.
type FeatureVector [FeatureDim]float32

// MLConfig holds runtime configuration for the ML subsystem.
type MLConfig struct {
	MetaWinThreshold      float64 `json:"meta_win_threshold" yaml:"meta_win_threshold"`
	MetaExtremeLow        float64 `json:"meta_extreme_low" yaml:"meta_extreme_low"`
	MetaExtremeHigh       float64 `json:"meta_extreme_high" yaml:"meta_extreme_high"`
	MetaPositionScaleCap  float64 `json:"meta_position_scale_cap" yaml:"meta_position_scale_cap"`
	InferenceTimeoutUs    int64   `json:"inference_timeout_us" yaml:"inference_timeout_us"`
	FeatureComputeBudgetUs int64  `json:"feature_compute_budget_us" yaml:"feature_compute_budget_us"`
}

// DefaultMLConfig returns safe production defaults matching orca/ml/config.py.
func DefaultMLConfig() MLConfig {
	return MLConfig{
		MetaWinThreshold:      0.55,
		MetaExtremeLow:        0.05,
		MetaExtremeHigh:       0.95,
		MetaPositionScaleCap:  1.50,
		InferenceTimeoutUs:    100,
		FeatureComputeBudgetUs: 50,
	}
}

// MetaLabelingResult is the output of the meta-labeling model.
type MetaLabelingResult struct {
	PWin      float64  `json:"p_win"`
	Threshold float64  `json:"threshold"`
	Accepted  bool     `json:"accepted"`
	Reason    string   `json:"reason,omitempty"`
}

// RejectionEntry is a single signal rejection record for the audit log.
type RejectionEntry struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Symbol     string    `json:"symbol"`
	Strategy   string    `json:"strategy"`
	ModelName  string    `json:"model_name"`
	ModelVer   string    `json:"model_version"`
	PWin       float64   `json:"p_win"`
	Threshold  float64   `json:"threshold"`
	RawSignal  string    `json:"raw_signal"`
	Features   string    `json:"feature_values"`
	SHAPValues string    `json:"feature_importance"`
}

// DriftStatus represents the severity of model drift.
type DriftStatus int

const (
	DriftNone DriftStatus = iota
	DriftModerate
	DriftSignificant
)

func (d DriftStatus) String() string {
	switch d {
	case DriftNone:
		return "no_drift"
	case DriftModerate:
		return "moderate_drift"
	case DriftSignificant:
		return "significant_drift"
	default:
		return "unknown"
	}
}
