package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/breaker"
)

type AlertLevel string

const (
	Info     AlertLevel = "INFO"
	Warning  AlertLevel = "WARNING"
	Critical AlertLevel = "CRITICAL"
)

type Alert struct {
	Level   AlertLevel `json:"level"`
	Message string     `json:"message"`
	Time    string     `json:"time"`
}

type TelegramBot struct {
	token          string
	chatID         string
	baseURL        string
	enabled        bool
	circuitBreaker *breaker.CircuitBreaker
}

func NewTelegramBot() *TelegramBot {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	return &TelegramBot{
		token:          token,
		chatID:         chatID,
		baseURL:        "https://api.telegram.org",
		enabled:        token != "" && chatID != "",
		circuitBreaker: breaker.NewCircuitBreaker(3, 30*time.Second),
	}
}

func (tb *TelegramBot) SendAlert(level AlertLevel, message string) {
	if !tb.enabled {
		return
	}

	alert := Alert{
		Level:   level,
		Message: message,
		Time:    time.Now().Format(time.RFC3339),
	}

	switch level {
	case Critical:
		tb.sendMessage(fmt.Sprintf("\xe2\x9a\xa0\xef\xb8\x8f *CRITICAL*: %s", message))
	case Warning:
		tb.sendMessage(fmt.Sprintf("\xe2\x9a\xa0 *WARNING*: %s", message))
	default:
		tb.sendMessage(message)
	}

	slog.Info("telegram alert", "level", string(level), "message", message, "component", "telegram")
	_ = alert
}

func (tb *TelegramBot) Send(level AlertLevel, msgType string, args ...interface{}) {
	var message string
	switch msgType {
	case "KillSwitchTriggered":
		message = fmt.Sprintf("KILL SWITCH: %v", args...)
	case "RegimeChanged":
		message = fmt.Sprintf("Regime: %v", args...)
	case "ConsistencyOutlier":
		message = fmt.Sprintf("Daily P&L exceeds threshold: %v", args...)
	case "CredentialExpiry":
		message = fmt.Sprintf("API key expires: %v", args...)
	case "MemoryScanDetected":
		message = fmt.Sprintf("Unauthorized memory access detected: %v", args...)
	default:
		message = fmt.Sprintf("%s: %v", msgType, args)
	}
	tb.SendAlert(level, message)
}

func (tb *TelegramBot) sendMessage(text string) {
	if !tb.circuitBreaker.Allow() {
		slog.Warn("circuit breaker open, dropping message", "component", "telegram")
		return
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", tb.baseURL, tb.token)
	payload := map[string]string{
		"chat_id":    tb.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		tb.circuitBreaker.RecordFailure()
		slog.Error("telegram send error", "error", err, "component", "telegram")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		tb.circuitBreaker.RecordFailure()
	} else {
		tb.circuitBreaker.RecordSuccess()
	}
}
