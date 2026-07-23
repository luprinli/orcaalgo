package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type StrategyParamSet struct {
	ID     string             `json:"id"`
	Kind   string             `json:"kind"`
	Params map[string]float64 `json:"params"`
}

type StrategyParamsConfig struct {
	Strategies []StrategyParamSet `json:"strategies"`
}

func LoadStrategyParams(path string) (*StrategyParamsConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg StrategyParamsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *StrategyParamsConfig) FindByID(id string) *StrategyParamSet {
	for i := range c.Strategies {
		if c.Strategies[i].ID == id {
			return &c.Strategies[i]
		}
	}
	return nil
}

func (c *StrategyParamsConfig) FindByKind(kind string) []*StrategyParamSet {
	var result []*StrategyParamSet
	for i := range c.Strategies {
		if c.Strategies[i].Kind == kind {
			result = append(result, &c.Strategies[i])
		}
	}
	return result
}

func DefaultStrategyParamsPath() string {
	return "configs/strategy_params.json"
}
