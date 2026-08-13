package backtest

import "time"

// TradeChange is a single append-only mutation to a trade's lifecycle state. It
// powers the trade drill-down view: each entry records what field changed, its
// previous and new values, and the reason, so a trader can replay exactly how a
// position evolved (entry → stop/target set → trailing updates → exit).
type TradeChange struct {
	Timestamp time.Time `json:"timestamp"`
	Field     string    `json:"field"`
	From      string    `json:"from,omitempty"`
	To        string    `json:"to,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// addChange appends a lifecycle change to the trade. It is the single place a
// change is recorded, so the change log is append-only and consistent across
// every engine code path that mutates a trade.
func (t *Trade) addChange(ts time.Time, field, from, to, reason string) {
	t.Changes = append(t.Changes, TradeChange{
		Timestamp: ts,
		Field:     field,
		From:      from,
		To:        to,
		Reason:    reason,
	})
}
