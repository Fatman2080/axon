package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type RuntimeConfig struct {
	AppBaseURL string `json:"appBaseUrl"`
	Server     struct {
		Port        int    `json:"port"`
		TokenSecret string `json:"tokenSecret"`
	} `json:"server"`
	Storage struct {
		DBPath string `json:"dbPath"`
	} `json:"storage"`
	AgentPool struct {
		FixedKey string `json:"fixedKey"`
	} `json:"agentPool"`
	Hyperliquid struct {
		BaseURL string `json:"baseURL"`
	} `json:"hyperliquid"`
	Frontend struct {
		Mode    string `json:"mode"`
		Release struct {
			WWWDistDir   string `json:"wwwDistDir"`
			AdminDistDir string `json:"adminDistDir"`
		} `json:"release"`
		Dev struct {
			WWWDevServer   string `json:"wwwDevServer"`
			AdminDevServer string `json:"adminDevServer"`
		} `json:"dev"`
	} `json:"frontend"`
	Log struct {
		Dir      string `json:"dir"`      // directory for log files; empty = file logging disabled
		Level    string `json:"level"`    // "debug"|"info"|"warn"|"error", default "info"
		MaxSize  int    `json:"maxSize"`  // MB per file before rotation, default 100
		MaxFiles int    `json:"maxFiles"` // compressed rotated files to keep, default 10
		Console  bool   `json:"console"` // also write to stderr, default true
	} `json:"log"`
}

func defaultRuntimeConfig() RuntimeConfig {
	cfg := RuntimeConfig{}
	cfg.AppBaseURL = "http://localhost:9333"
	cfg.Server.Port = 9333
	cfg.Server.TokenSecret = "openfi-dev-secret"
	cfg.Storage.DBPath = "../data/openfi.db"
	cfg.AgentPool.FixedKey = "01234567890123456789012345678901"
	cfg.Hyperliquid.BaseURL = "https://api.hyperliquid.xyz"
	cfg.Frontend.Mode = "release"
	cfg.Frontend.Release.WWWDistDir = "../../frontend-www/dist"
	cfg.Frontend.Release.AdminDistDir = "../../frontend-admin/dist"
	cfg.Frontend.Dev.WWWDevServer = "http://127.0.0.1:9334"
	cfg.Frontend.Dev.AdminDevServer = "http://127.0.0.1:9335"
	cfg.Log.Level = "info"
	cfg.Log.MaxSize = 100
	cfg.Log.MaxFiles = 10
	cfg.Log.Console = true
	return cfg
}

func loadRuntimeConfig(path string) (RuntimeConfig, error) {
	cfg := defaultRuntimeConfig()

	raw, err := os.ReadFile(path)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return RuntimeConfig{}, fmt.Errorf("invalid config json: %w", err)
	}

	configDir := filepath.Dir(path)
	cfg.Storage.DBPath = resolveConfigPath(configDir, cfg.Storage.DBPath)
	cfg.Frontend.Release.WWWDistDir = resolveConfigPath(configDir, cfg.Frontend.Release.WWWDistDir)
	cfg.Frontend.Release.AdminDistDir = resolveConfigPath(configDir, cfg.Frontend.Release.AdminDistDir)

	if cfg.Server.Port <= 0 {
		return RuntimeConfig{}, errors.New("server.port must be greater than 0")
	}
	if strings.TrimSpace(cfg.Server.TokenSecret) == "" {
		return RuntimeConfig{}, errors.New("server.tokenSecret is required")
	}
	if _, err := deriveAES256Key(cfg.AgentPool.FixedKey); err != nil {
		return RuntimeConfig{}, fmt.Errorf("agentPool.fixedKey invalid: %w", err)
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Frontend.Mode))
	if mode == "" {
		mode = "release"
	}
	if mode != "release" && mode != "dev" {
		return RuntimeConfig{}, errors.New("frontend.mode must be release or dev")
	}
	cfg.Frontend.Mode = mode

	cfg.AppBaseURL = strings.TrimRight(strings.TrimSpace(cfg.AppBaseURL), "/")
	if cfg.AppBaseURL == "" {
		cfg.AppBaseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}
	parsedAppBaseURL, err := url.Parse(cfg.AppBaseURL)
	if err != nil || parsedAppBaseURL.Scheme == "" || parsedAppBaseURL.Host == "" {
		return RuntimeConfig{}, errors.New("appBaseUrl must be a valid absolute url")
	}

	return cfg, nil
}

func resolveConfigPath(configDir string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Clean(filepath.Join(configDir, trimmed))
}
