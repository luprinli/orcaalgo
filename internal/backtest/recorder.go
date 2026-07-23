package backtest

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/model"
)

type BacktestRecorder struct {
	states []model.TradingState
}

func NewBacktestRecorder() *BacktestRecorder {
	return &BacktestRecorder{
		states: make([]model.TradingState, 0, 10000),
	}
}

func (r *BacktestRecorder) Record(state *model.TradingState, orders []*model.Order) {
	r.states = append(r.states, *state)
}

func (r *BacktestRecorder) Flush() error {
	return nil
}

func (r *BacktestRecorder) ToCSV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("backtest recorder: create CSV: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{
		"timestamp", "balance", "position", "mid_price", "fee",
		"trading_volume", "trading_value", "num_trades",
	}); err != nil {
		return err
	}

	for _, s := range r.states {
		if err := w.Write([]string{
			s.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.6f", s.Balance),
			fmt.Sprintf("%.6f", s.Position),
			fmt.Sprintf("%d", s.MidPrice),
			fmt.Sprintf("%.6f", s.Fee),
			fmt.Sprintf("%.6f", s.TradingVolume),
			fmt.Sprintf("%.6f", s.TradingValue),
			fmt.Sprintf("%d", s.NumTrades),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *BacktestRecorder) States() []model.TradingState {
	return r.states
}
