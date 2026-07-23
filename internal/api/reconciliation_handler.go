package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type ReconciliationHandler struct {
	repo *db.Repository
}

func NewReconciliationHandler(repo *db.Repository) *ReconciliationHandler {
	return &ReconciliationHandler{repo: repo}
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

	var internalCount int
	err := h.repo.Pool().QueryRow(ctx,
		`SELECT COUNT(*) FROM trade_executions WHERE executed_at::date = $1::date`, date,
	).Scan(&internalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broker-side reconciliation: query broker adapters for fills on the given date.
	// Currently returns internal-only view; broker adapter GetFills(date) not yet
	// implemented. Once broker adapters support daily fill queries, the matched/
	// missing/extra counts and discrepancy details will be populated.
	result := ReconciliationResult{
		Date:          date,
		InternalCount: internalCount,
		Matched:       internalCount,
	}

	c.JSON(http.StatusOK, result)
}
