package propfirm

import (
	"encoding/json"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`

	MaxDailyLossPct     float64  `json:"max_daily_loss_pct" yaml:"max_daily_loss_pct"`
	MaxDrawdownPct      float64  `json:"max_drawdown_pct" yaml:"max_drawdown_pct"`
	DrawdownType        string   `json:"drawdown_type" yaml:"drawdown_type"`
	MaxPositionPct      float64  `json:"max_position_pct" yaml:"max_position_pct"`
	MaxOpenPositions    int      `json:"max_open_positions" yaml:"max_open_positions"`
	MaxTradesPerDay     int      `json:"max_trades_per_day" yaml:"max_trades_per_day"`

	ConsistencyEnabled     bool    `json:"consistency_enabled" yaml:"consistency_enabled"`
	ConsistencyThresholdPct float64 `json:"consistency_threshold_pct" yaml:"consistency_threshold_pct"`
	ConsistencyPenalty     float64 `json:"consistency_penalty" yaml:"consistency_penalty"`

	ProfitTargetPctPhase1 float64 `json:"profit_target_pct_phase1" yaml:"profit_target_pct_phase1"`
	ProfitTargetPctPhase2 float64 `json:"profit_target_pct_phase2" yaml:"profit_target_pct_phase2"`
	MinTradingDays        int     `json:"min_trading_days" yaml:"min_trading_days"`

	WeekendHoldingAllowed bool       `json:"weekend_holding_allowed" yaml:"weekend_holding_allowed"`
	NewsTradingAllowed    bool       `json:"news_trading_allowed" yaml:"news_trading_allowed"`
	RegimeMultipliers     [4]float64 `json:"regime_multipliers" yaml:"regime_multipliers"`
}

type State struct {
	ProfileID       string  `json:"profile_id"`
	StartingBalance float64 `json:"starting_balance"`
	PeakBalance     float64 `json:"peak_balance"`
	DailyPnL        float64 `json:"daily_pnl"`
	DailyPnLPct     float64 `json:"daily_pnl_pct"`
	CumulativePnL   float64 `json:"cumulative_pnl"`
	ConsistencyMult float64 `json:"consistency_mult"`
	TradingDays     int     `json:"trading_days"`
	CurrentPhase    int     `json:"current_phase"`
	PhaseTargetMet  bool    `json:"phase_target_met"`
	Violated        bool    `json:"violated"`
	ViolationReason string  `json:"violation_reason"`
}

type Manager struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	active   string
	state    State
}

func NewManager() *Manager {
	return &Manager{
		profiles: make(map[string]*Profile),
		state: State{ConsistencyMult: 1.0, CurrentPhase: 1},
	}
}

func (m *Manager) LoadProfile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		if err := json.Unmarshal(data, &profile); err != nil {
			return err
		}
	}
	m.mu.Lock()
	m.profiles[profile.ID] = &profile
	m.mu.Unlock()
	return nil
}

func (m *Manager) ActivateProfile(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.profiles[id]
	if !ok {
		return os.ErrNotExist
	}
	m.active = id
	m.state = State{
		ProfileID:       id,
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	return nil
}

func (m *Manager) ActiveProfile() *Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiles[m.active]
}

func (m *Manager) ActiveID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) RecordFill(pnl float64, balance float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.DailyPnL += pnl
	m.state.CumulativePnL += pnl
	if balance > m.state.PeakBalance {
		m.state.PeakBalance = balance
	}
	if m.state.StartingBalance > 0 {
		m.state.DailyPnLPct = m.state.DailyPnL / m.state.StartingBalance * 100.0
	}
}

func (m *Manager) CheckDailyLimits() (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p := m.profiles[m.active]
	if p == nil {
		return true, ""
	}
	s := m.state
	currentBalance := s.StartingBalance + s.DailyPnL
	if p.MaxDailyLossPct > 0 && DailyLossExceeded(s.StartingBalance, currentBalance, p.MaxDailyLossPct) {
		return false, "daily_loss_limit"
	}
	if p.MaxDrawdownPct > 0 && DrawdownExceeded(s.PeakBalance, currentBalance, p.MaxDrawdownPct) {
		return false, "max_drawdown"
	}
	if p.ConsistencyEnabled && p.ConsistencyThresholdPct > 0 && ConsistencyBreached(s.DailyPnLPct, p.ConsistencyThresholdPct) {
		return false, "consistency_outlier"
	}
	return true, ""
}

func (m *Manager) DailyReset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.profiles[m.active]
	s := &m.state
	if p != nil && p.ConsistencyEnabled && s.DailyPnLPct > p.ConsistencyThresholdPct {
		s.ConsistencyMult = p.ConsistencyPenalty
	} else {
		s.ConsistencyMult = 1.0
	}
	s.DailyPnL = 0
	s.DailyPnLPct = 0
	s.TradingDays++
	if p != nil {
		target := p.ProfitTargetPctPhase1
		if s.CurrentPhase == 2 {
			target = p.ProfitTargetPctPhase2
		}
		totalReturn := s.CumulativePnL / s.StartingBalance * 100.0
		if totalReturn >= target && s.TradingDays >= p.MinTradingDays {
			s.PhaseTargetMet = true
		}
	}
}

func (m *Manager) AdvancePhase() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.CurrentPhase++
	m.state.PhaseTargetMet = false
	m.state.DailyPnL = 0
	m.state.DailyPnLPct = 0
	m.state.CumulativePnL = 0
	m.state.PeakBalance = m.state.StartingBalance
	m.state.ConsistencyMult = 1.0
}

func (m *Manager) MarkViolated(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Violated = true
	m.state.ViolationReason = reason
}

func (m *Manager) IsHalted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.Violated
}

func (m *Manager) AllProfiles() map[string]*Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	profiles := make(map[string]*Profile, len(m.profiles))
	for k, v := range m.profiles {
		profiles[k] = v
	}
	return profiles
}

func DefaultFTMOProfile() *Profile {
	return &Profile{
		ID:   "ftmo",
		Name: "FTMO",
		MaxDailyLossPct:         5.0,
		MaxDrawdownPct:          10.0,
		DrawdownType:            "static_hwm",
		MaxPositionPct:          2.0,
		MaxOpenPositions:        5,
		MaxTradesPerDay:         10,
		ConsistencyEnabled:      true,
		ConsistencyThresholdPct: 30.0,
		ConsistencyPenalty:      0.5,
		ProfitTargetPctPhase1:   10.0,
		ProfitTargetPctPhase2:   5.0,
		MinTradingDays:          4,
		WeekendHoldingAllowed:   true,
		NewsTradingAllowed:      true,
		RegimeMultipliers:       [4]float64{1.0, 0.85, 0.75, 0.5},
	}
}

func DefaultProfile() *Profile {
	return DefaultFTMOProfile()
}
