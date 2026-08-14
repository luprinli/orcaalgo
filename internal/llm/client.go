package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/breaker"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type Client struct {
	provider   Provider
	apiKey     string
	baseURL    string
	httpClient *http.Client
	breaker    *breaker.CircuitBreaker
}

func NewClient(provider Provider) *Client {
	return NewClientWithKey(provider, os.Getenv(string(provider)+"_API_KEY"), "")
}

// NewClientWithKey builds a client with an explicit API key and base URL
// (BYOK). An empty base URL falls back to the provider default. This is the
// constructor used for per-user keys; NewClient remains the env-var fallback
// for self-hosted/default deployments.
func NewClientWithKey(provider Provider, key, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL(provider)
	}
	return &Client{
		provider:   provider,
		apiKey:     key,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		breaker:    breaker.NewCircuitBreaker(5, 30*time.Second),
	}
}

func defaultBaseURL(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	case ProviderOllama:
		return "http://localhost:11434"
	default:
		return ""
	}
}

func (c *Client) Chat(req *ChatRequest) (*ChatResponse, error) {
	if !c.breaker.Allow() {
		return nil, fmt.Errorf("llm circuit open")
	}
	resp, err := c.doChat(req)
	if err != nil {
		c.breaker.RecordFailure()
		return nil, err
	}
	c.breaker.RecordSuccess()
	return resp, nil
}

func (c *Client) doChat(req *ChatRequest) (*ChatResponse, error) {
	var url string
	var bodyData []byte
	var err error

	switch c.provider {
	case ProviderAnthropic:
		url = c.baseURL + "/v1/messages"
		systemContent := ""
		var messages []map[string]interface{}
		for _, m := range req.Messages {
			if m.Role == "system" {
				systemContent = m.Content
			} else {
				messages = append(messages, map[string]interface{}{
					"role":    m.Role,
					"content": m.Content,
				})
			}
		}
		anthropicReq := map[string]interface{}{
			"model":        req.Model,
			"max_tokens":   req.MaxTokens,
			"temperature":  req.Temperature,
			"messages":     messages,
		}
		if systemContent != "" {
			anthropicReq["system"] = systemContent
		}
		bodyData, err = json.Marshal(anthropicReq)
	case ProviderOllama:
		url = c.baseURL + "/api/chat"
		bodyData, err = json.Marshal(req)
	default:
		url = c.baseURL + "/chat/completions"
		bodyData, err = json.Marshal(req)
	}
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyData))
	if err != nil { return nil, fmt.Errorf("create request: %w", err) }
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		switch c.provider {
		case ProviderAnthropic:
			httpReq.Header.Set("x-api-key", c.apiKey)
			httpReq.Header.Set("anthropic-version", "2023-06-01")
		default:
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil { return nil, fmt.Errorf("llm request: %w", err) }
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil { return nil, fmt.Errorf("read response: %w", err) }
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm error %d: %s", resp.StatusCode, string(data))
	}

	var chatResp ChatResponse
	switch c.provider {
	case ProviderAnthropic:
		var ar struct {
			ID      string `json:"id"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(data, &ar); err != nil {
			return nil, fmt.Errorf("llm parse: %w", err)
		}
		chatResp.ID = ar.ID
		if len(ar.Content) > 0 {
			chatResp.Choices = append(chatResp.Choices, struct {
				Message Message `json:"message"`
			}{Message: Message{Role: "assistant", Content: ar.Content[0].Text}})
		}
		chatResp.Usage.PromptTokens = ar.Usage.InputTokens
		chatResp.Usage.CompletionTokens = ar.Usage.OutputTokens
	default:
		if err := json.Unmarshal(data, &chatResp); err != nil {
			return nil, fmt.Errorf("llm parse: %w", err)
		}
	}
	return &chatResp, nil
}