package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/monitor"
)

type ComboProgress struct {
	Symbol     string `json:"symbol"`
	StrategyID string `json:"strategy_id"`
	Timeframe  string `json:"timeframe"`
	Status     string `json:"status"` // pending | running | completed | failed
	Error      string `json:"error,omitempty"`
}

type MatrixProgress struct {
	BatchID    string           `json:"batch_id"`
	Mode       string           `json:"mode"`
	Total      int              `json:"total"`
	Completed  int              `json:"completed"`
	Failed     int              `json:"failed"`
	Running    int              `json:"running"`
	StartTime  time.Time        `json:"start_time"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Combos     []ComboProgress  `json:"combos"`
	Results    []backtest.ComboResult `json:"results,omitempty"`
	Status     string           `json:"status"` // running | completed | failed
	NextSeq    int              `json:"next_seq"`
}

type ProgressStore struct {
	mu    sync.RWMutex
	runs  map[string]*MatrixProgress
	hub   *monitor.WSHub
}

func NewProgressStore() *ProgressStore {
	return &ProgressStore{runs: make(map[string]*MatrixProgress)}
}

func NewProgressStoreWithHub(hub *monitor.WSHub) *ProgressStore {
	return &ProgressStore{runs: make(map[string]*MatrixProgress), hub: hub}
}

func (ps *ProgressStore) SetHub(hub *monitor.WSHub) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.hub = hub
}

func (ps *ProgressStore) Create(batchID string, total int, combos []ComboProgress) *MatrixProgress {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	mp := &MatrixProgress{
		BatchID:   batchID,
		Mode:      "matrix",
		Total:     total,
		Combos:    combos,
		StartTime: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "running",
	}
	ps.runs[batchID] = mp
	return mp
}

func (ps *ProgressStore) UpdateCombo(batchID string, index int, status string, err string, result *backtest.ComboResult) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	mp, ok := ps.runs[batchID]
	if !ok || index >= len(mp.Combos) {
		return
	}
	prevStatus := mp.Combos[index].Status
	mp.Combos[index].Status = status
	if err != "" {
		mp.Combos[index].Error = err
		mp.Failed++
	}
	if status == "completed" || status == "failed" {
		if prevStatus == "running" {
			mp.Running--
		}
		mp.Completed++
		if result != nil {
			result.RunID = fmt.Sprintf("%s-%d", batchID, mp.NextSeq)
			mp.Results = append(mp.Results, *result)
			mp.NextSeq++
		}
	} else if status == "running" && prevStatus == "pending" {
		mp.Running++
	}
	mp.UpdatedAt = time.Now()
	if mp.Completed+mp.Failed >= mp.Total {
		mp.Status = "completed"
		mp.Running = 0
	}

	if ps.hub != nil {
		ps.hub.Broadcast("backtest_progress", mp)
	}
}

func (ps *ProgressStore) Get(batchID string) *MatrixProgress {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.runs[batchID]
}

func (ps *ProgressStore) GetSince(batchID string, sinceSeq int) ([]backtest.ComboResult, int, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	mp, ok := ps.runs[batchID]
	if !ok {
		return nil, 0, false
	}
	complete := mp.Status != "running"
	if sinceSeq >= len(mp.Results) {
		return nil, mp.NextSeq, complete
	}
	return mp.Results[sinceSeq:], mp.NextSeq, complete
}

func (ps *ProgressStore) Cancel(batchID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	mp, ok := ps.runs[batchID]
	if !ok {
		return
	}
	for i := range mp.Combos {
		if mp.Combos[i].Status == "pending" || mp.Combos[i].Status == "running" {
			mp.Combos[i].Status = "cancelled"
		}
	}
	mp.Status = "cancelled"
	mp.UpdatedAt = time.Now()
}
