package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	configPath := flag.String("config", "./config/config.json", "config file path")
	flag.Parse()

	cfg, err := loadRuntimeConfig(*configPath)
	if err != nil {
		logFatal("main", "failed to load config: %v", err)
	}

	// Parse raw config JSON for one-time migration of legacy fields to DB.
	legacySettings := parseLegacySettings(*configPath)

	if err := os.MkdirAll(filepath.Dir(cfg.Storage.DBPath), 0o755); err != nil {
		logFatal("main", "failed to create data directory: %v", err)
	}

	db, err := sql.Open("sqlite3", cfg.Storage.DBPath)
	if err != nil {
		logFatal("main", "failed to open sqlite: %v", err)
	}
	defer db.Close()

	store := newStore(db)
	if err := store.initSchema(); err != nil {
		logFatal("main", "failed to init schema: %v", err)
	}
	if err := store.seedIfEmpty(); err != nil {
		logFatal("main", "failed to seed data: %v", err)
	}

	// Load admin-managed settings from DB (xOAuth, contracts, sync).
	// On first run, migrate legacy values from config.json if DB is empty.
	migrateLegacySetting(store, "xoauth_client_id", legacySettings.XOAuth.ClientID)
	migrateLegacySetting(store, "xoauth_client_secret", legacySettings.XOAuth.ClientSecret)
	migrateLegacySetting(store, "xoauth_scopes", legacySettings.XOAuth.Scopes)
	migrateLegacySetting(store, "contracts_rpc_url", legacySettings.Contracts.RPCURL)
	migrateLegacySetting(store, "contracts_allocator_address", legacySettings.Contracts.AllocatorAddress)
	if legacySettings.Sync.IntervalSeconds > 0 {
		migrateLegacySetting(store, "sync_interval_seconds", strconv.Itoa(legacySettings.Sync.IntervalSeconds))
	}

	xoauthClientID := store.getSettingDefault("xoauth_client_id", "")
	xoauthClientSecret := store.getSettingDefault("xoauth_client_secret", "")
	xoauthScopes := store.getSettingDefault("xoauth_scopes", "users.read tweet.read offline.access")
	contractRPCURL := store.getSettingDefault("contracts_rpc_url", "")
	contractAllocator := store.getSettingDefault("contracts_allocator_address", "")
	syncIntervalStr := store.getSettingDefault("sync_interval_seconds", "1800")
	syncInterval, _ := strconv.Atoi(syncIntervalStr)
	if syncInterval < 0 {
		syncInterval = 0
	}

	if strings.TrimSpace(xoauthClientID) == "" {
		logWarn("main", "X OAuth is not configured (xoauth_client_id not set in admin settings)")
	}

	server := &Server{
		store:             store,
		tokenSecret:       cfg.Server.TokenSecret,
		agentFixedKey:     cfg.AgentPool.FixedKey,
		appBaseURL:        cfg.AppBaseURL,
		hyperliquid:       newHyperliquidClient(cfg.Hyperliquid.BaseURL),
		contractRPCURL:    contractRPCURL,
		contractAllocator: contractAllocator,
		xOAuth: XOAuthConfig{
			ClientID:           xoauthClientID,
			ClientSecret:       xoauthClientSecret,
			RedirectURL:        cfg.AppBaseURL + "/api/auth/x/callback",
			FrontendSuccessURL: cfg.AppBaseURL + "/auth/x/callback",
			FrontendFailureURL: cfg.AppBaseURL + "/auth/x/callback",
			Scopes:             xoauthScopes,
		},
	}

	// EVM client init is async — it can be slow to dial the RPC endpoint.
	if server.contractRPCURL != "" && server.contractAllocator != "" {
		go func() {
			logInfo("evm", "initializing EVM client (rpc: %s)", server.contractRPCURL)
			ec, err := NewEVMClient(server.contractRPCURL, server.contractAllocator)
			if err != nil {
				logWarn("evm", "EVM client init failed: %v", err)
				return
			}
			server.setEVMClient(ec)
			logInfo("evm", "EVM client ready")
		}()
	}

	staticHost, err := newStaticHost(cfg)
	if err != nil {
		logFatal("main", "failed to setup static host: %v", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(requestLoggerMiddleware())

	server.registerPublicRoutes(e.Group("/api"))
	server.registerAdminRoutes(e.Group("/admin/api"))
	staticHost.registerRoutes(e)

	server.startAutoSync(syncInterval)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logInfo("main", "OpenFi server listening on %s (db: %s, frontend: %s)", addr, cfg.Storage.DBPath, cfg.Frontend.Mode)
	if server.xOAuth.RedirectURL != "" {
		logInfo("main", "X OAuth redirect_uri: %s", server.xOAuth.RedirectURL)
	}
	if err := e.Start(addr); err != nil {
		logFatal("main", "server exited: %v", err)
	}
}
