package ingest

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/quickfixgo/quickfix"
)

type FIXClient struct {
	settings  *quickfix.Settings
	initiator *quickfix.Initiator
	ringBuf   *RingBuffer
	configFile string
}

func NewFIXClient(configFile string, ringBuf *RingBuffer) (*FIXClient, error) {
	return &FIXClient{
		ringBuf:    ringBuf,
		configFile: configFile,
	}, nil
}

func (c *FIXClient) Start(ctx context.Context) error {
	f, err := os.Open(c.configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	settings, err := quickfix.ParseSettings(f)
	if err != nil {
		return err
	}
	c.settings = settings

	app := &FIXApp{ringBuf: c.ringBuf}

	storeFactory := quickfix.NewMemoryStoreFactory()
	logFactory := quickfix.NewNullLogFactory()

	initiator, err := quickfix.NewInitiator(app, storeFactory, c.settings, logFactory)
	if err != nil {
		return err
	}
	c.initiator = initiator

	go func() {
		if err := initiator.Start(); err != nil {
			log.Printf("FIX initiator error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		initiator.Stop()
	}()

	return nil
}

type FIXApp struct {
	ringBuf *RingBuffer
}

func (a *FIXApp) OnCreate(sessionID quickfix.SessionID)                          {}
func (a *FIXApp) OnLogon(sessionID quickfix.SessionID)                           {}
func (a *FIXApp) OnLogout(sessionID quickfix.SessionID)                          {}
func (a *FIXApp) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID)     {}
func (a *FIXApp) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error { return nil }
func (a *FIXApp) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}
func (a *FIXApp) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	tick := parseMarketData(msg)
	if tick != nil {
		if !a.ringBuf.Push(tick) { log.Printf("fix ring buffer overflow for symbol %d", tick.SymbolID) }
	}
	return nil
}

func parseMarketData(msg *quickfix.Message) *GoMarketTick {
	body := msg.Body

	var symbol quickfix.FIXString
	var price quickfix.FIXFloat
	var volume quickfix.FIXFloat

	if err := body.GetField(55, &symbol); err != nil {
		return nil
	}
	if err := body.GetField(44, &price); err != nil {
		return nil
	}
	if vErr := body.GetField(38, &volume); vErr != nil { volume = 0 }

	return &GoMarketTick{
		Timestamp: time.Now().UnixNano(),
		PriceRaw:  int64(float64(price) * PRICE_SCALE),
		VolumeRaw: uint64(float64(volume)),
		SymbolID:  hashSymbol(string(symbol)),
		Side:      1,
	}
}
