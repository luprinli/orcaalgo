package risk

const (
	DefaultBufferPct         = 0.002
	DefaultCorrelationMult   = 0.5
	DefaultAdverseSelectPct  = 0.05
	DefaultKellyMultiplier   = 0.25
	DefaultPerTradeCap       = 0.02
	DefaultTotalExpCap       = 0.30
	DefaultMaxPositionPct    = 0.03
	DefaultVolatilityHalt    = 3.0
	DefaultOrderRateLimit    = 10
	DefaultStratDrawdownFrac = 0.5
	DefaultVIXHigh           = 35.0
	DefaultVIXMid            = 28.0
	DefaultVIXLow            = 20.0
	VIXHighMult              = 0.50
	VIXMidMult               = 0.75
	VIXLowMult               = 0.90
	SentimentExtremeLow      = 10
	SentimentExtremeHigh     = 90
	SentimentModerateLow     = 20
	SentimentModerateHigh    = 80
	SentimentExtremeMult     = 0.50
	SentimentModerateMult    = 0.75
	// Regime-based sizing attenuation (non-ML path): normal/expansion 1.0x,
	// moderate (regime 2) and crisis (regime 3+) scale down. RegimeMLScale is the
	// multiplier applied to the ML regime score when SetRegimeScore is used.
	RegimeModerateMult = 0.75
	RegimeCrisisMult   = 0.50
	RegimeMLScale      = 1.5
)
