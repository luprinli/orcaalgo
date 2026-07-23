package ingest

import (
	"context"
	"log/slog"
	"sync"
)

type SubscriptionManager struct {
	registry       *Registry
	wsClient       *WSClient
	fixClient      *FIXClient
	currentSymbols map[string]bool
	mu             sync.RWMutex
	logger         *slog.Logger
}

func NewSubscriptionManager(reg *Registry, ws *WSClient, fix *FIXClient, logger *slog.Logger) *SubscriptionManager {
	return &SubscriptionManager{
		registry:       reg,
		wsClient:       ws,
		fixClient:      fix,
		currentSymbols: make(map[string]bool),
		logger:         logger,
	}
}

func (s *SubscriptionManager) SyncSubscriptions(ctx context.Context, desired []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	desiredMap := make(map[string]bool, len(desired))
	for _, d := range desired {
		desiredMap[d] = true
	}

	var toAdd []string
	var toRemove []string

	for ticker := range s.currentSymbols {
		if !desiredMap[ticker] {
			toRemove = append(toRemove, ticker)
		}
	}
	for ticker := range desiredMap {
		if !s.currentSymbols[ticker] {
			toAdd = append(toAdd, ticker)
		}
	}

	for _, ticker := range toRemove {
		delete(s.currentSymbols, ticker)
		s.logger.InfoContext(ctx, "subscription_removed", "symbol", ticker)
	}

	for _, ticker := range toAdd {
		s.currentSymbols[ticker] = true
		if s.wsClient != nil {
			s.wsClient.Subscribe(ticker)
		}
		s.logger.InfoContext(ctx, "subscription_added", "symbol", ticker)
	}

	return nil
}

func (s *SubscriptionManager) GetAllSubscribed() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0, len(s.currentSymbols))
	for ticker := range s.currentSymbols {
		result = append(result, ticker)
	}
	return result
}
