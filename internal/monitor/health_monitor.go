package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/notify"
)

type ComponentStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "degraded", "failed"
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

type HealthStatus struct {
	Healthy    bool              `json:"healthy"`
	Components []ComponentStatus `json:"components"`
	Timestamp  string            `json:"timestamp"`
	Uptime     string            `json:"uptime"`
}

type HealthMonitor struct {
	mu           sync.RWMutex
	pool         *pgxpool.Pool
	accountMgr   *broker.AccountManager
	notifyMgr    *notify.Manager
	status       HealthStatus
	startTime    time.Time
	stop         chan struct{}
}

func NewHealthMonitor(pool *pgxpool.Pool, accountMgr *broker.AccountManager, notifyMgr *notify.Manager) *HealthMonitor {
	return &HealthMonitor{
		pool:       pool,
		accountMgr: accountMgr,
		notifyMgr:  notifyMgr,
		startTime:  time.Now(),
		stop:       make(chan struct{}),
	}
}

func (h *HealthMonitor) Start() {
	go h.run()
	slog.Info("health monitor started", "component", "health_monitor")
}

func (h *HealthMonitor) Stop() {
	close(h.stop)
}

func (h *HealthMonitor) Status() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func (h *HealthMonitor) run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	h.check()

	for {
		select {
		case <-h.stop:
			return
		case <-ticker.C:
			h.check()
		}
	}
}

func (h *HealthMonitor) check() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var components []ComponentStatus
	healthy := true
	anyAccountOk := false

	dbStatus := h.checkDatabase(ctx)
	components = append(components, dbStatus)
	if dbStatus.Status != "ok" {
		healthy = false
	}

	if h.accountMgr != nil {
		for _, acct := range h.accountMgr.ListAccountsByUser(ctx, "") {
			accStatus := h.checkAccount(ctx, acct.ID)
			components = append(components, accStatus)
			if accStatus.Status == "ok" {
				anyAccountOk = true
			}
			if accStatus.Status != "ok" {
				healthy = false
			}
		}
	}

	SetBrokerConnected(anyAccountOk)

	h.mu.Lock()
	h.status = HealthStatus{
		Healthy:    healthy,
		Components: components,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Uptime:     time.Since(h.startTime).Round(time.Second).String(),
	}
	h.mu.Unlock()

	if !healthy && h.notifyMgr != nil {
		h.notifyMgr.Publish(notify.NewEvent(
			notify.EventType("health_degraded"),
			notify.LevelWarning,
			"Health Check Failed",
			"One or more components are unhealthy",
		))
	}
}

func (h *HealthMonitor) checkDatabase(ctx context.Context) ComponentStatus {
	start := time.Now()
	if h.pool == nil {
		return ComponentStatus{Name: "database", Status: "failed", Message: "Not connected"}
	}
	if err := h.pool.Ping(ctx); err != nil {
		return ComponentStatus{Name: "database", Status: "failed", Message: err.Error()}
	}
	return ComponentStatus{Name: "database", Status: "ok", Latency: time.Since(start).Round(time.Millisecond).String()}
}

func (h *HealthMonitor) checkAccount(ctx context.Context, accountID string) ComponentStatus {
	acc, err := h.accountMgr.GetAccount(accountID)
	if err != nil {
		return ComponentStatus{Name: "account:" + accountID, Status: "failed", Message: err.Error()}
	}

	start := time.Now()
	_, err = acc.GetAccount(ctx)
	if err != nil {
		return ComponentStatus{Name: "account:" + accountID, Status: "degraded", Message: err.Error()}
	}

	return ComponentStatus{Name: "account:" + accountID, Status: "ok", Latency: time.Since(start).Round(time.Millisecond).String()}
}
