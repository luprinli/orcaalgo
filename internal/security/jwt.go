package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type Claims struct {
	UserID   string   `json:"sub"`
	Username string   `json:"name"`
	Roles    []string `json:"roles"`
	IssuedAt int64    `json:"iat"`
	Expires  int64    `json:"exp"`
	TokenID  string   `json:"jti"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func GenerateTokenPair(userID, username string, roles []string, secret []byte, ttl time.Duration) (*TokenPair, error) {
	now := time.Now().Unix()
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return nil, fmt.Errorf("generate token ID: %w", err)
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		IssuedAt: now,
		Expires:  now + int64(ttl.Seconds()),
		TokenID:  base64.RawURLEncoding.EncodeToString(jti),
	}

	accessToken, err := signToken(claims, secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken := make([]byte, 32)
	if _, err := rand.Read(refreshToken); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: base64.RawURLEncoding.EncodeToString(refreshToken),
		ExpiresIn:    int64(ttl.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func ValidateToken(tokenString string, secret []byte) (*Claims, error) {
	parts := splitToken(tokenString)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := computeHMAC(signingInput, secret)
	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	if !hmacEqual(expectedSig, actualSig) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	if time.Now().Unix() > claims.Expires {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func signToken(claims Claims, secret []byte) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := header + "." + payload
	sig := computeHMAC(signingInput, secret)
	signature := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + signature, nil
}

func computeHMAC(input string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func splitToken(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
