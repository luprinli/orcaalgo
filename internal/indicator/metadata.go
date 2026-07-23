package indicator

type OutputType string

const (
	OutputLine      OutputType = "line"
	OutputHistogram OutputType = "histogram"
)

type PlotOptions struct {
	Color     string  `json:"color"`
	LineWidth float64 `json:"lineWidth,omitempty"`
	Precision int     `json:"precision,omitempty"`
	MinMove   float64 `json:"minMove,omitempty"`
}

type OutputMeta struct {
	Name        string       `json:"name"`
	Type        OutputType   `json:"type"`
	PlotOptions *PlotOptions `json:"plotOptions"`
}

type IndicatorUIMetadata struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Overlay     bool         `json:"overlay"`
	Outputs     []OutputMeta `json:"outputs"`
}

var UIMetadata = map[string]IndicatorUIMetadata{
	"sma": {
		ID: "sma", Name: "Simple Moving Average",
		Description: "Average of closing prices over a window",
		Overlay:     true,
		Outputs: []OutputMeta{
			{Name: "sma", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#FCFC4E", LineWidth: 2}},
		},
	},
	"ema": {
		ID: "ema", Name: "Exponential Moving Average",
		Description: "Weighted moving average with exponential decay",
		Overlay:     true,
		Outputs: []OutputMeta{
			{Name: "ema", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#FF9800", LineWidth: 2}},
		},
	},
	"rsi": {
		ID: "rsi", Name: "Relative Strength Index",
		Description: "Momentum oscillator measuring speed and change of price movements (0-100)",
		Overlay: false,
		Outputs: []OutputMeta{
			{Name: "rsi", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#7E57C2", LineWidth: 2, Precision: 1}},
		},
	},
	"macd": {
		ID: "macd", Name: "MACD",
		Description: "Moving Average Convergence Divergence — trend-following momentum indicator",
		Overlay: false,
		Outputs: []OutputMeta{
			{Name: "macd", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#2962ff", LineWidth: 2, MinMove: 0.00001}},
			{Name: "signal", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#f23645", LineWidth: 2, MinMove: 0.00001}},
			{Name: "hist", Type: OutputHistogram, PlotOptions: &PlotOptions{Color: "#089981", MinMove: 0.00001}},
		},
	},
	"bbands": {
		ID: "bbands", Name: "Bollinger Bands",
		Description: "Volatility bands placed above and below a moving average",
		Overlay: true,
		Outputs: []OutputMeta{
			{Name: "upper", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#f23645", LineWidth: 2}},
			{Name: "mid", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#2962ff", LineWidth: 2}},
			{Name: "lower", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#089981", LineWidth: 2}},
		},
	},
	"atr": {
		ID: "atr", Name: "Average True Range",
		Description: "Measures market volatility by decomposing the entire range of an asset price",
		Overlay: false,
		Outputs: []OutputMeta{
			{Name: "atr", Type: OutputLine, PlotOptions: &PlotOptions{Color: "#00BCD4", LineWidth: 2, Precision: 4}},
		},
	},
}
