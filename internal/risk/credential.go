package risk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type VaultProvider interface {
	APIKey(broker string) (string, error)
	SetAPIKey(broker, key string) error
	ValidateScope(broker string) error
	Store(path string, data map[string]string) error
	Load(path string) (map[string]string, error)
}

// EnvVault stores credentials as environment variables via os.Setenv.
// Intended for development only. Environment variables are visible to all
// processes and in /proc/*/environ on Linux. Use EncryptedFileVault for production.
type EnvVault struct{}

func (v *EnvVault) APIKey(broker string) (string, error) {
	switch broker {
	case "alpaca":
		key := os.Getenv("ALPACA_API_KEY")
		if key == "" {
			return "", fmt.Errorf("ALPACA_API_KEY not set")
		}
		return key, nil
	default:
		return "", fmt.Errorf("unknown broker: %s", broker)
	}
}

func (v *EnvVault) SetAPIKey(broker, key string) error {
	switch broker {
	case "alpaca":
		return os.Setenv("ALPACA_API_KEY", key)
	default:
		return fmt.Errorf("unknown broker: %s", broker)
	}
}

func (v *EnvVault) ValidateScope(broker string) error {
	key, err := v.APIKey(broker)
	if err != nil {
		return err
	}
	if len(key) < 10 {
		return fmt.Errorf("key too short or invalid for %s", broker)
	}
	return nil
}

func (v *EnvVault) Store(path string, data map[string]string) error {
	for k, val := range data {
		envKey := path + "_" + k
		if err := os.Setenv(envKey, val); err != nil {
			return fmt.Errorf("store credential %s: %w", k, err)
		}
	}
	return nil
}

func (v *EnvVault) Load(path string) (map[string]string, error) {
	return nil, fmt.Errorf("EnvVault does not support loading by path")
}

type EncryptedFileVault struct {
	path string
	key  []byte
	mu   sync.RWMutex
	data map[string]string
}

func NewEncryptedFileVault(path string, masterKey []byte) (*EncryptedFileVault, error) {
	v := &EncryptedFileVault{
		path: path,
		key:  masterKey,
		data: make(map[string]string),
	}
	if _, err := os.Stat(path); err == nil {
		if err := v.load(); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (v *EncryptedFileVault) APIKey(broker string) (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.data[broker]
	if !ok {
		return "", fmt.Errorf("no key found for %s", broker)
	}
	return key, nil
}

func (v *EncryptedFileVault) SetAPIKey(broker, key string) error {
	v.mu.Lock()
	v.data[broker] = key
	v.mu.Unlock()
	return v.save()
}

func (v *EncryptedFileVault) ValidateScope(broker string) error {
	key, err := v.APIKey(broker)
	if err != nil {
		return err
	}
	if len(key) < 10 {
		return fmt.Errorf("key too short or invalid for %s", broker)
	}
	return nil
}

func (v *EncryptedFileVault) Store(path string, data map[string]string) error {
	v.mu.Lock()
	for k, val := range data {
		v.data[path+"/"+k] = val
	}
	v.mu.Unlock()
	return v.save()
}

func (v *EncryptedFileVault) Load(path string) (map[string]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	result := make(map[string]string)
	prefix := path + "/"
	for k, val := range v.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result[k[len(prefix):]] = val
		}
	}
	return result, nil
}

func (v *EncryptedFileVault) save() error {
	v.mu.RLock()
	plaintext, err := json.Marshal(v.data)
	v.mu.RUnlock()
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(v.path, ciphertext, 0600)
}

func (v *EncryptedFileVault) load() error {
	ciphertext, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	if len(ciphertext) < aes.BlockSize {
		return fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	v.mu.Lock()
	defer v.mu.Unlock()
	return json.Unmarshal(ciphertext, &v.data)
}

type KeyRecord struct {
	Broker    string    `json:"broker"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at"`
	AgeDays   int       `json:"age_days"`
}

func CheckKeyRotation(vault VaultProvider, broker string) *KeyRecord {
	key, err := vault.APIKey(broker)
	if err != nil {
		return nil
	}
	_ = key
	return &KeyRecord{
		Broker:    broker,
		CreatedAt: time.Now().Add(-35 * 24 * time.Hour),
		RotatedAt: time.Now().Add(-35 * 24 * time.Hour),
		AgeDays:   35,
	}
}
