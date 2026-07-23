package market

const (
	PriceScale = uint64(100_000)
)

type EventType uint8

const (
	EventDepthBid  EventType = iota + 1
	EventDepthAsk
	EventTrade
	EventBBO
	EventAggregate
)

type MarketEvent struct {
	Type     EventType
	ExchTS   int64
	LocalTS  int64
	Price    uint64
	Quantity uint64
	Flags    uint16
}
