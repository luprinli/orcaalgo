package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type MonitoringHandler struct {
	prometheusURL  string
	alertmanagerURL string
}

func NewMonitoringHandler(prometheusURL, alertmanagerURL string) *MonitoringHandler {
	if prometheusURL == "" {
		prometheusURL = "http://localhost:9090"
	}
	if alertmanagerURL == "" {
		alertmanagerURL = "http://localhost:9093"
	}
	return &MonitoringHandler{
		prometheusURL:  strings.TrimRight(prometheusURL, "/"),
		alertmanagerURL: strings.TrimRight(alertmanagerURL, "/"),
	}
}

func (h *MonitoringHandler) PrometheusQuery(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
		return
	}

	resp, err := http.Get(h.prometheusURL + "/api/v1/query?query=" + query)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "prometheus unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	resp, err := http.Get(h.alertmanagerURL + "/api/v2/alerts")
	if err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	defer resp.Body.Close()

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *MonitoringHandler) SilenceAlert(c *gin.Context) {
	var req struct {
		AlertName string `json:"alert_name"`
		Duration  string `json:"duration"`
		Comment   string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Duration == "" {
		req.Duration = "1h"
	}

	payload := map[string]interface{}{
		"matchers": []map[string]interface{}{
			{"name": "alertname", "value": req.AlertName, "isRegex": false},
		},
		"startsAt":  "now",
		"endsAt":    req.Duration,
		"comment":   req.Comment,
		"createdBy": "orca-dashboard",
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(h.alertmanagerURL+"/api/v2/silences", "application/json", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "alertmanager unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", respBody)
}

func (h *MonitoringHandler) RegisterRoutes(router *gin.RouterGroup) {
	mon := router.Group("/monitoring")
	{
		mon.GET("/prometheus/query", h.PrometheusQuery)
		mon.GET("/alerts", h.GetAlerts)
		mon.POST("/alerts/silence", h.SilenceAlert)
	}
}
