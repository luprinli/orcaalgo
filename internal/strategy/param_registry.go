package strategy

type ParamRegistry struct {
	definitions map[string][]ParamDef
}

func NewParamRegistry() *ParamRegistry {
	return &ParamRegistry{
		definitions: make(map[string][]ParamDef),
	}
}

func (pr *ParamRegistry) Register(name string, defs []ParamDef) {
	pr.definitions[name] = defs
}

func (pr *ParamRegistry) Get(name string) ([]ParamDef, bool) {
	defs, ok := pr.definitions[name]
	return defs, ok
}

func (pr *ParamRegistry) All() map[string][]ParamDef {
	return pr.definitions
}

func ApplyParams(target map[string]float64, updates map[string]float64) {
	for k, v := range updates {
		if _, ok := target[k]; ok {
			target[k] = v
		}
	}
}

func ClampParam(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
