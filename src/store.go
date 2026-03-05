package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var errInviteCodeNotFound = errors.New("invite code not found")

type Store struct {
	db *sql.DB
}

func newStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d%03d", prefix, time.Now().UnixNano(), rand.Intn(1000))
}

func (s *Store) initSchema() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			x_id TEXT NOT NULL UNIQUE,
			email TEXT,
			name TEXT,
			level TEXT,
			data_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS admins (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS strategies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			status TEXT NOT NULL,
			creator TEXT NOT NULL,
			agent_level TEXT NOT NULL,
			current_tvl REAL NOT NULL,
			max_tvl REAL NOT NULL,
			pnl_contribution REAL NOT NULL,
			rating REAL NOT NULL,
			data_json TEXT NOT NULL,
			review_note TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			strategy_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			status TEXT NOT NULL,
			investment REAL NOT NULL,
			total_profit REAL NOT NULL,
			data_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS vault (
			id TEXT PRIMARY KEY,
			data_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS invite_codes (
			id TEXT PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			description TEXT,
			max_uses INTEGER NOT NULL DEFAULT 0,
			used_count INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS invite_code_usages (
			id TEXT PRIMARY KEY,
			code_id TEXT NOT NULL,
			code TEXT NOT NULL,
			user_id TEXT NOT NULL,
			used_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agent_accounts (
			id TEXT PRIMARY KEY,
			public_key TEXT NOT NULL UNIQUE,
			encrypted_private_key TEXT NOT NULL,
			status TEXT NOT NULL,
			assigned_user_id TEXT,
			assigned_at TEXT,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agent_snapshots (
			id TEXT PRIMARY KEY,
			public_key TEXT NOT NULL,
			account_value REAL NOT NULL,
			unrealized_pnl REAL NOT NULL,
			source TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agent_fills (
			id TEXT PRIMARY KEY,
			public_key TEXT NOT NULL,
			fill_time INTEGER NOT NULL,
			data_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS nonces (
			nonce TEXT PRIMARY KEY,
			wallet_address TEXT,
			created_at TEXT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			state TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			code_verifier TEXT NOT NULL,
			invite_code TEXT,
			next_url TEXT,
			created_at TEXT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_strategies_status ON strategies(status);`,
		`CREATE INDEX IF NOT EXISTS idx_strategies_name ON strategies(name);`,
		`CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);`,
		`CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_users_x_id ON users(x_id);`,
		`CREATE INDEX IF NOT EXISTS idx_invite_code_usages_user ON invite_code_usages(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_accounts_status ON agent_accounts(status);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_snapshots_pub_created ON agent_snapshots(public_key, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_fills_pub_time ON agent_fills(public_key, fill_time);`,
		`CREATE INDEX IF NOT EXISTS idx_oauth_states_provider_created ON oauth_states(provider, created_at);`,

		// agent_accounts profile columns (safe to re-run)
		`ALTER TABLE agent_accounts ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_accounts ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_accounts ADD COLUMN category TEXT NOT NULL DEFAULT 'trend'`,
		`ALTER TABLE agent_accounts ADD COLUMN agent_level TEXT NOT NULL DEFAULT 'intern'`,

		// agent_reviews table
		`CREATE TABLE IF NOT EXISTS agent_reviews (
			id TEXT PRIMARY KEY,
			public_key TEXT NOT NULL,
			user_id TEXT NOT NULL,
			rating INTEGER NOT NULL,
			comment TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(public_key, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_reviews_pub ON agent_reviews(public_key)`,

		// vault / EVM columns
		`ALTER TABLE agent_accounts ADD COLUMN vault_address TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_accounts ADD COLUMN evm_balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_accounts ADD COLUMN agent_status TEXT NOT NULL DEFAULT 'inactive'`,

		// key-value settings (admin-managed config persisted to DB)
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// performance fee column
		`ALTER TABLE agent_accounts ADD COLUMN performance_fee REAL NOT NULL DEFAULT 0.2`,
	}

	for _, stmt := range schema {
		if _, err := s.db.Exec(stmt); err != nil {
			// ALTER TABLE ADD COLUMN fails if column already exists — ignore those
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *Store) saveNonce(nonce string, wallet string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO nonces(nonce, wallet_address, created_at, used) VALUES(?, ?, ?, 0)`,
		nonce,
		strings.ToLower(strings.TrimSpace(wallet)),
		nowISO(),
	)
	return err
}

func (s *Store) consumeNonce(nonce string, wallet string) (bool, error) {
	wallet = strings.ToLower(strings.TrimSpace(wallet))

	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var storedWallet string
	var createdAt string
	var used int
	err = tx.QueryRow(`SELECT IFNULL(wallet_address, ''), created_at, used FROM nonces WHERE nonce = ?`, nonce).Scan(&storedWallet, &createdAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if used == 1 {
		_ = tx.Rollback()
		return false, nil
	}
	if storedWallet != "" && wallet != "" && storedWallet != wallet {
		_ = tx.Rollback()
		return false, nil
	}

	created, parseErr := time.Parse(time.RFC3339, createdAt)
	if parseErr == nil {
		if time.Since(created) > 15*time.Minute {
			_ = tx.Rollback()
			return false, nil
		}
	}

	if _, err = tx.Exec(`UPDATE nonces SET used = 1 WHERE nonce = ?`, nonce); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) saveOAuthState(provider string, state string, codeVerifier string, inviteCode string, nextURL string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO oauth_states(state, provider, code_verifier, invite_code, next_url, created_at, used) VALUES(?, ?, ?, ?, ?, ?, 0)`,
		strings.TrimSpace(state),
		strings.ToLower(strings.TrimSpace(provider)),
		strings.TrimSpace(codeVerifier),
		strings.ToUpper(strings.TrimSpace(inviteCode)),
		strings.TrimSpace(nextURL),
		nowISO(),
	)
	return err
}

func (s *Store) consumeOAuthState(provider string, state string, ttl time.Duration) (OAuthState, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	state = strings.TrimSpace(state)
	if state == "" {
		return OAuthState{}, errors.New("empty oauth state")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return OAuthState{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item := OAuthState{}
	var used int
	err = tx.QueryRow(`
		SELECT provider, state, code_verifier, IFNULL(invite_code, ''), IFNULL(next_url, ''), created_at, used
		FROM oauth_states
		WHERE state = ?`,
		state,
	).Scan(&item.Provider, &item.State, &item.CodeVerifier, &item.InviteCode, &item.NextURL, &item.CreatedAt, &used)
	if err != nil {
		return OAuthState{}, err
	}
	if item.Provider != provider {
		return OAuthState{}, errors.New("oauth provider mismatch")
	}
	if used == 1 {
		return OAuthState{}, errors.New("oauth state already consumed")
	}
	createdAt, parseErr := time.Parse(time.RFC3339, item.CreatedAt)
	if parseErr != nil || time.Since(createdAt) > ttl {
		return OAuthState{}, errors.New("oauth state expired")
	}

	if _, err = tx.Exec(`UPDATE oauth_states SET used = 1 WHERE state = ?`, state); err != nil {
		return OAuthState{}, err
	}
	if _, err = tx.Exec(`DELETE FROM oauth_states WHERE provider = ? AND created_at < ?`, provider, time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339)); err != nil {
		return OAuthState{}, err
	}
	if err = tx.Commit(); err != nil {
		return OAuthState{}, err
	}
	return item, nil
}

func (s *Store) getUserByXID(xID string) (User, error) {
	xID = strings.ToLower(strings.TrimSpace(xID))
	var payload string
	err := s.db.QueryRow(`SELECT data_json FROM users WHERE x_id = ?`, xID).Scan(&payload)
	if err != nil {
		return User{}, err
	}
	var user User
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) getOrCreateUserByXID(xID string) (User, error) {
	xID = strings.ToLower(strings.TrimSpace(xID))
	user, err := s.getUserByXID(xID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}

	user = User{
		ID:        newID("user"),
		XID:       xID,
		Email:     xID + "@x.local",
		Name:      "User " + xID[:8],
		CreatedAt: nowISO(),
	}
	if err := s.saveUser(user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) saveUser(user User) error {
	if user.ID == "" {
		user.ID = newID("user")
	}
	if user.CreatedAt == "" {
		user.CreatedAt = nowISO()
	}
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}
	t := nowISO()
	_, err = s.db.Exec(`
		INSERT INTO users(id, x_id, email, name, data_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			x_id = excluded.x_id,
			email = excluded.email,
			name = excluded.name,
			data_json = excluded.data_json,
			updated_at = excluded.updated_at`,
		user.ID,
		strings.ToLower(user.XID),
		user.Email,
		user.Name,
		string(payload),
		t,
		t,
	)
	return err
}

func (s *Store) getUserByID(id string) (User, error) {
	var payload string
	err := s.db.QueryRow(`SELECT data_json FROM users WHERE id = ?`, id).Scan(&payload)
	if err != nil {
		return User{}, err
	}
	var user User
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) listUsers(search string) ([]User, error) {
	query := `SELECT data_json FROM users`
	args := make([]any, 0)
	if strings.TrimSpace(search) != "" {
		query += ` WHERE lower(name) LIKE ? OR lower(email) LIKE ? OR lower(x_id) LIKE ?`
		pattern := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		args = append(args, pattern, pattern, pattern)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var u User
		if err := json.Unmarshal([]byte(payload), &u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) listInviteCodes() ([]InviteCode, error) {
	rows, err := s.db.Query(`
		SELECT id, code, IFNULL(description, ''), status, IFNULL(max_uses, 0), used_count, created_at
		FROM invite_codes
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]InviteCode, 0)
	for rows.Next() {
		var item InviteCode
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Description,
			&item.Status,
			&item.MaxUses,
			&item.UsedCount,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) createInviteCode(code InviteCode) (InviteCode, error) {
	if code.ID == "" {
		code.ID = newID("invite")
	}
	if code.CreatedAt == "" {
		code.CreatedAt = nowISO()
	}
	if code.Status == "" {
		code.Status = "active"
	}
	if _, err := s.db.Exec(`
		INSERT INTO invite_codes(id, code, description, status, max_uses, used_count, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, code.ID, strings.ToUpper(strings.TrimSpace(code.Code)), code.Description, code.Status, code.MaxUses, code.UsedCount, code.CreatedAt); err != nil {
		return InviteCode{}, err
	}
	code.Code = strings.ToUpper(strings.TrimSpace(code.Code))
	return code, nil
}

func (s *Store) updateInviteCode(id string, patch InviteCode) (InviteCode, error) {
	var item InviteCode
	err := s.db.QueryRow(`
		SELECT id, code, IFNULL(description, ''), status, IFNULL(max_uses, 0), used_count, created_at
		FROM invite_codes WHERE id = ?`, id,
	).Scan(&item.ID, &item.Code, &item.Description, &item.Status, &item.MaxUses, &item.UsedCount, &item.CreatedAt)
	if err != nil {
		return InviteCode{}, err
	}

	if patch.Description != "" {
		item.Description = patch.Description
	}
	if patch.MaxUses > 0 {
		item.MaxUses = patch.MaxUses
	}
	if patch.Status != "" {
		item.Status = patch.Status
	}

	_, err = s.db.Exec(`
		UPDATE invite_codes
		SET description = ?, max_uses = ?, status = ?
		WHERE id = ?`,
		item.Description,
		item.MaxUses,
		item.Status,
		id,
	)
	if err != nil {
		return InviteCode{}, err
	}
	return item, nil
}

func (s *Store) createInviteCodesBatch(prefix string, count int, length int, maxUses int, description string) ([]InviteCode, error) {
	if count <= 0 {
		return []InviteCode{}, nil
	}
	if count > 5000 {
		count = 5000
	}
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	length = normalizeInviteCodeLength(length)

	suffixLength := length
	if prefix != "" {
		if len(prefix) > length-5 {
			return nil, errors.New("prefix too long for target length")
		}
		suffixLength = length - len(prefix) - 1
	}

	created := make([]InviteCode, 0, count)
	for i := 0; i < count; i++ {
		code := randomAlphaNum(length)
		if prefix != "" {
			code = fmt.Sprintf("%s-%s", prefix, randomAlphaNum(suffixLength))
		}
		item, err := s.createInviteCode(InviteCode{
			Code:        code,
			Description: description,
			MaxUses:     maxUses,
			Status:      "active",
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				i--
				continue
			}
			return nil, err
		}
		created = append(created, item)
	}
	return created, nil
}

func (s *Store) deleteInviteCodes(ids []string) (int, error) {
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	if len(filtered) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(filtered)), ",")
	args := make([]any, 0, len(filtered))
	for _, id := range filtered {
		args = append(args, id)
	}

	res, err := s.db.Exec(`DELETE FROM invite_codes WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (s *Store) listUnusedInviteCodes(limit int) ([]string, error) {
	if limit <= 0 || limit > 10000 {
		limit = 10000
	}
	rows, err := s.db.Query(`
		SELECT code
		FROM invite_codes
		WHERE status = 'active'
		  AND used_count = 0
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		items = append(items, code)
	}
	return items, rows.Err()
}

func (s *Store) consumeInviteCodeForUser(code string, userID string) (InviteCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return InviteCode{}, errors.New("empty invite code")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return InviteCode{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var invite InviteCode
	err = tx.QueryRow(`
		SELECT id, code, IFNULL(description, ''), status, IFNULL(max_uses, 0), used_count, created_at
		FROM invite_codes
		WHERE code = ?`, code,
	).Scan(&invite.ID, &invite.Code, &invite.Description, &invite.Status, &invite.MaxUses, &invite.UsedCount, &invite.CreatedAt)
	if err != nil {
		return InviteCode{}, err
	}

	if invite.Status != "active" {
		return InviteCode{}, errors.New("invite code inactive")
	}
	if invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses {
		return InviteCode{}, errors.New("invite code limit reached")
	}

	var existed int
	if err = tx.QueryRow(`SELECT COUNT(1) FROM invite_code_usages WHERE user_id = ?`, userID).Scan(&existed); err != nil {
		return InviteCode{}, err
	}
	if existed > 0 {
		return InviteCode{}, errors.New("user already used invite code")
	}

	if _, err = tx.Exec(`UPDATE invite_codes SET used_count = used_count + 1 WHERE id = ?`, invite.ID); err != nil {
		return InviteCode{}, err
	}
	if _, err = tx.Exec(`
		INSERT INTO invite_code_usages(id, code_id, code, user_id, used_at)
		VALUES(?, ?, ?, ?, ?)`,
		newID("invite_use"),
		invite.ID,
		invite.Code,
		userID,
		nowISO(),
	); err != nil {
		return InviteCode{}, err
	}
	if err = tx.Commit(); err != nil {
		return InviteCode{}, err
	}
	invite.UsedCount += 1
	return invite, nil
}

func (s *Store) importAgentPrivateKeys(privateKeys []string, fixedKey string) (AgentImportResult, error) {
	result := AgentImportResult{PublicKeys: make([]string, 0)}
	for _, pk := range privateKeys {
		pub, err := derivePublicKeyFromPrivateKey(pk)
		if err != nil {
			result.Invalid++
			continue
		}
		encrypted, err := encryptSecret(pk, fixedKey)
		if err != nil {
			return result, err
		}
		item := AgentAccount{
			ID:        newID("acct"),
			PublicKey: pub,
			Status:    "unused",
			CreatedAt: nowISO(),
		}
		_, err = s.db.Exec(`
			INSERT INTO agent_accounts(id, public_key, encrypted_private_key, status, created_at)
			VALUES(?, ?, ?, ?, ?)`,
			item.ID,
			item.PublicKey,
			encrypted,
			item.Status,
			item.CreatedAt,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				result.Duplicates++
				continue
			}
			return result, err
		}
		result.Imported++
		result.PublicKeys = append(result.PublicKeys, pub)
	}
	return result, nil
}

func (s *Store) listAgentAccounts(status string) ([]AgentAccount, error) {
	query := `
		SELECT a.id, a.public_key, a.status, IFNULL(a.assigned_user_id, ''), IFNULL(u.name, ''), IFNULL(a.assigned_at, ''), a.created_at
		FROM agent_accounts a
		LEFT JOIN users u ON u.id = a.assigned_user_id
		WHERE 1=1`
	args := make([]any, 0)
	if status != "" {
		query += ` AND a.status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY a.created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AgentAccount, 0)
	for rows.Next() {
		item := AgentAccount{}
		if err := rows.Scan(
			&item.ID,
			&item.PublicKey,
			&item.Status,
			&item.AssignedUserID,
			&item.AssignedUserName,
			&item.AssignedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) assignUnusedAgentAccount(userID string) (AgentAccount, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AgentAccount{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item := AgentAccount{}
	err = tx.QueryRow(`
		SELECT id, public_key, status, IFNULL(assigned_user_id, ''), IFNULL(assigned_at, ''), created_at
		FROM agent_accounts
		WHERE status = 'unused'
		ORDER BY created_at ASC
		LIMIT 1`,
	).Scan(&item.ID, &item.PublicKey, &item.Status, &item.AssignedUserID, &item.AssignedAt, &item.CreatedAt)
	if err != nil {
		return AgentAccount{}, err
	}

	item.Status = "assigned"
	item.AssignedUserID = userID
	item.AssignedAt = nowISO()
	if _, err = tx.Exec(`
		UPDATE agent_accounts
		SET status = 'assigned', assigned_user_id = ?, assigned_at = ?
		WHERE id = ?`,
		item.AssignedUserID,
		item.AssignedAt,
		item.ID,
	); err != nil {
		return AgentAccount{}, err
	}
	if err = tx.Commit(); err != nil {
		return AgentAccount{}, err
	}
	return item, nil
}

func (s *Store) consumeInviteAndAssignAccount(code string, userID string) (InviteCode, AgentAccount, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return InviteCode{}, AgentAccount{}, errors.New("empty invite code")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var invite InviteCode
	err = tx.QueryRow(`
		SELECT id, code, IFNULL(description, ''), status, IFNULL(max_uses, 0), used_count, created_at
		FROM invite_codes
		WHERE code = ?`, code,
	).Scan(&invite.ID, &invite.Code, &invite.Description, &invite.Status, &invite.MaxUses, &invite.UsedCount, &invite.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InviteCode{}, AgentAccount{}, errInviteCodeNotFound
		}
		return InviteCode{}, AgentAccount{}, err
	}
	if invite.Status != "active" {
		return InviteCode{}, AgentAccount{}, errors.New("invite code inactive")
	}
	if invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses {
		return InviteCode{}, AgentAccount{}, errors.New("invite code limit reached")
	}

	var usageCount int
	if err = tx.QueryRow(`SELECT COUNT(1) FROM invite_code_usages WHERE user_id = ?`, userID).Scan(&usageCount); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	if usageCount > 0 {
		return InviteCode{}, AgentAccount{}, errors.New("user already used invite code")
	}

	account := AgentAccount{}
	err = tx.QueryRow(`
		SELECT id, public_key, status, IFNULL(assigned_user_id, ''), IFNULL(assigned_at, ''), created_at
		FROM agent_accounts
		WHERE status = 'unused'
		ORDER BY created_at ASC
		LIMIT 1`,
	).Scan(&account.ID, &account.PublicKey, &account.Status, &account.AssignedUserID, &account.AssignedAt, &account.CreatedAt)
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}

	if _, err = tx.Exec(`UPDATE invite_codes SET used_count = used_count + 1 WHERE id = ?`, invite.ID); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	if _, err = tx.Exec(`
		INSERT INTO invite_code_usages(id, code_id, code, user_id, used_at)
		VALUES(?, ?, ?, ?, ?)`,
		newID("invite_use"),
		invite.ID,
		invite.Code,
		userID,
		nowISO(),
	); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}

	account.Status = "assigned"
	account.AssignedUserID = userID
	account.AssignedAt = nowISO()
	if _, err = tx.Exec(`
		UPDATE agent_accounts
		SET status = 'assigned', assigned_user_id = ?, assigned_at = ?
		WHERE id = ?`,
		account.AssignedUserID,
		account.AssignedAt,
		account.ID,
	); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}

	if err = tx.Commit(); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	invite.UsedCount += 1
	return invite, account, nil
}

func (s *Store) getAgentAccountByUserID(userID string) (AgentAccount, error) {
	item := AgentAccount{}
	err := s.db.QueryRow(`
		SELECT id, public_key, status, IFNULL(assigned_user_id, ''), IFNULL(assigned_at, ''), created_at
		FROM agent_accounts
		WHERE assigned_user_id = ?
		ORDER BY assigned_at DESC
		LIMIT 1`, userID,
	).Scan(&item.ID, &item.PublicKey, &item.Status, &item.AssignedUserID, &item.AssignedAt, &item.CreatedAt)
	if err != nil {
		return AgentAccount{}, err
	}
	return item, nil
}

func (s *Store) saveAgentSnapshot(publicKey string, accountValue float64, unrealizedPNL float64, source string) (AgentSnapshot, error) {
	item := AgentSnapshot{
		ID:            newID("snap"),
		PublicKey:     publicKey,
		AccountValue:  accountValue,
		UnrealizedPNL: unrealizedPNL,
		Source:        source,
		CreatedAt:     nowISO(),
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_snapshots(id, public_key, account_value, unrealized_pnl, source, created_at)
		VALUES(?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.PublicKey,
		item.AccountValue,
		item.UnrealizedPNL,
		item.Source,
		item.CreatedAt,
	)
	if err != nil {
		return AgentSnapshot{}, err
	}
	return item, nil
}

func (s *Store) listAgentSnapshots(publicKey string, limit int, period string) ([]AgentSnapshot, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	// Compute time filter from period
	var since string
	switch strings.ToUpper(strings.TrimSpace(period)) {
	case "1D":
		since = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	case "1W":
		since = time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	case "1M":
		since = time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	default:
		// ALL or empty — no time filter
	}

	args := []any{strings.ToLower(publicKey)}
	query := `
		SELECT id, public_key, account_value, unrealized_pnl, source, created_at
		FROM agent_snapshots
		WHERE public_key = ?`
	if since != "" {
		query += ` AND created_at >= ?`
		args = append(args, since)
	}
	query += `
		ORDER BY created_at DESC
		LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AgentSnapshot, 0)
	for rows.Next() {
		item := AgentSnapshot{}
		if err := rows.Scan(&item.ID, &item.PublicKey, &item.AccountValue, &item.UnrealizedPNL, &item.Source, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) listAgentStats(search string) ([]AgentMarketItem, error) {
	query := `
		SELECT a.public_key,
		       IFNULL(a.name, ''),
		       IFNULL(a.description, ''),
		       IFNULL(a.assigned_user_id, ''),
		       IFNULL(u.name, ''),
		       IFNULL(latest.account_value, 0),
		       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2)
		FROM agent_accounts a
		LEFT JOIN users u ON u.id = a.assigned_user_id
		LEFT JOIN (
			SELECT public_key, account_value, created_at
			FROM agent_snapshots s1
			WHERE s1.created_at = (
				SELECT MAX(s2.created_at) FROM agent_snapshots s2 WHERE s2.public_key = s1.public_key
			)
		) latest ON latest.public_key = a.public_key
		LEFT JOIN (
			SELECT public_key, account_value
			FROM agent_snapshots s1
			WHERE s1.created_at = (
				SELECT MIN(s2.created_at) FROM agent_snapshots s2 WHERE s2.public_key = s1.public_key
			)
		) first ON first.public_key = a.public_key
		WHERE a.status = 'assigned'`
	args := make([]any, 0)
	if s := strings.TrimSpace(search); s != "" {
		query += ` AND (lower(a.public_key) LIKE ? OR lower(u.name) LIKE ? OR lower(a.name) LIKE ?)`
		pattern := "%" + strings.ToLower(s) + "%"
		args = append(args, pattern, pattern, pattern)
	}
	query += ` ORDER BY a.assigned_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AgentMarketItem, 0)
	for rows.Next() {
		var item AgentMarketItem
		if err := rows.Scan(
			&item.PublicKey,
			&item.Name,
			&item.Description,
			&item.UserID,
			&item.UserName,
			&item.AccountValue,
			&item.TotalPnL,
			&item.LastSyncedAt,
			&item.VaultAddress,
			&item.EVMBalance,
			&item.AgentStatus,
			&item.PerformanceFee,
		); err != nil {
			return nil, err
		}
		item.TVL = item.AccountValue + item.EVMBalance
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) getAgentStats(publicKey string) (AgentMarketItem, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var item AgentMarketItem
	err := s.db.QueryRow(`
		SELECT a.public_key,
		       IFNULL(a.name, ''),
		       IFNULL(a.description, ''),
		       IFNULL(a.assigned_user_id, ''),
		       IFNULL(u.name, ''),
		       IFNULL(latest.account_value, 0),
		       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2)
		FROM agent_accounts a
		LEFT JOIN users u ON u.id = a.assigned_user_id
		LEFT JOIN (
			SELECT public_key, account_value, created_at
			FROM agent_snapshots s1
			WHERE s1.created_at = (
				SELECT MAX(s2.created_at) FROM agent_snapshots s2 WHERE s2.public_key = s1.public_key
			)
		) latest ON latest.public_key = a.public_key
		LEFT JOIN (
			SELECT public_key, account_value
			FROM agent_snapshots s1
			WHERE s1.created_at = (
				SELECT MIN(s2.created_at) FROM agent_snapshots s2 WHERE s2.public_key = s1.public_key
			)
		) first ON first.public_key = a.public_key
		WHERE a.public_key = ? AND a.status = 'assigned'`,
		publicKey,
	).Scan(
		&item.PublicKey,
		&item.Name,
		&item.Description,
		&item.UserID,
		&item.UserName,
		&item.AccountValue,
		&item.TotalPnL,
		&item.LastSyncedAt,
		&item.VaultAddress,
		&item.EVMBalance,
		&item.AgentStatus,
		&item.PerformanceFee,
	)
	if err != nil {
		return AgentMarketItem{}, err
	}
	item.TVL = item.AccountValue + item.EVMBalance
	return item, nil
}

func (s *Store) updateVaultData(publicKey string, vaultAddress string, evmBalance float64, agentStatus string) error {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	_, err := s.db.Exec(`
		UPDATE agent_accounts
		SET vault_address = ?, evm_balance = ?, agent_status = ?
		WHERE public_key = ?`,
		strings.TrimSpace(vaultAddress),
		evmBalance,
		strings.TrimSpace(agentStatus),
		publicKey,
	)
	return err
}

func (s *Store) updateAgentProfile(publicKey string, name string, description string, performanceFee float64) error {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	res, err := s.db.Exec(`
		UPDATE agent_accounts
		SET name = ?, description = ?, performance_fee = ?
		WHERE public_key = ?`,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		performanceFee,
		publicKey,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) upsertAgentFill(publicKey string, fillID string, fillTime int64, rawJSON string) error {
	if fillID == "" {
		fillID = fmt.Sprintf("%s_%d", strings.ToLower(publicKey), fillTime)
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_fills(id, public_key, fill_time, data_json, created_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			fill_time = excluded.fill_time,
			data_json = excluded.data_json`,
		fillID,
		strings.ToLower(publicKey),
		fillTime,
		rawJSON,
		nowISO(),
	)
	return err
}

func (s *Store) listAgentFills(publicKey string, limit int) ([]VaultFill, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT data_json FROM agent_fills
		WHERE public_key = ?
		ORDER BY fill_time DESC
		LIMIT ?`,
		strings.ToLower(publicKey),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fills := make([]VaultFill, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var f VaultFill
		if err := json.Unmarshal([]byte(raw), &f); err != nil {
			continue
		}
		fills = append(fills, f)
	}
	return fills, nil
}

func (s *Store) getAgentCreatedAt(publicKey string) string {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var createdAt string
	s.db.QueryRow(`SELECT created_at FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&createdAt)
	return createdAt
}

func (s *Store) verifyInviteCode(code string) (bool, string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var maxUses int
	var usedCount int
	var status string
	err := s.db.QueryRow(`SELECT IFNULL(max_uses, 0), used_count, status FROM invite_codes WHERE code = ?`, code).Scan(&maxUses, &usedCount, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "code_not_found", nil
	}
	if err != nil {
		return false, "", err
	}
	if status != "active" {
		return false, "code_inactive", nil
	}
	if maxUses > 0 && usedCount >= maxUses {
		return false, "code_limit_reached", nil
	}
	return true, "ok", nil
}

func (s *Store) listAdmins() ([]AdminUser, error) {
	rows, err := s.db.Query(`SELECT id, email, name, created_at FROM admins ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		var created string
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) createAdmin(email string, name string, password string) (AdminUser, error) {
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedName := strings.TrimSpace(name)
	trimmedPassword := strings.TrimSpace(password)
	if trimmedEmail == "" {
		return AdminUser{}, errors.New("admin email required")
	}
	if trimmedPassword == "" {
		return AdminUser{}, errors.New("admin password required")
	}
	if trimmedName == "" {
		trimmedName = "Admin"
	}

	id := newID("admin")
	createdAt := nowISO()
	_, err := s.db.Exec(`
		INSERT INTO admins(id, email, password_hash, name, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, id, trimmedEmail, hashPassword(trimmedPassword), trimmedName, createdAt)
	if err != nil {
		return AdminUser{}, err
	}
	return s.getAdminByID(id)
}

func (s *Store) updateAdminPassword(id string, newPassword string) error {
	trimmedID := strings.TrimSpace(id)
	trimmedPassword := strings.TrimSpace(newPassword)
	if trimmedID == "" {
		return errors.New("admin id required")
	}
	if trimmedPassword == "" {
		return errors.New("admin password required")
	}

	res, err := s.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?`, hashPassword(trimmedPassword), trimmedID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) deleteAdmin(id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("admin id required")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var total int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM admins`).Scan(&total); err != nil {
		return err
	}
	if total <= 1 {
		return errors.New("cannot_delete_last_admin")
	}

	res, err := tx.Exec(`DELETE FROM admins WHERE id = ?`, trimmedID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (s *Store) getAdminByEmail(email string) (AdminUser, string, error) {
	var admin AdminUser
	var passwordHash string
	var created string
	err := s.db.QueryRow(`SELECT id, email, name, password_hash, created_at FROM admins WHERE lower(email) = ?`, strings.ToLower(strings.TrimSpace(email))).Scan(
		&admin.ID,
		&admin.Email,
		&admin.Name,
		&passwordHash,
		&created,
	)
	if err != nil {
		return AdminUser{}, "", err
	}
	admin.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return admin, passwordHash, nil
}

func (s *Store) getAdminByID(id string) (AdminUser, error) {
	var admin AdminUser
	var created string
	err := s.db.QueryRow(`SELECT id, email, name, created_at FROM admins WHERE id = ?`, id).Scan(
		&admin.ID,
		&admin.Email,
		&admin.Name,
		&created,
	)
	if err != nil {
		return AdminUser{}, err
	}
	admin.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return admin, nil
}

func (s *Store) getAdminPasswordHashByID(id string) (string, error) {
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM admins WHERE id = ?`, id).Scan(&hash)
	return hash, err
}

func (s *Store) deleteUsers(ids []string) (int, error) {
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(filtered)), ",")
	args := make([]any, 0, len(filtered))
	for _, id := range filtered {
		args = append(args, id)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`UPDATE agent_accounts SET assigned_user_id = NULL, assigned_at = NULL WHERE assigned_user_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM invite_code_usages WHERE user_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM agent_reviews WHERE user_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	res, err := tx.Exec(`DELETE FROM users WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (s *Store) deleteAgentAccounts(publicKeys []string) (int, error) {
	filtered := make([]string, 0, len(publicKeys))
	for _, pk := range publicKeys {
		trimmed := strings.TrimSpace(pk)
		if trimmed != "" {
			filtered = append(filtered, strings.ToLower(trimmed))
		}
	}
	if len(filtered) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(filtered)), ",")
	args := make([]any, 0, len(filtered))
	for _, pk := range filtered {
		args = append(args, pk)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM agent_snapshots WHERE public_key IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM agent_fills WHERE public_key IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM agent_reviews WHERE public_key IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	res, err := tx.Exec(`DELETE FROM agent_accounts WHERE public_key IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (s *Store) getSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

func (s *Store) setSetting(key string, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings(key, value, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, nowISO(),
	)
	return err
}

func (s *Store) getSettingDefault(key string, defaultVal string) string {
	val, err := s.getSetting(key)
	if err != nil || strings.TrimSpace(val) == "" {
		return defaultVal
	}
	return val
}

func (s *Store) listAssignedPublicKeys() ([]string, error) {
	rows, err := s.db.Query(`SELECT public_key FROM agent_accounts WHERE status = 'assigned'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s *Store) dashboardStats() (DashboardStats, error) {
	stats := DashboardStats{}
	queries := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(1) FROM users`, &stats.TotalUsers},
		{`SELECT COUNT(1) FROM agent_accounts`, &stats.TotalAgentAccounts},
		{`SELECT COUNT(1) FROM agent_accounts WHERE status = 'assigned'`, &stats.AssignedAgents},
		{`SELECT COUNT(1) FROM agent_accounts WHERE status = 'unused'`, &stats.UnusedAgents},
		{`SELECT COUNT(1) FROM invite_codes`, &stats.TotalInviteCodes},
		{`SELECT COUNT(1) FROM invite_codes WHERE status = 'active'`, &stats.ActiveInviteCodes},
	}
	for _, item := range queries {
		if err := s.db.QueryRow(item.query).Scan(item.dest); err != nil {
			return DashboardStats{}, err
		}
	}
	return stats, nil
}

func (s *Store) getDailySlots() (DailySlots, error) {
	totalStr := s.getSettingDefault("daily_slots_total", "1000")
	resetHourStr := s.getSettingDefault("daily_slots_reset_hour", "0")
	consumedStr := s.getSettingDefault("daily_slots_consumed", "0")
	resetAtStr := s.getSettingDefault("daily_slots_reset_at", "")

	total, _ := strconv.Atoi(totalStr)
	if total <= 0 {
		total = 1000
	}
	resetHour, _ := strconv.Atoi(resetHourStr)
	if resetHour < 0 || resetHour > 23 {
		resetHour = 0
	}
	consumed, _ := strconv.Atoi(consumedStr)

	now := time.Now().UTC()
	cycleStart := time.Date(now.Year(), now.Month(), now.Day(), resetHour, 0, 0, 0, time.UTC)
	if now.Before(cycleStart) {
		cycleStart = cycleStart.Add(-24 * time.Hour)
	}
	cycleEnd := cycleStart.Add(24 * time.Hour)

	// Lazy reset: if stored reset_at is before current cycle start, reset consumed
	needsReset := false
	if resetAtStr == "" {
		needsReset = true
	} else {
		storedResetAt, err := time.Parse(time.RFC3339, resetAtStr)
		if err != nil || storedResetAt.Before(cycleStart) {
			needsReset = true
		}
	}
	if needsReset {
		consumed = 0
		_ = s.setSetting("daily_slots_consumed", "0")
		_ = s.setSetting("daily_slots_reset_at", cycleStart.Format(time.RFC3339))
	}

	remaining := total - consumed
	if remaining < 0 {
		remaining = 0
	}

	return DailySlots{
		Total:     total,
		Consumed:  consumed,
		Remaining: remaining,
		ResetHour: resetHour,
		ResetsAt:  cycleEnd,
	}, nil
}

func (s *Store) consumeDailySlot() (DailySlots, error) {
	// Ensure cycle is current first
	slots, err := s.getDailySlots()
	if err != nil {
		return DailySlots{}, err
	}
	if slots.Remaining <= 0 {
		return slots, errors.New("no_slots_remaining")
	}

	newConsumed := slots.Consumed + 1
	if err := s.setSetting("daily_slots_consumed", strconv.Itoa(newConsumed)); err != nil {
		return DailySlots{}, err
	}
	slots.Consumed = newConsumed
	slots.Remaining = slots.Total - newConsumed
	if slots.Remaining < 0 {
		slots.Remaining = 0
	}
	return slots, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullIfZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func randomAlphaNum(size int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if size <= 0 {
		size = 8
	}
	var b strings.Builder
	b.Grow(size)
	for i := 0; i < size; i++ {
		b.WriteByte(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func normalizeInviteCodeLength(length int) int {
	if length <= 0 {
		return 12
	}
	if length < 6 {
		return 6
	}
	if length > 64 {
		return 64
	}
	return length
}
