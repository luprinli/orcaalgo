package universe

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
)

type UniverseManager struct {
	repo          *db.Repository
	hub           WSHubBroadcaster
	cache         *UniverseCache
	configID      string
	configHash    string
	currentConfig db.UniverseConfig
	symbols       []db.Symbol
	logger        *slog.Logger
}

type WSHubBroadcaster interface {
	Broadcast(channel string, data interface{})
}

func NewUniverseManager(repo *db.Repository, hub WSHubBroadcaster, logger *slog.Logger) *UniverseManager {
	return &UniverseManager{
		repo:   repo,
		hub:    hub,
		cache:  NewUniverseCache(5 * time.Minute),
		logger: logger,
	}
}

func (m *UniverseManager) LoadInitialUniverse(ctx context.Context) error {
	m.logger.InfoContext(ctx, "universe_load_initial_started")

	cfg, err := m.repo.GetActiveUniverseConfig(ctx, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		m.logger.WarnContext(ctx, "universe_no_active_config", "error", err)
		cfg = &db.UniverseConfig{
			ID:                uuid.New().String(),
			UserID:            "00000000-0000-0000-0000-000000000001",
			Name:              "default",
			ProfileID:         "default",
			AssetClassFilters: map[string]interface{}{},
			DynamicTriggers:   map[string]interface{}{},
			IsActive:          true,
		}
	}

	m.configID = cfg.ID
	m.currentConfig = *cfg

	symbols, err := m.repo.LoadActiveSymbols(ctx)
	if err != nil {
		return fmt.Errorf("load active symbols: %w", err)
	}

	filtered, err := m.RunCoarseFilter(ctx, symbols)
	if err != nil {
		m.logger.WarnContext(ctx, "universe_coarse_filter_failed", "error", err)
		filtered = symbols
	}

	m.symbols = filtered
	m.cache.Set(m.symbols, m.configHash)

	m.logger.InfoContext(ctx, "universe_load_initial_complete",
		"total_symbols", len(symbols),
		"active_symbols", len(m.symbols),
	)

	return nil
}

func (m *UniverseManager) GetCurrentUniverse(ctx context.Context) ([]db.Symbol, error) {
	if cached, ok := m.cache.Get(); ok {
		return cached, nil
	}
	symbols, err := m.repo.LoadActiveSymbols(ctx)
	if err != nil {
		return nil, err
	}
	m.symbols = symbols
	m.cache.Set(symbols, m.configHash)
	return symbols, nil
}

func (m *UniverseManager) GetCurrentTickers(ctx context.Context) ([]string, error) {
	symbols, err := m.GetCurrentUniverse(ctx)
	if err != nil {
		return nil, err
	}
	tickers := make([]string, len(symbols))
	for i, s := range symbols {
		tickers[i] = s.Ticker
	}
	return tickers, nil
}

func (m *UniverseManager) RunCoarseFilter(ctx context.Context, symbols []db.Symbol) ([]db.Symbol, error) {
	filters := DefaultFilters()
	var passed []db.Symbol
	for _, s := range symbols {
		switch s.AssetType {
		case "equity":
			if filters.Equity.Apply(s) {
				passed = append(passed, s)
			}
		case "forex":
			if filters.Forex.Apply(s) {
				passed = append(passed, s)
			}
		case "crypto":
			if filters.Crypto.Apply(s) {
				passed = append(passed, s)
			}
		case "index":
			if filters.Index.Apply(s) {
				passed = append(passed, s)
			}
		case "commodity":
			if filters.Commodity.Apply(s) {
				passed = append(passed, s)
			}
		default:
			if filters.Equity.Apply(s) {
				passed = append(passed, s)
			}
		}
	}
	m.logger.InfoContext(ctx, "universe_coarse_filter",
		"before", len(symbols),
		"after", len(passed),
	)
	return passed, nil
}

func (m *UniverseManager) RunFineFilter(ctx context.Context, symbols []db.Symbol) ([]db.Symbol, error) {
	filters := DefaultFilters()
	var passed []db.Symbol
	for _, s := range symbols {
		switch s.AssetType {
		case "equity":
			if filters.Equity.Apply(s) {
				passed = append(passed, s)
			}
		case "forex":
			if filters.Forex.Apply(s) {
				passed = append(passed, s)
			}
		case "crypto":
			if filters.Crypto.Apply(s) {
				passed = append(passed, s)
			}
		case "index":
			if filters.Index.Apply(s) {
				passed = append(passed, s)
			}
		case "commodity":
			if filters.Commodity.Apply(s) {
				passed = append(passed, s)
			}
		default:
			if filters.Equity.Apply(s) {
				passed = append(passed, s)
			}
		}
	}
	m.logger.InfoContext(ctx, "universe_fine_filter",
		"before", len(symbols),
		"after", len(passed),
	)
	return passed, nil
}

func (m *UniverseManager) DetectDynamicTriggers(ctx context.Context, current []db.Symbol, snapshot MarketDataSnapshot) (additions []db.Symbol, removals []db.Symbol, err error) {
	m.logger.InfoContext(ctx, "universe_dynamic_triggers_started", "symbol_count", len(current))

	triggers := DefaultTriggers()
	currentMap := make(map[string]bool, len(current))
	for _, s := range current {
		currentMap[s.Ticker] = true
	}

	for ticker, metric := range snapshot.SymbolMetrics {
		if triggers.VolumeSpikeMultiplier > 0 && metric.CurrentVolume > metric.AvgVolume20D*triggers.VolumeSpikeMultiplier {
			if !currentMap[ticker] {
				sym, dbErr := m.repo.GetSymbolByTicker(ctx, ticker)
				if dbErr == nil && sym != nil {
					additions = append(additions, *sym)
					m.logger.InfoContext(ctx, "universe_trigger_addition",
						"symbol", ticker,
						"type", "volume_spike",
						"current_volume", metric.CurrentVolume,
						"avg_volume", metric.AvgVolume20D,
					)
				}
			}
		}
		if triggers.VolatilityMultiplier > 0 && metric.ATR14Pct > metric.ATR14Pct*triggers.VolatilityMultiplier {
			if !currentMap[ticker] {
				sym, dbErr := m.repo.GetSymbolByTicker(ctx, ticker)
				if dbErr == nil && sym != nil {
					additions = append(additions, *sym)
					m.logger.InfoContext(ctx, "universe_trigger_addition",
						"symbol", ticker,
						"type", "volatility_breakout",
						"atr_pct", metric.ATR14Pct,
					)
				}
			}
		}
	}

	_ = triggers.NewsSentimentAbsMin
	_ = triggers.MinLookbackDays
	_ = triggers.CooldownHoursAfterAdd
	_ = triggers.CooldownHoursAfterRemove

	return additions, removals, nil
}

func (m *UniverseManager) Refresh(ctx context.Context) error {
	m.logger.InfoContext(ctx, "universe_refresh_started")

	symbols, err := m.repo.LoadActiveSymbols(ctx)
	if err != nil {
		return fmt.Errorf("load symbols: %w", err)
	}

	coarseFiltered, err := m.RunCoarseFilter(ctx, symbols)
	if err != nil {
		m.logger.WarnContext(ctx, "universe_coarse_filter_error", "error", err)
		coarseFiltered = symbols
	}

	fineFiltered, err := m.RunFineFilter(ctx, coarseFiltered)
	if err != nil {
		m.logger.WarnContext(ctx, "universe_fine_filter_error", "error", err)
		fineFiltered = coarseFiltered
	}

	prevTickers := make(map[string]bool)
	for _, s := range m.symbols {
		prevTickers[s.Ticker] = true
	}

	newTickers := make(map[string]bool)
	for _, s := range fineFiltered {
		newTickers[s.Ticker] = true
	}

	var added []string
	for ticker := range newTickers {
		if !prevTickers[ticker] {
			added = append(added, ticker)
		}
	}
	var removed []string
	for ticker := range prevTickers {
		if !newTickers[ticker] {
			removed = append(removed, ticker)
		}
	}

	m.symbols = fineFiltered
	m.cache.Set(m.symbols, m.configHash)

	if err := m.SaveSnapshot(ctx, time.Now()); err != nil {
		m.logger.WarnContext(ctx, "universe_snapshot_save_failed", "error", err)
	}

	if m.hub != nil && (len(added) > 0 || len(removed) > 0) {
		m.hub.Broadcast("universe", map[string]interface{}{
			"type":       "universe_changed",
			"timestamp":  time.Now(),
			"added":      added,
			"removed":    removed,
			"total":      len(m.symbols),
			"snapshot_date": time.Now().Format("2006-01-02"),
		})
	}

	m.logger.InfoContext(ctx, "universe_refresh_complete",
		"total", len(m.symbols),
		"added", len(added),
		"removed", len(removed),
	)
	return nil
}

func (m *UniverseManager) SaveSnapshot(ctx context.Context, date time.Time) error {
	symbolIDs := make([]int32, len(m.symbols))
	for i, s := range m.symbols {
		symbolIDs[i] = s.ID
	}

	configHash, _ := ComputeConfigHash(DefaultFilters(), DefaultTriggers())
	snapshotHash := ComputeSnapshotHash(m.symbols, configHash)

	snap := &db.UniverseSnapshot{
		ID:           uuid.New().String(),
		UserID:       "00000000-0000-0000-0000-000000000001",
		SnapshotDate: date,
		SymbolIDs:    symbolIDs,
		ContentHash:  snapshotHash,
	}

	return m.repo.InsertUniverseSnapshot(ctx, snap)
}

func (m *UniverseManager) LoadSnapshot(ctx context.Context, date time.Time) ([]db.Symbol, error) {
	snap, err := m.repo.GetUniverseSnapshot(ctx, "00000000-0000-0000-0000-000000000001", date)
	if err != nil {
		return nil, fmt.Errorf("load snapshot for %s: %w", date.Format("2006-01-02"), err)
	}
	tickers, err := m.repo.ResolveSnapshotSymbols(ctx, snap.SymbolIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve snapshot symbols: %w", err)
	}
	symbols := make([]db.Symbol, len(tickers))
	for i, t := range tickers {
		symbols[i] = db.Symbol{Ticker: t, ID: snap.SymbolIDs[i]}
	}
	return symbols, nil
}

func (m *UniverseManager) LoadSnapshotsInRange(ctx context.Context, start, end time.Time) ([]db.UniverseSnapshot, error) {
	return m.repo.ListUniverseSnapshots(ctx, "00000000-0000-0000-0000-000000000001", start, end)
}

func (m *UniverseManager) AddManualOverride(ctx context.Context, ticker string) error {
	sym, err := m.repo.GetSymbolByTicker(ctx, ticker)
	if err != nil {
		return fmt.Errorf("symbol %s not found: %w", ticker, err)
	}
	m.symbols = append(m.symbols, *sym)
	m.cache.Invalidate()
	m.logger.InfoContext(ctx, "universe_manual_add", "symbol", ticker)
	return nil
}

func (m *UniverseManager) RemoveManualOverride(ctx context.Context, ticker string) error {
	filtered := make([]db.Symbol, 0, len(m.symbols))
	for _, s := range m.symbols {
		if s.Ticker != ticker {
			filtered = append(filtered, s)
		}
	}
	m.symbols = filtered
	m.cache.Invalidate()
	m.logger.InfoContext(ctx, "universe_manual_remove", "symbol", ticker)
	return nil
}

func (m *UniverseManager) SchedulePeriodicRefresh(cronExpr string) (stop func(), err error) {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				ctx := context.Background()
				if err := m.Refresh(ctx); err != nil {
					m.logger.ErrorContext(ctx, "universe_scheduled_refresh_failed", "error", err)
				}
			}
		}
	}()

	stop = func() {
		close(done)
	}
	return stop, nil
}

func (m *UniverseManager) ConfigHash() string {
	return m.configHash
}

func (m *UniverseManager) ConfigID() string {
	return m.configID
}

// SyncFromBrokerAssets ingests broker-discovered assets as inactive symbols so
// they are available for universe mapping, without disturbing the canonical
// universe. Returns the number of newly-added symbols.
func (m *UniverseManager) SyncFromBrokerAssets(ctx context.Context, assets []broker.Asset) (int, error) {
	added := 0
	for _, a := range assets {
		if a.Symbol == "" || !a.Tradable {
			continue
		}
		inserted, err := m.repo.UpsertSymbolFromAsset(ctx, a.Symbol, a.Exchange, mapAssetClass(a.Class))
		if err != nil {
			return added, err
		}
		if inserted {
			added++
		}
	}
	m.cache.Invalidate()
	m.logger.InfoContext(ctx, "universe_broker_sync", "total", len(assets), "added", added)
	return added, nil
}

// mapAssetClass maps a broker asset class to the internal symbol asset type.
func mapAssetClass(class string) string {
	switch class {
	case "us_equity":
		return "equity"
	case "crypto":
		return "crypto"
	default:
		return "equity"
	}
}
