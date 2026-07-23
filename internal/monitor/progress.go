package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const DefaultProgressDir = "data/progress"

type BatchProgress struct {
	BatchID       string                 `json:"batch_id"`
	Status        string                 `json:"status"`
	Description   string                 `json:"description"`
	ProgressPct   float64                `json:"progress_pct"`
	Completed     int                    `json:"completed"`
	Failed        int                    `json:"failed"`
	Total         int                    `json:"total"`
	ElapsedS      float64                `json:"elapsed_s"`
	EtaS          *float64               `json:"eta_s,omitempty"`
	StartedAt     string                 `json:"started_at,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
	HaltFileExists bool                  `json:"halt_file_exists"`
}

func ReadBatchProgress(batchID string) (*BatchProgress, error) {
	path := filepath.Join(DefaultProgressDir, batchID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bp BatchProgress
	if err := json.Unmarshal(data, &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}

func ListActiveBatches() ([]BatchProgress, error) {
	entries, err := os.ReadDir(DefaultProgressDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []BatchProgress
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(DefaultProgressDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var bp BatchProgress
		if err := json.Unmarshal(data, &bp); err != nil {
			continue
		}
		if bp.Status == "running" || bp.Status == "halted" || bp.Status == "pending" {
			results = append(results, bp)
		}
	}
	return results, nil
}

func HaltBatch(batchID string) error {
	dir := DefaultProgressDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	haltPath := filepath.Join(dir, ".halt_"+batchID)
	return os.WriteFile(haltPath, []byte(time.Now().Format(time.RFC3339)), 0644)
}

func ResumeBatch(batchID string) error {
	haltPath := filepath.Join(DefaultProgressDir, ".halt_"+batchID)
	return os.Remove(haltPath)
}
