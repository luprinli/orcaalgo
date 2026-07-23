package backtest

import (
	"context"
	"sync"
	"time"
)

// TaskStatus mirrors the intended backtest_tasks.status column.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
	TaskSkipped   TaskStatus = "skipped"
)

type taskRecord struct {
	BatchID string                 `json:"batch_id"`
	Seq     int                    `json:"seq"`
	Spec    map[string]interface{} `json:"spec"`
	Status  TaskStatus             `json:"status"`
	Result  *ComboResult           `json:"result,omitempty"`
	Created time.Time              `json:"created_at"`
}

// TaskQueue provides durable (in-memory, DB-path ready) storage for matrix
// backtest combos. When a DB is wired in (see TaskDatabase interface below),
// the queue becomes durable across restarts. Currently uses in-memory maps
// with the same semantics as the target backtest_tasks table.
//
// DB-path: implement the TaskDatabase interface against the backtest_tasks
// table (migration 000028), then replace the in-memory maps with DB calls.
type TaskQueue struct {
	mu    sync.RWMutex
	tasks map[string]map[int]*taskRecord // batchID → seq → record
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks: make(map[string]map[int]*taskRecord),
	}
}

func (tq *TaskQueue) Enqueue(ctx context.Context, batchID string, combos []ComboResult) (int, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	if _, ok := tq.tasks[batchID]; !ok {
		tq.tasks[batchID] = make(map[int]*taskRecord)
	}
	for i, c := range combos {
		tq.tasks[batchID][i] = &taskRecord{
			BatchID: batchID,
			Seq:     i,
			Spec: map[string]interface{}{
				"strategy_id": c.StrategyID,
				"symbol":      c.Symbol,
				"timeframe":   c.Timeframe,
			},
			Status:  TaskPending,
			Created: time.Now(),
		}
	}
	return len(combos), nil
}

func (tq *TaskQueue) CountPending(ctx context.Context, batchID string) (int, error) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	count := 0
	if batch, ok := tq.tasks[batchID]; ok {
		for _, t := range batch {
			if t.Status == TaskPending || t.Status == TaskRunning {
				count++
			}
		}
	}
	return count, nil
}

func (tq *TaskQueue) ClaimNext(ctx context.Context, batchID string) (int, map[string]interface{}, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	batch, ok := tq.tasks[batchID]
	if !ok {
		return -1, nil, nil
	}
	for seq, t := range batch {
		if t.Status == TaskPending {
			t.Status = TaskRunning
			return seq, t.Spec, nil
		}
	}
	return -1, nil, nil
}

func (tq *TaskQueue) Complete(ctx context.Context, batchID string, seq int, result *ComboResult) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	if batch, ok := tq.tasks[batchID]; ok {
		if t, ok := batch[seq]; ok {
			t.Status = TaskCompleted
			t.Result = result
		}
	}
	return nil
}

func (tq *TaskQueue) Fail(ctx context.Context, batchID string, seq int, errMsg string) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	if batch, ok := tq.tasks[batchID]; ok {
		if t, ok := batch[seq]; ok {
			t.Status = TaskFailed
			t.Result = &ComboResult{Error: errMsg}
		}
	}
	return nil
}

func (tq *TaskQueue) CancelBatch(ctx context.Context, batchID string) (int, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	count := 0
	if batch, ok := tq.tasks[batchID]; ok {
		for _, t := range batch {
			if t.Status == TaskPending || t.Status == TaskRunning {
				t.Status = TaskCancelled
				count++
			}
		}
	}
	return count, nil
}

func (tq *TaskQueue) GetResultsSince(ctx context.Context, batchID string, sinceSeq int) ([]ComboResult, int, error) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	batch, ok := tq.tasks[batchID]
	if !ok {
		return nil, 0, nil
	}
	var results []ComboResult
	maxSeq := 0
	for seq, t := range batch {
		if seq > maxSeq {
			maxSeq = seq
		}
		if seq >= sinceSeq && t.Result != nil && t.Status == TaskCompleted {
			results = append(results, *t.Result)
		}
	}
	return results, maxSeq + 1, nil
}

// TaskDatabase is the DB-backed persistence interface. Implement this against
// the backtest_tasks table (migration 000028) for durability across restarts.
//
//	type TaskDatabase interface {
//	    InsertTask(ctx, batchID string, seq int, spec map[string]interface{}) error
//	    CountTasksByStatus(ctx, batchID, status string) (int, error)
//	    ClaimPendingTask(ctx, batchID string) (seq int, spec map[string]interface{}, err error)
//	    UpdateTask(ctx, batchID string, seq int, status string, result *ComboResult) error
//	    CancelBatchTasks(ctx, batchID string) (int, error)
//	    GetTaskResults(ctx, batchID string, sinceSeq int) ([]ComboResult, int, error)
//	    DeleteTasksOlderThan(ctx, olderThan time.Duration) (int, error)
//	}
