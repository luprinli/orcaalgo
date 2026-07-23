package backtest

import "context"

func (e *Engine) Start(ctx context.Context) error { return nil }

func (e *Engine) Pause() error { return nil }

func (e *Engine) Resume() error { return nil }

func (e *Engine) Stop() error { return nil }

func (e *Engine) IsRunning() bool { return false }

func (e *Engine) Health(ctx context.Context) error { return nil }
