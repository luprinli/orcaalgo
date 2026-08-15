package backtest

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"
)

// Execution resource-management knobs (see docs/backtest_execution_framework_plan.md §2, §8).
// All are overridable via environment variables so the engine can be sized to the
// host without recompilation.

const (
	defaultMemBudgetMB   = 2048
	defaultPerBacktestMB = 128
	defaultDBReserve     = 4
	// DefaultDBPoolMax is the DB connection-pool size the matrix worker pool is
	// sized against. It mirrors db.Config.PoolMax's default (repository_core.go).
	// Keep this in sync if the DB pool default changes. Bumped from 20 to 40 so
	// 16 matrix workers x ~4 concurrent loads never starve the pool.
	DefaultDBPoolMax = 40
	// The execution framework (bounded global worker pool + LRU candle cache +
	// admission control + chunked streaming) makes the full realistic universe
	// (~11 strategies x 62 symbols x 7 timeframes ≈ 4,466 combos) safe to run with
	// peak memory bounded independently of matrix size. The cap is a runaway guard,
	// not a performance limit, so it sits above the full universe. Override with
	// ORCA_MATRIX_MAX_COMBOS.
	defaultMaxCombos     = 5000
	defaultEquityPoints  = 2000
	defaultChunkSize     = 50
	defaultCandleCap     = 6
	maxWorkersHardCap    = 16
)

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// MatrixWorkers returns a resource-aware worker count derived from CPU, a memory
// budget, and the DB connection pool — never the raw matrix size. This bounds
// peak resource usage as a function of the configured budget (plan §2.1).
//
// workers = clamp(min(NumCPU-1, MemBudget/PerBacktest, DBPoolMax-DBReserve), 1, hardCap)
func MatrixWorkers(dbPoolMax int) int {
	if forced := envInt("ORCA_MATRIX_WORKERS", 0); forced > 0 {
		return clampInt(forced, 1, maxWorkersHardCap)
	}

	cpu := runtime.NumCPU() - 1
	if cpu < 1 {
		cpu = 1
	}

	memBudget := envInt("ORCA_MATRIX_MEM_BUDGET_MB", defaultMemBudgetMB)
	perBT := envInt("ORCA_MATRIX_PER_BT_MB", defaultPerBacktestMB)
	memWorkers := memBudget / perBT
	if memWorkers < 1 {
		memWorkers = 1
	}

	dbReserve := envInt("ORCA_DB_RESERVE", defaultDBReserve)
	dbWorkers := dbPoolMax - dbReserve
	if dbWorkers < 1 {
		dbWorkers = 1
	}

	w := cpu
	if memWorkers < w {
		w = memWorkers
	}
	if dbWorkers < w {
		w = dbWorkers
	}
	return clampInt(w, 1, maxWorkersHardCap)
}

// MaxCombos is the hard ceiling on a single matrix submission. Oversized matrices
// are rejected with guidance to chunk (plan §0/§7).
func MaxCombos() int { return envInt("ORCA_MATRIX_MAX_COMBOS", defaultMaxCombos) }

// MemBudgetMB is the heap budget used for admission control and capacity reporting.
func MemBudgetMB() int { return envInt("ORCA_MATRIX_MEM_BUDGET_MB", defaultMemBudgetMB) }

// ChunkSize is the logical chunk size used for chunk telemetry and audit chunking.
func ChunkSize() int { return envInt("ORCA_CHUNK_SIZE", defaultChunkSize) }

// CandleCacheCap bounds the number of distinct candle sets held in memory at once
// (LRU capacity). With cache-grouped combo ordering this keeps candle memory
// bounded regardless of matrix size (plan §2.3/§3.4).
func CandleCacheCap() int { return envInt("ORCA_CANDLE_CACHE_CAP", defaultCandleCap) }

// softHeapLimitBytes is the heap level above which new combo starts are throttled.
func softHeapLimitBytes() uint64 {
	return uint64(MemBudgetMB()) * 1024 * 1024 * 85 / 100
}

// AwaitHeadroom throttles new combo starts under memory pressure: if heap is above
// the soft limit it forces a GC and waits (bounded) for it to recover, providing
// admission control that prevents host OOM under heavy matrices (plan §2.2).
// Returns immediately once heap is below the soft limit or the ceiling is reached.
func AwaitHeadroom(ctx context.Context) {
	var m runtime.MemStats
	limit := softHeapLimitBytes()
	for i := 0; i < 25; i++ {
		runtime.ReadMemStats(&m)
		if m.HeapInuse < limit {
			return
		}
		runtime.GC()
		select {
		case <-ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// EquityMaxPoints caps the resolution of equity curves returned/stored so a single
// backtest cannot balloon memory or payload size (plan §2.4).
func EquityMaxPoints() int { return envInt("ORCA_EQUITY_MAX_POINTS", defaultEquityPoints) }

// DownsampleEquity reduces an equity curve to at most maxPoints using stride
// sampling that always preserves the first and last points (so start capital and
// final equity are exact). Returns the input unchanged when already within budget.
func DownsampleEquity(points []EquityPoint, maxPoints int) []EquityPoint {
	if maxPoints <= 2 || len(points) <= maxPoints {
		return points
	}
	n := len(points)
	// Reserve the last slot for the final point; sample the rest evenly.
	stride := float64(n-1) / float64(maxPoints-1)
	out := make([]EquityPoint, 0, maxPoints)
	for i := 0; i < maxPoints-1; i++ {
		idx := int(float64(i) * stride)
		if idx >= n {
			idx = n - 1
		}
		out = append(out, points[idx])
	}
	out = append(out, points[n-1])
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
