package strategy

import "time"

// Session is a time-of-day trading window expressed in a fixed timezone offset.
// Centralizing it (Rule 8) prevents a strategy from shipping with a UTC default
// for an exchange-local session (the session-scalp timezone bug).
type Session struct {
	StartHour, StartMin int
	EndHour, EndMin     int
	TimezoneOffset      int // hours from UTC (ET = -4 EDT / -5 EST)
}

// NewETSession returns the default US-equity session (9:30–16:00 ET/EDT).
func NewETSession() Session {
	return Session{StartHour: 9, StartMin: 30, EndHour: 16, EndMin: 0, TimezoneOffset: -4}
}

// InWindow reports whether t falls within the session's [start, end) window in
// the session's timezone.
func (s Session) InWindow(t time.Time) bool {
	local := t.Add(time.Duration(s.TimezoneOffset) * time.Hour)
	total := local.Hour()*60 + local.Minute()
	start := s.StartHour*60 + s.StartMin
	end := s.EndHour*60 + s.EndMin
	return total >= start && total < end
}

// DayKey returns the session-local calendar day for t (for day-scoped counters).
func (s Session) DayKey(t time.Time) string {
	return t.Add(time.Duration(s.TimezoneOffset) * time.Hour).Format("2006-01-02")
}
