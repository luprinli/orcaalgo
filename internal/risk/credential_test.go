package risk

import (
	"os"
	"testing"
)

func TestEnvVault_APIKey(t *testing.T) {
	os.Setenv("ALPACA_API_KEY", "test-key-12345")
	defer os.Unsetenv("ALPACA_API_KEY")

	v := &EnvVault{}
	key, err := v.APIKey("alpaca")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "test-key-12345" {
		t.Errorf("expected 'test-key-12345', got %q", key)
	}
}

func TestEnvVault_MissingKey(t *testing.T) {
	os.Unsetenv("ALPACA_API_KEY")
	v := &EnvVault{}
	_, err := v.APIKey("alpaca")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestEnvVault_UnknownBroker(t *testing.T) {
	v := &EnvVault{}
	_, err := v.APIKey("unknown-broker")
	if err == nil {
		t.Error("expected error for unknown broker")
	}
}

func TestEnvVault_ValidateScope(t *testing.T) {
	os.Setenv("ALPACA_API_KEY", "pkey-valid-scope-12345")
	defer os.Unsetenv("ALPACA_API_KEY")

	v := &EnvVault{}
	err := v.ValidateScope("alpaca")
	if err != nil {
		t.Errorf("expected valid scope, got error: %v", err)
	}
}

func TestEnvVault_ValidateScopeTooShort(t *testing.T) {
	os.Setenv("ALPACA_API_KEY", "short")
	defer os.Unsetenv("ALPACA_API_KEY")

	v := &EnvVault{}
	err := v.ValidateScope("alpaca")
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestEncryptedFileVault_SetAndGet(t *testing.T) {
	path := t.TempDir() + "/vault.enc"
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	v, err := NewEncryptedFileVault(path, masterKey)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	err = v.SetAPIKey("alpaca", "test-api-key-1234567890")
	if err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	key, err := v.APIKey("alpaca")
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}
	if key != "test-api-key-1234567890" {
		t.Errorf("expected 'test-api-key-1234567890', got %q", key)
	}
}

func TestEncryptedFileVault_Persistence(t *testing.T) {
	path := t.TempDir() + "/vault.enc"
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	v1, err := NewEncryptedFileVault(path, masterKey)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}
	v1.SetAPIKey("alpaca", "persistent-key")

	v2, err := NewEncryptedFileVault(path, masterKey)
	if err != nil {
		t.Fatalf("failed to reopen vault: %v", err)
	}

	key, err := v2.APIKey("alpaca")
	if err != nil {
		t.Fatalf("failed to read persisted key: %v", err)
	}
	if key != "persistent-key" {
		t.Errorf("expected 'persistent-key', got %q", key)
	}
}

func TestCheckKeyRotation(t *testing.T) {
	os.Setenv("ALPACA_API_KEY", "test-key-12345")
	defer os.Unsetenv("ALPACA_API_KEY")

	v := &EnvVault{}
	record := CheckKeyRotation(v, "alpaca")
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record.Broker != "alpaca" {
		t.Errorf("expected broker 'alpaca', got %q", record.Broker)
	}
	if record.AgeDays <= 0 {
		t.Errorf("expected positive age, got %d", record.AgeDays)
	}
}
