package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const (
	testFixedKeyUTF8 = "choHivaiS7ou4Dahbooghae4Ahighae1"
	testFixedKeyHex  = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

func TestDecryptPrivateKeyPayload_EncryptedDataHexJSON(t *testing.T) {
	expected := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	encrypted := encryptKeyList(t, expected, testFixedKeyUTF8)
	wrapped := map[string]any{
		"status":         "ok",
		"format":         "AES-GCM-256",
		"encrypted_data": encrypted,
		"count":          len(expected),
	}
	raw, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	actual, err := decryptPrivateKeyPayload(string(raw), testFixedKeyUTF8)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected result: %+v", actual)
	}
}

func TestDecryptPrivateKeyPayload_EncryptedDataWithWhitespace(t *testing.T) {
	expected := []string{"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
	encrypted := encryptKeyList(t, expected, testFixedKeyUTF8)
	encryptedWithWhitespace := strings.Join(splitBy(encrypted, 96), "\n")
	wrapped := map[string]any{
		"status":         "ok",
		"format":         "AES-GCM-256",
		"encrypted_data": encryptedWithWhitespace,
		"count":          len(expected),
	}
	raw, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	actual, err := decryptPrivateKeyPayload(string(raw), testFixedKeyUTF8)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected result: %+v", actual)
	}
}

func TestDecryptPrivateKeyPayload_SupportsJSONStringInput(t *testing.T) {
	expected := []string{"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}
	encoded := encryptKeyList(t, expected, testFixedKeyUTF8)
	payload, err := json.Marshal(encoded)
	if err != nil {
		t.Fatalf("marshal string payload failed: %v", err)
	}

	actual, err := decryptPrivateKeyPayload(string(payload), testFixedKeyUTF8)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected result: %+v", actual)
	}
}

func TestDecryptPrivateKeyPayload_HexKey(t *testing.T) {
	expected := []string{"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	encrypted := encryptKeyList(t, expected, testFixedKeyHex)
	wrapped := map[string]any{
		"status":         "ok",
		"format":         "AES-GCM-256",
		"encrypted_data": encrypted,
		"count":          len(expected),
	}
	raw, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	actual, err := decryptPrivateKeyPayload(string(raw), testFixedKeyHex)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected result: %+v", actual)
	}
}

func TestDeriveAES256Key_UTF8Fallback(t *testing.T) {
	key, err := deriveAES256Key(testFixedKeyUTF8)
	if err != nil {
		t.Fatalf("derive key failed: %v", err)
	}
	if !bytes.Equal(key, []byte(testFixedKeyUTF8)) {
		t.Fatalf("unexpected key bytes")
	}
}

func TestDeriveAES256Key_HexPreferred(t *testing.T) {
	key, err := deriveAES256Key(testFixedKeyHex)
	if err != nil {
		t.Fatalf("derive key failed: %v", err)
	}
	expected, err := hex.DecodeString(testFixedKeyHex)
	if err != nil {
		t.Fatalf("decode expected hex key failed: %v", err)
	}
	if !bytes.Equal(key, expected) {
		t.Fatalf("unexpected key bytes")
	}
}

func encryptKeyList(t *testing.T, keys []string, fixedKey string) string {
	t.Helper()
	key, err := deriveAES256Key(fixedKey)
	if err != nil {
		t.Fatalf("derive key failed: %v", err)
	}
	nonce := make([]byte, 16)
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	raw, err := json.Marshal(keys)
	if err != nil {
		t.Fatalf("marshal keys failed: %v", err)
	}
	ciphertextWithTag, err := aesGCMEncrypt(key, nonce, raw)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if len(ciphertextWithTag) <= 16 {
		t.Fatalf("ciphertext too short")
	}
	tag := ciphertextWithTag[len(ciphertextWithTag)-16:]
	ciphertext := ciphertextWithTag[:len(ciphertextWithTag)-16]
	combined := append(append(append([]byte{}, nonce...), tag...), ciphertext...)
	return hex.EncodeToString(combined)
}

func splitBy(input string, size int) []string {
	if size <= 0 || len(input) <= size {
		return []string{input}
	}
	out := make([]string, 0, (len(input)+size-1)/size)
	for start := 0; start < len(input); start += size {
		end := start + size
		if end > len(input) {
			end = len(input)
		}
		out = append(out, input[start:end])
	}
	return out
}
