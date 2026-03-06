package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type tokenClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
}

func issueToken(secret string, sub string, role string, ttl time.Duration) (string, error) {
	claims := tokenClaims{
		Sub:  sub,
		Role: role,
		Exp:  time.Now().Add(ttl).Unix(),
		Iat:  time.Now().Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	part := base64.RawURLEncoding.EncodeToString(payload)
	sig := signPart(secret, part)
	return part + "." + sig, nil
}

func parseToken(secret string, token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	if signPart(secret, parts[0]) != parts[1] {
		return nil, errors.New("invalid token signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	claims := &tokenClaims{}
	if err := json.Unmarshal(raw, claims); err != nil {
		return nil, err
	}
	if claims.Exp < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return claims, nil
}

func signPart(secret string, part string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(part))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashPasswordStrong(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(password string, storedHash string) bool {
	trimmed := strings.TrimSpace(storedHash)
	return bcrypt.CompareHashAndPassword([]byte(trimmed), []byte(password)) == nil
}

func parseSIWEMessage(message string) (wallet string, nonce string, err error) {
	walletRegex := regexp.MustCompile(`0x[a-fA-F0-9]{40}`)
	wallet = walletRegex.FindString(message)
	if wallet == "" {
		return "", "", fmt.Errorf("wallet address not found in SIWE message")
	}
	wallet = strings.ToLower(wallet)

	nonceRegex := regexp.MustCompile(`(?m)^Nonce:\s*([A-Za-z0-9]+)$`)
	matches := nonceRegex.FindStringSubmatch(message)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("nonce not found in SIWE message")
	}
	nonce = matches[1]
	return wallet, nonce, nil
}

func parseBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}
