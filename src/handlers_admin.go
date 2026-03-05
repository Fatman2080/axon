package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"net/http"
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

	secured.GET("/users", s.handleAdminListUsers)
	secured.DELETE("/users", s.handleAdminBatchDeleteUsers)
	secured.DELETE("/agent-stats", s.handleAdminBatchDeleteAgents)

	secured.GET("/invite-codes", s.handleAdminListInviteCodes)
	secured.POST("/invite-codes", s.handleAdminCreateInviteCode)
	secured.POST("/invite-codes/batch", s.handleAdminCreateInviteCodesBatch)
	secured.DELETE("/invite-codes", s.handleAdminDeleteInviteCodes)
	secured.PATCH("/invite-codes/:id", s.handleAdminUpdateInviteCode)
	secured.GET("/invite-codes/unused/export", s.handleAdminExportUnusedInviteCodes)

	secured.GET("/agent-accounts", s.handleAdminListAgentAccounts)
	secured.POST("/agent-accounts/import", s.handleAdminImportAgentAccounts)
	secured.DELETE("/agent-accounts", s.handleAdminBatchDeleteAgentPool)

	secured.GET("/agent-stats", s.handleAdminListAgentStats)
	secured.POST("/agent-stats/:publicKey/sync", s.handleAdminSyncAgentData)
	secured.PATCH("/agent-stats/:publicKey/profile", s.handleAdminUpdateAgentProfile)

	secured.GET("/settings/sync", s.handleAdminGetSyncSettings)
	secured.PATCH("/settings/sync", s.handleAdminUpdateSyncSettings)
	secured.GET("/settings/xoauth", s.handleAdminGetXOAuthSettings)
	secured.PATCH("/settings/xoauth", s.handleAdminUpdateXOAuthSettings)
	secured.GET("/settings/contracts", s.handleAdminGetContractsSettings)
	secured.PATCH("/settings/contracts", s.handleAdminUpdateContractsSettings)
	secured.GET("/settings/daily-slots", s.handleAdminGetDailySlotsSettings)
	secured.PATCH("/settings/daily-slots", s.handleAdminUpdateDailySlotsSettings)
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
	if hashPassword(req.Password) != passwordHash {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_credentials"})
	}
	token, err := issueToken(s.tokenSecret, admin.ID, "admin", 12*time.Hour)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_issue_token"})
	}
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
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminDashboard(c echo.Context) error {
	stats, err := s.store.dashboardStats()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_dashboard"})
	}
	return c.JSON(http.StatusOK, stats)
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
	return c.JSON(http.StatusCreated, item)
}

func (s *Server) handleAdminUpdateInviteCode(c echo.Context) error {
	id := c.Param("id")
	req := struct {
		Description string `json:"description"`
		MaxUses     int    `json:"maxUses"`
		Status      string `json:"status"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	patch := InviteCode{
		Description: req.Description,
		MaxUses:     req.MaxUses,
		Status:      req.Status,
	}
	item, err := s.store.updateInviteCode(id, patch)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "invite_code_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_update_invite_code"})
	}
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
	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminListAgentStats(c echo.Context) error {
	search := c.QueryParam("search")
	items, err := s.store.listAgentStats(search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_list_agent_stats"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminSyncAgentData(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	item, err := s.syncByPublicKey(publicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_sync_agent_data"})
	}
	return c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminUpdateAgentProfile(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}
	req := struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		PerformanceFee float64 `json:"performanceFee"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if err := s.store.updateAgentProfile(publicKey, req.Name, req.Description, req.PerformanceFee); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "agent_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_update_agent_profile"})
	}
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminGetSyncSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"intervalSeconds": s.syncIntervalSecs,
	})
}

func (s *Server) handleAdminUpdateSyncSettings(c echo.Context) error {
	req := struct {
		IntervalSeconds int `json:"intervalSeconds"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if req.IntervalSeconds < 0 {
		req.IntervalSeconds = 0
	}
	_ = s.store.setSetting("sync_interval_seconds", strconv.Itoa(req.IntervalSeconds))
	s.startAutoSync(req.IntervalSeconds)
	return c.JSON(http.StatusOK, echo.Map{
		"intervalSeconds": s.syncIntervalSecs,
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
	}()

	logInfo("admin", "contracts settings updated (rpc: %s)", rpcURL)
	return c.JSON(http.StatusOK, echo.Map{"success": true})
}

func (s *Server) handleAdminGetDailySlotsSettings(c echo.Context) error {
	slots, err := s.store.getDailySlots()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_daily_slots"})
	}
	return c.JSON(http.StatusOK, slots)
}

func (s *Server) handleAdminUpdateDailySlotsSettings(c echo.Context) error {
	req := struct {
		Total         *int  `json:"total"`
		ResetHour     *int  `json:"resetHour"`
		ResetConsumed *bool `json:"resetConsumed"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}

	if req.Total != nil {
		v := *req.Total
		if v < 1 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "total must be >= 1"})
		}
		_ = s.store.setSetting("daily_slots_total", strconv.Itoa(v))
	}
	if req.ResetHour != nil {
		v := *req.ResetHour
		if v < 0 || v > 23 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "resetHour must be 0-23"})
		}
		_ = s.store.setSetting("daily_slots_reset_hour", strconv.Itoa(v))
	}
	if req.ResetConsumed != nil && *req.ResetConsumed {
		_ = s.store.setSetting("daily_slots_consumed", "0")
		_ = s.store.setSetting("daily_slots_reset_at", time.Now().UTC().Format(time.RFC3339))
	}

	slots, err := s.store.getDailySlots()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_daily_slots"})
	}
	return c.JSON(http.StatusOK, slots)
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
	if hashPassword(req.Password) != storedHash {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	deleted, err := s.store.deleteAgentAccounts(req.PublicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_agent_accounts"})
	}
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
	if hashPassword(req.Password) != storedHash {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	deleted, err := s.store.deleteUsers(req.IDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_users"})
	}
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}

func (s *Server) handleAdminBatchDeleteAgents(c echo.Context) error {
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
	if hashPassword(req.Password) != storedHash {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_password"})
	}

	deleted, err := s.store.deleteAgentAccounts(req.PublicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_delete_agents"})
	}
	return c.JSON(http.StatusOK, echo.Map{"deleted": deleted})
}
