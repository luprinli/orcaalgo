package temporal

import (
	"context"
	"testing"
	"time"
)

func TestWithTemporalContext_PropagatesNow(t *testing.T) {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	tc := NewBacktest(now)
	ctx := WithTemporalContext(context.Background(), tc)

	got, ok := GetTemporalContext(ctx)
	if !ok {
		t.Fatal("expected temporal context to be present")
	}
	if !got.Now.Equal(now) {
		t.Errorf("expected now=%v, got %v", now, got.Now)
	}
	if got.IsLive {
		t.Error("expected IsLive=false for backtest context")
	}
}

func TestGetTemporalContext_Missing(t *testing.T) {
	_, ok := GetTemporalContext(context.Background())
	if ok {
		t.Error("expected no temporal context in plain context")
	}
}

func TestGetMaxTime_Present(t *testing.T) {
	now := time.Now()
	ctx := WithTemporalContext(context.Background(), NewLive())
	got, ok := GetMaxTime(ctx)
	if !ok {
		t.Fatal("expected max time")
	}
	if got.Before(now) {
		t.Errorf("got %v, expected at least %v", got, now)
	}
}

func TestIsLive(t *testing.T) {
	ctx := WithTemporalContext(context.Background(), NewLive())
	if !IsLive(ctx) {
		t.Error("expected live context")
	}
	ctx2 := WithTemporalContext(context.Background(), NewBacktest(time.Now()))
	if IsLive(ctx2) {
		t.Error("expected non-live context")
	}
}
