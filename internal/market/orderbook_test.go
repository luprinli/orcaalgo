package market

import "testing"

func TestOrderBook_BestBidAsk(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 500)
	ob.UpdateAsk(0, 10050, 300)

	bidPrice, bidQty := ob.BestBid()
	if bidPrice != 10000 || bidQty != 500 {
		t.Errorf("expected bid 10000@500, got %d@%d", bidPrice, bidQty)
	}

	askPrice, askQty := ob.BestAsk()
	if askPrice != 10050 || askQty != 300 {
		t.Errorf("expected ask 10050@300, got %d@%d", askPrice, askQty)
	}
}

func TestOrderBook_MidPrice(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 500)
	ob.UpdateAsk(0, 10050, 300)

	mid := ob.MidPrice()
	if mid != 10025 {
		t.Errorf("expected mid 10025, got %d", mid)
	}
}

func TestOrderBook_Spread(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 500)
	ob.UpdateAsk(0, 10050, 300)

	spread := ob.Spread()
	if spread != 50 {
		t.Errorf("expected spread 50, got %d", spread)
	}
}

func TestOrderBook_Empty(t *testing.T) {
	ob := NewOrderBook(1)

	bid, _ := ob.BestBid()
	ask, _ := ob.BestAsk()
	if bid != 0 || ask != 0 {
		t.Error("empty book should return 0")
	}
	if ob.MidPrice() != 0 {
		t.Error("empty book mid should be 0")
	}
	if ob.Spread() != 0 {
		t.Error("empty book spread should be 0")
	}
}

func TestOrderBook_ImbalanceRatio(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 600)
	ob.UpdateAsk(0, 10050, 400)

	ratio := ob.ImbalanceRatio()
	if ratio < 0.59 || ratio > 0.61 {
		t.Errorf("expected imbalance ~0.6, got %.4f", ratio)
	}
}

func TestOrderBook_ImbalanceEqual(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 500)
	ob.UpdateAsk(0, 10050, 500)

	ratio := ob.ImbalanceRatio()
	if ratio != 0.5 {
		t.Errorf("expected 0.5, got %.4f", ratio)
	}
}

func TestOrderBook_ImbalanceEmpty(t *testing.T) {
	ob := NewOrderBook(1)
	ratio := ob.ImbalanceRatio()
	if ratio != 0.5 {
		t.Errorf("expected 0.5 for empty book, got %.4f", ratio)
	}
}

func TestOrderBook_MultipleLevels(t *testing.T) {
	ob := NewOrderBook(1)
	for i := 0; i < 5; i++ {
		ob.UpdateBid(i, uint64(10000-i*10), 100)
		ob.UpdateAsk(i, uint64(10050+i*10), 100)
	}

	if ob.BidCount != 5 || ob.AskCount != 5 {
		t.Errorf("expected 5 levels, got bid=%d ask=%d", ob.BidCount, ob.AskCount)
	}

	bid, _ := ob.BestBid()
	ask, _ := ob.BestAsk()
	if bid != 10000 || ask != 10050 {
		t.Errorf("best prices should be 10000/10050, got %d/%d", bid, ask)
	}
}

func TestOrderBook_Reset(t *testing.T) {
	ob := NewOrderBook(1)
	ob.UpdateBid(0, 10000, 500)
	ob.UpdateAsk(0, 10050, 300)

	ob.Reset()
	if ob.BidCount != 0 || ob.AskCount != 0 {
		t.Error("reset should clear levels")
	}
}
