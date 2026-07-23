package risk

import "testing"

func TestAdversarialRejectSpike(t *testing.T) {
	state := &AdversarialState{}
	now := int64(1_000_000_000_000)

	for i := 0; i < 6; i++ {
		RecordReject(state, now)
	}
	result := CheckAdversarial(state, 100, 1, now)
	if !result.TriggerKill {
		t.Error("should trigger kill on reject spike")
	}
}

func TestAdversarialNormalSize(t *testing.T) {
	state := &AdversarialState{}
	now := int64(10 * 3600 * 1_000_000_000)

	for i := 0; i < 10; i++ {
		state.Last10Sizes[i] = 100
	}
	updateAvgSize(state)

	result := CheckAdversarial(state, 150, 1, now)
	if result.Reject {
		t.Error("should not reject normal size after building average")
	}
	if state.AvgTradeSize == 0 {
		t.Error("avg trade size should be non-zero")
	}
}

func TestAdversarialUnusualSize(t *testing.T) {
	state := &AdversarialState{}
	state.AvgTradeSize = 100
	now := int64(10 * 3600 * 1_000_000_000)

	result := CheckAdversarial(state, 250, 1, now)
	if !result.Reject {
		t.Error(">2x avg should be rejected as unusual_size")
	}
}

func TestAdversarialWatchlist(t *testing.T) {
	state := &AdversarialState{}
	state.WatchlistCount = 2
	state.Watchlist[0] = 1
	state.Watchlist[1] = 2
	now := int64(10 * 3600 * 1_000_000_000)

	result := CheckAdversarial(state, 100, 3, now)
	if !result.Reject || result.Reason != "unusual_symbol" {
		t.Error("should reject non-watchlisted symbol")
	}

	state2 := &AdversarialState{}
	state2.WatchlistCount = 1
	state2.Watchlist[0] = 1

	result2 := CheckAdversarial(state2, 100, 1, now)
	if result2.Reject {
		t.Error("should approve watchlisted symbol")
	}
}

func TestAdversarialAfterHours(t *testing.T) {
	if !isAfterHours(2 * 3600 * 1_000_000_000) {
		t.Error("2 AM UTC should be after hours (<9 AM)")
	}
	if !isAfterHours(18 * 3600 * 1_000_000_000) {
		t.Error("6 PM UTC should be after hours (>=4 PM)")
	}
	if isAfterHours(10 * 3600 * 1_000_000_000) {
		t.Error("10 AM Monday UTC should be during hours")
	}
}

func TestAdversarialApprove(t *testing.T) {
	state := &AdversarialState{}
	now := int64(10 * 3600 * 1_000_000_000)

	result := CheckAdversarial(state, 100, 1, now)
	if !result.Approve {
		t.Error("should approve normal trade")
	}
	if state.SizeIdx != 1 {
		t.Errorf("size idx should increment to 1, got %d", state.SizeIdx)
	}
}
