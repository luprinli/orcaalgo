package temporal

import (
	"context"
	"time"
)

type contextKey string

const temporalKey contextKey = "orca_temporal"

type TemporalContext struct {
	Now     time.Time
	IsLive  bool
	IsTrain bool
}

func WithTemporalContext(ctx context.Context, tc TemporalContext) context.Context {
	return context.WithValue(ctx, temporalKey, tc)
}

func GetTemporalContext(ctx context.Context) (TemporalContext, bool) {
	v := ctx.Value(temporalKey)
	if v == nil {
		return TemporalContext{}, false
	}
	tc, ok := v.(TemporalContext)
	return tc, ok
}

func GetMaxTime(ctx context.Context) (time.Time, bool) {
	tc, ok := GetTemporalContext(ctx)
	if !ok {
		return time.Time{}, false
	}
	return tc.Now, true
}

func IsLive(ctx context.Context) bool {
	tc, ok := GetTemporalContext(ctx)
	return ok && tc.IsLive
}

func NewLive() TemporalContext {
	return TemporalContext{
		Now:     time.Now(),
		IsLive:  true,
		IsTrain: false,
	}
}

func NewBacktest(now time.Time) TemporalContext {
	return TemporalContext{
		Now:     now,
		IsLive:  false,
		IsTrain: true,
	}
}

func NewTrain(now time.Time) TemporalContext {
	return TemporalContext{
		Now:     now,
		IsLive:  false,
		IsTrain: true,
	}
}
