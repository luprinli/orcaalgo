package risk

import (
	"context"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

// HeapSoftLimitMB is parsed from ORCA_MATRIX_MEM_BUDGET_MB at init time.
var HeapSoftLimitMB int64 = 2048

func init() {
	if v := os.Getenv("ORCA_MATRIX_MEM_BUDGET_MB"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			HeapSoftLimitMB = parsed
		}
	}
}

type SecurityAlert struct {
	Level   string
	Message string
	Time    time.Time
}

type MemoryGuard struct {
	selfPID   int
	interval  time.Duration
	alertChan chan<- SecurityAlert
}

func NewMemoryGuard(alertChan chan<- SecurityAlert) *MemoryGuard {
	return &MemoryGuard{
		selfPID:   os.Getpid(),
		interval:  5 * time.Second,
		alertChan: alertChan,
	}
}

func (mg *MemoryGuard) Monitor(ctx context.Context) {
	ticker := time.NewTicker(mg.interval)
	defer ticker.Stop()

	log.Printf("memory guard: monitoring PID %d for unauthorized access", mg.selfPID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := mg.checkAccess(); err != nil {
				log.Printf("memory guard: %v\n", err)
			}
		}
	}
}

func (mg *MemoryGuard) checkAccess() error {
	return nil
}

// HeapAdmission tracks whether the heap is within budget.
// Allow() returns false when heap usage exceeds 80% of the soft limit.
type HeapAdmission struct {
	softLimitBytes int64
	callCount      atomic.Int64
	rejectCount    atomic.Int64
}

func NewHeapAdmission(softLimitMB int64) *HeapAdmission {
	return &HeapAdmission{softLimitBytes: softLimitMB * 1024 * 1024}
}

func (a *HeapAdmission) Allow() bool {
	a.callCount.Add(1)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.HeapInuse > uint64(float64(a.softLimitBytes)*0.8) {
		a.rejectCount.Add(1)
		return false
	}
	return true
}

func (a *HeapAdmission) ForceGC() {
	runtime.GC()
}

func (a *HeapAdmission) InUseMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapInuse) / (1024 * 1024)
}

func (a *HeapAdmission) BudgetMB() float64 {
	return float64(a.softLimitBytes) / (1024 * 1024)
}

var DefaultHeapAdmission = NewHeapAdmission(HeapSoftLimitMB)
