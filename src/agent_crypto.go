package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

type importPayload struct {
	EncryptedData string `json:"encrypted_data"`
}

func deriveAES256Key(passphrase string) ([]byte, error) {
	key := strings.TrimSpace(passphrase)
	if key == "" {
		return nil, errors.New("empty key")
	}

	if isHexString(key) {
		decoded, err := hex.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("invalid hex key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid hex key length %d, expected 32 bytes", len(decoded))
		}
		return decoded, nil
	}

	raw := []byte(key)
	if len(raw) != 32 {
		return nil, fmt.Errorf("invalid utf-8 key length %d, expected 32 bytes", len(raw))
	}
	return raw, nil
}

func decryptPrivateKeyPayload(input string, fixedKey string) ([]string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errors.New("empty payload")
	}

	encoded := trimmed
	var payload importPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
		if strings.TrimSpace(payload.EncryptedData) != "" {
			encoded = payload.EncryptedData
		}
	} else {
		var raw string
		if err := json.Unmarshal([]byte(trimmed), &raw); err == nil && strings.TrimSpace(raw) != "" {
			encoded = raw
		}
	}

	cleaned := strings.TrimSpace(encoded)
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if len(cleaned) >= 2 && cleaned[0] == '"' && cleaned[len(cleaned)-1] == '"' {
		cleaned = strings.TrimSpace(cleaned[1 : len(cleaned)-1])
	}
	if cleaned == "" {
		return nil, errors.New("empty encrypted data")
	}

	combined, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, errors.New("unsupported encoding, expected hex")
	}
	if len(combined) <= 32 {
		return nil, errors.New("invalid encrypted data")
	}

	key, err := deriveAES256Key(fixedKey)
	if err != nil {
		return nil, err
	}
	nonce := combined[:16]
	tag := combined[16:32]
	ciphertext := combined[32:]
	ciphertextWithTag := append(append(make([]byte, 0, len(ciphertext)+len(tag)), ciphertext...), tag...)

	plaintext, err := aesGCMDecrypt(key, nonce, ciphertextWithTag)
	if err != nil {
		return nil, err
	}
	arr := make([]string, 0)
	if err := json.Unmarshal(plaintext, &arr); err != nil {
		return nil, err
	}
	return normalizePrivateKeys(arr), nil
}

func encryptSecret(plaintext string, fixedKey string) (string, error) {
	key, err := deriveAES256Key(fixedKey)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext, err := aesGCMEncrypt(key, nonce, []byte(plaintext))
	if err != nil {
		return "", err
	}
	combined := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

func decryptSecret(ciphertextBase64 string, fixedKey string) (string, error) {
	key, err := deriveAES256Key(fixedKey)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	if len(raw) <= 12 {
		return "", errors.New("invalid secret payload")
	}
	plaintext, err := aesGCMDecrypt(key, raw[:12], raw[12:])
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func aesGCMEncrypt(key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

func aesGCMDecrypt(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func isHexString(input string) bool {
	if input == "" {
		return false
	}
	for _, c := range input {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func normalizePrivateKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, item := range keys {
		trimmed := strings.TrimSpace(item)
		trimmed = strings.TrimPrefix(trimmed, "0x")
		if trimmed == "" {
			continue
		}
		out = append(out, strings.ToLower(trimmed))
	}
	return out
}

func derivePublicKeyFromPrivateKey(privateKeyHex string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(strings.TrimPrefix(privateKeyHex, "0x")))
	if err != nil {
		return "", err
	}
	key, err := crypto.ToECDSA(raw)
	if err != nil {
		return "", err
	}
	return strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex()), nil
}
