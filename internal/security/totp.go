package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

type TOTPSecret struct {
	Secret    string `json:"secret"`
	Issuer    string `json:"issuer"`
	Account   string `json:"account"`
	Digits    int    `json:"digits"`
	Period    int64  `json:"period"`
}

func GenerateTOTPSecret(issuer, account string) *TOTPSecret {
	key := make([]byte, 20)
	rand.Read(key)
	return &TOTPSecret{
		Secret:  base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(key),
		Issuer:  issuer,
		Account: account,
		Digits:  6,
		Period:  30,
	}
}

func ValidateTOTP(secret string, code string) (bool, error) {
	if len(code) != 6 {
		return false, fmt.Errorf("TOTP code must be 6 digits")
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false, fmt.Errorf("decode TOTP secret: %w", err)
	}
	now := time.Now().Unix()
	window := int64(1)

	for t := now - window; t <= now+window; t++ {
		expected := computeTOTP(key, t, 30)
		if expected == code {
			return true, nil
		}
	}
	return false, nil
}

func computeTOTP(key []byte, timestamp int64, period int64) string {
	counter := timestamp / period
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	binary := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	otp := int(binary) % int(math.Pow10(6))

	return fmt.Sprintf("%06d", otp)
}

func GetTOTPURI(secret *TOTPSecret) string {
	issuer := strings.ReplaceAll(secret.Issuer, " ", "%20")
	account := strings.ReplaceAll(secret.Account, " ", "%20")
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer, account, secret.Secret, issuer, secret.Digits, secret.Period)
}
