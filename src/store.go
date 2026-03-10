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
var errNoSlotsRemaining = errors.New("no_slots_remaining")

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
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			x_id TEXT NOT NULL UNIQUE,
			email TEXT,
			name TEXT,
			level TEXT,
			x_username TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			show_x_on_leaderboard INTEGER NOT NULL DEFAULT 0,
			invite_code_used TEXT NOT NULL DEFAULT '',
			agent_public_key TEXT NOT NULL DEFAULT '',
			agent_assigned_at TEXT NOT NULL DEFAULT '',
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
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS vault (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			total_tvl REAL NOT NULL DEFAULT 0,
			share_price REAL NOT NULL DEFAULT 0,
			apy REAL NOT NULL DEFAULT 0,
			total_profit REAL NOT NULL DEFAULT 0,
			agent_count INTEGER NOT NULL DEFAULT 0,
			depositor_count INTEGER NOT NULL DEFAULT 0,
			chain_balance REAL NOT NULL DEFAULT 0,
			cluster_balance REAL NOT NULL DEFAULT 0,
			hype_balance REAL NOT NULL DEFAULT 0,
			agent_balance REAL NOT NULL DEFAULT 0,
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
		`CREATE TABLE IF NOT EXISTS agent_vault_orders (
			id TEXT PRIMARY KEY,
			vault_address TEXT NOT NULL DEFAULT '',
			agent_vault_user TEXT NOT NULL DEFAULT '',
			fill_time INTEGER NOT NULL,
			coin TEXT NOT NULL DEFAULT '',
			side TEXT NOT NULL DEFAULT '',
			size REAL NOT NULL DEFAULT 0,
			price REAL NOT NULL DEFAULT 0,
			fee REAL NOT NULL DEFAULT 0,
			closed_pnl REAL NOT NULL DEFAULT 0,
			fill_hash TEXT NOT NULL DEFAULT '',
			start_position REAL NOT NULL DEFAULT 0,
			direction TEXT NOT NULL DEFAULT '',
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
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_invite_code_usages_user_unique ON invite_code_usages(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_accounts_status ON agent_accounts(status);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_accounts_assigned_user_unique ON agent_accounts(assigned_user_id) WHERE assigned_user_id IS NOT NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_agent_snapshots_pub_created ON agent_snapshots(public_key, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_vault_orders_user_time ON agent_vault_orders(agent_vault_user, fill_time);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_vault_orders_vault_time ON agent_vault_orders(vault_address, fill_time);`,
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

		// initial capital from on-chain vault
		`ALTER TABLE agent_accounts ADD COLUMN initial_capital REAL NOT NULL DEFAULT 0`,

		// agent_vaults — discovered from Allocator contract
		`CREATE TABLE IF NOT EXISTS agent_vaults (
			vault_address TEXT PRIMARY KEY,
			user_address TEXT NOT NULL DEFAULT '',
			evm_balance REAL NOT NULL DEFAULT 0,
			initial_capital REAL NOT NULL DEFAULT 0,
			valid INTEGER NOT NULL DEFAULT 0,
			allocator_address TEXT NOT NULL DEFAULT '',
			account_value REAL NOT NULL DEFAULT 0,
			unrealized_pnl REAL NOT NULL DEFAULT 0,
			last_synced_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_vaults_user ON agent_vaults(user_address)`,

		// sync status tracking for agent_vaults
		`ALTER TABLE agent_vaults ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'ok'`,
		`ALTER TABLE agent_vaults ADD COLUMN sync_error TEXT NOT NULL DEFAULT ''`,

		// treasury snapshots — global treasury stats time series
		`CREATE TABLE IF NOT EXISTS treasury_snapshots (
			id TEXT PRIMARY KEY,
			vault_evm REAL NOT NULL DEFAULT 0,
			vault_perps REAL NOT NULL DEFAULT 0,
			vault_spot REAL NOT NULL DEFAULT 0,
			vault_pnl REAL NOT NULL DEFAULT 0,
			vault_capital REAL NOT NULL DEFAULT 0,
			allocator_evm REAL NOT NULL DEFAULT 0,
			allocator_perps REAL NOT NULL DEFAULT 0,
			allocator_spot REAL NOT NULL DEFAULT 0,
			owner_evm REAL NOT NULL DEFAULT 0,
			owner_perps REAL NOT NULL DEFAULT 0,
			owner_spot REAL NOT NULL DEFAULT 0,
			total_funds REAL NOT NULL DEFAULT 0,
			vault_count INTEGER NOT NULL DEFAULT 0,
			active_vault_count INTEGER NOT NULL DEFAULT 0,
			allocator_address TEXT NOT NULL DEFAULT '',
			owner_address TEXT NOT NULL DEFAULT '',
			extra_usdc REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_treasury_snapshots_created ON treasury_snapshots(created_at)`,

		// platform snapshots — platform-level time series
		`CREATE TABLE IF NOT EXISTS platform_snapshots (
			id TEXT PRIMARY KEY,
			total_tvl REAL NOT NULL DEFAULT 0,
			total_pnl REAL NOT NULL DEFAULT 0,
			total_capital REAL NOT NULL DEFAULT 0,
			user_count INTEGER NOT NULL DEFAULT 0,
			active_agent_count INTEGER NOT NULL DEFAULT 0,
			total_agent_count INTEGER NOT NULL DEFAULT 0,
			total_trades INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_snapshots_created ON platform_snapshots(created_at)`,
		`ALTER TABLE treasury_snapshots ADD COLUMN extra_usdc REAL NOT NULL DEFAULT 0`,

		// users structured columns (safe to re-run)
		`ALTER TABLE users ADD COLUMN x_username TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN avatar TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN show_x_on_leaderboard INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN invite_code_used TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN agent_public_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN agent_assigned_at TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_users_agent_public_key ON users(agent_public_key);`,

		// vault structured columns (safe to re-run)
		`ALTER TABLE vault ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vault ADD COLUMN total_tvl REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN share_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN apy REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN total_profit REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN agent_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN depositor_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN chain_balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN cluster_balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN hype_balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE vault ADD COLUMN agent_balance REAL NOT NULL DEFAULT 0`,

		// agent_vault_orders structured columns (safe to re-run)
		`ALTER TABLE agent_vault_orders ADD COLUMN coin TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_vault_orders ADD COLUMN side TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_vault_orders ADD COLUMN size REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_vault_orders ADD COLUMN price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_vault_orders ADD COLUMN fee REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_vault_orders ADD COLUMN closed_pnl REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_vault_orders ADD COLUMN fill_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_vault_orders ADD COLUMN start_position REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE agent_vault_orders ADD COLUMN direction TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_vault_orders ADD COLUMN vault_address TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_vault_orders ADD COLUMN agent_vault_user TEXT NOT NULL DEFAULT ''`,
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
	if err := s.migrateUsersDataJSONIfNeeded(); err != nil {
		return err
	}
	if err := s.migrateStrategiesDataJSONIfNeeded(); err != nil {
		return err
	}
	if err := s.migrateAgentsDataJSONIfNeeded(); err != nil {
		return err
	}
	if err := s.migrateVaultDataJSONIfNeeded(); err != nil {
		return err
	}
	if err := s.migrateAgentFillsToAgentVaultOrdersIfNeeded(); err != nil {
		return err
	}
	return nil
}

type legacyUserDataJSON struct {
	Email              string `json:"email"`
	Name               string `json:"name"`
	XID                string `json:"xId"`
	XUsername          string `json:"xUsername"`
	Avatar             string `json:"avatar"`
	ShowXOnLeaderboard *bool  `json:"showXOnLeaderboard,omitempty"`
	InviteCodeUsed     string `json:"inviteCodeUsed"`
	AgentPublicKey     string `json:"agentPublicKey"`
	AgentAssignedAt    string `json:"agentAssignedAt"`
	CreatedAt          string `json:"createdAt"`
}

type legacyStrategyDataJSON struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Status          string   `json:"status"`
	Creator         string   `json:"creator"`
	AgentLevel      string   `json:"agentLevel"`
	CurrentTVL      *float64 `json:"currentTvl"`
	MaxTVL          *float64 `json:"maxTvl"`
	PnLContribution *float64 `json:"pnlContribution"`
	Rating          *float64 `json:"rating"`
	ReviewNote      string   `json:"reviewNote"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

type legacyAgentDataJSON struct {
	Name        string   `json:"name"`
	StrategyID  string   `json:"strategyId"`
	UserID      string   `json:"userId"`
	Status      string   `json:"status"`
	Investment  *float64 `json:"investment"`
	TotalProfit *float64 `json:"totalProfit"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type legacyVaultDataJSON struct {
	Name           string   `json:"name"`
	TotalTVL       *float64 `json:"totalTvl"`
	SharePrice     *float64 `json:"sharePrice"`
	APY            *float64 `json:"apy"`
	TotalProfit    *float64 `json:"totalProfit"`
	AgentCount     *int     `json:"agentCount"`
	DepositorCount *int     `json:"depositorCount"`
	ChainBalance   *float64 `json:"chainBalance"`
	ClusterBalance *float64 `json:"clusterBalance"`
	HypeBalance    *float64 `json:"hypeBalance"`
	AgentBalance   *float64 `json:"agentBalance"`
	UpdatedAt      string   `json:"updatedAt"`
}

func (s *Store) tableHasColumn(tableName string, columnName string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) tableExists(tableName string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) migrateUsersDataJSONIfNeeded() error {
	hasDataJSON, err := s.tableHasColumn("users", "data_json")
	if err != nil {
		return err
	}
	if !hasDataJSON {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`
		SELECT id,
		       IFNULL(email, ''),
		       IFNULL(name, ''),
		       IFNULL(x_id, ''),
		       IFNULL(created_at, ''),
		       IFNULL(x_username, ''),
		       IFNULL(avatar, ''),
		       IFNULL(invite_code_used, ''),
		       IFNULL(agent_public_key, ''),
		       IFNULL(agent_assigned_at, ''),
		       IFNULL(show_x_on_leaderboard, 0),
		       IFNULL(data_json, '')
		FROM users`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                 string
			email              string
			name               string
			xID                string
			createdAt          string
			xUsername          string
			avatar             string
			inviteCodeUsed     string
			agentPublicKey     string
			agentAssignedAt    string
			showXInt           int
			payload            string
			showXOnLeaderboard bool
			legacy             legacyUserDataJSON
		)
		if err := rows.Scan(
			&id,
			&email,
			&name,
			&xID,
			&createdAt,
			&xUsername,
			&avatar,
			&inviteCodeUsed,
			&agentPublicKey,
			&agentAssignedAt,
			&showXInt,
			&payload,
		); err != nil {
			return err
		}

		showXOnLeaderboard = showXInt != 0
		payload = strings.TrimSpace(payload)
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
				return fmt.Errorf("users.data_json invalid (id=%s): %w", id, err)
			} else {
				if email == "" {
					email = strings.TrimSpace(legacy.Email)
				}
				if name == "" {
					name = strings.TrimSpace(legacy.Name)
				}
				if xID == "" {
					xID = strings.ToLower(strings.TrimSpace(legacy.XID))
				}
				if xUsername == "" {
					xUsername = strings.TrimSpace(legacy.XUsername)
				}
				if avatar == "" {
					avatar = strings.TrimSpace(legacy.Avatar)
				}
				if inviteCodeUsed == "" {
					inviteCodeUsed = strings.TrimSpace(legacy.InviteCodeUsed)
				}
				if agentPublicKey == "" {
					agentPublicKey = strings.ToLower(strings.TrimSpace(legacy.AgentPublicKey))
				}
				if agentAssignedAt == "" {
					agentAssignedAt = strings.TrimSpace(legacy.AgentAssignedAt)
				}
				if createdAt == "" {
					createdAt = strings.TrimSpace(legacy.CreatedAt)
				}
				if !showXOnLeaderboard && legacy.ShowXOnLeaderboard != nil {
					showXOnLeaderboard = *legacy.ShowXOnLeaderboard
				}
			}
		}

		if createdAt == "" {
			createdAt = nowISO()
		}
		xID = strings.ToLower(strings.TrimSpace(xID))
		agentPublicKey = strings.ToLower(strings.TrimSpace(agentPublicKey))

		if _, err := tx.Exec(`
			UPDATE users
			SET email = ?,
			    name = ?,
			    x_id = ?,
			    x_username = ?,
			    avatar = ?,
			    invite_code_used = ?,
			    agent_public_key = ?,
			    agent_assigned_at = ?,
			    show_x_on_leaderboard = ?,
			    created_at = ?,
			    updated_at = ?,
			    data_json = ''
			WHERE id = ?`,
			email,
			name,
			xID,
			xUsername,
			avatar,
			inviteCodeUsed,
			agentPublicKey,
			agentAssignedAt,
			boolToInt(showXOnLeaderboard),
			createdAt,
			nowISO(),
			id,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE users DROP COLUMN data_json`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logInfo("db", "users.data_json migrated into structured columns and dropped")
	return nil
}

func (s *Store) migrateStrategiesDataJSONIfNeeded() error {
	hasDataJSON, err := s.tableHasColumn("strategies", "data_json")
	if err != nil {
		return err
	}
	if !hasDataJSON {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`
		SELECT id,
		       IFNULL(name, ''),
		       IFNULL(category, ''),
		       IFNULL(status, ''),
		       IFNULL(creator, ''),
		       IFNULL(agent_level, ''),
		       IFNULL(current_tvl, 0),
		       IFNULL(max_tvl, 0),
		       IFNULL(pnl_contribution, 0),
		       IFNULL(rating, 0),
		       IFNULL(review_note, ''),
		       IFNULL(created_at, ''),
		       IFNULL(updated_at, ''),
		       IFNULL(data_json, '')
		FROM strategies`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id              string
			name            string
			category        string
			status          string
			creator         string
			agentLevel      string
			currentTVL      float64
			maxTVL          float64
			pnlContribution float64
			rating          float64
			reviewNote      string
			createdAt       string
			updatedAt       string
			payload         string
			legacy          legacyStrategyDataJSON
		)
		if err := rows.Scan(
			&id,
			&name,
			&category,
			&status,
			&creator,
			&agentLevel,
			&currentTVL,
			&maxTVL,
			&pnlContribution,
			&rating,
			&reviewNote,
			&createdAt,
			&updatedAt,
			&payload,
		); err != nil {
			return err
		}

		payload = strings.TrimSpace(payload)
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
				return fmt.Errorf("strategies.data_json invalid (id=%s): %w", id, err)
			} else {
				if name == "" {
					name = strings.TrimSpace(legacy.Name)
				}
				if category == "" {
					category = strings.TrimSpace(legacy.Category)
				}
				if status == "" {
					status = strings.TrimSpace(legacy.Status)
				}
				if creator == "" {
					creator = strings.TrimSpace(legacy.Creator)
				}
				if agentLevel == "" {
					agentLevel = strings.TrimSpace(legacy.AgentLevel)
				}
				if currentTVL == 0 && legacy.CurrentTVL != nil {
					currentTVL = *legacy.CurrentTVL
				}
				if maxTVL == 0 && legacy.MaxTVL != nil {
					maxTVL = *legacy.MaxTVL
				}
				if pnlContribution == 0 && legacy.PnLContribution != nil {
					pnlContribution = *legacy.PnLContribution
				}
				if rating == 0 && legacy.Rating != nil {
					rating = *legacy.Rating
				}
				if reviewNote == "" {
					reviewNote = strings.TrimSpace(legacy.ReviewNote)
				}
				if createdAt == "" {
					createdAt = strings.TrimSpace(legacy.CreatedAt)
				}
				if updatedAt == "" {
					updatedAt = strings.TrimSpace(legacy.UpdatedAt)
				}
			}
		}

		if createdAt == "" {
			createdAt = nowISO()
		}
		if updatedAt == "" {
			updatedAt = nowISO()
		}

		if _, err := tx.Exec(`
			UPDATE strategies
			SET name = ?,
			    category = ?,
			    status = ?,
			    creator = ?,
			    agent_level = ?,
			    current_tvl = ?,
			    max_tvl = ?,
			    pnl_contribution = ?,
			    rating = ?,
			    review_note = ?,
			    created_at = ?,
			    updated_at = ?,
			    data_json = ''
			WHERE id = ?`,
			name,
			category,
			status,
			creator,
			agentLevel,
			currentTVL,
			maxTVL,
			pnlContribution,
			rating,
			reviewNote,
			createdAt,
			updatedAt,
			id,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE strategies DROP COLUMN data_json`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logInfo("db", "strategies.data_json migrated into structured columns and dropped")
	return nil
}

func (s *Store) migrateAgentsDataJSONIfNeeded() error {
	hasDataJSON, err := s.tableHasColumn("agents", "data_json")
	if err != nil {
		return err
	}
	if !hasDataJSON {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`
		SELECT id,
		       IFNULL(name, ''),
		       IFNULL(strategy_id, ''),
		       IFNULL(user_id, ''),
		       IFNULL(status, ''),
		       IFNULL(investment, 0),
		       IFNULL(total_profit, 0),
		       IFNULL(created_at, ''),
		       IFNULL(updated_at, ''),
		       IFNULL(data_json, '')
		FROM agents`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id          string
			name        string
			strategyID  string
			userID      string
			status      string
			investment  float64
			totalProfit float64
			createdAt   string
			updatedAt   string
			payload     string
			legacy      legacyAgentDataJSON
		)
		if err := rows.Scan(
			&id,
			&name,
			&strategyID,
			&userID,
			&status,
			&investment,
			&totalProfit,
			&createdAt,
			&updatedAt,
			&payload,
		); err != nil {
			return err
		}

		payload = strings.TrimSpace(payload)
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
				return fmt.Errorf("agents.data_json invalid (id=%s): %w", id, err)
			} else {
				if name == "" {
					name = strings.TrimSpace(legacy.Name)
				}
				if strategyID == "" {
					strategyID = strings.TrimSpace(legacy.StrategyID)
				}
				if userID == "" {
					userID = strings.TrimSpace(legacy.UserID)
				}
				if status == "" {
					status = strings.TrimSpace(legacy.Status)
				}
				if investment == 0 && legacy.Investment != nil {
					investment = *legacy.Investment
				}
				if totalProfit == 0 && legacy.TotalProfit != nil {
					totalProfit = *legacy.TotalProfit
				}
				if createdAt == "" {
					createdAt = strings.TrimSpace(legacy.CreatedAt)
				}
				if updatedAt == "" {
					updatedAt = strings.TrimSpace(legacy.UpdatedAt)
				}
			}
		}

		if createdAt == "" {
			createdAt = nowISO()
		}
		if updatedAt == "" {
			updatedAt = nowISO()
		}

		if _, err := tx.Exec(`
			UPDATE agents
			SET name = ?,
			    strategy_id = ?,
			    user_id = ?,
			    status = ?,
			    investment = ?,
			    total_profit = ?,
			    created_at = ?,
			    updated_at = ?,
			    data_json = ''
			WHERE id = ?`,
			name,
			strategyID,
			userID,
			status,
			investment,
			totalProfit,
			createdAt,
			updatedAt,
			id,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE agents DROP COLUMN data_json`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logInfo("db", "agents.data_json migrated into structured columns and dropped")
	return nil
}

func (s *Store) migrateVaultDataJSONIfNeeded() error {
	hasDataJSON, err := s.tableHasColumn("vault", "data_json")
	if err != nil {
		return err
	}
	if !hasDataJSON {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`
		SELECT id,
		       IFNULL(name, ''),
		       IFNULL(total_tvl, 0),
		       IFNULL(share_price, 0),
		       IFNULL(apy, 0),
		       IFNULL(total_profit, 0),
		       IFNULL(agent_count, 0),
		       IFNULL(depositor_count, 0),
		       IFNULL(chain_balance, 0),
		       IFNULL(cluster_balance, 0),
		       IFNULL(hype_balance, 0),
		       IFNULL(agent_balance, 0),
		       IFNULL(updated_at, ''),
		       IFNULL(data_json, '')
		FROM vault`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id             string
			name           string
			totalTVL       float64
			sharePrice     float64
			apy            float64
			totalProfit    float64
			agentCount     int
			depositorCount int
			chainBalance   float64
			clusterBalance float64
			hypeBalance    float64
			agentBalance   float64
			updatedAt      string
			payload        string
			legacy         legacyVaultDataJSON
		)
		if err := rows.Scan(
			&id,
			&name,
			&totalTVL,
			&sharePrice,
			&apy,
			&totalProfit,
			&agentCount,
			&depositorCount,
			&chainBalance,
			&clusterBalance,
			&hypeBalance,
			&agentBalance,
			&updatedAt,
			&payload,
		); err != nil {
			return err
		}

		payload = strings.TrimSpace(payload)
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
				return fmt.Errorf("vault.data_json invalid (id=%s): %w", id, err)
			} else {
				if name == "" {
					name = strings.TrimSpace(legacy.Name)
				}
				if totalTVL == 0 && legacy.TotalTVL != nil {
					totalTVL = *legacy.TotalTVL
				}
				if sharePrice == 0 && legacy.SharePrice != nil {
					sharePrice = *legacy.SharePrice
				}
				if apy == 0 && legacy.APY != nil {
					apy = *legacy.APY
				}
				if totalProfit == 0 && legacy.TotalProfit != nil {
					totalProfit = *legacy.TotalProfit
				}
				if agentCount == 0 && legacy.AgentCount != nil {
					agentCount = *legacy.AgentCount
				}
				if depositorCount == 0 && legacy.DepositorCount != nil {
					depositorCount = *legacy.DepositorCount
				}
				if chainBalance == 0 && legacy.ChainBalance != nil {
					chainBalance = *legacy.ChainBalance
				}
				if clusterBalance == 0 && legacy.ClusterBalance != nil {
					clusterBalance = *legacy.ClusterBalance
				}
				if hypeBalance == 0 && legacy.HypeBalance != nil {
					hypeBalance = *legacy.HypeBalance
				}
				if agentBalance == 0 && legacy.AgentBalance != nil {
					agentBalance = *legacy.AgentBalance
				}
				if updatedAt == "" {
					updatedAt = strings.TrimSpace(legacy.UpdatedAt)
				}
			}
		}

		if updatedAt == "" {
			updatedAt = nowISO()
		}

		if _, err := tx.Exec(`
			UPDATE vault
			SET name = ?,
			    total_tvl = ?,
			    share_price = ?,
			    apy = ?,
			    total_profit = ?,
			    agent_count = ?,
			    depositor_count = ?,
			    chain_balance = ?,
			    cluster_balance = ?,
			    hype_balance = ?,
			    agent_balance = ?,
			    updated_at = ?,
			    data_json = ''
			WHERE id = ?`,
			name,
			totalTVL,
			sharePrice,
			apy,
			totalProfit,
			agentCount,
			depositorCount,
			chainBalance,
			clusterBalance,
			hypeBalance,
			agentBalance,
			updatedAt,
			id,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE vault DROP COLUMN data_json`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logInfo("db", "vault.data_json migrated into structured columns and dropped")
	return nil
}

func (s *Store) migrateAgentFillsToAgentVaultOrdersIfNeeded() error {
	oldExists, err := s.tableExists("agent_fills")
	if err != nil {
		return err
	}
	if !oldExists {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	hasDataJSON, err := s.tableHasColumn("agent_fills", "data_json")
	if err != nil {
		return err
	}
	hasStructuredCols, err := s.tableHasColumn("agent_fills", "closed_pnl")
	if err != nil {
		return err
	}

	if hasDataJSON {
		query := `SELECT
			IFNULL(id, ''),
			IFNULL(public_key, ''),
			IFNULL(fill_time, 0), `
		if hasStructuredCols {
			query += `IFNULL(coin, ''),
			          IFNULL(side, ''),
			          IFNULL(size, 0),
			          IFNULL(price, 0),
			          IFNULL(fee, 0),
			          IFNULL(closed_pnl, 0),
			          IFNULL(fill_hash, ''),
			          IFNULL(start_position, 0),
			          IFNULL(direction, ''), `
		}
		query += `IFNULL(data_json, ''),
		          IFNULL(created_at, '')
		          FROM agent_fills`

		rows, qErr := tx.Query(query)
		if qErr != nil {
			return qErr
		}
		defer rows.Close()

		for rows.Next() {
			var (
				id            string
				publicKey     string
				fillTime      int64
				coin          string
				side          string
				size          float64
				price         float64
				fee           float64
				closedPnl     float64
				fillHash      string
				startPosition float64
				direction     string
				payload       string
				createdAt     string
				fill          VaultFill
				vaultAddress  string
			)
			if hasStructuredCols {
				if err := rows.Scan(
					&id,
					&publicKey,
					&fillTime,
					&coin,
					&side,
					&size,
					&price,
					&fee,
					&closedPnl,
					&fillHash,
					&startPosition,
					&direction,
					&payload,
					&createdAt,
				); err != nil {
					return err
				}
			} else {
				if err := rows.Scan(
					&id,
					&publicKey,
					&fillTime,
					&payload,
					&createdAt,
				); err != nil {
					return err
				}
			}

			payload = strings.TrimSpace(payload)
			if payload != "" {
				if err := json.Unmarshal([]byte(payload), &fill); err != nil {
					return fmt.Errorf("agent_fills.data_json invalid (id=%s): %w", id, err)
				} else {
					if fillTime == 0 && fill.Time > 0 {
						fillTime = fill.Time
					}
					if coin == "" {
						coin = strings.TrimSpace(fill.Coin)
					}
					if side == "" {
						side = strings.TrimSpace(fill.Side)
					}
					if size == 0 {
						size = fill.Size
					}
					if price == 0 {
						price = fill.Price
					}
					if fee == 0 {
						fee = fill.Fee
					}
					if closedPnl == 0 {
						closedPnl = fill.ClosedPnl
					}
					if fillHash == "" {
						fillHash = strings.TrimSpace(fill.Hash)
					}
					if startPosition == 0 {
						startPosition = fill.StartPosition
					}
					if direction == "" {
						direction = strings.TrimSpace(fill.Direction)
					}
				}
			}
			if fillHash == "" {
				fillHash = id
			}
			if createdAt == "" {
				createdAt = nowISO()
			}

			_ = tx.QueryRow(
				`SELECT IFNULL(vault_address, '') FROM agent_accounts WHERE lower(public_key) = lower(?) LIMIT 1`,
				strings.ToLower(strings.TrimSpace(publicKey)),
			).Scan(&vaultAddress)

			if _, err := tx.Exec(`
				INSERT INTO agent_vault_orders(
					id, vault_address, agent_vault_user, fill_time, coin, side, size, price, fee, closed_pnl, fill_hash, start_position, direction, created_at
				)
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					vault_address = excluded.vault_address,
					agent_vault_user = excluded.agent_vault_user,
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
				id,
				strings.ToLower(strings.TrimSpace(vaultAddress)),
				strings.ToLower(strings.TrimSpace(publicKey)),
				fillTime,
				coin,
				side,
				size,
				price,
				fee,
				closedPnl,
				fillHash,
				startPosition,
				direction,
				createdAt,
			); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	} else if hasStructuredCols {
		if _, err := tx.Exec(`
			INSERT INTO agent_vault_orders(
				id, vault_address, agent_vault_user, fill_time, coin, side, size, price, fee, closed_pnl, fill_hash, start_position, direction, created_at
			)
			SELECT f.id,
			       lower(trim(IFNULL(a.vault_address, ''))),
			       lower(trim(IFNULL(f.public_key, ''))),
			       IFNULL(f.fill_time, 0),
			       IFNULL(f.coin, ''),
			       IFNULL(f.side, ''),
			       IFNULL(f.size, 0),
			       IFNULL(f.price, 0),
			       IFNULL(f.fee, 0),
			       IFNULL(f.closed_pnl, 0),
			       IFNULL(f.fill_hash, ''),
			       IFNULL(f.start_position, 0),
			       IFNULL(f.direction, ''),
			       IFNULL(f.created_at, '')
			FROM agent_fills f
			LEFT JOIN agent_accounts a ON lower(a.public_key) = lower(f.public_key)
			ON CONFLICT(id) DO UPDATE SET
				vault_address = excluded.vault_address,
				agent_vault_user = excluded.agent_vault_user,
				fill_time = excluded.fill_time,
				coin = excluded.coin,
				side = excluded.side,
				size = excluded.size,
				price = excluded.price,
				fee = excluded.fee,
				closed_pnl = excluded.closed_pnl,
				fill_hash = excluded.fill_hash,
				start_position = excluded.start_position,
				direction = excluded.direction`); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DROP TABLE agent_fills`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	logInfo("db", "migrated agent_fills -> agent_vault_orders and dropped legacy table")
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
	defer func() { _ = tx.Rollback() }()

	var storedWallet string
	var createdAt string
	var used int
	err = tx.QueryRow(`SELECT IFNULL(wallet_address, ''), created_at, used FROM nonces WHERE nonce = ?`, nonce).Scan(&storedWallet, &createdAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if used == 1 {
		return false, nil
	}
	if storedWallet != "" && wallet != "" && storedWallet != wallet {
		return false, nil
	}

	created, parseErr := time.Parse(time.RFC3339, createdAt)
	if parseErr == nil {
		if time.Since(created) > 15*time.Minute {
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
	defer func() { _ = tx.Rollback() }()

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
	createdAtTime, parseErr := time.Parse(time.RFC3339, item.CreatedAt)
	if parseErr != nil || time.Since(createdAtTime) > ttl {
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
	return s.getUserByWhereClause(`x_id = ?`, xID)
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
	user.XID = strings.ToLower(strings.TrimSpace(user.XID))
	user.AgentPublicKey = strings.ToLower(strings.TrimSpace(user.AgentPublicKey))
	t := nowISO()
	_, err := s.db.Exec(`
		INSERT INTO users(
			id, x_id, email, name,
			x_username, avatar, show_x_on_leaderboard,
			invite_code_used, agent_public_key, agent_assigned_at,
			created_at, updated_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			x_id = excluded.x_id,
			email = excluded.email,
			name = excluded.name,
			x_username = excluded.x_username,
			avatar = excluded.avatar,
			show_x_on_leaderboard = excluded.show_x_on_leaderboard,
			invite_code_used = excluded.invite_code_used,
			agent_public_key = excluded.agent_public_key,
			agent_assigned_at = excluded.agent_assigned_at,
			updated_at = excluded.updated_at`,
		user.ID,
		user.XID,
		user.Email,
		user.Name,
		user.XUsername,
		user.Avatar,
		boolToInt(user.ShowXOnLeaderboard),
		user.InviteCodeUsed,
		user.AgentPublicKey,
		user.AgentAssignedAt,
		t,
		t,
	)
	return err
}

func (s *Store) getUserByID(id string) (User, error) {
	return s.getUserByWhereClause(`id = ?`, id)
}

func (s *Store) updateUserShowXOnLeaderboard(id string, enabled bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return sql.ErrNoRows
	}
	res, err := s.db.Exec(`
		UPDATE users
		SET show_x_on_leaderboard = ?, updated_at = ?
		WHERE id = ?`,
		boolToInt(enabled),
		nowISO(),
		id,
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

func (s *Store) getUserByName(name string) (User, error) {
	return s.getUserByWhereClause(`lower(name) = lower(?)`, strings.TrimSpace(name))
}

func (s *Store) listUsers(search string) ([]User, error) {
	query := `SELECT
		id,
		IFNULL(email, ''),
		IFNULL(name, ''),
		IFNULL(x_id, ''),
		IFNULL(x_username, ''),
		IFNULL(avatar, ''),
		IFNULL(show_x_on_leaderboard, 0),
		IFNULL(invite_code_used, ''),
		IFNULL(agent_public_key, ''),
		IFNULL(agent_assigned_at, ''),
		created_at
	FROM users`
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
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) getUserByWhereClause(whereClause string, arg any) (User, error) {
	query := `SELECT
		id,
		IFNULL(email, ''),
		IFNULL(name, ''),
		IFNULL(x_id, ''),
		IFNULL(x_username, ''),
		IFNULL(avatar, ''),
		IFNULL(show_x_on_leaderboard, 0),
		IFNULL(invite_code_used, ''),
		IFNULL(agent_public_key, ''),
		IFNULL(agent_assigned_at, ''),
		created_at
	FROM users WHERE ` + whereClause + ` LIMIT 1`
	row := s.db.QueryRow(query, arg)
	return scanUserRow(row)
}

type userRowScanner interface {
	Scan(dest ...any) error
}

func scanUserRow(scanner userRowScanner) (User, error) {
	var (
		user     User
		showXInt int
	)
	err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.XID,
		&user.XUsername,
		&user.Avatar,
		&showXInt,
		&user.InviteCodeUsed,
		&user.AgentPublicKey,
		&user.AgentAssignedAt,
		&user.CreatedAt,
	)
	if err != nil {
		return User{}, err
	}
	user.ShowXOnLeaderboard = showXInt != 0
	user.XID = strings.ToLower(strings.TrimSpace(user.XID))
	user.AgentPublicKey = strings.ToLower(strings.TrimSpace(user.AgentPublicKey))
	return user, nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch load usage info for codes that have been used
	usedCodeIDs := make([]string, 0)
	codeIndexMap := make(map[string]int)
	for i, item := range items {
		if item.UsedCount > 0 {
			usedCodeIDs = append(usedCodeIDs, item.ID)
			codeIndexMap[item.ID] = i
		}
	}

	if len(usedCodeIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(usedCodeIDs)), ",")
		args := make([]any, 0, len(usedCodeIDs))
		for _, id := range usedCodeIDs {
			args = append(args, id)
		}
		usageRows, err := s.db.Query(`
			SELECT u.code_id, u.user_id, IFNULL(usr.name, ''), u.used_at
			FROM invite_code_usages u
			LEFT JOIN users usr ON usr.id = u.user_id
			WHERE u.code_id IN (`+placeholders+`)
			ORDER BY u.used_at DESC`, args...)
		if err == nil {
			defer usageRows.Close()
			for usageRows.Next() {
				var codeID, userID, userName, usedAt string
				if err := usageRows.Scan(&codeID, &userID, &userName, &usedAt); err != nil {
					continue
				}
				if idx, ok := codeIndexMap[codeID]; ok {
					items[idx].UsedByUsers = append(items[idx].UsedByUsers, InviteCodeUser{
						UserID:   userID,
						UserName: userName,
						UsedAt:   usedAt,
					})
				}
			}
		}
	}

	return items, nil
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

func (s *Store) updateInviteCode(id string, description *string, maxUses *int, status *string) (InviteCode, error) {
	var item InviteCode
	err := s.db.QueryRow(`
		SELECT id, code, IFNULL(description, ''), status, IFNULL(max_uses, 0), used_count, created_at
		FROM invite_codes WHERE id = ?`, id,
	).Scan(&item.ID, &item.Code, &item.Description, &item.Status, &item.MaxUses, &item.UsedCount, &item.CreatedAt)
	if err != nil {
		return InviteCode{}, err
	}

	if description != nil {
		item.Description = strings.TrimSpace(*description)
	}
	if maxUses != nil {
		if *maxUses < 0 {
			item.MaxUses = 0
		} else {
			item.MaxUses = *maxUses
		}
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		item.Status = strings.TrimSpace(*status)
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
	defer func() { _ = tx.Rollback() }()

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
	passwordHash, err := hashPasswordStrong(trimmedPassword)
	if err != nil {
		return AdminUser{}, err
	}
	_, err = s.db.Exec(`
		INSERT INTO admins(id, email, password_hash, name, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, id, trimmedEmail, passwordHash, trimmedName, createdAt)
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

	passwordHash, err := hashPasswordStrong(trimmedPassword)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?`, passwordHash, trimmedID)
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
	defer func() { _ = tx.Rollback() }()

	// Release agent accounts bound to deleted users.
	if _, err = tx.Exec(`UPDATE agent_accounts
		SET status = 'unused',
		    assigned_user_id = NULL,
		    assigned_at = NULL,
		    vault_address = '',
		    evm_balance = 0,
		    agent_status = 'inactive',
		    initial_capital = 0
		WHERE assigned_user_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}

	// Roll back invite code usage counters before removing usage rows.
	usageRows, qErr := tx.Query(`SELECT code_id, COUNT(1) FROM invite_code_usages WHERE user_id IN (`+placeholders+`) GROUP BY code_id`, args...)
	if qErr != nil {
		return 0, qErr
	}
	type usageDelta struct {
		codeID string
		count  int
	}
	deltas := make([]usageDelta, 0)
	for usageRows.Next() {
		var d usageDelta
		if err = usageRows.Scan(&d.codeID, &d.count); err != nil {
			_ = usageRows.Close()
			return 0, err
		}
		deltas = append(deltas, d)
	}
	if err = usageRows.Err(); err != nil {
		_ = usageRows.Close()
		return 0, err
	}
	_ = usageRows.Close()
	for _, d := range deltas {
		if _, err = tx.Exec(`UPDATE invite_codes
			SET used_count = CASE
				WHEN used_count > ? THEN used_count - ?
				ELSE 0
			END
			WHERE id = ?`, d.count, d.count, d.codeID); err != nil {
			return 0, err
		}
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

func (s *Store) setSettingTx(tx *sql.Tx, key string, value string) error {
	_, err := tx.Exec(`
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

func (s *Store) getSettingDefaultTx(tx *sql.Tx, key string, defaultVal string) (string, error) {
	var val string
	err := tx.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(val) == "" {
		return defaultVal, nil
	}
	return val, nil
}

func (s *Store) getDailySlots() (DailySlots, error) {
	totalStr := s.getSettingDefault("intern_slots_total", "100")
	total, _ := strconv.Atoi(totalStr)
	if total <= 0 {
		total = 100
	}
	var consumed int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM agent_accounts a
		JOIN agent_vaults v ON lower(a.vault_address) = lower(v.vault_address)
		WHERE a.vault_address != '' AND v.valid = 1 AND a.status = 'assigned'`).Scan(&consumed)
	remaining := total - consumed
	if remaining < 0 {
		remaining = 0
	}
	return DailySlots{Total: total, Consumed: consumed, Remaining: remaining}, nil
}

func (s *Store) getTvlOffset() float64 {
	str := s.getSettingDefault("tvl_offset", "0")
	v, _ := strconv.ParseFloat(str, 64)
	return v
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
