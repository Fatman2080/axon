package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type Server struct {
	store            *Store
	tokenSecret      string
	agentFixedKey    string
	appBaseURL       string
	hyperliquid      *HyperliquidClient
	evmMu            sync.RWMutex
	evmClient        *EVMClient
	xOAuth           XOAuthConfig
	contractRPCURL   string
	contractAllocator string
	syncIntervalSecs int
	syncStop         chan struct{}
}

func (s *Server) getEVMClient() *EVMClient {
	s.evmMu.RLock()
	defer s.evmMu.RUnlock()
	return s.evmClient
}

func (s *Server) setEVMClient(c *EVMClient) {
	s.evmMu.Lock()
	defer s.evmMu.Unlock()
	s.evmClient = c
}

func (s *Server) registerPublicRoutes(g *echo.Group) {
	g.GET("/health", s.handleHealth)
	g.POST("/auth/x-login", s.handleXLogin)
	g.GET("/auth/x/start", s.handleXOAuthStart)
	g.GET("/auth/x/callback", s.handleXOAuthCallback)
	g.GET("/auth/twitter", s.handleTwitterAuth)

	g.GET("/agent-market", s.handleAgentMarket)
	g.GET("/agent-market/:publicKey", s.handleAgentMarketDetail)

	g.GET("/vault/stats", s.handleVaultStats)
	g.GET("/daily-slots", s.handleDailySlots)
	g.GET("/vault/overview", s.handleVaultOverview)

	g.GET("/invite-codes/verify", s.handleVerifyInviteCode)
	g.POST("/invite-codes/consume", s.handleConsumeInviteCode, s.requireRole("user"))
	g.GET("/user/me", s.handleGetMe, s.requireRole("user"))
	g.GET("/user/agent/history", s.handleUserAgentHistory, s.requireRole("user"))
	g.GET("/user/agent/stats", s.handleUserAgentStats, s.requireRole("user"))
}

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"ok":   true,
		"time": time.Now().UTC().Format(time.RFC3339),
	})
}

type xLoginError struct {
	Status int
	Code   string
}

func (e *xLoginError) Error() string {
	return e.Code
}

func (s *Server) loginWithXIdentity(xID string, xUsername string, avatar string, inviteCode string) (string, User, *xLoginError) {
	xID = strings.TrimSpace(xID)
	if xID == "" {
		return "", User{}, &xLoginError{Status: http.StatusBadRequest, Code: "x_id_required"}
	}

	user, err := s.store.getOrCreateUserByXID(xID)
	if err != nil {
		return "", User{}, &xLoginError{Status: http.StatusInternalServerError, Code: "failed_to_load_user"}
	}
	user.XID = xID
	user.XUsername = strings.TrimSpace(xUsername)
	if user.XUsername != "" && !strings.HasPrefix(user.XUsername, "@") {
		user.XUsername = "@" + user.XUsername
	}
	if strings.TrimSpace(avatar) != "" {
		user.Avatar = strings.TrimSpace(avatar)
	}
	if user.Email == "" || strings.HasSuffix(strings.ToLower(user.Email), "@x.local") {
		user.Email = xID + "@x.local"
	}
	displayName := strings.TrimPrefix(user.XUsername, "@")
	if displayName != "" && (user.Name == "" || strings.HasPrefix(user.Name, "User ")) {
		user.Name = displayName
	}

	// Invite code is separate from authentication.
	// If the user hasn't used an invite code yet and one is provided, try to consume it.
	// If no invite code is provided, the user is still authenticated — they just won't have an agent account yet.
	if user.InviteCodeUsed == "" && strings.TrimSpace(inviteCode) != "" {
		invite, acct, err := s.store.consumeInviteAndAssignAccount(inviteCode, user.ID)
		if err != nil {
			if errors.Is(err, errInviteCodeNotFound) {
				return "", User{}, &xLoginError{Status: http.StatusForbidden, Code: "invalid_invite_code"}
			}
			if errors.Is(err, sql.ErrNoRows) {
				return "", User{}, &xLoginError{Status: http.StatusConflict, Code: "agent_account_pool_empty"}
			}
			return "", User{}, &xLoginError{Status: http.StatusForbidden, Code: "invalid_invite_code"}
		}
		user.InviteCodeUsed = invite.Code
		user.AgentPublicKey = acct.PublicKey
		user.AgentAssignedAt = acct.AssignedAt
	} else if user.InviteCodeUsed != "" && user.AgentPublicKey == "" {
		acct, err := s.store.assignUnusedAgentAccount(user.ID)
		if err == nil {
			user.AgentPublicKey = acct.PublicKey
			user.AgentAssignedAt = acct.AssignedAt
		}
	}

	if err := s.store.saveUser(user); err != nil {
		return "", User{}, &xLoginError{Status: http.StatusInternalServerError, Code: "failed_to_save_user"}
	}

	token, err := issueToken(s.tokenSecret, user.ID, "user", 72*time.Hour)
	if err != nil {
		return "", User{}, &xLoginError{Status: http.StatusInternalServerError, Code: "failed_to_issue_token"}
	}
	return token, user, nil
}

func (s *Server) handleXLogin(c echo.Context) error {
	req := struct {
		XID        string `json:"xId"`
		XUsername  string `json:"xUsername"`
		Avatar     string `json:"avatar"`
		InviteCode string `json:"inviteCode"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	token, user, loginErr := s.loginWithXIdentity(req.XID, req.XUsername, req.Avatar, req.InviteCode)
	if loginErr != nil {
		return c.JSON(loginErr.Status, echo.Map{"error": loginErr.Code})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleXOAuthStart(c echo.Context) error {
	if !s.xOAuth.Enabled() {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "x_oauth_not_configured"})
	}
	inviteCode := strings.TrimSpace(c.QueryParam("inviteCode"))
	nextURL := strings.TrimSpace(c.QueryParam("next"))
	if nextURL == "" {
		nextURL = "/"
	}

	state, err := randomURLSafeString(24)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_generate_oauth_state"})
	}
	codeVerifier, err := randomURLSafeString(48)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_generate_pkce_verifier"})
	}
	if err := s.store.saveOAuthState("x", state, codeVerifier, inviteCode, nextURL); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_store_oauth_state"})
	}

	return c.Redirect(http.StatusFound, s.buildXAuthorizeURL(state, buildCodeChallenge(codeVerifier)))
}

func (s *Server) handleXOAuthCallback(c echo.Context) error {
	failureBase := s.xOAuth.FrontendFailureURL
	successBase := s.xOAuth.FrontendSuccessURL
	if strings.TrimSpace(failureBase) == "" {
		failureBase = s.appBaseURL + "/auth/x/callback"
	}
	if strings.TrimSpace(successBase) == "" {
		successBase = s.appBaseURL + "/auth/x/callback"
	}

	redirectFailure := func(code string, nextURL string) error {
		return c.Redirect(http.StatusFound, buildRedirectURL(failureBase, map[string]string{
			"error": code,
			"next":  nextURL,
		}))
	}

	if errCode := strings.TrimSpace(c.QueryParam("error")); errCode != "" {
		return redirectFailure("x_oauth_"+errCode, strings.TrimSpace(c.QueryParam("next")))
	}
	code := strings.TrimSpace(c.QueryParam("code"))
	state := strings.TrimSpace(c.QueryParam("state"))
	if code == "" || state == "" {
		return redirectFailure("invalid_oauth_callback", "")
	}

	oauthState, err := s.store.consumeOAuthState("x", state, 10*time.Minute)
	if err != nil {
		return redirectFailure("invalid_or_expired_oauth_state", "")
	}

	accessToken, err := s.exchangeXOAuthToken(code, oauthState.CodeVerifier)
	if err != nil {
		logError("oauth", "x oauth token exchange failed: %v", err)
		return redirectFailure("x_oauth_token_exchange_failed", oauthState.NextURL)
	}
	xUser, err := s.fetchXOAuthUser(accessToken)
	if err != nil {
		logError("oauth", "x oauth userinfo fetch failed: %v", err)
		return redirectFailure("x_oauth_userinfo_failed", oauthState.NextURL)
	}

	token, _, loginErr := s.loginWithXIdentity(xUser.ID, xUser.Username, xUser.ProfileImageURL, oauthState.InviteCode)
	if loginErr != nil {
		logError("oauth", "x oauth login failed: xid=%s username=%s code=%s", xUser.ID, xUser.Username, loginErr.Code)
		return redirectFailure(loginErr.Code, oauthState.NextURL)
	}
	logInfo("oauth", "x oauth login success: xid=%s username=%s", xUser.ID, xUser.Username)
	return c.Redirect(http.StatusFound, buildRedirectURL(successBase, map[string]string{
		"token": token,
		"next":  oauthState.NextURL,
	}))
}

func (s *Server) handleTwitterAuth(c echo.Context) error {
	query := c.QueryParams()
	target := url.Values{}
	if inviteCode := strings.TrimSpace(query.Get("inviteCode")); inviteCode != "" {
		target.Set("inviteCode", inviteCode)
	}
	if next := strings.TrimSpace(query.Get("next")); next != "" {
		target.Set("next", next)
	}
	redirectPath := "/api/auth/x/start"
	if encoded := target.Encode(); encoded != "" {
		redirectPath += "?" + encoded
	}
	return c.Redirect(http.StatusFound, redirectPath)
}

func (s *Server) handleGetMe(c echo.Context) error {
	userID := c.Get("subject").(string)
	user, err := s.store.getUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_user"})
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) handleVerifyInviteCode(c echo.Context) error {
	code := c.QueryParam("code")
	if strings.TrimSpace(code) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "code_required"})
	}
	ok, reason, err := s.store.verifyInviteCode(code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_verify_code"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"valid":  ok,
		"reason": reason,
	})
}

func (s *Server) handleConsumeInviteCode(c echo.Context) error {
	userID := c.Get("subject").(string)
	req := struct {
		Code string `json:"code"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid_payload"})
	}
	if strings.TrimSpace(req.Code) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "code_required"})
	}

	user, err := s.store.getUserByID(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
	}
	if user.InviteCodeUsed != "" {
		return c.JSON(http.StatusOK, echo.Map{
			"success":    true,
			"inviteCode": user.InviteCodeUsed,
			"publicKey":  user.AgentPublicKey,
		})
	}

	invite, account, err := s.store.consumeInviteAndAssignAccount(req.Code, userID)
	if err != nil {
		if errors.Is(err, errInviteCodeNotFound) {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "invalid_invite_code"})
		}
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusConflict, echo.Map{"error": "agent_account_pool_empty"})
		}
		return c.JSON(http.StatusForbidden, echo.Map{"error": "invalid_invite_code"})
	}
	user.InviteCodeUsed = invite.Code
	user.AgentPublicKey = account.PublicKey
	user.AgentAssignedAt = account.AssignedAt
	if err := s.store.saveUser(user); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_save_user"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"success":    true,
		"inviteCode": invite.Code,
		"publicKey":  account.PublicKey,
		"user":       user,
	})
}

func (s *Server) handleAgentMarket(c echo.Context) error {
	search := c.QueryParam("search")
	items, err := s.store.listAgentStats(search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_agent_market"})
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleAgentMarketDetail(c echo.Context) error {
	publicKey := strings.ToLower(strings.TrimSpace(c.Param("publicKey")))
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "public_key_required"})
	}

	agent, err := s.store.getAgentStats(publicKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "agent_not_found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_agent"})
	}

	period := strings.TrimSpace(c.QueryParam("period"))
	snapshots, err := s.store.listAgentSnapshots(publicKey, 120, period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_history"})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})
	history := make([]float64, 0, len(snapshots))
	for _, snap := range snapshots {
		history = append(history, snap.AccountValue)
	}

	// Fetch live positions and fills from Hyperliquid
	positions := make([]VaultPosition, 0)
	recentFills := make([]VaultFill, 0)
	if s.hyperliquid != nil {
		if pos, err := s.hyperliquid.FetchPositions(publicKey); err == nil {
			positions = pos
		}
		if fills, err := s.hyperliquid.FetchUserFills(publicKey); err == nil {
			limit := 50
			if len(fills) < limit {
				limit = len(fills)
			}
			recentFills = fills[:limit]
		}
	}

	createdAt := s.store.getAgentCreatedAt(publicKey)

	return c.JSON(http.StatusOK, echo.Map{
		"agent":       agent,
		"history":     history,
		"positions":   positions,
		"recentFills": recentFills,
		"createdAt":   createdAt,
	})
}

func (s *Server) handleUserAgentHistory(c echo.Context) error {
	userID := c.Get("subject").(string)
	user, err := s.store.getUserByID(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
	}
	if strings.TrimSpace(user.AgentPublicKey) == "" {
		return c.JSON(http.StatusOK, echo.Map{"history": []float64{}, "trades": []interface{}{}})
	}
	period := strings.TrimSpace(c.QueryParam("period"))
	snapshots, err := s.store.listAgentSnapshots(user.AgentPublicKey, 120, period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_history"})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})
	history := make([]float64, 0, len(snapshots))
	for _, item := range snapshots {
		history = append(history, item.AccountValue)
	}
	return c.JSON(http.StatusOK, echo.Map{
		"publicKey": user.AgentPublicKey,
		"history":   history,
		"trades":    []interface{}{},
	})
}

func (s *Server) handleVaultStats(c echo.Context) error {
	items, err := s.store.listAgentStats("")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_stats"})
	}

	var totalTvl, totalL1Value, totalEvmBalance float64
	var activeCount int
	for _, item := range items {
		if item.AgentStatus != AgentStatusActive {
			continue
		}
		totalTvl += item.TVL
		totalL1Value += item.AccountValue
		totalEvmBalance += item.EVMBalance
		activeCount++
	}

	return c.JSON(http.StatusOK, echo.Map{
		"totalTvl":        totalTvl,
		"totalEvmBalance": totalEvmBalance,
		"totalL1Value":    totalL1Value,
		"agentCount":      activeCount,
	})
}

func (s *Server) handleVaultOverview(c echo.Context) error {
	items, err := s.store.listAgentStats("")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_stats"})
	}

	overview := VaultOverview{
		Positions:   make([]VaultPosition, 0),
		RecentFills: make([]VaultFill, 0),
	}
	for _, item := range items {
		if item.AgentStatus != AgentStatusActive {
			continue
		}
		overview.TotalTvl += item.TVL
		overview.TotalL1Value += item.AccountValue
		overview.TotalEvmBalance += item.EVMBalance
		overview.TotalPnl += item.TotalPnL
		overview.AgentCount++
	}

	if s.hyperliquid != nil {
		keys, err := s.store.listAssignedPublicKeys()
		if err == nil {
			for _, key := range keys {
				positions, err := s.hyperliquid.FetchPositions(key)
				if err != nil {
					continue
				}
				overview.Positions = append(overview.Positions, positions...)
			}
			// Fetch fills from first few agents (limit to avoid timeout)
			limit := 5
			if len(keys) < limit {
				limit = len(keys)
			}
			for _, key := range keys[:limit] {
				fills, err := s.hyperliquid.FetchUserFills(key)
				if err != nil {
					continue
				}
				overview.RecentFills = append(overview.RecentFills, fills...)
			}
			// Sort fills by time descending, limit to 50
			sort.Slice(overview.RecentFills, func(i, j int) bool {
				return overview.RecentFills[i].Time > overview.RecentFills[j].Time
			})
			if len(overview.RecentFills) > 50 {
				overview.RecentFills = overview.RecentFills[:50]
			}
		}
	}

	return c.JSON(http.StatusOK, overview)
}

func (s *Server) handleUserAgentStats(c echo.Context) error {
	userID := c.Get("subject").(string)
	user, err := s.store.getUserByID(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user_not_found"})
	}

	result := echo.Map{
		"publicKey":    user.AgentPublicKey,
		"accountValue": 0.0,
		"totalPnl":     0.0,
		"positions":    []VaultPosition{},
		"recentFills":  []VaultFill{},
	}

	pk := strings.TrimSpace(user.AgentPublicKey)
	if pk == "" {
		return c.JSON(http.StatusOK, result)
	}

	// Get latest snapshot data
	snapshots, err := s.store.listAgentSnapshots(pk, 120, "ALL")
	if err == nil && len(snapshots) > 0 {
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i].CreatedAt < snapshots[j].CreatedAt
		})
		latest := snapshots[len(snapshots)-1]
		first := snapshots[0]
		result["accountValue"] = latest.AccountValue
		result["totalPnl"] = latest.AccountValue - first.AccountValue
	}

	// Fetch live positions and fills from Hyperliquid
	if s.hyperliquid != nil {
		if positions, err := s.hyperliquid.FetchPositions(pk); err == nil {
			result["positions"] = positions
		}
		if fills, err := s.hyperliquid.FetchUserFills(pk); err == nil {
			limit := 30
			if len(fills) < limit {
				limit = len(fills)
			}
			result["recentFills"] = fills[:limit]
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleDailySlots(c echo.Context) error {
	slots, err := s.store.getDailySlots()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_get_daily_slots"})
	}
	return c.JSON(http.StatusOK, slots)
}

func (s *Server) requireRole(requiredRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := parseBearerToken(c.Request().Header.Get("Authorization"))
			if token == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing_token"})
			}
			claims, err := parseToken(s.tokenSecret, token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid_token"})
			}
			if requiredRole != "" && claims.Role != requiredRole {
				return c.JSON(http.StatusForbidden, echo.Map{"error": "permission_denied"})
			}
			c.Set("subject", claims.Sub)
			c.Set("role", claims.Role)
			return next(c)
		}
	}
}
