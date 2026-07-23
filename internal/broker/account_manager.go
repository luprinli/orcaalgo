package broker

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/types"
)

type ManagedAccount struct {
	ID                string
	BrokerType        string
	Name              string
	PropFirmProfileID string
	Balance           types.Price
	Equity            types.Price
	DailyPnL          float64
	HighWaterMark     float64
	IsDefault         bool
	adapter           Adapter
}

func NewManagedAccount(id, brokerType, name string, adapter Adapter) *ManagedAccount {
	return &ManagedAccount{
		ID:         id,
		BrokerType: brokerType,
		Name:       name,
		adapter:    adapter,
	}
}

func (ma *ManagedAccount) SetBalance(balance, equity types.Price, dailyPnL, highWaterMark float64) {
	ma.Balance = balance
	ma.Equity = equity
	ma.DailyPnL = dailyPnL
	ma.HighWaterMark = highWaterMark
}

func (ma *ManagedAccount) Adapter() Adapter {
	return ma.adapter
}

func (ma *ManagedAccount) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	return ma.adapter.PlaceOrder(ctx, req)
}

func (ma *ManagedAccount) CancelOrder(ctx context.Context, orderID string) error {
	return ma.adapter.CancelOrder(ctx, orderID)
}

func (ma *ManagedAccount) CancelAllOrders(ctx context.Context) error {
	return ma.adapter.CancelAllOrders(ctx)
}

func (ma *ManagedAccount) CloseAllPositions(ctx context.Context) error {
	return ma.adapter.CloseAllPositions(ctx)
}

func (ma *ManagedAccount) GetPositions(ctx context.Context) ([]Position, error) {
	return ma.adapter.GetPositions(ctx)
}

func (ma *ManagedAccount) GetAccount(ctx context.Context) (*Account, error) {
	return ma.adapter.GetAccount(ctx)
}

func (ma *ManagedAccount) ValidateCredentials(ctx context.Context) error {
	return ma.adapter.ValidateCredentials(ctx)
}

func (ma *ManagedAccount) ToDBAccount() *db.Account {
	return &db.Account{
		ID:                ma.ID,
		BrokerType:        ma.BrokerType,
		Name:              ma.Name,
		PropFirmProfileID: ma.PropFirmProfileID,
		Balance:           ma.Balance.Float64(),
		Equity:            ma.Equity.Float64(),
		DailyPnL:          ma.DailyPnL,
		HighWaterMark:     ma.HighWaterMark,
		IsDefault:         ma.IsDefault,
	}
}

func (ma *ManagedAccount) ApplyFromDBAccount(a *db.Account) {
	ma.ID = a.ID
	ma.BrokerType = a.BrokerType
	ma.Name = a.Name
	ma.PropFirmProfileID = a.PropFirmProfileID
	ma.Balance = types.FromFloat64(a.Balance)
	ma.Equity = types.FromFloat64(a.Equity)
	ma.DailyPnL = a.DailyPnL
	ma.HighWaterMark = a.HighWaterMark
	ma.IsDefault = a.IsDefault
}

type AccountManager struct {
	mu       sync.RWMutex
	accounts map[string]*ManagedAccount
	repo     *db.Repository
	registry *BrokerDriverRegistry
}

func NewAccountManager(repo *db.Repository, registry *BrokerDriverRegistry) *AccountManager {
	return &AccountManager{
		accounts: make(map[string]*ManagedAccount),
		repo:     repo,
		registry: registry,
	}
}

func (am *AccountManager) RegisterAccount(acct *ManagedAccount) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.accounts[acct.ID]; exists {
		return fmt.Errorf("account %s already registered", acct.ID)
	}

	// Register the adapter into the broker driver registry.
	// Non-CapableAdapter implementations are skipped — they can't
	// participate in capability-based routing.
	if ca, ok := acct.adapter.(CapableAdapter); ok {
		m := ca.Manifest()
		m.ID = acct.BrokerType + ":" + acct.ID
		if err := am.registry.Register(ca); err != nil {
			return fmt.Errorf("register adapter for account %s: %w", acct.ID, err)
		}
	}

	am.accounts[acct.ID] = acct
	return nil
}

func (am *AccountManager) UnregisterAccount(id string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	acct, ok := am.accounts[id]
	if !ok {
		return
	}

	registryKey := acct.BrokerType + ":" + id
	am.registry.Unregister(registryKey)
	delete(am.accounts, id)
}

func (am *AccountManager) GetAccount(id string) (*ManagedAccount, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	acct, ok := am.accounts[id]
	if !ok {
		return nil, fmt.Errorf("account %s not found", id)
	}
	return acct, nil
}

func (am *AccountManager) GetDefaultAccount() (*ManagedAccount, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, acct := range am.accounts {
		if acct.IsDefault {
			return acct, nil
		}
	}
	if len(am.accounts) == 1 {
		for _, acct := range am.accounts {
			return acct, nil
		}
	}
	return nil, fmt.Errorf("no default account found")
}

func (am *AccountManager) GetDefaultAccountID() string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for id, acct := range am.accounts {
		if acct.IsDefault {
			return id
		}
	}
	if len(am.accounts) == 1 {
		for id := range am.accounts {
			return id
		}
	}
	return ""
}

func (am *AccountManager) ListAccountsByUser(ctx context.Context, userID string) []*ManagedAccount {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if userID == "" {
		result := make([]*ManagedAccount, 0, len(am.accounts))
		for _, acct := range am.accounts {
			result = append(result, acct)
		}
		return result
	}

	if am.repo != nil {
		dbAccounts, err := am.repo.ListAccountsByUser(ctx, userID)
		if err == nil {
			result := make([]*ManagedAccount, 0, len(dbAccounts))
			for _, dba := range dbAccounts {
				if acct, ok := am.accounts[dba.ID]; ok {
					acct.ApplyFromDBAccount(&dba)
					result = append(result, acct)
				}
			}
			return result
		}
	}

	result := make([]*ManagedAccount, 0)
	for _, acct := range am.accounts {
		result = append(result, acct)
	}
	return result
}

func (am *AccountManager) GetAdapter(accountID string) (Adapter, error) {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	return acct.Adapter(), nil
}

func (am *AccountManager) GetDefaultAdapter() (Adapter, error) {
	acct, err := am.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	return acct.Adapter(), nil
}

func (am *AccountManager) PlaceOrder(ctx context.Context, accountID string, req *OrderRequest) (*OrderResponse, error) {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	return acct.PlaceOrder(ctx, req)
}

func (am *AccountManager) CancelOrder(ctx context.Context, accountID, orderID string) error {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return err
	}
	return acct.CancelOrder(ctx, orderID)
}

func (am *AccountManager) CancelAllOrders(ctx context.Context, accountID string) error {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return err
	}
	return acct.CancelAllOrders(ctx)
}

func (am *AccountManager) CloseAllPositions(ctx context.Context, accountID string) error {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return err
	}
	return acct.CloseAllPositions(ctx)
}

func (am *AccountManager) GetPositions(ctx context.Context, accountID string) ([]Position, error) {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	return acct.GetPositions(ctx)
}

func (am *AccountManager) GetAccountInfo(ctx context.Context, accountID string) (*Account, error) {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	return acct.GetAccount(ctx)
}

func (am *AccountManager) SyncAccountState(ctx context.Context, accountID string) error {
	acct, err := am.GetAccount(accountID)
	if err != nil {
		return err
	}

	brokerAcct, err := acct.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("sync account %s: %w", accountID, err)
	}

	am.mu.Lock()
	acct.Balance = brokerAcct.Balance
	acct.Equity = brokerAcct.Equity
	acct.DailyPnL = brokerAcct.DailyPL
	if brokerAcct.Equity.Float64() > acct.HighWaterMark {
		acct.HighWaterMark = brokerAcct.Equity.Float64()
	}
	am.mu.Unlock()

	if am.repo != nil {
		if err := am.repo.UpsertAccountBalance(ctx, accountID, acct.Balance.Float64(), acct.Equity.Float64(), acct.DailyPnL, acct.HighWaterMark); err != nil {
			log.Printf("account_manager: failed to persist account %s balance: %v", accountID, err)
		}
	}

	positions, err := acct.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("sync positions for account %s: %w", accountID, err)
	}

	if am.repo != nil {
		for _, pos := range positions {
			ap := &db.AccountPosition{
				AccountID:     accountID,
				Symbol:        pos.Symbol,
				Quantity:      pos.Quantity,
				AvgEntryPrice: pos.AvgEntryPrice.Float64(),
				MarketValue:   pos.MarketValue.Float64(),
				UnrealizedPL:  pos.UnrealizedPL,
			}
			if err := am.repo.UpsertAccountPosition(ctx, ap); err != nil {
				log.Printf("account_manager: failed to persist position for account %s: %v", accountID, err)
			}
		}
	}

	return nil
}

func (am *AccountManager) SyncAll(ctx context.Context) {
	for _, acct := range am.ListAccountsByUser(ctx, "") {
		if err := am.SyncAccountState(ctx, acct.ID); err != nil {
			log.Printf("account_manager: sync failed for account %s: %v", acct.ID, err)
		}
	}
}

func (am *AccountManager) CloseAllPositionsAcrossAll(ctx context.Context) {
	for _, acct := range am.ListAccountsByUser(ctx, "") {
		if err := acct.CloseAllPositions(ctx); err != nil {
			log.Printf("account_manager: CloseAllPositions failed for account %s: %v", acct.ID, err)
		}
	}
}

func (am *AccountManager) CancelAllOrdersAcrossAll(ctx context.Context) {
	for _, acct := range am.ListAccountsByUser(ctx, "") {
		if err := acct.CancelAllOrders(ctx); err != nil {
			log.Printf("account_manager: CancelAllOrders failed for account %s: %v", acct.ID, err)
		}
	}
}

func (am *AccountManager) AccountCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.accounts)
}

func (am *AccountManager) SetDefaultAccount(ctx context.Context, id string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	acct, ok := am.accounts[id]
	if !ok {
		return fmt.Errorf("account %s not found", id)
	}

	for _, a := range am.accounts {
		a.IsDefault = false
	}
	acct.IsDefault = true

	if am.repo != nil {
		return am.repo.SetDefaultAccount(ctx, id)
	}
	return nil
}
