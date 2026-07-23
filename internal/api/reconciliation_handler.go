package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
)

type ReconciliationHandler struct {
	repo         *db.Repository
	fillProvider broker.FillProvider
}

func NewReconciliationHandler(repo *db.Repository) *ReconciliationHandler {
	return &ReconciliationHandler{repo: repo}
}

func (h *ReconciliationHandler) SetFillProvider(fp broker.FillProvider) {
	h.fillProvider = fp
}

func (h *ReconciliationHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/reconciliation/daily", h.DailyReconciliation)
}

type ReconciliationResult struct {
	Date            string          `json:"date"`
	InternalCount   int             `json:"internal_count"`
	Matched         int             `json:"matched"`
	Missing         int             `json:"missing"`
	Extra           int             `json:"extra"`
	PriceDiscrepancies int          `json:"price_discrepancies"`
	Discrepancies   []DiscrepancyDetail `json:"discrepancies,omitempty"`
}

type DiscrepancyDetail struct {
	OrderID    string  `json:"order_id"`
	Internal   float64 `json:"internal_fill"`
	Broker     float64 `json:"broker_fill"`
	DiffBps    float64 `json:"diff_bps"`
}

func (h *ReconciliationHandler) DailyReconciliation(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter required (YYYY-MM-DD)"})
		return
	}

	ctx := c.Request.Context()

	rows, err := h.repo.Pool().Query(ctx,
		`SELECT id, strategy_id, symbol, side, quantity, price, executed_at, broker_order_id
		 FROM trade_executions WHERE executed_at::date = $1::date ORDER BY executed_at`, date,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var internal []db.TradeExecution
	for rows.Next() {
		var exec db.TradeExecution
		if err := rows.Scan(&exec.ID, &exec.StrategyID, &exec.Symbol, &exec.Side,
			&exec.Quantity, &exec.Price, &exec.ExecutedAt, &exec.BrokerOrderID); err != nil {
			continue
		}
		internal = append(internal, exec)
	}

	var brokerFills []broker.TradeFill
	if h.fillProvider != nil {
		fills, fillErr := h.fillProvider.GetFills(ctx, date)
		if fillErr != nil {
			slog.Warn("reconciliation: broker fill query failed", "date", date, "err", fillErr)
		} else {
			brokerFills = fills
		}
	}

	result := MatchReconciliation(internal, brokerFills)
	result.Date = date

	c.JSON(http.StatusOK, result)
}
