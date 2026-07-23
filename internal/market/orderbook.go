package market

const MaxLevels = 32

type PriceLevel struct {
	Price    uint64
	Quantity uint64
	Orders   int
}

type OrderBook struct {
	SymbolID uint32
	Bids     [MaxLevels]PriceLevel
	Asks     [MaxLevels]PriceLevel
	BidCount int
	AskCount int
	VWAP     float64
	TotalVol uint64
}

func NewOrderBook(symbolID uint32) *OrderBook {
	return &OrderBook{SymbolID: symbolID}
}

func (ob *OrderBook) UpdateBid(index int, price, quantity uint64) {
	if index >= 0 && index < MaxLevels {
		ob.Bids[index] = PriceLevel{Price: price, Quantity: quantity, Orders: 1}
		if index+1 > ob.BidCount {
			ob.BidCount = index + 1
		}
	}
}

func (ob *OrderBook) UpdateAsk(index int, price, quantity uint64) {
	if index >= 0 && index < MaxLevels {
		ob.Asks[index] = PriceLevel{Price: price, Quantity: quantity, Orders: 1}
		if index+1 > ob.AskCount {
			ob.AskCount = index + 1
		}
	}
}

func (ob *OrderBook) BestBid() (uint64, uint64) {
	if ob.BidCount == 0 {
		return 0, 0
	}
	best := ob.Bids[0]
	return best.Price, best.Quantity
}

func (ob *OrderBook) BestAsk() (uint64, uint64) {
	if ob.AskCount == 0 {
		return 0, 0
	}
	best := ob.Asks[0]
	return best.Price, best.Quantity
}

func (ob *OrderBook) MidPrice() uint64 {
	bidPrice, _ := ob.BestBid()
	askPrice, _ := ob.BestAsk()
	if bidPrice == 0 || askPrice == 0 {
		return 0
	}
	return (bidPrice + askPrice) / 2
}

func (ob *OrderBook) Spread() uint64 {
	bid, _ := ob.BestBid()
	ask, _ := ob.BestAsk()
	if bid == 0 || ask == 0 {
		return 0
	}
	return ask - bid
}

func (ob *OrderBook) ImbalanceRatio() float64 {
	var bidTotal, askTotal uint64
	for i := 0; i < ob.BidCount; i++ {
		bidTotal += ob.Bids[i].Quantity
	}
	for i := 0; i < ob.AskCount; i++ {
		askTotal += ob.Asks[i].Quantity
	}
	total := bidTotal + askTotal
	if total == 0 {
		return 0.5
	}
	return float64(bidTotal) / float64(total)
}

func (ob *OrderBook) Reset() {
	ob.BidCount = 0
	ob.AskCount = 0
	ob.VWAP = 0
	ob.TotalVol = 0
	for i := range ob.Bids {
		ob.Bids[i] = PriceLevel{}
		ob.Asks[i] = PriceLevel{}
	}
}
