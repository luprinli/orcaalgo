package strategy

type ParamDef struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Step        float64 `json:"step"`
	Group       string  `json:"group"`
	Description string  `json:"description"`
}

const (
	ParamContinuous  = "continuous"
	ParamInteger     = "integer"
	ParamCategorical = "categorical"
)

func (d ParamDef) Clamp(value float64) float64 {
	if value < d.Min {
		return d.Min
	}
	if value > d.Max {
		return d.Max
	}
	return value
}
