package risk

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrAlreadyHalted   = errors.New("kill switch already halted")
	ErrLocked          = errors.New("kill switch re-entrancy guard locked")
	ErrAlreadyInFlight = errors.New("kill switch already in flight")
)

// KillSwitch implements Hard Prohibition #8: re-entrancy guard with triple-check pattern.
// Architecture:
//   1. isLocked (atomic CAS) — prevents concurrent Trigger() calls
//   2. killSwitchReady (atomic) — 1=ready to fire, 0=already executing
//   3. halted (atomic) — prevents re-triggering when already halted
// Multi-account: if accountCancel is set, CloseAllPositions is called on each account.
type KillSwitch struct {
	halted           int32
	isLocked         atomic.Int32
	killSwitchReady  atomic.Int32
	lastTrigger      time.Time
	reason           string
	mu               sync.RWMutex
	broker           BrokerCancel
	onTrigger        []func(reason string, time time.Time)
	accountCancel    AccountCanceller
}

type AccountCanceller interface {
	CancelAllOrdersAcrossAll(ctx context.Context)
	CloseAllPositionsAcrossAll(ctx context.Context)
}

type BrokerCancel interface {
	CancelAllOrders(ctx context.Context) error
	CloseAllPositions(ctx context.Context) error
}

func NewKillSwitch(broker BrokerCancel) *KillSwitch {
	ks := &KillSwitch{
		broker: broker,
	}
	ks.killSwitchReady.Store(1)
	return ks
}

func (ks *KillSwitch) SetAccountCanceller(ac AccountCanceller) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.accountCancel = ac
}

func (ks *KillSwitch) Trigger(reason string) error {
	if atomic.LoadInt32(&ks.halted) == 1 {
		return ErrAlreadyHalted
	}
	if !ks.isLocked.CompareAndSwap(0, 1) {
		return ErrLocked
	}
	defer ks.isLocked.Store(0)

	if ks.killSwitchReady.Load() == 0 {
		return ErrAlreadyInFlight
	}
	ks.killSwitchReady.Store(0)

	ks.mu.Lock()
	ks.reason = reason
	ks.lastTrigger = time.Now()
	ks.mu.Unlock()

	atomic.StoreInt32(&ks.halted, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if ks.accountCancel != nil {
		ks.accountCancel.CancelAllOrdersAcrossAll(ctx)
		ks.accountCancel.CloseAllPositionsAcrossAll(ctx)
	} else {
		if err := ks.broker.CancelAllOrders(ctx); err != nil {
			log.Printf("kill switch: failed to cancel orders: %v", err)
		}
		if err := ks.broker.CloseAllPositions(ctx); err != nil {
			log.Printf("kill switch: failed to close positions: %v", err)
		}
	}

	ks.mu.RLock()
	callbacks := make([]func(reason string, time time.Time), len(ks.onTrigger))
	copy(callbacks, ks.onTrigger)
	triggerTime := ks.lastTrigger
	ks.mu.RUnlock()

	for _, fn := range callbacks {
		fn(reason, triggerTime)
	}

	ks.killSwitchReady.Store(1)
	return nil
}

func (ks *KillSwitch) Resume() {
	atomic.StoreInt32(&ks.halted, 0)
	ks.mu.Lock()
	ks.reason = ""
	ks.mu.Unlock()
}

func (ks *KillSwitch) IsHalted() bool {
	return atomic.LoadInt32(&ks.halted) == 1
}

func (ks *KillSwitch) IsInFlight() bool {
	return ks.killSwitchReady.Load() == 0
}

func (ks *KillSwitch) IsFlightReady() bool {
	return ks.killSwitchReady.Load() == 1
}

func (ks *KillSwitch) IsLocked() bool {
	return ks.isLocked.Load() == 1
}

func (ks *KillSwitch) Status() (bool, string, time.Time) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.IsHalted(), ks.reason, ks.lastTrigger
}

func (ks *KillSwitch) OnTrigger(fn func(reason string, time time.Time)) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.onTrigger = append(ks.onTrigger, fn)
}

func (ks *KillSwitch) MonitorRejectRate(ctx context.Context, checkInterval time.Duration, rejectChan <-chan int) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	var rejectCount int
	windowStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case count := <-rejectChan:
			if time.Since(windowStart) > 5*time.Minute {
				rejectCount = 0
				windowStart = time.Now()
			}
			rejectCount += count
			if rejectCount > 3 {
				_ = ks.Trigger("reject_spike")
			}
		case <-ticker.C:
			if time.Since(windowStart) > 5*time.Minute {
				rejectCount = 0
				windowStart = time.Now()
			}
		}
	}
}
