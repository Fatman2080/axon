package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HyperliquidClient struct {
	baseURL string
	client  *http.Client
}

type HyperliquidAccountData struct {
	PublicKey     string
	AccountValue  float64
	UnrealizedPNL float64
	LastSyncAt    string
	Source        string
}

func newHyperliquidClient(baseURL string) *HyperliquidClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.hyperliquid.xyz"
	}
	return &HyperliquidClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *HyperliquidClient) FetchAccountData(publicKey string) (HyperliquidAccountData, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	now := time.Now().UTC().Format(time.RFC3339)
	result := HyperliquidAccountData{
		PublicKey:  publicKey,
		LastSyncAt: now,
		Source:     "hyperliquid",
	}

	statePayload := map[string]any{
		"type": "clearinghouseState",
		"user": publicKey,
	}
	stateResp := map[string]any{}
	if err := c.postInfo(statePayload, &stateResp); err != nil {
		return result, err
	}
	result.AccountValue = parseFloatField(stateResp, "marginSummary", "accountValue")
	if result.AccountValue == 0 {
		result.AccountValue = parseFloatField(stateResp, "crossMarginSummary", "accountValue")
	}
	assetPositions, _ := stateResp["assetPositions"].([]any)
	for _, item := range assetPositions {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pos, _ := mapped["position"].(map[string]any)
		result.UnrealizedPNL += readAnyFloat(pos["unrealizedPnl"])
	}

	return result, nil
}

func (c *HyperliquidClient) FetchPositions(publicKey string) ([]VaultPosition, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	statePayload := map[string]any{
		"type": "clearinghouseState",
		"user": publicKey,
	}
	stateResp := map[string]any{}
	if err := c.postInfo(statePayload, &stateResp); err != nil {
		return nil, err
	}

	positions := make([]VaultPosition, 0)
	assetPositions, _ := stateResp["assetPositions"].([]any)
	for _, item := range assetPositions {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pos, _ := mapped["position"].(map[string]any)
		if pos == nil {
			continue
		}
		size := readAnyFloat(pos["szi"])
		if size == 0 {
			continue
		}
		positions = append(positions, VaultPosition{
			Coin:            readStringField(pos, "coin"),
			Size:            size,
			EntryPrice:      readAnyFloat(pos["entryPx"]),
			MarkPrice:       readAnyFloat(pos["markPx"]),
			UnrealizedPnl:   readAnyFloat(pos["unrealizedPnl"]),
			ReturnOnEquity:  readAnyFloat(pos["returnOnEquity"]),
			PositionValue:   readAnyFloat(pos["positionValue"]),
			Leverage:        readLeverageField(mapped),
			LiquidationPrice: readAnyFloat(pos["liquidationPx"]),
		})
	}
	return positions, nil
}

func (c *HyperliquidClient) FetchUserFills(publicKey string) ([]VaultFill, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	payload := map[string]any{
		"type": "userFills",
		"user": publicKey,
	}
	var rawFills []map[string]any
	if err := c.postInfo(payload, &rawFills); err != nil {
		return nil, err
	}

	fills := make([]VaultFill, 0, len(rawFills))
	for _, f := range rawFills {
		fills = append(fills, VaultFill{
			Coin:          readStringField(f, "coin"),
			Side:          readStringField(f, "side"),
			Size:          readAnyFloat(f["sz"]),
			Price:         readAnyFloat(f["px"]),
			Time:          int64(readAnyFloat(f["time"])),
			Fee:           readAnyFloat(f["fee"]),
			ClosedPnl:     readAnyFloat(f["closedPnl"]),
			Hash:          readStringField(f, "hash"),
			StartPosition: readAnyFloat(f["startPosition"]),
			Direction:     readStringField(f, "dir"),
		})
	}
	return fills, nil
}

func readStringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func readLeverageField(m map[string]any) float64 {
	lev, ok := m["leverage"].(map[string]any)
	if !ok {
		return 0
	}
	return readAnyFloat(lev["value"])
}

func (c *HyperliquidClient) postInfo(payload map[string]any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.baseURL+"/info", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("hyperliquid api status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func parseFloatField(root map[string]any, field string, sub string) float64 {
	obj, ok := root[field].(map[string]any)
	if !ok {
		return 0
	}
	return readAnyFloat(obj[sub])
}

func readAnyFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}
