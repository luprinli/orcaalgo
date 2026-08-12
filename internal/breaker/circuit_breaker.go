package breaker

import (
	"sync"
	"time"
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern with three states:
// Closed (normal operation), Open (rejects all requests), and HalfOpen
// (allows a limited number of trial requests to test recovery).
type CircuitBreaker struct {
	mu                 sync.Mutex
	state              CircuitState
	failureThreshold   int
	resetTimeout       time.Duration
	halfOpenMaxSuccess int
	failureCount       int
	successCount       int
	lastFailureTime    time.Time
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
	}
}

// Allow checks whether a request should be permitted based on the current
// circuit state. Returns false when the circuit is Open and the reset timeout
// has not elapsed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.successCount < cb.halfOpenMaxSuccess
	default:
		return false
	}
}

// RecordSuccess resets the failure count. In HalfOpen state, it increments
// the success counter and transitions back to Closed once the threshold is met.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxSuccess {
			cb.state = CircuitClosed
		}
	}
}

// RecordFailure increments the failure count. Opens the circuit immediately
// if in HalfOpen state, or once the failure threshold is reached while Closed.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()
	if cb.state == CircuitHalfOpen || cb.failureCount >= cb.failureThreshold {
		cb.state = CircuitOpen
	}
}
