package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	defer func() { _ = tx.Rollback() }()

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
	res, err := tx.Exec(`
		UPDATE agent_accounts
		SET status = 'assigned', assigned_user_id = ?, assigned_at = ?
		WHERE id = ? AND status = 'unused'`,
		item.AssignedUserID,
		item.AssignedAt,
		item.ID,
	)
	if err != nil {
		return AgentAccount{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return AgentAccount{}, err
	}
	if affected == 0 {
		return AgentAccount{}, sql.ErrNoRows
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
	defer func() { _ = tx.Rollback() }()

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
	var assignedCount int
	if err = tx.QueryRow(`SELECT COUNT(1) FROM agent_accounts WHERE assigned_user_id = ?`, userID).Scan(&assignedCount); err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	if assignedCount > 0 {
		return InviteCode{}, AgentAccount{}, errors.New("user already has agent")
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
	// Check remaining slots: configured total minus actually consumed
	totalStr, err := s.getSettingDefaultTx(tx, "intern_slots_total", "100")
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	slotsTotal, _ := strconv.Atoi(totalStr)
	if slotsTotal <= 0 {
		slotsTotal = 100
	}
	var slotsConsumed int
	err = tx.QueryRow(`SELECT COUNT(1) FROM agent_accounts a
		JOIN agent_vaults v ON lower(a.vault_address) = lower(v.vault_address)
		WHERE a.vault_address != '' AND v.valid = 1 AND a.status = 'assigned'`).Scan(&slotsConsumed)
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	if slotsConsumed >= slotsTotal {
		return InviteCode{}, AgentAccount{}, errNoSlotsRemaining
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
	res, err := tx.Exec(`
		UPDATE agent_accounts
		SET status = 'assigned', assigned_user_id = ?, assigned_at = ?
		WHERE id = ? AND status = 'unused'`,
		account.AssignedUserID,
		account.AssignedAt,
		account.ID,
	)
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return InviteCode{}, AgentAccount{}, err
	}
	if affected == 0 {
		return InviteCode{}, AgentAccount{}, sql.ErrNoRows
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

func (s *Store) revokeUserAgent(publicKey string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var assignedUserID string
	err = tx.QueryRow(`SELECT IFNULL(assigned_user_id, '') FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&assignedUserID)
	if err != nil {
		return err
	}
	if assignedUserID == "" {
		return errors.New("agent not assigned")
	}

	if _, err = tx.Exec(`UPDATE agent_accounts SET status = 'unused', assigned_user_id = NULL, assigned_at = NULL WHERE public_key = ?`, publicKey); err != nil {
		return err
	}

	var payload string
	err = tx.QueryRow(`SELECT data_json FROM users WHERE id = ?`, assignedUserID).Scan(&payload)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// user may have been deleted, just commit agent changes
	} else {
		var user User
		if err = json.Unmarshal([]byte(payload), &user); err != nil {
			return err
		}
		user.AgentPublicKey = ""
		user.AgentAssignedAt = ""
		userPayload, err2 := json.Marshal(user)
		if err2 != nil {
			err = err2
			return err
		}
		if _, err = tx.Exec(`UPDATE users SET data_json = ?, updated_at = ? WHERE id = ?`, string(userPayload), nowISO(), assignedUserID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) revokeUserInviteCode(userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var payload string
	err = tx.QueryRow(`SELECT data_json FROM users WHERE id = ?`, userID).Scan(&payload)
	if err != nil {
		return err
	}
	var user User
	if err = json.Unmarshal([]byte(payload), &user); err != nil {
		return err
	}

	// If user has an assigned agent, revoke it first (within the same tx)
	if user.AgentPublicKey != "" {
		if _, err = tx.Exec(`UPDATE agent_accounts SET status = 'unused', assigned_user_id = NULL, assigned_at = NULL WHERE public_key = ?`, user.AgentPublicKey); err != nil {
			return err
		}
		user.AgentPublicKey = ""
		user.AgentAssignedAt = ""
	}

	// Find usage record and decrement invite code used_count
	var codeID string
	err = tx.QueryRow(`SELECT code_id FROM invite_code_usages WHERE user_id = ? LIMIT 1`, userID).Scan(&codeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if codeID != "" {
		if _, err = tx.Exec(`UPDATE invite_codes SET used_count = used_count - 1 WHERE id = ? AND used_count > 0`, codeID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`DELETE FROM invite_code_usages WHERE user_id = ?`, userID); err != nil {
		return err
	}

	user.InviteCodeUsed = ""
	userPayload, err2 := json.Marshal(user)
	if err2 != nil {
		err = err2
		return err
	}
	if _, err = tx.Exec(`UPDATE users SET data_json = ?, updated_at = ? WHERE id = ?`, string(userPayload), nowISO(), userID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) reassignAgent(publicKey string, userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRow(`SELECT status FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&status)
	if err != nil {
		return err
	}
	if status != "unused" {
		return errors.New("agent not available")
	}

	var payload string
	err = tx.QueryRow(`SELECT data_json FROM users WHERE id = ?`, userID).Scan(&payload)
	if err != nil {
		return err
	}
	var user User
	if err = json.Unmarshal([]byte(payload), &user); err != nil {
		return err
	}
	if user.AgentPublicKey != "" {
		return errors.New("user already has agent")
	}

	now := nowISO()
	if _, err = tx.Exec(`UPDATE agent_accounts SET status = 'assigned', assigned_user_id = ?, assigned_at = ? WHERE public_key = ?`, userID, now, publicKey); err != nil {
		return err
	}

	user.AgentPublicKey = publicKey
	user.AgentAssignedAt = now
	userPayload, err2 := json.Marshal(user)
	if err2 != nil {
		err = err2
		return err
	}
	if _, err = tx.Exec(`UPDATE users SET data_json = ?, updated_at = ? WHERE id = ?`, string(userPayload), nowISO(), userID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
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
		since = timeAgo(1 * 24)
	case "1W":
		since = timeAgo(7 * 24)
	case "1M":
		since = timeAgo(30 * 24)
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
		       IFNULL(a.category, 'trend'),
		       IFNULL(a.assigned_user_id, ''),
		       IFNULL(u.name, ''),
		       IFNULL(latest.account_value, 0),
		       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2),
		       IFNULL(a.initial_capital, 0)
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
			&item.Category,
			&item.UserID,
			&item.UserName,
			&item.AccountValue,
			&item.TotalPnL,
			&item.LastSyncedAt,
			&item.VaultAddress,
			&item.EVMBalance,
			&item.AgentStatus,
			&item.PerformanceFee,
			&item.InitialCapital,
		); err != nil {
			return nil, err
		}
		item.TVL = item.AccountValue
		// When initialCapital is available, use it for more accurate PnL
		if item.InitialCapital > 0 {
			item.TotalPnL = item.AccountValue - item.InitialCapital
		}
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
		       IFNULL(a.category, 'trend'),
		       IFNULL(a.assigned_user_id, ''),
		       IFNULL(u.name, ''),
		       IFNULL(latest.account_value, 0),
		       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2),
		       IFNULL(a.initial_capital, 0)
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
		&item.Category,
		&item.UserID,
		&item.UserName,
		&item.AccountValue,
		&item.TotalPnL,
		&item.LastSyncedAt,
		&item.VaultAddress,
		&item.EVMBalance,
		&item.AgentStatus,
		&item.PerformanceFee,
		&item.InitialCapital,
	)
	if err != nil {
		return AgentMarketItem{}, err
	}
	item.TVL = item.AccountValue
	if item.InitialCapital > 0 {
		item.TotalPnL = item.AccountValue - item.InitialCapital
	}
	return item, nil
}

func (s *Store) updateVaultData(publicKey string, vaultAddress string, evmBalance float64, agentStatus string, initialCapital float64) error {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	_, err := s.db.Exec(`
		UPDATE agent_accounts
		SET vault_address = ?, evm_balance = ?, agent_status = ?, initial_capital = ?
		WHERE public_key = ?`,
		strings.TrimSpace(vaultAddress),
		evmBalance,
		strings.TrimSpace(agentStatus),
		initialCapital,
		publicKey,
	)
	return err
}

func (s *Store) updateAgentProfile(publicKey string, name *string, description *string, category *string, performanceFee *float64) error {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var item AgentMarketItem
	err := s.db.QueryRow(`
		SELECT IFNULL(name, ''), IFNULL(description, ''), IFNULL(category, 'trend'), IFNULL(performance_fee, 0.2)
		FROM agent_accounts
		WHERE public_key = ?`,
		publicKey,
	).Scan(&item.Name, &item.Description, &item.Category, &item.PerformanceFee)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}

	if name != nil {
		item.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		item.Description = strings.TrimSpace(*description)
	}
	if category != nil {
		item.Category = strings.ToLower(strings.TrimSpace(*category))
	}
	if performanceFee != nil {
		item.PerformanceFee = *performanceFee
	}

	res, err := s.db.Exec(`
		UPDATE agent_accounts
		SET name = ?, description = ?, category = ?, performance_fee = ?
		WHERE public_key = ?`,
		item.Name,
		item.Description,
		item.Category,
		item.PerformanceFee,
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

func (s *Store) listRecentFillsForActiveAgents(limit int) ([]VaultFill, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT f.data_json
		FROM agent_fills f
		INNER JOIN agent_accounts a ON lower(a.public_key) = lower(f.public_key)
		WHERE a.status = 'assigned' AND a.agent_status = 'active'
		ORDER BY f.fill_time DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fills := make([]VaultFill, 0, limit)
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
	return fills, rows.Err()
}

func (s *Store) getAgentCreatedAt(publicKey string) string {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var createdAt string
	s.db.QueryRow(`SELECT created_at FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&createdAt)
	return createdAt
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
	defer func() { _ = tx.Rollback() }()

	// Clear stale references from user profiles before deleting accounts.
	keySet := make(map[string]struct{}, len(filtered))
	for _, pk := range filtered {
		keySet[pk] = struct{}{}
	}
	userRows, err := tx.Query(`SELECT id, data_json FROM users`)
	if err != nil {
		return 0, err
	}
	type userPatch struct {
		id      string
		payload string
	}
	patches := make([]userPatch, 0)
	for userRows.Next() {
		var id string
		var payload string
		if err := userRows.Scan(&id, &payload); err != nil {
			_ = userRows.Close()
			return 0, err
		}
		var user User
		if err := json.Unmarshal([]byte(payload), &user); err != nil {
			continue
		}
		if _, ok := keySet[strings.ToLower(strings.TrimSpace(user.AgentPublicKey))]; !ok {
			continue
		}
		user.AgentPublicKey = ""
		user.AgentAssignedAt = ""
		updated, err := json.Marshal(user)
		if err != nil {
			_ = userRows.Close()
			return 0, err
		}
		patches = append(patches, userPatch{
			id:      id,
			payload: string(updated),
		})
	}
	if err := userRows.Err(); err != nil {
		_ = userRows.Close()
		return 0, err
	}
	_ = userRows.Close()
	for _, p := range patches {
		if _, err = tx.Exec(`UPDATE users SET data_json = ?, updated_at = ? WHERE id = ?`, p.payload, nowISO(), p.id); err != nil {
			return 0, err
		}
	}

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

func (s *Store) syncAgentVaultState(records []VaultRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Reset existing projection and then upsert active mappings from latest on-chain snapshot.
	if _, err = tx.Exec(`
		UPDATE agent_accounts
		SET vault_address = '',
		    evm_balance = 0,
		    agent_status = 'inactive',
		    initial_capital = 0`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		UPDATE agent_accounts
		SET vault_address = ?,
		    evm_balance = ?,
		    agent_status = ?,
		    initial_capital = ?
		WHERE lower(public_key) = lower(?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if strings.TrimSpace(r.UserAddress) == "" {
			continue
		}
		status := AgentStatusInactive
		if r.Valid {
			status = AgentStatusActive
		}
		if _, err = stmt.Exec(
			strings.ToLower(strings.TrimSpace(r.VaultAddress)),
			r.EVMBalance,
			status,
			r.InitialCapital,
			strings.ToLower(strings.TrimSpace(r.UserAddress)),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) getAgentEncryptedPrivateKey(publicKey string) (string, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var encrypted string
	err := s.db.QueryRow(`SELECT encrypted_private_key FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&encrypted)
	return encrypted, err
}

func (s *Store) getAgentDispatchInfo(publicKey string) (string, string, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var encrypted, vaultAddress string
	err := s.db.QueryRow(`SELECT encrypted_private_key, IFNULL(vault_address, '') FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&encrypted, &vaultAddress)
	if err != nil {
		return "", "", err
	}
	return encrypted, strings.ToLower(strings.TrimSpace(vaultAddress)), nil
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

// --- agent_vaults CRUD ---

func (s *Store) upsertAgentVault(v VaultRecord) error {
	now := nowISO()
	if v.CreatedAt == "" {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	validInt := 0
	if v.Valid {
		validInt = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_vaults(vault_address, user_address, evm_balance, initial_capital, valid, allocator_address, account_value, unrealized_pnl, last_synced_at, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vault_address) DO UPDATE SET
			user_address = excluded.user_address,
			evm_balance = excluded.evm_balance,
			initial_capital = excluded.initial_capital,
			valid = excluded.valid,
			allocator_address = excluded.allocator_address,
			updated_at = excluded.updated_at`,
		v.VaultAddress, v.UserAddress, v.EVMBalance, v.InitialCapital, validInt, v.AllocatorAddress, v.AccountValue, v.UnrealizedPnl, v.LastSyncedAt, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (s *Store) batchUpsertAgentVaults(vaults []VaultRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO agent_vaults(vault_address, user_address, evm_balance, initial_capital, valid, allocator_address, account_value, unrealized_pnl, last_synced_at, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vault_address) DO UPDATE SET
			user_address = excluded.user_address,
			evm_balance = excluded.evm_balance,
			initial_capital = excluded.initial_capital,
			valid = excluded.valid,
			allocator_address = excluded.allocator_address,
			updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := nowISO()
	for _, v := range vaults {
		if v.CreatedAt == "" {
			v.CreatedAt = now
		}
		v.UpdatedAt = now
		validInt := 0
		if v.Valid {
			validInt = 1
		}
		if _, err := stmt.Exec(v.VaultAddress, v.UserAddress, v.EVMBalance, v.InitialCapital, validInt, v.AllocatorAddress, v.AccountValue, v.UnrealizedPnl, v.LastSyncedAt, v.CreatedAt, v.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) listAgentVaults() ([]VaultRecord, error) {
	rows, err := s.db.Query(`
		SELECT vault_address, user_address, evm_balance, initial_capital, valid, allocator_address, account_value, unrealized_pnl, last_synced_at, IFNULL(sync_status,'ok'), IFNULL(sync_error,''), created_at, updated_at
		FROM agent_vaults
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]VaultRecord, 0)
	for rows.Next() {
		var v VaultRecord
		var validInt int
		if err := rows.Scan(&v.VaultAddress, &v.UserAddress, &v.EVMBalance, &v.InitialCapital, &validInt, &v.AllocatorAddress, &v.AccountValue, &v.UnrealizedPnl, &v.LastSyncedAt, &v.SyncStatus, &v.SyncError, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Valid = validInt != 0
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) deleteAgentVaults(vaultAddresses []string) (int, error) {
	filtered := make([]string, 0, len(vaultAddresses))
	seen := make(map[string]struct{}, len(vaultAddresses))
	for _, addr := range vaultAddresses {
		normalized := strings.ToLower(strings.TrimSpace(addr))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		filtered = append(filtered, normalized)
	}
	if len(filtered) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(filtered)), ",")
	args := make([]any, 0, len(filtered))
	for _, v := range filtered {
		args = append(args, v)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec(`
		UPDATE agent_accounts
		SET vault_address = '',
		    evm_balance = 0,
		    agent_status = 'inactive',
		    initial_capital = 0
		WHERE lower(vault_address) IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}

	res, err := tx.Exec(`DELETE FROM agent_vaults WHERE lower(vault_address) IN (`+placeholders+`)`, args...)
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

func (s *Store) getAgentVaultByUser(userAddress string) (VaultRecord, error) {
	userAddress = strings.ToLower(strings.TrimSpace(userAddress))
	var v VaultRecord
	var validInt int
	err := s.db.QueryRow(`
		SELECT vault_address, user_address, evm_balance, initial_capital, valid, allocator_address, account_value, unrealized_pnl, last_synced_at, IFNULL(sync_status,'ok'), IFNULL(sync_error,''), created_at, updated_at
		FROM agent_vaults
		WHERE user_address = ?`, userAddress).Scan(&v.VaultAddress, &v.UserAddress, &v.EVMBalance, &v.InitialCapital, &validInt, &v.AllocatorAddress, &v.AccountValue, &v.UnrealizedPnl, &v.LastSyncedAt, &v.SyncStatus, &v.SyncError, &v.CreatedAt, &v.UpdatedAt)
	v.Valid = validInt != 0
	return v, err
}

func (s *Store) updateVaultHyperliquidData(vaultAddress string, accountValue float64, unrealizedPnl float64) error {
	_, err := s.db.Exec(`
		UPDATE agent_vaults
		SET account_value = ?, unrealized_pnl = ?, last_synced_at = ?, updated_at = ?,
		    sync_status = 'ok', sync_error = ''
		WHERE vault_address = ?`,
		accountValue, unrealizedPnl, nowISO(), nowISO(), vaultAddress,
	)
	return err
}

func (s *Store) updateVaultSyncError(vaultAddress string, syncErr string) error {
	_, err := s.db.Exec(`
		UPDATE agent_vaults
		SET sync_status = 'error', sync_error = ?, last_synced_at = ?, updated_at = ?
		WHERE vault_address = ?`,
		syncErr, nowISO(), nowISO(), vaultAddress,
	)
	return err
}
