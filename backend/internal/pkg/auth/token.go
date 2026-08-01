package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	UserID    string `json:"uid"`
	OpenID    string `json:"openid"`
	ExpiresAt int64  `json:"exp"`
}

func SignToken(secret string, ttl time.Duration, userID, openid string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("token secret required")
	}
	claims := Claims{
		UserID:    strings.TrimSpace(userID),
		OpenID:    strings.TrimSpace(openid),
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(secret, body)
	return body + "." + signature, nil
}

func ParseToken(secret, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}
	body := parts[0]
	signature := parts[1]
	if subtle.ConstantTimeCompare([]byte(sign(secret, body)), []byte(signature)) != 1 {
		return Claims{}, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.OpenID) == "" {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
