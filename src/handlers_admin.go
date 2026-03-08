package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (s *Server) registerAdminRoutes(g *echo.Group) {
	g.POST("/login", s.handleAdminLogin)

	secured := g.Group("", s.requireRole("admin"))
	secured.GET("/me", s.handleAdminMe)
	secured.GET("/admins", s.handleAdminListAdmins)
	secured.POST("/admins", s.handleAdminCreateAdmin)
	secured.PATCH("/admins/:id/password", s.handleAdminUpdateAdminPassword)
	secured.DELETE("/admins/:id", s.handleAdminDeleteAdmin)
	secured.GET("/dashboard", s.handleAdminDashboard)
	secured.GET("/dashboard/trends", s.handleAdminDashboardTrends)

	secured.GET("/agents/:publicKey/performance", s.handleAdminAgentPerformance)
	secured.GET("/agents/leaderboard", s.handleAdminAgentLeaderboard)

	secured.GET("/users", s.handleAdminListUsers)
	secured.DELETE("/users", s.handleAdminBatchDeleteUsers)
	secured.GET("/invite-codes", s.handleAdminListInviteCodes)
	secured.POST("/invite-codes", s.handleAdminCreateInviteCode)
	secured.POST("/invite-codes/batch", s.handleAdminCreateInviteCodesBatch)
	secured.DELETE("/invite-codes", s.handleAdminDeleteInviteCodes)
	secured.PATCH("/invite-codes/:id", s.handleAdminUpdateInviteCode)
	secured.GET("/invite-codes/unused/export", s.handleAdminExportUnusedInviteCodes)

	secured.GET("/agent-accounts", s.handleAdminListAgentAccounts)
	secured.POST("/agent-accounts/import", s.handleAdminImportAgentAccounts)
	secured.DELETE("/agent-accounts", s.handleAdminBatchDeleteAgentPool)
	secured.POST("/agent-accounts/:publicKey/revoke", s.handleAdminRevokeAgent)
	secured.POST("/agent-accounts/:publicKey/reassign", s.handleAdminReassignAgent)
	secured.POST("/users/:id/revoke-invite", s.handleAdminRevokeUserInvite)
	secured.POST("/users/:id/revoke-agent", s.handleAdminRevokeUserAgent)

	secured.GET("/agent-vaults", s.handleAdminListAgentVaults)
	secured.DELETE("/agent-vaults", s.handleAdminBatchDeleteAgentVaults)
	secured.PATCH("/agent-accounts/:publicKey/profile", s.handleAdminUpdateAgentProfile)
	secured.POST("/agent-accounts/:publicKey/privatekey", s.handleAdminGetAgentPrivateKey)

	secured.GET("/settings/sync", s.handleAdminGetSyncSettings)
	secured.PATCH("/settings/sync", s.handleAdminUpdateSyncSettings)
	secured.GET("/settings/xoauth", s.handleAdminGetXOAuthSettings)
	secured.PATCH("/settings/xoauth", s.handleAdminUpdateXOAuthSettings)
	secured.GET("/settings/contracts", s.handleAdminGetContractsSettings)
	secured.PATCH("/settings/contracts", s.handleAdminUpdateContractsSettings)
	secured.GET("/settings/intern-slots", s.handleAdminGetInternSlots)
	secured.PATCH("/settings/intern-slots", s.handleAdminUpdateInternSlots)
	secured.GET("/settings/tvl-offset", s.handleAdminGetTvlOffset)
	secured.PATCH("/settings/tvl-offset", s.handleAdminUpdateTvlOffset)
	secured.GET("/settings/dispatch", s.handleAdminGetDispatchSettings)
	secured.PATCH("/settings/dispatch", s.handleAdminUpdateDispatchSettings)
	secured.POST("/agent-accounts/:publicKey/dispatch", s.handleAdminDispatchAgent)

	secured.GET("/treasury", s.handleAdminTreasury)
	secured.GET("/treasury/history", s.handleAdminTreasuryHistory)

	secured.GET("/settings/backup", s.handleAdminGetBackupSettings)
	secured.PATCH("/settings/backup", s.handleAdminUpdateBackupSettings)
	secured.GET("/backups", s.handleAdminListBackups)
	secured.POST("/backups", s.handleAdminCreateBackup)
	secured.POST("/backups/restore", s.handleAdminRestoreBackup)
}

func (s *Server) handleAdminLogin(c echo.Context) error {
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	admin, passwordHash, err := s.store.getAdminByEmail(req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_credentials"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_login"})
	}
	if !verifyPassword(req.Password, passwordHash) {
		logWarn("audit", "login failed: email=%s", req.Email)
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_credentials"})
	}
	token, err := issueToken(s.tokenSecret, admin.ID, "admin", 12*time.Hour)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_issue_token"})
	}
	logInfo("audit", "login success: email=%s", req.Email)
	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
		"admin": admin,
	})
}

func (s *Server) handleAdminMe(c echo.Context) error {
	adminID := c.Get("subject").(string)
	admin, err := s.store.getAdminByID(adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "admin_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_admin"})
	}
	return c.JSON(http.StatusOK, admin)
}

func (s *Server) handleAdminListAdmins(c echo.Context) error {
	items, err := s.store.listAdmins()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_admins"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminCreateAdmin(c echo.Context) error {
	req := struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.Email) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "email_required"})
	}
	if len(strings.TrimSpace(req.Password)) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_too_short"})
	}

	item, err := s.store.createAdmin(req.Email, req.Name, req.Password)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return c.JSON(http.StatusConflict, echo.Map{"error": "admin_email_exists"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_create_admin"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: created admin email=%s", adminID, req.Email)
	return c.JSON(http.StatusCreated, item)
}

func (s *Server) handleAdminUpdateAdminPassword(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	req := struct {
		Password string `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if len(strings.TrimSpace(req.Password)) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_too_short"})
	}
	if err := s.store.updateAdminPassword(id, req.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "admin_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_update_admin_password"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: changed password for admin %s", adminID, id)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminDeleteAdmin(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	operatorID := c.Get("subject").(string)
	if id == operatorID {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "cannot_delete_current_admin"})
	}

	if err := s.store.deleteAdmin(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "admin_not_found"})
		}
		if strings.Contains(err.Error(), "cannot_delete_last_admin") {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "cannot_delete_last_admin"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_admin"})
	}
	logInfo("audit", "admin %s: deleted admin %s", operatorID, id)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminDashboard(c echo.Context) error {
	stats, err := s.store.dashboardStatsEnhanced()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_dashboard"})
	}
	return c.JSON(http.StatusOK, stats)
}

func (s *Server) handleAdminDashboardTrends(c echo.Context) error {
	period := strings.TrimSpace(c.QueryParam("period"))
	if period == "" {
		period = "7d"
	}
	limit := 200
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	snapshots, err := s.store.listPlatformSnapshots(limit, period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_trends"})
	}
	growth := s.store.getPlatformGrowth(period)
	return c.JSON(http.StatusOK, echo.Map{
		"snapshots": snapshots,
		"growth":    growth,
	})
}

func (s *Server) handleAdminAgentPerformance(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	perf, err := s.store.getAgentPerformance(publicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_performance"})
	}
	return c.JSON(http.StatusOK, perf)
}

func (s *Server) handleAdminAgentLeaderboard(c echo.Context) error {
	sortBy := strings.TrimSpace(c.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "pnl"
	}
	limit := 10
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := s.store.getAgentLeaderboard(sortBy, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_leaderboard"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminListUsers(c echo.Context) error {
	search := c.QueryParam("search")
	items, err := s.store.listUsers(search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_users"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminListInviteCodes(c echo.Context) error {
	items, err := s.store.listInviteCodes()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_invite_codes"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminCreateInviteCode(c echo.Context) error {
	req := InviteCode{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.Code) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "code_required"})
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	if req.Status == "" {
		req.Status = "active"
	}
	item, err := s.store.createInviteCode(req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return c.JSON(http.StatusConflict, echo.Map{"error": "invite_code_exists"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_create_invite_code"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: created invite code", adminID)
	return c.JSON(http.StatusCreated, item)
}

func (s *Server) handleAdminUpdateInviteCode(c echo.Context) error {
	id := c.Param("id")
	req := struct {
		Description *string `json:"description"`
		MaxUses     *int    `json:"maxUses"`
		Status      *string `json:"status"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	item, err := s.store.updateInviteCode(id, req.Description, req.MaxUses, req.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "invite_code_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_update_invite_code"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated invite code %s", adminID, id)
	return c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminCreateInviteCodesBatch(c echo.Context) error {
	req := struct {
		Prefix      string `json:"prefix"`
		Length      int    `json:"length"`
		Count       int    `json:"count"`
		MaxUses     int    `json:"maxUses"`
		Description string `json:"description"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.Count <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "count_must_be_positive"})
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	items, err := s.store.createInviteCodesBatch(req.Prefix, req.Count, req.Length, req.MaxUses, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_create_invite_codes"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: created %d invite codes", adminID, len(items))
	return c.JSON(http.StatusCreated, items)
}

func (s *Server) handleAdminDeleteInviteCodes(c echo.Context) error {
	req := struct {
		IDs []string `json:"ids"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "ids_required"})
	}
	deleted, err := s.store.deleteInviteCodes(req.IDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_invite_codes"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: deleted %d invite codes", adminID, deleted)
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}

func (s *Server) handleAdminExportUnusedInviteCodes(c echo.Context) error {
	limit := 10000
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	format := strings.ToLower(strings.TrimSpace(c.QueryParam("format")))
	if format == "" {
		format = "json"
	}
	items, err := s.store.listUnusedInviteCodes(limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_export_invite_codes"})
	}

	if format == "csv" {
		buf := &bytes.Buffer{}
		writer := csv.NewWriter(buf)
		_ = writer.Write([]string{"code"})
		for _, item := range items {
			_ = writer.Write([]string{item})
		}
		writer.Flush()
		c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
		c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=unused-invite-codes.csv")
		return c.String(http.StatusOK, buf.String())
	}
	return c.JSON(http.StatusOK, echo.Map{"codes": items})
}

func (s *Server) handleAdminListAgentAccounts(c echo.Context) error {
	status := strings.ToLower(strings.TrimSpace(c.QueryParam("status")))
	items, err := s.store.listAgentAccounts(status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_agent_accounts"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminImportAgentAccounts(c echo.Context) error {
	req := struct {
		EncryptedPayload string `json:"encryptedPayload"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.EncryptedPayload) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "encrypted_payload_required"})
	}
	keys, err := decryptPrivateKeyPayload(req.EncryptedPayload, s.agentFixedKey)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "decrypt_failed"})
	}
	result, err := s.store.importAgentPrivateKeys(keys, s.agentFixedKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_import_agent_accounts"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: imported %d agent accounts", adminID, result.Imported)
	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminListAgentVaults(c echo.Context) error {
	items, err := s.store.listAgentVaults()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_agent_vaults"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminBatchDeleteAgentVaults(c echo.Context) error {
	req := struct {
		VaultAddresses []string `json:"vaultAddresses"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if len(req.VaultAddresses) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "vault_addresses_required"})
	}

	targetSet := make(map[string]struct{})
	for _, addr := range req.VaultAddresses {
		normalized := strings.ToLower(strings.TrimSpace(addr))
		if normalized == "" {
			continue
		}
		targetSet[normalized] = struct{}{}
	}
	targets := make([]string, 0, len(targetSet))
	for addr := range targetSet {
		targets = append(targets, addr)
	}

	deleted, err := s.store.deleteAgentVaults(targets)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_agent_vaults"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: deleted %d agent vaults", adminID, deleted)
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}

func (s *Server) handleAdminUpdateAgentProfile(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	req := struct {
		Name           *string  `json:"name"`
		Description    *string  `json:"description"`
		Category       *string  `json:"category"`
		PerformanceFee *float64 `json:"performanceFee"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.Category != nil {
		category := strings.ToLower(strings.TrimSpace(*req.Category))
		switch category {
		case "trend", "arbitrage", "grid", "martingale":
			*req.Category = category
		default:
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_category"})
		}
	}
	if req.PerformanceFee != nil {
		if *req.PerformanceFee < 0 || *req.PerformanceFee > 1 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "performance_fee_out_of_range"})
		}
	}
	if err := s.store.updateAgentProfile(publicKey, req.Name, req.Description, req.Category, req.PerformanceFee); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "agent_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_update_agent_profile"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated profile for agent %s", adminID, publicKey)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminGetSyncSettings(c echo.Context) error {
	hlConcurrency := 5
	if v := s.store.getSettingDefault("sync_hl_concurrency", "5"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hlConcurrency = n
		}
	}
	return c.JSON(http.StatusOK, echo.Map{
		"intervalSeconds": s.syncIntervalSecs,
		"hlConcurrency":   hlConcurrency,
	})
}

func (s *Server) handleAdminUpdateSyncSettings(c echo.Context) error {
	req := struct {
		IntervalSeconds *int `json:"intervalSeconds"`
		HLConcurrency   *int `json:"hlConcurrency"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.IntervalSeconds != nil {
		v := *req.IntervalSeconds
		if v < 0 {
			v = 0
		}
		_ = s.store.setSetting("sync_interval_seconds", strconv.Itoa(v))
		s.startAutoSync(v)
	}
	if req.HLConcurrency != nil {
		v := *req.HLConcurrency
		if v < 1 {
			v = 1
		}
		_ = s.store.setSetting("sync_hl_concurrency", strconv.Itoa(v))
	}

	hlConcurrency := 5
	if v := s.store.getSettingDefault("sync_hl_concurrency", "5"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hlConcurrency = n
		}
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated sync settings", adminID)
	return c.JSON(http.StatusOK, echo.Map{
		"intervalSeconds": s.syncIntervalSecs,
		"hlConcurrency":   hlConcurrency,
	})
}

func (s *Server) handleAdminGetXOAuthSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"clientId":     s.xOAuth.ClientID,
		"clientSecret": s.xOAuth.ClientSecret,
		"scopes":       s.xOAuth.Scopes,
	})
}

func (s *Server) handleAdminUpdateXOAuthSettings(c echo.Context) error {
	req := struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Scopes       string `json:"scopes"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.ClientID != "" {
		s.xOAuth.ClientID = req.ClientID
		_ = s.store.setSetting("xoauth_client_id", req.ClientID)
	}
	if req.ClientSecret != "" {
		s.xOAuth.ClientSecret = req.ClientSecret
		_ = s.store.setSetting("xoauth_client_secret", req.ClientSecret)
	}
	if req.Scopes != "" {
		s.xOAuth.Scopes = req.Scopes
		_ = s.store.setSetting("xoauth_scopes", req.Scopes)
	}
	logInfo("admin", "xOAuth settings updated (clientId: %s...)", truncate(s.xOAuth.ClientID, 8))
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminGetContractsSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"rpcURL":           s.contractRPCURL,
		"allocatorAddress": s.contractAllocator,
	})
}

func (s *Server) handleAdminUpdateContractsSettings(c echo.Context) error {
	req := struct {
		RPCURL           string `json:"rpcURL"`
		AllocatorAddress string `json:"allocatorAddress"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}

	rpcURL := strings.TrimSpace(req.RPCURL)
	allocator := strings.TrimSpace(req.AllocatorAddress)

	if rpcURL == "" || allocator == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "rpcURL and allocatorAddress are required"})
	}

	s.contractRPCURL = rpcURL
	s.contractAllocator = allocator
	_ = s.store.setSetting("contracts_rpc_url", rpcURL)
	_ = s.store.setSetting("contracts_allocator_address", allocator)

	// Reinitialize EVM client asynchronously
	go func() {
		logInfo("evm", "reinitializing EVM client (rpc: %s)", rpcURL)
		ec, err := NewEVMClient(rpcURL, allocator)
		if err != nil {
			logWarn("evm", "EVM client reinit failed: %v", err)
			return
		}
		s.setEVMClient(ec)
		logInfo("evm", "EVM client ready")
		go s.runSyncRound()
	}()

	logInfo("admin", "contracts settings updated (rpc: %s)", rpcURL)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminGetInternSlots(c echo.Context) error {
	slots, err := s.store.getDailySlots()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_slots"})
	}
	return c.JSON(http.StatusOK, slots)
}

func (s *Server) handleAdminUpdateInternSlots(c echo.Context) error {
	req := struct {
		Total *int `json:"total"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.Total != nil {
		v := *req.Total
		if v < 1 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "total must be >= 1"})
		}
		_ = s.store.setSetting("intern_slots_total", strconv.Itoa(v))
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated intern slots", adminID)
	slots, err := s.store.getDailySlots()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_slots"})
	}
	return c.JSON(http.StatusOK, slots)
}

func (s *Server) handleAdminGetTvlOffset(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{"tvlOffset": s.store.getTvlOffset()})
}

func (s *Server) handleAdminUpdateTvlOffset(c echo.Context) error {
	req := struct {
		TvlOffset *float64 `json:"tvlOffset"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.TvlOffset == nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tvlOffset is required"})
	}
	if err := s.store.setSetting("tvl_offset", strconv.FormatFloat(*req.TvlOffset, 'f', -1, 64)); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_save"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated tvl offset to %f", adminID, *req.TvlOffset)
	return c.JSON(http.StatusOK, echo.Map{"tvlOffset": *req.TvlOffset})
}

func (s *Server) handleAdminBatchDeleteAgentPool(c echo.Context) error {
	req := struct {
		PublicKeys []string `json:"publicKeys"`
		Password   string   `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if len(req.PublicKeys) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_keys_required"})
	}
	if strings.TrimSpace(req.Password) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_required"})
	}

	adminID := c.Get("subject").(string)
	storedHash, err := s.store.getAdminPasswordHashByID(adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_verify_password"})
	}
	if !verifyPassword(req.Password, storedHash) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	deleted, err := s.store.deleteAgentAccounts(req.PublicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_agent_accounts"})
	}
	logInfo("audit", "admin %s: deleted %d agent accounts", adminID, deleted)
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}

func (s *Server) handleAdminBatchDeleteUsers(c echo.Context) error {
	req := struct {
		IDs      []string `json:"ids"`
		Password string   `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "ids_required"})
	}
	if strings.TrimSpace(req.Password) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_required"})
	}

	adminID := c.Get("subject").(string)
	storedHash, err := s.store.getAdminPasswordHashByID(adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_verify_password"})
	}
	if !verifyPassword(req.Password, storedHash) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	deleted, err := s.store.deleteUsers(req.IDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_users"})
	}
	logInfo("audit", "admin %s: deleted %d users", adminID, deleted)
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}

func (s *Server) handleAdminGetDispatchSettings(c echo.Context) error {
	cmd := s.store.getSettingDefault("dispatch_command", "")
	return c.JSON(http.StatusOK, echo.Map{"command": cmd})
}

func (s *Server) handleAdminUpdateDispatchSettings(c echo.Context) error {
	req := struct {
		Command string `json:"command"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if err := s.store.setSetting("dispatch_command", strings.TrimSpace(req.Command)); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_save"})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated dispatch settings", adminID)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminDispatchAgent(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}

	cmdTemplate := s.store.getSettingDefault("dispatch_command", "")
	if cmdTemplate == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dispatch_command_not_configured"})
	}

	encrypted, vaultAddress, err := s.store.getAgentDispatchInfo(publicKey)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "agent_not_found"})
	}

	privateKey, err := decryptSecret(encrypted, s.agentFixedKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_decrypt_key"})
	}

	cmd := buildDispatchCommand(cmdTemplate, privateKey, publicKey, vaultAddress)

	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: dispatching agent %s", adminID, publicKey)
	go func() {
		out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		if err != nil {
			logError("dispatch", "agent %s failed: %v — %s", publicKey, err, string(out))
		} else {
			logInfo("dispatch", "agent %s done: %s", publicKey, string(out))
		}
	}()

	return c.JSON(http.StatusOK, echo.Map{"success": true, "message": "dispatch started"})
}

func (s *Server) handleAdminGetAgentPrivateKey(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}

	req := struct {
		Password string `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.Password) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_required"})
	}

	adminID := c.Get("subject").(string)
	storedHash, err := s.store.getAdminPasswordHashByID(adminID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin_not_found"})
	}
	if !verifyPassword(req.Password, storedHash) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	encrypted, err := s.store.getAgentEncryptedPrivateKey(publicKey)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "agent_not_found"})
	}

	privateKey, err := decryptSecret(encrypted, s.agentFixedKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_decrypt_key"})
	}

	logInfo("admin", "admin %s viewed private key for agent %s", adminID, publicKey)
	return c.JSON(http.StatusOK, echo.Map{"privateKey": privateKey})
}

func (s *Server) handleAdminRevokeAgent(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	if err := s.store.revokeUserAgent(publicKey); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: revoked agent %s", adminID, publicKey)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminReassignAgent(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	req := struct {
		UserID   string `json:"userId"`
		UserName string `json:"userName"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}

	userID := strings.TrimSpace(req.UserID)
	// If userName is provided, look up the user by name
	if userID == "" && strings.TrimSpace(req.UserName) != "" {
		user, err := s.store.getUserByName(req.UserName)
		if err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
		}
		userID = user.ID
	}
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id_or_user_name_required"})
	}
	if err := s.store.reassignAgent(publicKey, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: reassigned agent %s to %s", adminID, publicKey, req.UserName)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminRevokeUserInvite(c echo.Context) error {
	userID := strings.TrimSpace(c.Param("id"))
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id_required"})
	}
	if err := s.store.revokeUserInviteCode(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: revoked invite for user %s", adminID, userID)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminRevokeUserAgent(c echo.Context) error {
	userID := strings.TrimSpace(c.Param("id"))
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id_required"})
	}
	user, err := s.store.getUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if user.AgentPublicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_has_no_agent"})
	}
	if err := s.store.revokeUserAgent(user.AgentPublicKey); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: revoked agent for user %s", adminID, userID)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminTreasury(c echo.Context) error {
	snap, err := s.store.getLatestTreasurySnapshot()
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{})
	}
	return c.JSON(http.StatusOK, snap)
}

func (s *Server) handleAdminTreasuryHistory(c echo.Context) error {
	period := strings.TrimSpace(c.QueryParam("period"))
	if period == "" {
		period = "7d"
	}
	limit := 200
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := s.store.listTreasurySnapshots(limit, period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_treasury_history"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) backupSettingsJSON() echo.Map {
	return echo.Map{
		"intervalHours": getPositiveIntSetting(s.store, "backup_interval_hours", 24),
		"retainHourly":  getPositiveIntSetting(s.store, "backup_retain_hourly", 3),
		"retainDaily":   getPositiveIntSetting(s.store, "backup_retain_daily", 3),
		"retainWeekly":  getPositiveIntSetting(s.store, "backup_retain_weekly", 3),
		"lastBackupAt":  s.store.getSettingDefault("backup_last_at", ""),
	}
}

func (s *Server) handleAdminGetBackupSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, s.backupSettingsJSON())
}

func (s *Server) handleAdminUpdateBackupSettings(c echo.Context) error {
	req := struct {
		IntervalHours *int `json:"intervalHours"`
		RetainHourly  *int `json:"retainHourly"`
		RetainDaily   *int `json:"retainDaily"`
		RetainWeekly  *int `json:"retainWeekly"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}

	if req.IntervalHours != nil {
		v := *req.IntervalHours
		if v < 0 {
			v = 0
		}
		_ = s.store.setSetting("backup_interval_hours", strconv.Itoa(v))
	}
	if req.RetainHourly != nil {
		v := *req.RetainHourly
		if v < 1 {
			v = 1
		}
		_ = s.store.setSetting("backup_retain_hourly", strconv.Itoa(v))
	}
	if req.RetainDaily != nil {
		v := *req.RetainDaily
		if v < 1 {
			v = 1
		}
		_ = s.store.setSetting("backup_retain_daily", strconv.Itoa(v))
	}
	if req.RetainWeekly != nil {
		v := *req.RetainWeekly
		if v < 1 {
			v = 1
		}
		_ = s.store.setSetting("backup_retain_weekly", strconv.Itoa(v))
	}

	// Run cleanup with new settings, then restart goroutine
	s.cleanupOldBackups()
	s.startAutoBackup()

	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: updated backup settings", adminID)
	return c.JSON(http.StatusOK, s.backupSettingsJSON())
}

func (s *Server) handleAdminListBackups(c echo.Context) error {
	backups, err := s.listBackups()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_backups"})
	}
	return c.JSON(http.StatusOK, backups)
}

func (s *Server) handleAdminCreateBackup(c echo.Context) error {
	adminID := c.Get("subject").(string)
	logInfo("audit", "admin %s: triggered manual backup", adminID)
	info, err := s.performBackup()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, info)
}

func (s *Server) handleAdminRestoreBackup(c echo.Context) error {
	req := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.Name) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name_required"})
	}
	if strings.TrimSpace(req.Password) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password_required"})
	}

	adminID := c.Get("subject").(string)
	storedHash, err := s.store.getAdminPasswordHashByID(adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_verify_password"})
	}
	if !verifyPassword(req.Password, storedHash) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	logWarn("audit", "admin %s: restoring from backup %s", adminID, req.Name)
	if err := s.restoreFromBackup(req.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true, "restored": req.Name})
}
