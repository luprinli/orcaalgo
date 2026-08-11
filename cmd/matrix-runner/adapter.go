package main

import (
	"context"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
)

type repoAdapter struct{ repo *db.Repository }

func (a *repoAdapter) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]backtest.Candle, error) {
	c, err := a.repo.LoadCandles(ctx, symbols, start, end)
	return convertCandles(c), err
}
func (a *repoAdapter) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]backtest.Candle, error) {
	return a.LoadCandles(ctx, symbols, start, end)
}
func (a *repoAdapter) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, tf string) ([][]backtest.Candle, error) {
	c, err := a.repo.LoadCandlesByTimeframe(ctx, symbols, start, end, tf)
	return convertCandles(c), err
}
func (a *repoAdapter) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, tf, source string) ([][]backtest.Candle, error) {
	return a.LoadCandlesTF(ctx, symbols, start, end, tf)
}
func (a *repoAdapter) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, tf string) (map[string][]backtest.Candle, error) {
	return nil, nil
}
func (a *repoAdapter) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]backtest.RegimeLog, error) {
	logs, err := a.repo.LoadRegimeLogs(ctx, start, end)
	if err != nil { return nil, nil }
	out := make([]backtest.RegimeLog, len(logs))
	for i, l := range logs {
		out[i] = backtest.RegimeLog{Time: l.Time, HMMState: l.HMMState, Confidence: l.Confidence, Symbol: l.Symbol}
	}
	return out, nil
}
func (a *repoAdapter) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]backtest.VIXLog, error) {
	logs, err := a.repo.LoadVIXLogs(ctx, start, end)
	if err != nil { return nil, nil }
	out := make([]backtest.VIXLog, len(logs))
	for i, l := range logs {
		out[i] = backtest.VIXLog{Time: l.Time, VIXValue: l.VIXValue, VIXChange: l.VIXChange}
	}
	return out, nil
}
func (a *repoAdapter) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]backtest.SentimentLog, error) {
	return nil, nil
}
func (a *repoAdapter) CountCandles(context.Context) (int64, error)         { return 0, nil }
func (a *repoAdapter) CountSyntheticCandles(context.Context) (int64, error) { return 0, nil }
func (a *repoAdapter) CountRegimeLogs(context.Context) (int64, error)       { return 0, nil }
func (a *repoAdapter) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]backtest.UniverseSnapshot, error) {
	return nil, nil
}

func convertCandles(c [][]db.Candle) [][]backtest.Candle {
	out := make([][]backtest.Candle, len(c))
	for i, row := range c {
		r := make([]backtest.Candle, len(row))
		for j, candle := range row {
			r[j] = backtest.Candle{
				Symbol: candle.Symbol, Time: candle.Time,
				Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close,
				Volume: candle.Volume,
			}
		}
		out[i] = r
	}
	return out
}
