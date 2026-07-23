package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStrategyParams(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "strategy_params.json")

	testCfg := StrategyParamsConfig{
		Strategies: []StrategyParamSet{
			{
				ID:   "orca-intraday-mr-v1",
				Kind: "mean_reversion",
				Params: map[string]float64{
					"lookback":  20,
					"entry_z":   2.0,
					"exit_z":    0.5,
				},
			},
		},
	}
	data, _ := json.MarshalIndent(testCfg, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	cfg, err := LoadStrategyParams(cfgPath)
	if err != nil {
		t.Fatalf("LoadStrategyParams failed: %v", err)
	}

	mr := cfg.FindByID("orca-intraday-mr-v1")
	if mr == nil {
		t.Fatal("FindByID returned nil")
	}
	if mr.Kind != "mean_reversion" {
		t.Errorf("expected kind 'mean_reversion', got '%s'", mr.Kind)
	}
	if mr.Params["lookback"] != 20 {
		t.Errorf("expected lookback 20, got %v", mr.Params["lookback"])
	}

	byKind := cfg.FindByKind("mean_reversion")
	if len(byKind) != 1 {
		t.Errorf("expected 1 mean_reversion, got %d", len(byKind))
	}

	notFound := cfg.FindByID("nonexistent")
	if notFound != nil {
		t.Error("FindByID should return nil for nonexistent")
	}
}
