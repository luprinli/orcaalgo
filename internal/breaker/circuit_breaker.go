package breaker

import (
	"sync"
	"time"
)

var (
	globalObserverMu sync.Mutex
	globalObserver   func(BreakerEvent)
)

// SetGlobalObserver registers a process-wide observer that is attached to every
// CircuitBreaker created after this call. It is the hook used at startup to
// route breaker transitions into the audit log. May be nil to disable.
func SetGlobalObserver(observer func(BreakerEvent)) {
	globalObserverMu.Lock()
	globalObserver = observer
	globalObserverMu.Unlock()
}

func currentGlobalObserver() func(BreakerEvent) {
	globalObserverMu.Lock()
	defer globalObserverMu.Unlock()
	return globalObserver
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// BreakerEvent records a circuit state transition (or warning) for the audit
// trail (R7). Emitted via an optional observer; observers must not re-enter the
// breaker (they are invoked outside the breaker lock).
type BreakerEvent struct {
	From   CircuitState `json:"from"`
	To     CircuitState `json:"to"`
	Warn   bool         `json:"warn"`
	Reason string       `json:"reason"`
}

// CircuitBreaker implements the circuit breaker pattern with three states:
// Closed (normal operation), Open (rejects all requests), and HalfOpen
// (allows a limited number of trial requests to test recovery).
type CircuitBreaker struct {
	mu                 sync.Mutex
	state              CircuitState
	failureThreshold   int
	warnThreshold      int
	resetTimeout       time.Duration
	halfOpenMaxSuccess int
	failureCount       int
	successCount       int
	lastFailureTime    time.Time
	warnEmitted        bool
	observer           func(BreakerEvent)
}

// NewCircuitBreaker creates a new circuit breaker that opens after
// failureThreshold consecutive failures and waits resetTimeout before
// transitioning to HalfOpen.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:              CircuitClosed,
		failureThreshold:   failureThreshold,
		resetTimeout:       resetTimeout,
		halfOpenMaxSuccess: 1,
		observer:           currentGlobalObserver(),
	}
}

// SetWarnThreshold sets the consecutive-failure count at which a warning event
// is emitted (before the breaker hard-opens at failureThreshold). Zero disables
// warnings.
func (cb *CircuitBreaker) SetWarnThreshold(threshold int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.warnThreshold = threshold
}

// SetObserver registers a callback invoked on every state transition and
// warning. It may be nil to disable observation.
func (cb *CircuitBreaker) SetObserver(observer func(BreakerEvent)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.observer = observer
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Allow checks whether a request should be permitted based on the current
// circuit state. Returns false when the circuit is Open and the reset timeout
// has not elapsed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	state := cb.state
	switch state {
	case CircuitClosed:
		cb.mu.Unlock()
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.notify(BreakerEvent{From: CircuitOpen, To: CircuitHalfOpen, Reason: "reset_timeout"})
			return true
		}
		cb.mu.Unlock()
		return false
	case CircuitHalfOpen:
		allow := cb.successCount < cb.halfOpenMaxSuccess
		cb.mu.Unlock()
		return allow
	default:
		cb.mu.Unlock()
		return false
	}
}

// RecordSuccess resets the failure count. In HalfOpen state, it increments
// the success counter and transitions back to Closed once the threshold is met.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	from := cb.state
	cb.failureCount = 0
	cb.warnEmitted = false
	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxSuccess {
			cb.state = CircuitClosed
		}
	}
	to := cb.state
	cb.mu.Unlock()

	if from == CircuitHalfOpen && to == CircuitClosed {
		cb.notify(BreakerEvent{From: from, To: to, Reason: "recovered"})
	}
}

// RecordFailure increments the failure count. Opens the circuit immediately
// if in HalfOpen state, or once the failure threshold is reached while Closed.
// A warning is emitted when the failure count first reaches warnThreshold.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	from := cb.state
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	var event *BreakerEvent
	if cb.state == CircuitHalfOpen || cb.failureCount >= cb.failureThreshold {
		cb.state = CircuitOpen
		event = &BreakerEvent{From: from, To: CircuitOpen, Reason: "failure_threshold"}
	} else if cb.warnThreshold > 0 && cb.failureCount >= cb.warnThreshold && !cb.warnEmitted {
		cb.warnEmitted = true
		event = &BreakerEvent{From: from, To: from, Warn: true, Reason: "warn_threshold"}
	}
	cb.mu.Unlock()

	if event != nil {
		cb.notify(*event)
	}
}

func (cb *CircuitBreaker) notify(event BreakerEvent) {
	cb.mu.Lock()
	observer := cb.observer
	cb.mu.Unlock()
	if observer != nil {
		observer(event)
	}
}
