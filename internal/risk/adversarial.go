package risk

const (
	RejectThreshold = 3
	FiveMinNS       = int64(300_000_000_000)
)

type AdversarialState struct {
	RejectCount        int
	RejectWindowStart  int64
	AvgTradeSize       uint64
	Last10Sizes        [10]uint64
	SizeIdx            int
	Last10Symbols      [10]uint32
	SymbolIdx          int
	Watchlist          [32]uint32
	WatchlistCount     int
	AfterHoursActivity bool
}

type AdversarialResult struct {
	Approve      bool
	TriggerKill  bool
	Reject       bool
	Reason       string
}

func CheckAdversarial(state *AdversarialState, quantity uint64, symbol uint32, now int64) AdversarialResult {
	if now-state.RejectWindowStart < FiveMinNS {
		if state.RejectCount > RejectThreshold {
			return AdversarialResult{TriggerKill: true, Reason: "reject_spike"}
		}
	}

	if state.AvgTradeSize > 0 && quantity > 2*state.AvgTradeSize {
		return AdversarialResult{Reject: true, Reason: "unusual_size"}
	}

	if state.WatchlistCount > 0 && !isWatchlisted(state, symbol) {
		return AdversarialResult{Reject: true, Reason: "unusual_symbol"}
	}

	if isAfterHours(now) {
		return AdversarialResult{Reject: true, Reason: "after_hours"}
	}

	state.SizeIdx = (state.SizeIdx + 1) % 10
	state.Last10Sizes[state.SizeIdx] = quantity
	updateAvgSize(state)

	state.SymbolIdx = (state.SymbolIdx + 1) % 10
	state.Last10Symbols[state.SymbolIdx] = symbol

	return AdversarialResult{Approve: true}
}

func updateAvgSize(state *AdversarialState) {
	var sum uint64
	count := 0
	for i := 0; i < 10; i++ {
		if state.Last10Sizes[i] > 0 {
			sum += state.Last10Sizes[i]
			count++
		}
	}
	if count > 0 {
		state.AvgTradeSize = sum / uint64(count)
	}
}

func isWatchlisted(state *AdversarialState, symbol uint32) bool {
	for i := 0; i < state.WatchlistCount; i++ {
		if state.Watchlist[i] == symbol {
			return true
		}
	}
	return false
}

func isAfterHours(timestampNS int64) bool {
	sec := timestampNS / 1_000_000_000
	hour := (sec / 3600) % 24
	weekday := ((sec / 86400) + 4) % 7
	if weekday >= 5 {
		return true
	}
	if hour < 9 || hour >= 16 {
		return true
	}
	return false
}

func RecordReject(state *AdversarialState, now int64) {
	if now-state.RejectWindowStart > FiveMinNS {
		state.RejectCount = 0
		state.RejectWindowStart = now
	}
	state.RejectCount++
}
