package engine

import (
	"context"
	"fmt"
)

type Runner interface {
	Start(ctx context.Context) error
	Stop() error
	Pause() error
	IsRunning() bool
	Health(ctx context.Context) error
}

func (e *LiveEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Running {
		return fmt.Errorf("live engine: already running")
	}
	e.Running = true
	e.Halted = false
	return nil
}

func (e *LiveEngine) Pause() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Halted = true
	return nil
}

func (e *LiveEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Running = false
	return nil
}

func (e *LiveEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Running && !e.Halted
}

func (e *LiveEngine) Health(ctx context.Context) error {
	if !e.Running {
		return fmt.Errorf("live engine: not running")
	}
	if e.Halted {
		return fmt.Errorf("live engine: halted")
	}
	return nil
}

var _ Runner = (*LiveEngine)(nil)
