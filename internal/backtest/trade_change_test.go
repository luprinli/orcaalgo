package backtest

import (
	"testing"
	"time"
)

func TestTradeAddChange_AppendsInOrder(t *testing.T) {
	tr := &Trade{}
	ts1 := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 1, 16, 0, 0, 0, time.UTC)

	tr.addChange(ts1, "entry", "", "100.00", "BUY")
	tr.addChange(ts1, "stop", "", "98.00", "initial")
	tr.addChange(ts2, "stop", "98.00", "99.00", "trailing")
	tr.addChange(ts2, "exit", "100.00", "101.00", "take_profit")

	if len(tr.Changes) != 4 {
		t.Fatalf("expected 4 changes, got %d", len(tr.Changes))
	}
	if tr.Changes[0].Field != "entry" || tr.Changes[0].To != "100.00" {
		t.Errorf("first change should be entry: %+v", tr.Changes[0])
	}
	if tr.Changes[2].Field != "stop" || tr.Changes[2].From != "98.00" || tr.Changes[2].To != "99.00" {
		t.Errorf("trailing stop change wrong: %+v", tr.Changes[2])
	}
	if tr.Changes[3].Field != "exit" || tr.Changes[3].Reason != "take_profit" {
		t.Errorf("exit change wrong: %+v", tr.Changes[3])
	}
}
