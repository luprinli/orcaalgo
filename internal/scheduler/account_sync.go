package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/monitor"
)

type AccountSyncConfig struct {
	Interval    time.Duration
	WsHub       *monitor.WSHub
	AccountMgr  *broker.AccountManager
}

type AccountSyncService struct {
	cfg AccountSyncConfig
	ctx context.Context
	stop context.CancelFunc
}

func NewAccountSyncService(cfg AccountSyncConfig) *AccountSyncService {
	ctx, stop := context.WithCancel(context.Background())
	return &AccountSyncService{
		cfg:  cfg,
		ctx:  ctx,
		stop: stop,
	}
}

func (s *AccountSyncService) Start() {
	if s.cfg.AccountMgr == nil {
		slog.Warn("AccountManager not configured, skipping", "component", "account_sync")
		return
	}
	if s.cfg.Interval <= 0 {
		s.cfg.Interval = 5 * time.Second
	}

	go s.run()
	slog.Info("account sync started", "interval", s.cfg.Interval, "component", "account_sync")
}

func (s *AccountSyncService) Stop() {
	s.stop()
}

func (s *AccountSyncService) run() {
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	s.sync()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.sync()
		}
	}
}

func (s *AccountSyncService) sync() {
	ctx := context.Background()
	accounts := s.cfg.AccountMgr.ListAccountsByUser(ctx, "")
	for _, acct := range accounts {
		if err := s.cfg.AccountMgr.SyncAccountState(ctx, acct.ID); err != nil {
			slog.Error("failed to sync account", "account_id", acct.ID, "error", err, "component", "account_sync")
			continue
		}

		if s.cfg.WsHub != nil {
			s.cfg.WsHub.Broadcast("account_status", map[string]interface{}{
				"account_id":       acct.ID,
				"name":             acct.Name,
				"broker_type":      acct.BrokerType,
				"balance":          acct.Balance,
				"equity":           acct.Equity,
				"daily_pnl":        acct.DailyPnL,
				"high_water_mark":  acct.HighWaterMark,
				"is_default":       acct.IsDefault,
				"timestamp":        time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
}

func (s *AccountSyncService) SyncNow() {
	s.sync()
}
