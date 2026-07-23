package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
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
	BatchID        string                 `json:"batch_id"`
	Mode           string                 `json:"mode"`
	Total          int                    `json:"total"`
	Completed      int                    `json:"completed"`
	Failed         int                    `json:"failed"`
	Running        int                    `json:"running"`
	Passed         int                    `json:"passed"`
	StartTime      time.Time              `json:"start_time"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Combos         []ComboProgress        `json:"combos"`
	Results        []backtest.ComboResult `json:"results,omitempty"`
	Status         string                 `json:"status"`
	NextSeq        int                    `json:"next_seq"`
	bestSharpe     float64
	bestStrategy   string
	bestSymbol     string
	totalTrades    int
	skipped        int
	phase          string
}

type ProgressStore struct {
	mu         sync.RWMutex
	runs       map[string]*MatrixProgress
	hub        *monitor.WSHub
	repo       *db.Repository
	cleanupStop chan struct{}
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

func (ps *ProgressStore) SetRepo(repo *db.Repository) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.repo = repo
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

	ps.persistAsync(batchID)

	return mp
}

func (ps *ProgressStore) persistAsync(batchID string) {
	if ps.repo == nil {
		return
	}
	go func() {
		ps.mu.RLock()
		mp, ok := ps.runs[batchID]
		if !ok {
			ps.mu.RUnlock()
			return
		}
		combosJSON, _ := json.Marshal(mp.Combos)
		resultsJSON, _ := json.Marshal(mp.Results)
		rec := &db.MatrixProgressRecord{
			BatchID: mp.BatchID, Mode: mp.Mode, Total: mp.Total,
			Completed: mp.Completed, Failed: mp.Failed, Running: mp.Running,
			Passed: mp.Passed, Status: mp.Status,
			StartTime: mp.StartTime, UpdatedAt: mp.UpdatedAt,
			CombosJSON: combosJSON, ResultsJSON: resultsJSON,
			BestSharpe: mp.bestSharpe, BestStrategy: mp.bestStrategy,
			BestSymbol: mp.bestSymbol, TotalTrades: mp.totalTrades,
		}
		ps.mu.RUnlock()
		if err := ps.repo.UpsertMatrixProgress(context.Background(), rec); err != nil {
			slog.Error("progress_store: failed to persist", "batch_id", batchID, "err", err)
		}
	}()
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
			if result.GatePassed != nil && *result.GatePassed {
				mp.Passed++
			}
			mp.totalTrades += result.NumTrades
			if result.SharpeRatio > mp.bestSharpe {
				mp.bestSharpe = result.SharpeRatio
				mp.bestStrategy = result.StrategyID
				mp.bestSymbol = result.Symbol
			}
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

	ps.persistAsync(batchID)
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

func (ps *ProgressStore) GetSummary(batchID string) gin.H {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	mp, ok := ps.runs[batchID]
	if !ok {
		return gin.H{}
	}
	elapsed := time.Since(mp.StartTime).Seconds()
	throughputPerMin := 0.0
	if elapsed > 0 && mp.Completed > 0 {
		throughputPerMin = float64(mp.Completed) / elapsed * 60
	}
	etaSeconds := 0.0
	if throughputPerMin > 0 && mp.Total > mp.Completed {
		etaSeconds = float64(mp.Total-mp.Completed) / throughputPerMin * 60
	}
	percent := 0.0
	if mp.Total > 0 {
		percent = float64(mp.Completed) / float64(mp.Total) * 100
	}
	return gin.H{
		"total_combos":      mp.Total,
		"passed":            mp.Passed,
		"failed":            mp.Failed,
		"total_trades":      mp.totalTrades,
		"best_sharpe":       mp.bestSharpe,
		"best_strategy":     mp.bestStrategy,
		"best_symbol":       mp.bestSymbol,
		"status":            mp.Status,
		"completed":         mp.Completed,
		"running":           mp.Running,
		"skipped":           mp.skipped,
		"phase":             mp.phase,
		"percent":           percent,
		"throughput_per_min": throughputPerMin,
		"eta_seconds":       etaSeconds,
		"seq":               mp.NextSeq,
	}
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

	ps.persistAsync(batchID)
}

func (ps *ProgressStore) RecoverFromDB(ctx context.Context) {
	if ps.repo == nil {
		return
	}
	active, err := ps.repo.ListActiveMatrices(ctx)
	if err != nil {
		slog.Error("progress_store: recovery query failed", "err", err)
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, rec := range active {
		if _, exists := ps.runs[rec.BatchID]; exists {
			continue
		}
		var combos []ComboProgress
		var results []backtest.ComboResult
		json.Unmarshal(rec.CombosJSON, &combos)
		json.Unmarshal(rec.ResultsJSON, &results)
		mp := &MatrixProgress{
			BatchID: rec.BatchID, Mode: rec.Mode, Total: rec.Total,
			Completed: rec.Completed, Failed: rec.Failed, Running: rec.Running,
			Passed: rec.Passed, Status: rec.Status,
			StartTime: rec.StartTime, UpdatedAt: rec.UpdatedAt,
			Combos: combos, Results: results,
			NextSeq: len(results),
			bestSharpe: rec.BestSharpe, bestStrategy: rec.BestStrategy,
			bestSymbol: rec.BestSymbol, totalTrades: rec.TotalTrades,
		}
		ps.runs[rec.BatchID] = mp
		slog.Info("progress_store: recovered matrix from db", "batch_id", rec.BatchID, "completed", rec.Completed)
	}
}

func (ps *ProgressStore) StartCleanupJob() {
	if ps.repo == nil {
		return
	}
	ps.cleanupStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, err := ps.repo.CleanupOldMatrices(context.Background())
				if err != nil {
					slog.Error("progress_store: cleanup failed", "err", err)
				} else if n > 0 {
					slog.Info("progress_store: cleaned up old matrices", "count", n)
				}
			case <-ps.cleanupStop:
				return
			}
		}
	}()
}

func (ps *ProgressStore) StopCleanupJob() {
	if ps.cleanupStop != nil {
		close(ps.cleanupStop)
	}
}
