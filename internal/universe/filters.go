package universe

import (
	"github.com/lee-econ/orca-core/internal/db"
)

type AssetClassFilter struct {
	MinNotionalVolume int64   `json:"min_notional_volume"`
	MinPrice          float64 `json:"min_price"`
	MaxPrice          float64 `json:"max_price"`
	MinMarketCap      int64   `json:"min_market_cap"`
	MinATRPercent     float64 `json:"min_atr_percent"`
	MaxATRPercent     float64 `json:"max_atr_percent"`
	MinRSI            float64 `json:"min_rsi"`
	MaxRSI            float64 `json:"max_rsi"`
}

type UniverseConfigFilters struct {
	Equity    AssetClassFilter `json:"equity"`
	Forex     AssetClassFilter `json:"forex"`
	Crypto    AssetClassFilter `json:"crypto"`
	Index     AssetClassFilter `json:"index"`
	Commodity AssetClassFilter `json:"commodity"`
}

type DynamicTriggerThresholds struct {
	VolumeSpikeMultiplier    float64 `json:"volume_spike_multiplier"`
	VolatilityMultiplier     float64 `json:"volatility_multiplier"`
	NewsSentimentAbsMin      float64 `json:"news_sentiment_abs_min"`
	MinLookbackDays          int     `json:"min_lookback_days"`
	CooldownHoursAfterAdd    int     `json:"cooldown_hours_after_add"`
	CooldownHoursAfterRemove int     `json:"cooldown_hours_after_remove"`
}

func DefaultFilters() UniverseConfigFilters {
	return UniverseConfigFilters{
		Equity: AssetClassFilter{
			MinNotionalVolume: 50_000_000,
			MinPrice:          5.0,
			MaxPrice:          5000.0,
			MinMarketCap:      500_000_000_000,
			MinATRPercent:     0.5,
			MaxATRPercent:     5.0,
			MinRSI:            25,
			MaxRSI:            75,
		},
		Forex: AssetClassFilter{
			MinNotionalVolume: 100_000_000,
			MinATRPercent:     0.3,
			MaxATRPercent:     3.0,
			MinRSI:            20,
			MaxRSI:            80,
		},
		Crypto: AssetClassFilter{
			MinNotionalVolume: 10_000_000,
			MinPrice:          1.0,
			MaxPrice:          100_000.0,
			MinATRPercent:     1.0,
			MaxATRPercent:     8.0,
			MinRSI:            20,
			MaxRSI:            80,
		},
		Index: AssetClassFilter{
			MinNotionalVolume: 1_000_000_000,
			MinPrice:          100.0,
			MinATRPercent:     0.3,
			MaxATRPercent:     4.0,
			MinRSI:            25,
			MaxRSI:            75,
		},
		Commodity: AssetClassFilter{
			MinNotionalVolume: 50_000_000,
			MinPrice:          10.0,
			MinATRPercent:     0.5,
			MaxATRPercent:     5.0,
			MinRSI:            20,
			MaxRSI:            80,
		},
	}
}

func DefaultTriggers() DynamicTriggerThresholds {
	return DynamicTriggerThresholds{
		VolumeSpikeMultiplier:    2.5,
		VolatilityMultiplier:     2.0,
		NewsSentimentAbsMin:      0.7,
		MinLookbackDays:          20,
		CooldownHoursAfterAdd:    48,
		CooldownHoursAfterRemove: 24,
	}
}

func (f AssetClassFilter) filterForClass(assetType string) AssetClassFilter {
	switch assetType {
	case "equity":
		return f
	case "forex":
		cf := f
		cf.MinMarketCap = 0
		cf.MinPrice = 0
		cf.MaxPrice = 0
		return cf
	case "crypto":
		cf := f
		cf.MinMarketCap = 0
		return cf
	case "index":
		cf := f
		cf.MinMarketCap = 0
		cf.MinPrice = 0
		cf.MaxPrice = 0
		return cf
	case "commodity":
		cf := f
		cf.MinMarketCap = 0
		return cf
	default:
		return f
	}
}

func (f AssetClassFilter) Apply(s db.Symbol) bool {
	eff := f.filterForClass(s.AssetType)

	if eff.MinNotionalVolume > 0 && s.LastVolume > 0 && s.LastVolume < eff.MinNotionalVolume {
		return false
	}
	if eff.MinMarketCap > 0 && s.MarketCap > 0 && s.MarketCap < eff.MinMarketCap {
		return false
	}
	if eff.MinPrice > 0 && s.LastPrice > 0 && s.LastPrice < eff.MinPrice {
		return false
	}
	if eff.MaxPrice > 0 && s.LastPrice > eff.MaxPrice {
		return false
	}
	if eff.MinATRPercent > 0 && s.LastATRPct > 0 && s.LastATRPct < eff.MinATRPercent {
		return false
	}
	if eff.MaxATRPercent > 0 && s.LastATRPct > eff.MaxATRPercent {
		return false
	}
	if eff.MinRSI > 0 && s.LastRSI > 0 && s.LastRSI < eff.MinRSI {
		return false
	}
	if eff.MaxRSI > 0 && s.LastRSI > eff.MaxRSI {
		return false
	}
	return true
}
