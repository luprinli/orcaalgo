package risk

import (
	"fmt"
	"sync"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

type MultiAccountCapitalPool struct {
	mu    sync.RWMutex
	pools map[string]*CapitalPoolManager
}

func NewMultiAccountCapitalPool() *MultiAccountCapitalPool {
	return &MultiAccountCapitalPool{
		pools: make(map[string]*CapitalPoolManager),
	}
}

func (m *MultiAccountCapitalPool) RegisterPool(accountID string, profile *propfirm.Profile, state *propfirm.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pools[accountID] = NewCapitalPoolManagerWithAccount(accountID, profile, state)
}

func (m *MultiAccountCapitalPool) RegisterPoolDirect(accountID string, pool *CapitalPoolManager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pool.SetAccountID(accountID)
	m.pools[accountID] = pool
}

func (m *MultiAccountCapitalPool) GetPool(accountID string) (*CapitalPoolManager, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pool, ok := m.pools[accountID]
	if !ok {
		return nil, fmt.Errorf("capital pool for account %s not found", accountID)
	}
	return pool, nil
}

func (m *MultiAccountCapitalPool) RequestCapital(accountID string, req CapitalRequest) (CapitalResult, error) {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return CapitalResult{ApprovedSize: 0, Reason: "account_not_found"}, err
	}
	return pool.RequestCapital(req), nil
}

func (m *MultiAccountCapitalPool) RecordFill(accountID, strategyID, symbol, side string, pnl float64, quantity float64) error {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return err
	}
	pool.RecordFill(strategyID, symbol, side, pnl, quantity)
	return nil
}

func (m *MultiAccountCapitalPool) ResetDaily(accountID string) error {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return err
	}
	pool.ResetDaily()
	return nil
}

func (m *MultiAccountCapitalPool) ResetAllDaily() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, pool := range m.pools {
		pool.ResetDaily()
	}
}

func (m *MultiAccountCapitalPool) TotalBalance(accountID string) (float64, error) {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return 0, err
	}
	return pool.TotalBalance(), nil
}

func (m *MultiAccountCapitalPool) AggregateBalance() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var total float64
	for _, pool := range m.pools {
		total += pool.TotalBalance()
	}
	return total
}

func (m *MultiAccountCapitalPool) TotalExposure(accountID string) (float64, error) {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return 0, err
	}
	return pool.TotalExposure(), nil
}

func (m *MultiAccountCapitalPool) StrategyMetrics(accountID string) ([]StrategyAllocation, error) {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return nil, err
	}
	return pool.StrategyMetrics(), nil
}

func (m *MultiAccountCapitalPool) SetProfile(accountID string, profile *propfirm.Profile) error {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return err
	}
	pool.SetProfile(profile)
	return nil
}

func (m *MultiAccountCapitalPool) UpdateState(accountID string, state *propfirm.State) error {
	pool, err := m.GetPool(accountID)
	if err != nil {
		return err
	}
	pool.UpdateState(state)
	return nil
}

func (m *MultiAccountCapitalPool) AccountIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.pools))
	for id := range m.pools {
		ids = append(ids, id)
	}
	return ids
}

func (m *MultiAccountCapitalPool) PoolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pools)
}
