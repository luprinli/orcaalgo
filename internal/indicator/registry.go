package indicator

import "sort"

type ParamType string

const (
	ParamInt    ParamType = "int"
	ParamFloat  ParamType = "float"
	ParamString ParamType = "string"
)

type ParamDef struct {
	Name        string      `json:"name"`
	Type        ParamType   `json:"type"`
	Default     interface{} `json:"default"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	Step        *float64    `json:"step,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Description string      `json:"description,omitempty"`
}

type IndicatorSpec struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Overlay     bool         `json:"overlay"`
	Parameters  []ParamDef   `json:"parameters"`
	Outputs     []OutputMeta `json:"outputs"`
	Warmup      int          `json:"warmup"`
}

func (s *IndicatorSpec) RequiredFields() []string {
	for _, p := range s.Parameters {
		if p.Name == "source" {
			return []string{"open", "high", "low", "close"}
		}
	}
	return []string{"close"}
}

var Registry = map[string]*IndicatorSpec{
	"sma": {
		ID: "sma", Name: "Simple Moving Average",
		Description: "Average of closing prices over a window",
		Overlay:     true,
		Warmup:      20,
		Parameters: []ParamDef{
			{Name: "period", Type: ParamInt, Default: 20, Min: f64ptr(1), Max: f64ptr(500), Description: "Lookback window"},
			{Name: "source", Type: ParamString, Default: "close", Options: []string{"close", "open", "high", "low"}, Description: "Price source"},
		},
		Outputs: []OutputMeta{
			{Name: "sma", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#FCFC4E", LineWidth: 2}},
		},
	},
	"ema": {
		ID: "ema", Name: "Exponential Moving Average",
		Description: "Weighted moving average with exponential decay",
		Overlay:     true,
		Warmup:      20,
		Parameters: []ParamDef{
			{Name: "period", Type: ParamInt, Default: 20, Min: f64ptr(1), Max: f64ptr(500), Description: "Lookback window"},
			{Name: "source", Type: ParamString, Default: "close", Options: []string{"close", "open", "high", "low"}, Description: "Price source"},
		},
		Outputs: []OutputMeta{
			{Name: "ema", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#FF9800", LineWidth: 2}},
		},
	},
	"rsi": {
		ID: "rsi", Name: "Relative Strength Index",
		Description: "Momentum oscillator measuring speed and change of price movements (0-100)",
		Overlay:     false,
		Warmup:      15,
		Parameters: []ParamDef{
			{Name: "period", Type: ParamInt, Default: 14, Min: f64ptr(1), Max: f64ptr(100), Description: "Lookback period"},
			{Name: "source", Type: ParamString, Default: "close", Options: []string{"close", "open", "high", "low"}, Description: "Price source"},
		},
		Outputs: []OutputMeta{
			{Name: "rsi", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#7E57C2", LineWidth: 2, Precision: 1}},
		},
	},
	"macd": {
		ID: "macd", Name: "MACD",
		Description: "Moving Average Convergence Divergence — trend-following momentum indicator",
		Overlay:     false,
		Warmup:      35,
		Parameters: []ParamDef{
			{Name: "fast", Type: ParamInt, Default: 12, Min: f64ptr(2), Max: f64ptr(100), Description: "Fast EMA period"},
			{Name: "slow", Type: ParamInt, Default: 26, Min: f64ptr(2), Max: f64ptr(200), Description: "Slow EMA period"},
			{Name: "signal", Type: ParamInt, Default: 9, Min: f64ptr(1), Max: f64ptr(100), Description: "Signal line EMA period"},
			{Name: "source", Type: ParamString, Default: "close", Options: []string{"close", "open", "high", "low"}, Description: "Price source"},
		},
		Outputs: []OutputMeta{
			{Name: "macd", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#2962ff", LineWidth: 2, MinMove: 0.00001}},
			{Name: "signal", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#f23645", LineWidth: 2, MinMove: 0.00001}},
			{Name: "hist", Type: OutputHistogram, PlotOptions: &PlotOptions{Color: "#089981", MinMove: 0.00001}},
		},
	},
	"bbands": {
		ID: "bbands", Name: "Bollinger Bands",
		Description: "Volatility bands placed above and below a moving average",
		Overlay:     true,
		Warmup:      20,
		Parameters: []ParamDef{
			{Name: "period", Type: ParamInt, Default: 20, Min: f64ptr(1), Max: f64ptr(200), Description: "Lookback window"},
			{Name: "std_dev", Type: ParamFloat, Default: 2.0, Min: f64ptr(0.1), Max: f64ptr(5.0), Step: f64ptr(0.1), Description: "Standard deviation multiplier"},
			{Name: "source", Type: ParamString, Default: "close", Options: []string{"close", "open", "high", "low"}, Description: "Price source"},
		},
		Outputs: []OutputMeta{
			{Name: "upper", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#f23645", LineWidth: 2}},
			{Name: "mid", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#2962ff", LineWidth: 2}},
			{Name: "lower", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#089981", LineWidth: 2}},
		},
	},
	"atr": {
		ID: "atr", Name: "Average True Range",
		Description: "Measures market volatility by decomposing the entire range of an asset price",
		Overlay:     false,
		Warmup:      15,
		Parameters: []ParamDef{
			{Name: "period", Type: ParamInt, Default: 14, Min: f64ptr(1), Max: f64ptr(100), Description: "Lookback window"},
		},
		Outputs: []OutputMeta{
			{Name: "atr", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#00BCD4", LineWidth: 2, Precision: 4}},
		},
	},
}

func List() []*IndicatorSpec {
	var specs []*IndicatorSpec
	for _, s := range Registry {
		specs = append(specs, s)
	}
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].ID < specs[j].ID
	})
	return specs
}

func Get(id string) (*IndicatorSpec, bool) {
	s, ok := Registry[id]
	return s, ok
}

func f64ptr(v float64) *float64 {
	return &v
}
