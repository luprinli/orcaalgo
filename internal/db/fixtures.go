package db

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
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
	VIXLogs          []VIXLogSeed
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
	Price    types.Price
	Volume   float64
	BidPrice types.Price
	AskPrice types.Price
	BidSize  float64
	AskSize  float64
}

type CandleSeed struct {
	Time     time.Time
	Symbol   string
	Open     types.Price
	High     types.Price
	Low      types.Price
	Close    types.Price
	Volume   float64
	Timeframe string
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

type VIXLogSeed struct {
	Time      time.Time
	VIXValue  float64
	VIXChange float64
}

type TradeHistorySeed struct {
	Time        time.Time
	Symbol      string
	Side        string
	Quantity    float64
	Price       types.Price
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
	candles := loadCandlesFromDataDir()
	symbols := []SymbolSeed{
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
		{Ticker: "SPY", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "QQQ", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "AAPL", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "MSFT", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "GOOGL", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "META", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "AMZN", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "NVDA", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "TSLA", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "VOO", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "DIA", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "IWM", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "GLD", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "USO", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
		{Ticker: "CL", Exchange: "STOOQ", AssetType: "commodity", TickSize: 0.01, LotSize: 100},
		{Ticker: "NQ", Exchange: "STOOQ", AssetType: "index", TickSize: 0.25, LotSize: 1},
		{Ticker: "ES", Exchange: "STOOQ", AssetType: "index", TickSize: 0.25, LotSize: 1},
		{Ticker: "TLT", Exchange: "STOOQ", AssetType: "stock", TickSize: 0.01, LotSize: 1},
	}

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
			{Name: "Intraday Mean Reversion", Type: "intraday_mr", Parameters: map[string]interface{}{"lookback": 20, "entry_z": 2.0, "exit_z": 0.5, "max_hold": 60}, Enabled: true},
			{Name: "Grid Trading", Type: "grid_trading", Parameters: map[string]interface{}{"grid_levels": 5, "position_scale": 1.0, "max_open": 10}, Enabled: false},
			{Name: "Opening Range Breakout (5m)", Type: "opening_range_breakout", Parameters: map[string]interface{}{"range_minutes": 5, "volume_mult": 1.5, "atr_stop": 2.0}, Enabled: true},
			{Name: "Trend Following", Type: "trend_following", Parameters: map[string]interface{}{"ema_fast": 20, "ema_slow": 50, "adx_threshold": 25}, Enabled: true},
			{Name: "Session Scalping", Type: "session_scalp", Parameters: map[string]interface{}{"range_minutes": 5, "volume_mult": 1.5, "atr_period": 14}, Enabled: true},
			{Name: "Pairs Trading", Type: "pairs_trading", Parameters: map[string]interface{}{"pair": []string{"SPY", "QQQ"}, "z_entry": 2.0, "z_exit": 0.0}, Enabled: true},
			{Name: "Volatility Harvesting", Type: "volatility_harvesting", Parameters: map[string]interface{}{"vix_threshold": 25, "delta_hedge": true, "max_vega": 500}, Enabled: true},
			{Name: "Dragon Capital Trend", Type: "dragon_trend", Parameters: map[string]interface{}{"ema_periods": []int{8, 21, 50, 200}, "min_aligned": 3, "adx_threshold": 20}, Enabled: true},
			{Name: "VWAP Mean Reversion", Type: "vwap_mr", Parameters: map[string]interface{}{"lookback": 20, "entry_z": 1.5, "exit_z": 0.5, "mode": "vwap"}, Enabled: true},
			{Name: "15-Minute ORB", Type: "orb_15m", Parameters: map[string]interface{}{"range_minutes": 15, "volume_mult": 1.5}, Enabled: true},
			{Name: "Volume-Weighted Scalp", Type: "volume_scalp", Parameters: map[string]interface{}{"range_minutes": 5, "volume_multiplier": 2.0}, Enabled: true},
			{Name: "VIX Futures Carry", Type: "vix_futures_carry", Parameters: map[string]interface{}{"contango_threshold": 22, "fade_entry_z": 1.5}, Enabled: true},
			{Name: "Volatility-Adjusted Grid", Type: "vol_grid", Parameters: map[string]interface{}{"grid_levels": 5, "adjust_by_volatility": true}, Enabled: false},
			{Name: "MA Crossover", Type: "ma_crossover", Parameters: map[string]interface{}{"fast_period": 10, "slow_period": 30}, Enabled: true},
			{Name: "RSI2 Reversion", Type: "rsi2_reversion", Parameters: map[string]interface{}{"rsi_period": 2, "entry_threshold": 10, "exit_threshold": 50}, Enabled: true},
			{Name: "Donchian Breakout", Type: "donchian_breakout", Parameters: map[string]interface{}{"channel_length": 20}, Enabled: true},
			{Name: "Keltner MACD", Type: "keltner_macd", Parameters: map[string]interface{}{"keltner_period": 20, "atr_multiplier": 1.5}, Enabled: true},
			{Name: "Ichimoku Cloud", Type: "ichimoku_cloud", Parameters: map[string]interface{}{"tenkan_period": 9, "kijun_period": 26, "senkou_b_period": 52}, Enabled: true},
		},
		Symbols:       symbols,
		Candles:       append(candles, generateSyntheticCandles(today, symbols)...),
		MarketTicks:   nil,
		RegimeLogs:    generateRegimeLogs(today, symbols),
		VIXLogs:       generateVIXLogs(today, candles),
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

func resolveDataDir() string {
	if root := os.Getenv("ORCA_PROJECT_ROOT"); root != "" {
		return filepath.Join(root, "data", "daily", "world", "stooq stocks indices")
	}
	wd, _ := os.Getwd()
	for dir := wd; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "kilo.json")); err == nil {
			return filepath.Join(dir, "data", "daily", "world", "stooq stocks indices")
		}
	}
	return filepath.Join("data", "daily", "world", "stooq stocks indices")
}

func loadCandlesFromDataDir() []CandleSeed {
	dataDir := resolveDataDir()
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
				Symbol: mappedSym, Timeframe: "1d",
				Open:   types.FromFloat64(open),
				High:   types.FromFloat64(high),
				Low:    types.FromFloat64(low),
				Close:  types.FromFloat64(close_),
				Volume: vol,
			})
		}
	}
	return allCandles
}

func generateVIXLogs(today time.Time, candles []CandleSeed) []VIXLogSeed {
	type dayRange struct {
		sum   float64
		count int
	}
	dailyVol := make(map[string]*dayRange)
	for _, c := range candles {
		if c.High.IsZero() || c.Low.IsZero() || c.Close.IsZero() {
			continue
		}
		high := c.High.Float64()
		low := c.Low.Float64()
		close_ := c.Close.Float64()
		if close_ == 0 || high <= low {
			continue
		}
		key := c.Time.Format("2006-01-02")
		if _, ok := dailyVol[key]; !ok {
			dailyVol[key] = &dayRange{}
		}
		dailyVol[key].sum += (high - low) / close_
		dailyVol[key].count++
	}

	type dayVIX struct {
		time time.Time
		raw  float64
	}
	var rawDays []dayVIX
	start := today.AddDate(0, 0, -400)
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		v := 12.0
		if dv, ok := dailyVol[key]; ok && dv.count > 0 {
			avgRange := (dv.sum / float64(dv.count)) * 100.0
			v = 10.0 + avgRange*15.0
		}
		sawtooth := float64((int(d.Unix()/86400)*7+13)%100-50) / 50.0 * 1.5
		v += sawtooth
		if v > 55.0 {
			v = 55.0
		}
		if v < 10.0 {
			v = 10.0
		}
		rawDays = append(rawDays, dayVIX{time: d, raw: v})
	}

	// VIX spike injection: add 6 synthetic volatility event windows per year.
	// Each event ramps up over 3-5 days, peaks at VIX 30-45 for 2-3 days,
	// and ramps down over 3-5 days. This ensures VIX-dependent strategies
	// have sufficient entry opportunities.
	spikeEvents := []struct {
		startDay  int
		duration  int
		peakVIX   float64
	}{
		{60, 10, 38.0}, {130, 12, 42.0}, {190, 8, 35.0},
		{250, 14, 45.0}, {310, 10, 32.0}, {360, 11, 40.0},
	}
	for _, ev := range spikeEvents {
		for offset := 0; offset < ev.duration && ev.startDay+offset < len(rawDays); offset++ {
			mid := float64(ev.duration) / 2.0
			pos := float64(offset)
			var spikeMult float64
			if pos < mid-1.5 {
				spikeMult = (pos + 1) / (mid - 1.5)
			} else if pos > mid+1.5 {
				spikeMult = 1.0 - (pos-mid-1.5)/(float64(ev.duration)-mid-1.5)
			} else {
				spikeMult = 1.0
			}
			if spikeMult > 1.0 {
				spikeMult = 1.0
			}
			if spikeMult < 0 {
				spikeMult = 0
			}
			extra := (ev.peakVIX - rawDays[ev.startDay+offset].raw) * spikeMult
			if extra < 0 {
				extra = 0
			}
			rawDays[ev.startDay+offset].raw += extra
		}
	}

	logs := make([]VIXLogSeed, 0, len(rawDays))
	var smoothed []float64
	window := 5
	for i := range rawDays {
		sum := 0.0
		n := 0
		for j := i - window; j <= i+window; j++ {
			if j >= 0 && j < len(rawDays) {
				sum += rawDays[j].raw
				n++
			}
		}
		smoothed = append(smoothed, sum/float64(n))
	}

	prev := smoothed[0]
	for i, v := range smoothed {
		change := v - prev
		logs = append(logs, VIXLogSeed{
			Time:      rawDays[i].time,
			VIXValue:  float64(int(v*100)) / 100,
			VIXChange: float64(int(change*100)) / 100,
		})
		prev = v
	}
	return logs
}

func generateRegimeLogs(today time.Time, symbols []SymbolSeed) []RegimeLogSeed {
	var logs []RegimeLogSeed
	start := today.AddDate(0, 0, -400)
	seedByDay := make(map[int64][4]float64)
	for i := int64(0); i < 400; i++ {
		day := start.AddDate(0, 0, int(i))
		key := day.Unix() / 86400
		r := float64((key*13+int64(len(symbols))*7)%100) / 100.0
		var state int8
		var conf float64
		switch {
		case r < 0.50:
			state, conf = 0, 0.75+r*0.2
		case r < 0.85:
			state, conf = 1, 0.65+r*0.1
		case r < 0.95:
			state, conf = 2, 0.55+r*0.1
		default:
			state, conf = 3, 0.40+r*0.2
		}
		if conf > 1.0 {
			conf = 1.0
		}
		seedByDay[key] = [4]float64{float64(state), conf, float64(state), conf}
		_ = seedByDay
		for _, sym := range symbols {
			r2 := float64((key*17+int64(len(sym.Ticker))*31+int64(sym.Ticker[0])*7)%100) / 100.0
			var s int8
			var c float64
			switch {
			case r2 < 0.50:
				s, c = 0, 0.75+r2*0.2
			case r2 < 0.85:
				s, c = 1, 0.65+r2*0.1
			case r2 < 0.95:
				s, c = 2, 0.55+r2*0.1
			default:
				s, c = 3, 0.40+r2*0.2
			}
			if c > 1.0 {
				c = 1.0
			}
			logs = append(logs, RegimeLogSeed{
				Time: day, Symbol: sym.Ticker, HMMState: s, Confidence: c,
			})
		}
	}
	return logs
}

func generateSyntheticCandles(today time.Time, symbols []SymbolSeed) []CandleSeed {
	basePrices := map[string]float64{
		"SPY": 580, "QQQ": 480, "AAPL": 220, "MSFT": 440, "GOOGL": 180, "META": 560, "AMZN": 220,
		"NVDA": 120, "TSLA": 250, "VOO": 530, "DIA": 420, "IWM": 210, "GLD": 220, "USO": 72,
		"CL": 70, "NQ": 21000, "ES": 6000, "TLT": 88,
		"EURUSD": 1.08, "GBPUSD": 1.28, "USDJPY": 148, "USDCHF": 0.88, "AUDUSD": 0.66,
		"USDCAD": 1.36, "NZDUSD": 0.60, "XAUUSD": 2350, "XAGUSD": 28,
		"BTCUSD": 68000, "ETHUSD": 3400, "US30": 41000, "SPX500": 5900,
		"NAS100": 21000, "UK100": 8300, "GER40": 21000, "JPN225": 41000,
	}
	volMap := map[string]float64{
		"SPY": 0.008, "QQQ": 0.012, "AAPL": 0.014, "MSFT": 0.011, "GOOGL": 0.013, "META": 0.016,
		"AMZN": 0.015, "NVDA": 0.025, "TSLA": 0.028, "VOO": 0.008, "DIA": 0.007, "IWM": 0.012,
		"GLD": 0.009, "USO": 0.015, "CL": 0.018, "NQ": 0.014, "ES": 0.009, "TLT": 0.008,
		"EURUSD": 0.005, "GBPUSD": 0.006, "USDJPY": 0.007, "USDCHF": 0.005, "AUDUSD": 0.006,
		"USDCAD": 0.005, "NZDUSD": 0.007, "XAUUSD": 0.010, "XAGUSD": 0.016,
		"BTCUSD": 0.030, "ETHUSD": 0.035, "US30": 0.007, "SPX500": 0.008,
		"NAS100": 0.012, "UK100": 0.008, "GER40": 0.010, "JPN225": 0.009,
	}
	var out []CandleSeed
	start := today.AddDate(0, 0, -400)
	for _, sym := range symbols {
		base, ok := basePrices[sym.Ticker]
		if !ok { base = 100 }
		vol, ok := volMap[sym.Ticker]
		if !ok { vol = 0.01 }
		price := base * (0.8 + 0.4*float64(sym.Ticker[0]%97)/97.0)
		for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday { continue }
			drift := 0.0001
			noise := (float64(int(d.Unix()/86400*int64(len(sym.Ticker))%200)-100) / 100.0) * vol
			price *= 1.0 + drift + noise
			high := price * (1.0 + vol*0.5)
			low := price * (1.0 - vol*0.5)
			open := low + (high-low)*(float64(int(d.Unix()/3600)%100)/100.0)
			closeP := price
			out = append(out, CandleSeed{
				Symbol: sym.Ticker, Time: d, Timeframe: "1d",
				Open: types.PriceFromFloat(float64(int(open*100)) / 100),
				High: types.PriceFromFloat(float64(int(high*100)) / 100),
				Low:  types.PriceFromFloat(float64(int(low*100)) / 100),
				Close: types.PriceFromFloat(float64(int(closeP*100)) / 100),
				Volume: 1000000 + float64(int(d.Unix()%10000))*100,
			})

			// Generate intraday sub-bars from the daily OHLC.
			subBarCounts := map[string]int{"4h": 6, "1h": 24}
			for tf, n := range subBarCounts {
				step := closeP - open
				dailyVol := vol / math.Sqrt(float64(n))
				prev := open
				barHi := prev
				barLo := prev
				for j := 0; j < n; j++ {
					t := d.Add(time.Duration(j+1) * (24 * time.Hour / time.Duration(n)))
					if t.Equal(d) { t = t.Add(time.Second) }
					noise := (float64(int((d.Unix()+int64(j))*7)%200) - 100) / 100.0 * dailyVol
					subClose := prev + step/float64(n) + noise*prev
					if subClose > high { subClose = high*0.995 }
					if subClose < low { subClose = low*1.005 }
					subOpen := prev
					subHigh := math.Max(subOpen, subClose) * (1.0 + dailyVol*0.3)
					subLow := math.Min(subOpen, subClose) * (1.0 - dailyVol*0.3)
					if subHigh > high { subHigh = high }
					if subLow < low { subLow = low }
					if subHigh < barHi { barHi = subHigh }
					if subLow > barLo { barLo = subLow }
					out = append(out, CandleSeed{
						Symbol: sym.Ticker, Time: t, Timeframe: tf,
						Open: types.PriceFromFloat(float64(int(subOpen*100)) / 100),
						High: types.PriceFromFloat(float64(int(subHigh*100)) / 100),
						Low:  types.PriceFromFloat(float64(int(subLow*100)) / 100),
						Close: types.PriceFromFloat(float64(int(subClose*100)) / 100),
						Volume: (1000000 + float64(int(d.Unix()%10000))*100) / float64(n),
					})
					prev = subClose
				}
				_ = barHi; _ = barLo
			}
		}
	}
	return out
}
