package api

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/metrics"
)

func (s *Server) getLiveMetrics(c *gin.Context) {
	windowStr := c.DefaultQuery("window", "30d")

	windowDays := 30
	if w := c.Query("window"); w != "" {
		w = w[:len(w)-1]
		if v, err := strconv.Atoi(w); err == nil && v > 0 {
			windowDays = v
		}
	}

	snap := metrics.PerformanceSnapshot{Timestamp: time.Now().UTC()}

	snap.Equity = 0
	snap.Balance = 0

	if s.adapter != nil {
		acct, err := s.adapter.GetAccount(c.Request.Context())
		if err == nil {
			snap.Equity = acct.Equity.Float64()
			snap.Balance = acct.Balance.Float64()
			snap.DailyPnL = acct.DailyPL
			if acct.Balance.Float64() > 0 {
				snap.DailyPnLPct = acct.DailyPL / acct.Balance.Float64() * 100.0
			}
		}
	}

	if s.killSwitch != nil {
		_, _, _ = s.killSwitch.Status()
	}

	_ = windowStr
	_ = windowDays

	c.JSON(200, snap)
}

func (s *Server) getLiveEquity(c *gin.Context) {
	windowStr := c.DefaultQuery("window", "90d")

	windowDays := 90
	if w := c.Query("window"); w != "" {
		w = w[:len(w)-1]
		if v, err := strconv.Atoi(w); err == nil && v > 0 {
			windowDays = v
		}
	}

	_ = windowStr
	_ = windowDays

	equity := []metrics.MetricEquityPoint{}

	c.JSON(200, equity)
}

func (s *Server) getLiveTrades(c *gin.Context) {
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	_ = page
	_ = limit

	c.JSON(200, gin.H{"trades": []metrics.TradeSummary{}, "total": 0})
}

func (s *Server) getLiveDailyReturns(c *gin.Context) {
	c.JSON(200, []metrics.DailyReturn{})
}

func (s *Server) getLiveRollingSharpe(c *gin.Context) {
	windowStr := c.DefaultQuery("window", "30d")
	windowDays := 30
	if w := c.Query("window"); w != "" {
		w = w[:len(w)-1]
		if v, err := strconv.Atoi(w); err == nil && v > 0 {
			windowDays = v
		}
	}
	_ = windowStr
	_ = windowDays

	c.JSON(200, []metrics.RollingMetric{})
}

var _ = errors.New
var _ = slog.Info
