package persist

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultSnapshotDir = "data/snapshots"

type StateManager struct {
	mu          sync.RWMutex
	snapshotDir string
	snapshotCh  chan snapshotRequest
}

type snapshotRequest struct {
	key  string
	data interface{}
}

func NewStateManager(snapshotDir string) *StateManager {
	if snapshotDir == "" {
		snapshotDir = defaultSnapshotDir
	}
	os.MkdirAll(snapshotDir, 0755)

	sm := &StateManager{
		snapshotDir: snapshotDir,
		snapshotCh:  make(chan snapshotRequest, 256),
	}
	go sm.processSnapshots()
	return sm
}

func (sm *StateManager) processSnapshots() {
	for req := range sm.snapshotCh {
		data, err := json.Marshal(req.data)
		if err != nil {
			log.Printf("state manager: marshal %s: %v", req.key, err)
			continue
		}
		path := filepath.Join(sm.snapshotDir, req.key+".json")
		if err := WriteFileAtomic(path, data); err != nil {
			log.Printf("state manager: write %s: %v", req.key, err)
		}
	}
}

func (sm *StateManager) Snapshot(key string, data interface{}) {
	select {
	case sm.snapshotCh <- snapshotRequest{key: key, data: data}:
	default:
		log.Printf("state manager: snapshot channel full, dropping %s", key)
	}
}

func (sm *StateManager) Restore(key string, target interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	path := filepath.Join(sm.snapshotDir, key+".json")
	data, err := ReadFileWithRecovery(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

type OrderSnapshot struct {
	OrderID      string    `json:"order_id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Quantity     float64   `json:"quantity"`
	LimitPrice   float64   `json:"limit_price"`
	StopPrice    float64   `json:"stop_price"`
	Status       string    `json:"status"`
	StrategyID   string    `json:"strategy_id"`
	FilledQty    float64   `json:"filled_qty"`
	AvgFillPrice float64   `json:"avg_fill_price"`
	CreatedAt    time.Time `json:"created_at"`
}

type PositionSnapshot struct {
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	AvgEntryPrice float64 `json:"avg_entry_price"`
	MarketValue   float64 `json:"market_value"`
	UnrealizedPL  float64 `json:"unrealized_pl"`
}

type RiskSnapshot struct {
	Halted                 bool    `json:"halted"`
	Reason                 string  `json:"reason"`
	DailyPnLPct            float64 `json:"daily_pnl_pct"`
	DrawdownPct            float64 `json:"drawdown_pct"`
	Balance                float64 `json:"balance"`
	Equity                 float64 `json:"equity"`
	ConsistencyMultiplier  float64 `json:"consistency_multiplier"`
}
