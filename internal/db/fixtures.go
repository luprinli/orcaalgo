package db

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type SeedData struct {
	AdminUsers      []AdminUserSeed
	BrokerProviders []BrokerProviderSeed
	LLMProviders     []LLMProviderSeed
	Strategies       []StrategySeed
	Symbols          []SymbolSeed
	Candles          []CandleSeed
	MarketTicks      []MarketTickSeed
	RegimeLogs       []RegimeLogSeed
	TradeHistory     []TradeHistorySeed
	BacktestResults  []BacktestResultSeed
	UniverseConfigs  []UniverseConfigSeed
	UniverseSnapshots []UniverseSnapshotSeed
}

type AdminUserSeed struct {
	Username     string
	Password     string
	Roles        []string
	TOTPEnabled  bool
}

type BrokerProviderSeed struct {
	Name   string
	Type   string
	Driver string
	Config map[string]interface{}
}

type LLMProviderSeed struct {
	Name     string
	Provider string
	Model    string
	APIKey   string
}

type StrategySeed struct {
	Name       string
	Type       string
	Parameters map[string]interface{}
	Enabled    bool
}

type SymbolSeed struct {
	Ticker    string
	Exchange  string
	AssetType string
	TickSize  float64
	LotSize   float64
}

type MarketTickSeed struct {
	Time     time.Time
	Symbol   string
	Price    float64
	Volume   float64
	BidPrice float64
	AskPrice float64
	BidSize  float64
	AskSize  float64
}

type CandleSeed struct {
	Time     time.Time
	Symbol   string
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

type UniverseConfigSeed struct {
	Name              string
	ProfileID         string
	AssetClassFilters map[string]interface{}
	DynamicTriggers   map[string]interface{}
	ContentHash       string
	IsActive          bool
}

type UniverseSnapshotSeed struct {
	SnapshotDate  time.Time
	SymbolTickers []string
	ContentHash   string
}

type RegimeLogSeed struct {
	Time       time.Time
	Symbol     string
	HMMState   int8
	Confidence float64
}

type TradeHistorySeed struct {
	Time        time.Time
	Symbol      string
	Side        string
	Quantity    float64
	Price       float64
	HMMRegime   int8
	StrategyID  string
	OutcomePnL  float64
}

type BacktestResultSeed struct {
	StrategyName string
	Symbols      []string
	StartDate    time.Time
	EndDate      time.Time
	SharpeRatio  float64
	MaxDrawdown  float64
	TotalReturn  float64
	WinRate      float64
	NumTrades    int
}

func GenerateSeedData() *SeedData {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return &SeedData{
		AdminUsers: []AdminUserSeed{
			{Username: "admin", Password: "admin123", Roles: []string{"admin", "trader"}, TOTPEnabled: false},
			{Username: "trader1", Password: "trader123", Roles: []string{"trader"}, TOTPEnabled: false},
		},
		BrokerProviders: []BrokerProviderSeed{
			{Name: "Alpaca Paper", Type: "broker", Driver: "alpaca", Config: map[string]interface{}{"paper": true, "base_url": "https://paper-api.alpaca.markets"}},
			{Name: "Alpaca Live", Type: "broker", Driver: "alpaca", Config: map[string]interface{}{"paper": false, "base_url": "https://api.alpaca.markets"}},
			{Name: "Interactive Brokers", Type: "broker", Driver: "ibkr", Config: map[string]interface{}{"host": "127.0.0.1", "port": 7497}},
			{Name: "Polygon.io", Type: "data_source", Driver: "polygon", Config: map[string]interface{}{"feed": "sip"}},
			{Name: "Binance", Type: "both", Driver: "binance", Config: map[string]interface{}{"futures": true}},
		},
		LLMProviders: []LLMProviderSeed{
			{Name: "OpenAI GPT-4o", Provider: "openai", Model: "gpt-4o"},
			{Name: "Anthropic Claude", Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
		},
		Strategies: []StrategySeed{
			{Name: "Intraday Mean Reversion", Type: "mean_reversion", Parameters: map[string]interface{}{"lookback": 20, "entry_z": 2.0, "exit_z": 0.5, "max_hold": 60}, Enabled: true},
			{Name: "Grid Trading", Type: "grid", Parameters: map[string]interface{}{"grid_levels": 5, "position_scale": 1.0, "max_open": 10}, Enabled: true},
			{Name: "Opening Range Breakout", Type: "breakout", Parameters: map[string]interface{}{"range_minutes": 5, "volume_mult": 1.5, "atr_stop": 2.0}, Enabled: true},
			{Name: "Trend Following", Type: "trend", Parameters: map[string]interface{}{"ema_fast": 20, "ema_slow": 50, "adx_threshold": 25}, Enabled: true},
			{Name: "Session Scalping", Type: "scalp", Parameters: map[string]interface{}{"range_minutes": 5, "volume_mult": 1.5, "atr_period": 14, "take_profit_atr_mult": 1.5, "stop_loss_atr_mult": 0.75, "time_exit_minutes": 90}, Enabled: true},
			{Name: "Pairs Trading", Type: "stat_arb", Parameters: map[string]interface{}{"pair": []string{"SPY", "QQQ"}, "z_entry": 2.0, "z_exit": 0.0}, Enabled: true},
			{Name: "Volatility Harvesting", Type: "vol_arb", Parameters: map[string]interface{}{"vix_threshold": 25, "delta_hedge": true, "max_vega": 500}, Enabled: true},
		},
		Symbols: []SymbolSeed{
			{Ticker: "EURUSD", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "GBPUSD", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "USDJPY", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.001, LotSize: 1000},
			{Ticker: "USDCHF", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "AUDUSD", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "USDCAD", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "NZDUSD", Exchange: "STOOQ", AssetType: "forex", TickSize: 0.00001, LotSize: 1000},
			{Ticker: "US30", Exchange: "STOOQ", AssetType: "index", TickSize: 1.0, LotSize: 1},
			{Ticker: "SPX500", Exchange: "STOOQ", AssetType: "index", TickSize: 0.25, LotSize: 1},
			{Ticker: "NAS100", Exchange: "STOOQ", AssetType: "index", TickSize: 0.25, LotSize: 1},
			{Ticker: "UK100", Exchange: "STOOQ", AssetType: "index", TickSize: 1.0, LotSize: 1},
			{Ticker: "GER40", Exchange: "STOOQ", AssetType: "index", TickSize: 1.0, LotSize: 1},
			{Ticker: "JPN225", Exchange: "STOOQ", AssetType: "index", TickSize: 1.0, LotSize: 1},
			{Ticker: "XAUUSD", Exchange: "STOOQ", AssetType: "commodity", TickSize: 0.01, LotSize: 100},
			{Ticker: "XAGUSD", Exchange: "STOOQ", AssetType: "commodity", TickSize: 0.01, LotSize: 100},
			{Ticker: "BTCUSD", Exchange: "STOOQ", AssetType: "crypto", TickSize: 0.01, LotSize: 1},
			{Ticker: "ETHUSD", Exchange: "STOOQ", AssetType: "crypto", TickSize: 0.01, LotSize: 1},
		},
		Candles:       loadCandlesFromDataDir(),
		MarketTicks:   nil,
		RegimeLogs: []RegimeLogSeed{
			{Time: today.AddDate(0, 0, -30), Symbol: "EURUSD", HMMState: 0, Confidence: 0.85},
			{Time: today.AddDate(0, 0, -25), Symbol: "EURUSD", HMMState: 1, Confidence: 0.72},
			{Time: today.AddDate(0, 0, -20), Symbol: "EURUSD", HMMState: 0, Confidence: 0.91},
			{Time: today.AddDate(0, 0, -15), Symbol: "EURUSD", HMMState: 1, Confidence: 0.68},
			{Time: today.AddDate(0, 0, -10), Symbol: "EURUSD", HMMState: 2, Confidence: 0.55},
			{Time: today.AddDate(0, 0, -8), Symbol: "EURUSD", HMMState: 0, Confidence: 0.88},
			{Time: today.AddDate(0, 0, -5), Symbol: "EURUSD", HMMState: 1, Confidence: 0.79},
			{Time: today.AddDate(0, 0, -3), Symbol: "EURUSD", HMMState: 0, Confidence: 0.92},
			{Time: today.AddDate(0, 0, -1), Symbol: "EURUSD", HMMState: 1, Confidence: 0.64},
			{Time: today, Symbol: "EURUSD", HMMState: 0, Confidence: 0.87},
		},
		TradeHistory: nil,
		BacktestResults: nil,
		UniverseConfigs: []UniverseConfigSeed{
			{
				Name:      "default",
				ProfileID: "default",
				AssetClassFilters: map[string]interface{}{
					"equity":    map[string]interface{}{"min_notional_volume": 50000000, "min_price": 5.0, "max_price": 5000.0, "min_market_cap": 500000000000, "min_atr_percent": 0.5, "max_atr_percent": 5.0, "min_rsi": 25, "max_rsi": 75},
					"forex":     map[string]interface{}{"min_notional_volume": 100000000, "min_atr_percent": 0.3, "max_atr_percent": 3.0, "min_rsi": 20, "max_rsi": 80},
					"crypto":    map[string]interface{}{"min_notional_volume": 10000000, "min_price": 1.0, "max_price": 100000.0, "min_atr_percent": 1.0, "max_atr_percent": 8.0, "min_rsi": 20, "max_rsi": 80},
				},
				DynamicTriggers: map[string]interface{}{
					"volume_spike_multiplier": 2.5, "volatility_multiplier": 2.0, "news_sentiment_abs_min": 0.7,
					"min_lookback_days": 20, "cooldown_hours_after_add": 48, "cooldown_hours_after_remove": 24,
				},
				ContentHash: "abc123default",
				IsActive:    true,
			},
		},
		UniverseSnapshots: []UniverseSnapshotSeed{
			{
				SnapshotDate:  today,
				SymbolTickers: []string{"EURUSD", "GBPUSD", "USDJPY", "BTCUSD", "ETHUSD", "US30", "SPX500"},
				ContentHash:   "snap001",
			},
		},
	}
}

func generateRecentMarketTicks(today time.Time) []MarketTickSeed {
	return nil
}

func generateRecentTrades(today time.Time) []TradeHistorySeed {
	return nil
}

func loadCandlesFromDataDir() []CandleSeed {
	dataDir := filepath.Join("data", "daily", "world", "stooq stocks indices")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil
	}

	tickerMap := map[string]string{
		"^_us.txt":   "SPX500",
		"^_usnq.txt": "NAS100",
		"^_uk.txt":   "UK100",
		"^_de.txt":   "GER40",
		"^_jp.txt":   "JPN225",
	}

	var allCandles []CandleSeed
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		mappedSym, ok := tickerMap[entry.Name()]
		if !ok {
			continue
		}

		path := filepath.Join(dataDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == 0 || line == "" {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) < 9 {
				continue
			}
			ymd := strings.TrimSpace(parts[2])
			if len(ymd) != 8 {
				continue
			}
			y, _ := strconv.Atoi(ymd[0:4])
			m, _ := strconv.Atoi(ymd[4:6])
			d, _ := strconv.Atoi(ymd[6:8])
			open, _ := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
			high, _ := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
			low, _ := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)
			close_, _ := strconv.ParseFloat(strings.TrimSpace(parts[7]), 64)
			vol, _ := strconv.ParseFloat(strings.TrimSpace(parts[8]), 64)

			if close_ == 0 {
				continue
			}
			allCandles = append(allCandles, CandleSeed{
				Time:   time.Date(y, time.Month(m), d, 16, 0, 0, 0, time.UTC),
				Symbol: mappedSym,
				Open:   open,
				High:   high,
				Low:    low,
				Close:  close_,
				Volume: vol,
			})
		}
	}
	return allCandles
}
