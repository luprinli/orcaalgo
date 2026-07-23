package universe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/lee-econ/orca-core/internal/db"
)

func ComputeConfigHash(filters UniverseConfigFilters, triggers DynamicTriggerThresholds) (string, error) {
	data := map[string]interface{}{
		"filters":  filters,
		"triggers": triggers,
	}
	canonical, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal config hash: %w", err)
	}
	h := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", h), nil
}

func ComputeSnapshotHash(symbols []db.Symbol, configHash string) string {
	tickers := make([]string, len(symbols))
	for i, s := range symbols {
		tickers[i] = s.Ticker
	}
	sort.Strings(tickers)
	data := map[string]interface{}{
		"tickers":     tickers,
		"config_hash": configHash,
	}
	canonical, _ := json.Marshal(data)
	h := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", h)
}
