package llm

import (
	"testing"
)

func TestNewClient_OpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-key")
	c := NewClient(ProviderOpenAI)
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.provider != ProviderOpenAI {
		t.Errorf("Expected openai, got %s", c.provider)
	}
	if c.apiKey != "sk-test-key" {
		t.Errorf("Expected sk-test-key, got %s", c.apiKey)
	}
	if c.baseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected openai base URL, got %s", c.baseURL)
	}
}

func TestNewClient_Anthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	c := NewClient(ProviderAnthropic)
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.baseURL != "https://api.anthropic.com" {
		t.Errorf("Expected anthropic base URL, got %s", c.baseURL)
	}
}

func TestNewClient_Ollama(t *testing.T) {
	c := NewClient(ProviderOllama)
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.baseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama base URL, got %s", c.baseURL)
	}
}

func TestNewClient_UnknownProvider(t *testing.T) {
	c := NewClient("unknown")
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.baseURL != "" {
		t.Errorf("Expected empty base URL for unknown provider, got %s", c.baseURL)
	}
}

func TestNewClientWithKey_UsesPassedKeyAndBaseURL(t *testing.T) {
	c := NewClientWithKey(ProviderOpenAI, "sk-byok", "https://proxy.example.com/v1")
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.apiKey != "sk-byok" {
		t.Errorf("Expected sk-byok, got %s", c.apiKey)
	}
	if c.baseURL != "https://proxy.example.com/v1" {
		t.Errorf("Expected custom base URL, got %s", c.baseURL)
	}
}

func TestNewClientWithKey_DefaultBaseURL(t *testing.T) {
	c := NewClientWithKey(ProviderAnthropic, "sk-ant-byok", "")
	if c.baseURL != "https://api.anthropic.com" {
		t.Errorf("Expected default anthropic base URL, got %s", c.baseURL)
	}
}

func TestNewClientWithKey_EmptyKeyAllowed(t *testing.T) {
	c := NewClientWithKey(ProviderOllama, "", "")
	if c.apiKey != "" {
		t.Errorf("Expected empty key (Ollama is keyless), got %s", c.apiKey)
	}
	if c.baseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama base URL, got %s", c.baseURL)
	}
}

func TestChatRequest_Defaults(t *testing.T) {
	req := &ChatRequest{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	if req.Model != "gpt-4" {
		t.Error("Model mismatch")
	}
	if req.Temperature != 0.7 {
		t.Error("Temperature mismatch")
	}
}

func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Analyze this trade",
	}
	if msg.Role != "user" {
		t.Error("Role mismatch")
	}
	if msg.Content != "Analyze this trade" {
		t.Error("Content mismatch")
	}
}

func TestProvider_Constants(t *testing.T) {
	if ProviderOpenAI != "openai" {
		t.Errorf("Expected 'openai', got '%s'", ProviderOpenAI)
	}
	if ProviderAnthropic != "anthropic" {
		t.Errorf("Expected 'anthropic', got '%s'", ProviderAnthropic)
	}
	if ProviderOllama != "ollama" {
		t.Errorf("Expected 'ollama', got '%s'", ProviderOllama)
	}
}
