package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type TelegramNotifier struct {
	botToken string
	devChat  string
	opsChat  string
	baseURL  string
	enabled  bool
}

func NewTelegramNotifier(chatIDs ...string) *TelegramNotifier {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	devChat := os.Getenv("TELEGRAM_DEV_CHAT")
	opsChat := os.Getenv("TELEGRAM_OPS_CHAT")

	if len(chatIDs) == 0 && devChat == "" && opsChat == "" {
		legacy := os.Getenv("TELEGRAM_CHAT_ID")
		if legacy != "" {
			devChat = legacy
			opsChat = legacy
		}
	}

	return &TelegramNotifier{
		botToken: token,
		devChat:  devChat,
		opsChat:  opsChat,
		baseURL:  "https://api.telegram.org",
		enabled:  token != "" && (devChat != "" || opsChat != ""),
	}
}

func (t *TelegramNotifier) Name() string {
	return "telegram"
}

func (t *TelegramNotifier) IsEnabled() bool {
	return t.enabled
}

func (t *TelegramNotifier) Send(event Event) error {
	if !t.enabled {
		return nil
	}

	switch event.Level {
	case LevelInfo:
		return nil
	case LevelCritical:
		text := "\xe2\x9a\xa0\xef\xb8\x8f *CRITICAL*: " + event.Title + "\n" + event.Message
		if event.Details != "" {
			text += fmt.Sprintf("\n```\n%s\n```", event.Details)
		}
		for _, chatID := range t.criticalChats() {
			t.sendMessage(chatID, text)
		}
		log.Printf("telegram notifier: sent CRITICAL event to %d chats", len(t.criticalChats()))
	case LevelWarning:
		text := "\xe2\x9a\xa0 *WARNING*: " + event.Title + "\n" + event.Message
		if event.Details != "" {
			text += fmt.Sprintf("\n```\n%s\n```", event.Details)
		}
		for _, chatID := range t.warningChats() {
			t.sendMessage(chatID, text)
		}
		log.Printf("telegram notifier: sent %s event to %d chats", event.Type, len(t.warningChats()))
	}

	return nil
}

func (t *TelegramNotifier) warningChats() []string {
	if t.devChat != "" {
		return []string{t.devChat}
	}
	if t.opsChat != "" {
		return []string{t.opsChat}
	}
	return nil
}

func (t *TelegramNotifier) criticalChats() []string {
	var chats []string
	seen := map[string]bool{}
	if t.opsChat != "" && !seen[t.opsChat] {
		chats = append(chats, t.opsChat)
		seen[t.opsChat] = true
	}
	if t.devChat != "" && !seen[t.devChat] {
		chats = append(chats, t.devChat)
		seen[t.devChat] = true
	}
	return chats
}

func (t *TelegramNotifier) sendMessage(chatID, text string) {
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.botToken)
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("telegram send error: %v", err)
		return
	}
	resp.Body.Close()
}
