package ingest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lee-econ/orca-core/internal/types"
)

type WSClient struct {
	url       string
	conn      *websocket.Conn
	ringBuf   *RingBuffer
	done      chan struct{}
	mu        sync.Mutex
	reconnect time.Duration
	subs      []string
	symbolMap map[string]uint32
	authHeader http.Header
}

type TickMessage struct {
	Symbol   string      `json:"symbol"`
	Price    types.Price `json:"price"`
	BidPrice types.Price `json:"bid_price"`
	AskPrice types.Price `json:"ask_price"`
	Volume   float64     `json:"volume"`
	BidSize  float64     `json:"bid_size"`
	AskSize  float64     `json:"ask_size"`
	Side     string      `json:"side"`
	Time     int64       `json:"timestamp"`
}

func (t *TickMessage) UnmarshalJSON(data []byte) error {
	type alias TickMessage
	var obj alias
	if err := json.Unmarshal(data, &obj); err == nil && obj.Symbol != "" {
		*t = TickMessage(obj)
		return nil
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) >= 2 {
		if s, ok := arr[0].(string); ok {
			t.Symbol = s
		}
		if v, ok := arr[1].(float64); ok {
			t.Price = types.FromFloat64(v)
		}
	}
	if len(arr) >= 3 {
		if v, ok := arr[2].(float64); ok {
			t.Volume = v
		}
	}
	if len(arr) >= 4 {
		if v, ok := arr[3].(float64); ok {
			t.BidPrice = types.FromFloat64(v)
		}
	}
	if len(arr) >= 5 {
		if v, ok := arr[4].(float64); ok {
			t.AskPrice = types.FromFloat64(v)
		}
	}
	if len(arr) >= 6 {
		if v, ok := arr[5].(float64); ok {
			t.Time = int64(v)
		}
	}
	return nil
}

const PRICE_SCALE = 100_000

func NewWSClient(url string, ringBuf *RingBuffer) *WSClient {
	return &WSClient{
		url:       url,
		ringBuf:   ringBuf,
		done:      make(chan struct{}),
		reconnect: time.Second,
		symbolMap: make(map[string]uint32),
	}
}

func (c *WSClient) RegisterSymbol(ticker string, id uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.symbolMap[ticker] = id
}

func (c *WSClient) Subscribe(symbols ...string) {
	c.subs = append(c.subs, symbols...)
}

func (c *WSClient) SetAuth(apiKey, apiSecret string) {
	c.authHeader = http.Header{
		"APCA-API-KEY-ID":     {apiKey},
		"APCA-API-SECRET-KEY": {apiSecret},
	}
}

func (c *WSClient) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   65536,
		WriteBufferSize:  4096,
	}
	headers := c.authHeader
	if headers == nil {
		headers = http.Header{}
	}
	conn, _, err := dialer.DialContext(ctx, c.url, headers)
	if err != nil {
		return err
	}
	c.conn = conn

	if len(c.subs) > 0 {
		subMsg := map[string]interface{}{
			"action":  "subscribe",
			"symbols": c.subs,
		}
		if err := conn.WriteJSON(subMsg); err != nil {
			return err
		}
	}
	return nil
}

func (c *WSClient) ReadLoop(ctx context.Context) {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("ws read error: %v, reconnecting...\n", err)
			time.Sleep(c.reconnect)
			c.reconnect = minDuration(c.reconnect*2, 30*time.Second)
			if err := c.Connect(ctx); err != nil {
				continue
			}
			c.reconnect = time.Second
			continue
		}

		var tick TickMessage
		if err := json.Unmarshal(msg, &tick); err != nil {
			log.Printf("ws parse error: %v\n", err)
			continue
		}

		side := uint8(1)
		if tick.Side == "ask" || tick.Side == "sell" {
			side = 2
		}

		c.mu.Lock()
		symID := c.symbolMap[tick.Symbol]
		c.mu.Unlock()
		if symID == 0 {
			symID = hashSymbol(tick.Symbol)
		}

		goTick := &GoMarketTick{
			Timestamp: tick.Time,
			PriceRaw:  tick.Price.Int64(),
			BidPrice:  tick.BidPrice.Int64(),
			AskPrice:  tick.AskPrice.Int64(),
			VolumeRaw: uint64(tick.Volume),
			BidSize:   uint64(tick.BidSize),
			AskSize:   uint64(tick.AskSize),
			SymbolID:  symID,
			Side:      side,
		}

		if !c.ringBuf.Push(goTick) {
			log.Printf("ws ring buffer overflow, tick dropped for %s\n", tick.Symbol)
		}
	}
}

func (c *WSClient) Close() {
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

func hashSymbol(symbol string) uint32 {
	var h uint32
	for i := 0; i < len(symbol); i++ {
		h = h*31 + uint32(symbol[i])
	}
	return h
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
