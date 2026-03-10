package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (s *Store) pickUnusedAgentAccountTx(tx *sql.Tx) (AgentAccount, error) {
	item := AgentAccount{}
	err := tx.QueryRow(`
		SELECT id, public_key, status, IFNULL(assigned_user_id, ''), IFNULL(assigned_at, ''), created_at
		FROM agent_accounts
		WHERE status = 'unused'
		ORDER BY
			CASE
				WHEN IFNULL(agent_status, 'inactive') = 'active' AND IFNULL(vault_address, '') != '' THEN 0
				ELSE 1
			END,
			created_at ASC
		LIMIT 1`,
	).Scan(&item.ID, &item.PublicKey, &item.Status, &item.AssignedUserID, &item.AssignedAt, &item.CreatedAt)
	if err != nil {
		return AgentAccount{}, err
	}
	return item, nil
}

func (s *Store) bindUserAgentTx(tx *sql.Tx, userID string, inviteCode *string, publicKey string, assignedAt string) error {
	if strings.TrimSpace(assignedAt) == "" {
		assignedAt = nowISO()
	}
	var (
		res sql.Result
		err error
	)
	if inviteCode != nil {
		res, err = tx.Exec(`
			UPDATE users
			SET invite_code_used = ?,
			    agent_public_key = ?,
			    agent_assigned_at = ?,
			    updated_at = ?
			WHERE id = ?`,
			strings.ToUpper(strings.TrimSpace(*inviteCode)),
			strings.ToLower(strings.TrimSpace(publicKey)),
			assignedAt,
			nowISO(),
			strings.TrimSpace(userID),
		)
	} else {
		res, err = tx.Exec(`
			UPDATE users
			SET agent_public_key = ?,
			    agent_assigned_at = ?,
			    updated_at = ?
			WHERE id = ?`,
			strings.ToLower(strings.TrimSpace(publicKey)),
			assignedAt,
			nowISO(),
			strings.TrimSpace(userID),
		)
	}
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

	// If this user is already bound in agent_accounts (legacy inconsistency),
	// repair users.agent_public_key instead of assigning a second account.
	existing := AgentAccount{}
	err = tx.QueryRow(`
		SELECT id, public_key, status, IFNULL(assigned_user_id, ''), IFNULL(assigned_at, ''), created_at
		FROM agent_accounts
		WHERE assigned_user_id = ? AND status = 'assigned'
		ORDER BY assigned_at DESC
		LIMIT 1`,
		strings.TrimSpace(userID),
	).Scan(&existing.ID, &existing.PublicKey, &existing.Status, &existing.AssignedUserID, &existing.AssignedAt, &existing.CreatedAt)
	if err == nil {
		if bindErr := s.bindUserAgentTx(tx, userID, nil, existing.PublicKey, existing.AssignedAt); bindErr != nil {
			return AgentAccount{}, bindErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return AgentAccount{}, commitErr
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AgentAccount{}, err
	}

	item, err := s.pickUnusedAgentAccountTx(tx)
	if err != nil {
		return AgentAccount{}, err
	}

	item.Status = "assigned"
	item.AssignedUserID = strings.TrimSpace(userID)
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
	if err := s.bindUserAgentTx(tx, userID, nil, item.PublicKey, item.AssignedAt); err != nil {
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

	account, err := s.pickUnusedAgentAccountTx(tx)
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
	if err = s.bindUserAgentTx(tx, userID, &invite.Code, account.PublicKey, account.AssignedAt); err != nil {
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

	err = tx.QueryRow(`SELECT id FROM users WHERE id = ?`, assignedUserID).Scan(&assignedUserID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// user may have been deleted, just commit agent changes
	} else {
		if _, err = tx.Exec(`
			UPDATE users
			SET agent_public_key = '',
			    agent_assigned_at = '',
			    updated_at = ?
			WHERE id = ?`,
			nowISO(),
			assignedUserID,
		); err != nil {
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

	var userAgentPublicKey string
	err = tx.QueryRow(`SELECT IFNULL(agent_public_key, '') FROM users WHERE id = ?`, userID).Scan(&userAgentPublicKey)
	if err != nil {
		return err
	}

	// If user has an assigned agent, revoke it first (within the same tx)
	if strings.TrimSpace(userAgentPublicKey) != "" {
		if _, err = tx.Exec(`UPDATE agent_accounts SET status = 'unused', assigned_user_id = NULL, assigned_at = NULL WHERE public_key = ?`, strings.ToLower(strings.TrimSpace(userAgentPublicKey))); err != nil {
			return err
		}
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

	if _, err = tx.Exec(`
		UPDATE users
		SET invite_code_used = '',
		    agent_public_key = '',
		    agent_assigned_at = '',
		    updated_at = ?
		WHERE id = ?`,
		nowISO(),
		userID,
	); err != nil {
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

	var currentAgentPublicKey string
	err = tx.QueryRow(`SELECT IFNULL(agent_public_key, '') FROM users WHERE id = ?`, userID).Scan(&currentAgentPublicKey)
	if err != nil {
		return err
	}
	if strings.TrimSpace(currentAgentPublicKey) != "" {
		return errors.New("user already has agent")
	}

	now := nowISO()
	if _, err = tx.Exec(`UPDATE agent_accounts SET status = 'assigned', assigned_user_id = ?, assigned_at = ? WHERE public_key = ?`, userID, now, publicKey); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		UPDATE users
		SET agent_public_key = ?,
		    agent_assigned_at = ?,
		    updated_at = ?
		WHERE id = ?`,
		strings.ToLower(strings.TrimSpace(publicKey)),
		now,
		nowISO(),
		userID,
	); err != nil {
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
	       CASE
	         WHEN IFNULL(u.show_x_on_leaderboard, 0) = 1 THEN IFNULL(u.x_username, '')
	         ELSE ''
	       END,
	       CASE
	         WHEN IFNULL(u.show_x_on_leaderboard, 0) = 1 THEN IFNULL(u.avatar, '')
	         ELSE ''
	       END,
	       IFNULL(latest.account_value, 0),
	       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2),
		       IFNULL(a.initial_capital, 0),
		       IFNULL(a.assigned_at, '')
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
		query += ` AND (lower(a.public_key) LIKE ? OR lower(CASE WHEN IFNULL(u.show_x_on_leaderboard, 0) = 1 THEN IFNULL(u.x_username, '') ELSE '' END) LIKE ? OR lower(a.name) LIKE ?)`
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
			&item.Avatar,
			&item.AccountValue,
			&item.TotalPnL,
			&item.LastSyncedAt,
			&item.VaultAddress,
			&item.EVMBalance,
			&item.AgentStatus,
			&item.PerformanceFee,
			&item.InitialCapital,
			&item.StartedAt,
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
	       CASE
	         WHEN IFNULL(u.show_x_on_leaderboard, 0) = 1 THEN IFNULL(u.x_username, '')
	         ELSE ''
	       END,
	       CASE
	         WHEN IFNULL(u.show_x_on_leaderboard, 0) = 1 THEN IFNULL(u.avatar, '')
	         ELSE ''
	       END,
	       IFNULL(latest.account_value, 0),
	       IFNULL(latest.account_value, 0) - IFNULL(first.account_value, 0),
		       IFNULL(latest.created_at, ''),
		       IFNULL(a.vault_address, ''),
		       IFNULL(a.evm_balance, 0),
		       IFNULL(a.agent_status, 'inactive'),
		       IFNULL(a.performance_fee, 0.2),
		       IFNULL(a.initial_capital, 0),
		       IFNULL(a.assigned_at, '')
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
		&item.Avatar,
		&item.AccountValue,
		&item.TotalPnL,
		&item.LastSyncedAt,
		&item.VaultAddress,
		&item.EVMBalance,
		&item.AgentStatus,
		&item.PerformanceFee,
		&item.InitialCapital,
		&item.StartedAt,
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

func (s *Store) upsertAgentVaultOrder(vaultAddress string, agentVaultUser string, fillID string, fillTime int64, rawJSON string) error {
	fill := VaultFill{}
	if strings.TrimSpace(rawJSON) != "" {
		if err := json.Unmarshal([]byte(rawJSON), &fill); err != nil {
			return err
		}
	}
	if fillTime <= 0 && fill.Time > 0 {
		fillTime = fill.Time
	}
	if fillID == "" {
		fillID = fmt.Sprintf("%s_%d", strings.ToLower(strings.TrimSpace(agentVaultUser)), fillTime)
	}
	fillHash := strings.TrimSpace(fill.Hash)
	if fillHash == "" {
		fillHash = fillID
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_vault_orders(
			id, vault_address, agent_vault_user, fill_time, coin, side, size, price, fee, closed_pnl, fill_hash, start_position, direction, created_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			-- Keep first bound ownership; only backfill when legacy row has empty owner/vault.
			vault_address = CASE
				WHEN IFNULL(agent_vault_orders.vault_address, '') = '' THEN excluded.vault_address
				ELSE agent_vault_orders.vault_address
			END,
			agent_vault_user = CASE
				WHEN IFNULL(agent_vault_orders.agent_vault_user, '') = '' THEN excluded.agent_vault_user
				ELSE agent_vault_orders.agent_vault_user
			END,
			fill_time = excluded.fill_time,
			coin = excluded.coin,
			side = excluded.side,
			size = excluded.size,
			price = excluded.price,
			fee = excluded.fee,
			closed_pnl = excluded.closed_pnl,
			fill_hash = excluded.fill_hash,
			start_position = excluded.start_position,
			direction = excluded.direction`,
		fillID,
		strings.ToLower(strings.TrimSpace(vaultAddress)),
		strings.ToLower(strings.TrimSpace(agentVaultUser)),
		fillTime,
		strings.TrimSpace(fill.Coin),
		strings.TrimSpace(fill.Side),
		fill.Size,
		fill.Price,
		fill.Fee,
		fill.ClosedPnl,
		fillHash,
		fill.StartPosition,
		strings.TrimSpace(fill.Direction),
		nowISO(),
	)
	return err
}

func (s *Store) listAgentVaultOrders(publicKey string, limit int) ([]VaultFill, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT coin, side, size, price, fill_time, fee, closed_pnl, fill_hash, start_position, direction
		FROM agent_vault_orders
		WHERE lower(agent_vault_user) = ?
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
		var f VaultFill
		if err := rows.Scan(
			&f.Coin,
			&f.Side,
			&f.Size,
			&f.Price,
			&f.Time,
			&f.Fee,
			&f.ClosedPnl,
			&f.Hash,
			&f.StartPosition,
			&f.Direction,
		); err != nil {
			return nil, err
		}
		fills = append(fills, f)
	}
	return fills, rows.Err()
}

func (s *Store) listRecentVaultOrdersForActiveAgents(limit int) ([]VaultFill, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT f.coin, f.side, f.size, f.price, f.fill_time, f.fee, f.closed_pnl, f.fill_hash, f.start_position, f.direction
		FROM agent_vault_orders f
		INNER JOIN agent_accounts a ON lower(a.public_key) = lower(f.agent_vault_user)
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
		var f VaultFill
		if err := rows.Scan(
			&f.Coin,
			&f.Side,
			&f.Size,
			&f.Price,
			&f.Time,
			&f.Fee,
			&f.ClosedPnl,
			&f.Hash,
			&f.StartPosition,
			&f.Direction,
		); err != nil {
			return nil, err
		}
		fills = append(fills, f)
	}
	return fills, rows.Err()
}

func (s *Store) getAgentCreatedAt(publicKey string) string {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	var createdAt string
	s.db.QueryRow(`SELECT IFNULL(assigned_at, '') FROM agent_accounts WHERE public_key = ?`, publicKey).Scan(&createdAt)
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

	// Clear stale references from users before deleting accounts.
	if _, err = tx.Exec(`
		UPDATE users
		SET agent_public_key = '',
		    agent_assigned_at = '',
		    updated_at = ?
		WHERE lower(agent_public_key) IN (`+placeholders+`)`,
		append([]any{nowISO()}, args...)...,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec(`DELETE FROM agent_snapshots WHERE public_key IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM agent_vault_orders WHERE lower(agent_vault_user) IN (`+placeholders+`)`, args...); err != nil {
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
