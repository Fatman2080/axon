package main

import (
	"math"
	"sort"
	"strings"
	"time"
)

// --- Dashboard ---

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

func (s *Store) dashboardStatsEnhanced() (DashboardStatsEnhanced, error) {
	var stats DashboardStatsEnhanced

	// 1) Original counts
	basic, err := s.dashboardStats()
	if err != nil {
		return stats, err
	}
	stats.TotalUsers = basic.TotalUsers
	stats.TotalAgentAccounts = basic.TotalAgentAccounts
	stats.AssignedAgents = basic.AssignedAgents
	stats.UnusedAgents = basic.UnusedAgents
	stats.TotalInviteCodes = basic.TotalInviteCodes
	stats.ActiveInviteCodes = basic.ActiveInviteCodes

	// 2) Financial from treasury snapshot (total assets = vault + allocator + owner, each has EVM + Perps + Spot)
	snap, err := s.getLatestTreasurySnapshot()
	if err == nil && snap != nil {
		stats.TotalTVL = snap.TotalFunds
		stats.TotalPnL = snap.VaultPnl
		stats.TotalCapital = snap.VaultCapital
		if stats.TotalCapital > 0 {
			stats.FundGrowthRate = (stats.TotalTVL - stats.TotalCapital) / stats.TotalCapital
		}
	}

	// 3) User growth
	today := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	weekAgo := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE created_at >= ?`, today).Scan(&stats.NewUsersToday)
	s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE created_at >= ?`, weekAgo).Scan(&stats.NewUsersWeek)
	if stats.TotalUsers > 0 {
		stats.ConversionRate = float64(stats.AssignedAgents) / float64(stats.TotalUsers)
	}

	// 4) Invite code stats — codes are single-use, rate = used / total
	if stats.TotalInviteCodes > 0 {
		var totalUsed int
		s.db.QueryRow(`SELECT COUNT(1) FROM invite_codes WHERE used_count > 0`).Scan(&totalUsed)
		stats.InviteConversionRate = float64(totalUsed) / float64(stats.TotalInviteCodes)
	}
	stats.TopInviteCodes = s.getTopInviteCodes(5)

	// 4.5) Agent performance summary
	s.fillAgentPerfSummary(&stats)

	// 5) System health
	stats.LastSyncAt = s.getSettingDefault("last_sync_at", "")
	stats.SyncRoundCount = syncRoundCounter
	latestPlatform, err := s.getLatestPlatformSnapshot()
	if err == nil && latestPlatform != nil {
		t, parseErr := time.Parse(time.RFC3339, latestPlatform.CreatedAt)
		if parseErr == nil {
			stats.DataFreshness = int(time.Since(t).Seconds())
		}
	}

	return stats, nil
}

func (s *Store) fillAgentPerfSummary(stats *DashboardStatsEnhanced) {
	// Query per-agent PnL from agent_vaults
	rows, err := s.db.Query(`
		SELECT v.user_address,
		       IFNULL(a.name, ''),
		       IFNULL(v.account_value, 0),
		       IFNULL(v.initial_capital, 0)
		FROM agent_vaults v
		LEFT JOIN agent_accounts a ON lower(a.public_key) = lower(v.user_address)
		WHERE v.user_address != '' AND v.initial_capital > 0`)
	if err != nil {
		return
	}
	defer rows.Close()

	var agents []agentPnl
	var totalROI float64

	for rows.Next() {
		var ap agentPnl
		var accountValue, initialCapital float64
		if err := rows.Scan(&ap.publicKey, &ap.name, &accountValue, &initialCapital); err != nil {
			continue
		}
		ap.initialCapital = initialCapital
		ap.pnl = accountValue - initialCapital
		if initialCapital > 0 {
			ap.roi = ap.pnl / initialCapital
		}
		totalROI += ap.roi
		agents = append(agents, ap)
	}

	if len(agents) > 0 {
		stats.AverageROI = totalROI / float64(len(agents))
	}

	// Sort by PnL descending for best, ascending for worst
	best := make([]agentPnl, len(agents))
	copy(best, agents)
	sortAgentPnl(best)

	limit := 5
	if len(best) < limit {
		limit = len(best)
	}
	for i := 0; i < limit; i++ {
		stats.BestAgents = append(stats.BestAgents, AgentPerfSummary{
			PublicKey: best[i].publicKey,
			Name:      best[i].name,
			PnL:       best[i].pnl,
			ROI:       best[i].roi,
		})
	}
	for i := len(best) - 1; i >= 0 && len(stats.WorstAgents) < 5; i-- {
		stats.WorstAgents = append(stats.WorstAgents, AgentPerfSummary{
			PublicKey: best[i].publicKey,
			Name:      best[i].name,
			PnL:       best[i].pnl,
			ROI:       best[i].roi,
		})
	}
}

type agentPnl struct {
	publicKey      string
	name           string
	pnl            float64
	roi            float64
	initialCapital float64
}

func sortAgentPnl(agents []agentPnl) {
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].pnl > agents[j].pnl
	})
}

func (s *Store) getTopInviteCodes(limit int) []InviteCodeSummary {
	rows, err := s.db.Query(`
		SELECT code, used_count, IFNULL(max_uses, 0)
		FROM invite_codes
		WHERE used_count > 0
		ORDER BY used_count DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []InviteCodeSummary
	for rows.Next() {
		var item InviteCodeSummary
		if err := rows.Scan(&item.Code, &item.UsedCount, &item.MaxUses); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

// --- Treasury snapshots ---

func (s *Store) saveTreasurySnapshot(snap TreasurySnapshot) error {
	snap.ID = newID("tsnap")
	snap.CreatedAt = nowISO()
	_, err := s.db.Exec(
		`INSERT INTO treasury_snapshots(id, vault_evm, vault_perps, vault_spot, vault_pnl, vault_capital,
			allocator_evm, allocator_perps, allocator_spot,
			owner_evm, owner_perps, owner_spot,
			total_funds, vault_count, active_vault_count, allocator_address, owner_address, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		snap.ID, snap.VaultEvm, snap.VaultPerps, snap.VaultSpot, snap.VaultPnl, snap.VaultCapital,
		snap.AllocatorEvm, snap.AllocatorPerps, snap.AllocatorSpot,
		snap.OwnerEvm, snap.OwnerPerps, snap.OwnerSpot,
		snap.TotalFunds, snap.VaultCount, snap.ActiveVaultCount, snap.AllocatorAddress, snap.OwnerAddress, snap.CreatedAt,
	)
	return err
}

func (s *Store) getLatestTreasurySnapshot() (*TreasurySnapshot, error) {
	row := s.db.QueryRow(
		`SELECT id, vault_evm, vault_perps, vault_spot, vault_pnl, vault_capital,
			allocator_evm, allocator_perps, allocator_spot,
			owner_evm, owner_perps, owner_spot,
			total_funds, vault_count, active_vault_count, allocator_address, owner_address, created_at
		FROM treasury_snapshots ORDER BY created_at DESC LIMIT 1`,
	)
	var snap TreasurySnapshot
	err := row.Scan(
		&snap.ID, &snap.VaultEvm, &snap.VaultPerps, &snap.VaultSpot, &snap.VaultPnl, &snap.VaultCapital,
		&snap.AllocatorEvm, &snap.AllocatorPerps, &snap.AllocatorSpot,
		&snap.OwnerEvm, &snap.OwnerPerps, &snap.OwnerSpot,
		&snap.TotalFunds, &snap.VaultCount, &snap.ActiveVaultCount, &snap.AllocatorAddress, &snap.OwnerAddress, &snap.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (s *Store) listTreasurySnapshots(limit int, period string) ([]TreasurySnapshot, error) {
	if limit <= 0 {
		limit = 200
	}
	cutoff := ""
	switch period {
	case "1h":
		cutoff = timeAgo(1)
	case "6h":
		cutoff = timeAgo(6)
	case "1d":
		cutoff = timeAgo(24)
	case "7d":
		cutoff = timeAgo(7 * 24)
	case "30d":
		cutoff = timeAgo(30 * 24)
	case "90d":
		cutoff = timeAgo(90 * 24)
	}

	query := `SELECT id, vault_evm, vault_perps, vault_spot, vault_pnl, vault_capital,
		allocator_evm, allocator_perps, allocator_spot,
		owner_evm, owner_perps, owner_spot,
		total_funds, vault_count, active_vault_count, allocator_address, owner_address, created_at
		FROM treasury_snapshots`
	var args []any
	if cutoff != "" {
		query += ` WHERE created_at >= ?`
		args = append(args, cutoff)
	}
	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TreasurySnapshot
	for rows.Next() {
		var snap TreasurySnapshot
		if err := rows.Scan(
			&snap.ID, &snap.VaultEvm, &snap.VaultPerps, &snap.VaultSpot, &snap.VaultPnl, &snap.VaultCapital,
			&snap.AllocatorEvm, &snap.AllocatorPerps, &snap.AllocatorSpot,
			&snap.OwnerEvm, &snap.OwnerPerps, &snap.OwnerSpot,
			&snap.TotalFunds, &snap.VaultCount, &snap.ActiveVaultCount, &snap.AllocatorAddress, &snap.OwnerAddress, &snap.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, snap)
	}
	return results, nil
}

// --- Platform snapshots ---

func (s *Store) savePlatformSnapshot(snap PlatformSnapshot) error {
	snap.ID = newID("psnap")
	snap.CreatedAt = nowISO()
	_, err := s.db.Exec(`
		INSERT INTO platform_snapshots(id, total_tvl, total_pnl, total_capital, user_count, active_agent_count, total_agent_count, total_trades, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ID, snap.TotalTVL, snap.TotalPnL, snap.TotalCapital, snap.UserCount, snap.ActiveAgentCount, snap.TotalAgentCount, snap.TotalTrades, snap.CreatedAt,
	)
	return err
}

func (s *Store) getLatestPlatformSnapshot() (*PlatformSnapshot, error) {
	var snap PlatformSnapshot
	err := s.db.QueryRow(`
		SELECT id, total_tvl, total_pnl, total_capital, user_count, active_agent_count, total_agent_count, total_trades, created_at
		FROM platform_snapshots ORDER BY created_at DESC LIMIT 1`,
	).Scan(&snap.ID, &snap.TotalTVL, &snap.TotalPnL, &snap.TotalCapital, &snap.UserCount, &snap.ActiveAgentCount, &snap.TotalAgentCount, &snap.TotalTrades, &snap.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (s *Store) listPlatformSnapshots(limit int, period string) ([]PlatformSnapshot, error) {
	if limit <= 0 {
		limit = 200
	}
	cutoff := periodToCutoff(period)
	query := `SELECT id, total_tvl, total_pnl, total_capital, user_count, active_agent_count, total_agent_count, total_trades, created_at
		FROM platform_snapshots`
	var args []any
	if cutoff != "" {
		query += ` WHERE created_at >= ?`
		args = append(args, cutoff)
	}
	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PlatformSnapshot
	for rows.Next() {
		var snap PlatformSnapshot
		if err := rows.Scan(&snap.ID, &snap.TotalTVL, &snap.TotalPnL, &snap.TotalCapital, &snap.UserCount, &snap.ActiveAgentCount, &snap.TotalAgentCount, &snap.TotalTrades, &snap.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, snap)
	}
	return results, nil
}

func (s *Store) getPlatformGrowth(period string) map[string]float64 {
	result := map[string]float64{}
	latest, err := s.getLatestPlatformSnapshot()
	if err != nil || latest == nil {
		return result
	}

	hours := periodToHours(period)
	cutoff := timeAgo(hours)
	var older PlatformSnapshot
	err = s.db.QueryRow(`
		SELECT id, total_tvl, total_pnl, total_capital, user_count, active_agent_count, total_agent_count, total_trades, created_at
		FROM platform_snapshots
		WHERE created_at <= ?
		ORDER BY created_at DESC LIMIT 1`, cutoff,
	).Scan(&older.ID, &older.TotalTVL, &older.TotalPnL, &older.TotalCapital, &older.UserCount, &older.ActiveAgentCount, &older.TotalAgentCount, &older.TotalTrades, &older.CreatedAt)
	if err != nil {
		return result
	}

	if older.TotalTVL > 0 {
		result["tvlGrowth"] = (latest.TotalTVL - older.TotalTVL) / older.TotalTVL
	}
	if older.UserCount > 0 {
		result["userGrowth"] = float64(latest.UserCount-older.UserCount) / float64(older.UserCount)
	}
	if older.ActiveAgentCount > 0 {
		result["agentGrowth"] = float64(latest.ActiveAgentCount-older.ActiveAgentCount) / float64(older.ActiveAgentCount)
	}
	if older.TotalTrades > 0 {
		result["tradeGrowth"] = float64(latest.TotalTrades-older.TotalTrades) / float64(older.TotalTrades)
	}

	return result
}

// --- Agent Performance ---

func (s *Store) getAgentPerformance(publicKey string) (AgentPerformance, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	perf := AgentPerformance{PublicKey: publicKey}

	// ROI from agent_vaults
	var accountValue, initialCapital float64
	err := s.db.QueryRow(`
			SELECT IFNULL(account_value, 0), IFNULL(initial_capital, 0)
			FROM agent_vaults WHERE lower(user_address) = ?`, publicKey).Scan(&accountValue, &initialCapital)
	if err != nil {
		// Fallback: latest snapshot value + initial capital in agent_accounts
		s.db.QueryRow(`
			SELECT IFNULL(
			         (SELECT account_value
			          FROM agent_snapshots
			          WHERE public_key = ?
			          ORDER BY created_at DESC
			          LIMIT 1),
			         IFNULL(evm_balance, 0)
			       ),
			       IFNULL(initial_capital, 0)
			FROM agent_accounts
			WHERE public_key = ?`, publicKey, publicKey).Scan(&accountValue, &initialCapital)
	}
	if initialCapital > 0 {
		perf.ROI = (accountValue - initialCapital) / initialCapital
	}

	// Win rate + PnL from agent_fills using json_extract
	var totalFills, profitableFills int
	var totalClosedPnl, sumWin, sumLoss float64
	err = s.db.QueryRow(`
		SELECT COUNT(1),
		       IFNULL(SUM(CASE WHEN CAST(json_extract(data_json, '$.closedPnl') AS REAL) > 0 THEN 1 ELSE 0 END), 0),
		       IFNULL(SUM(CAST(json_extract(data_json, '$.closedPnl') AS REAL)), 0),
		       IFNULL(SUM(CASE WHEN CAST(json_extract(data_json, '$.closedPnl') AS REAL) > 0 THEN CAST(json_extract(data_json, '$.closedPnl') AS REAL) ELSE 0 END), 0),
		       IFNULL(SUM(CASE WHEN CAST(json_extract(data_json, '$.closedPnl') AS REAL) < 0 THEN CAST(json_extract(data_json, '$.closedPnl') AS REAL) ELSE 0 END), 0)
		FROM agent_fills WHERE public_key = ?`, publicKey,
	).Scan(&totalFills, &profitableFills, &totalClosedPnl, &sumWin, &sumLoss)
	if err == nil {
		perf.TotalFills = totalFills
		perf.ProfitableFills = profitableFills
		perf.TotalClosedPnl = totalClosedPnl
		if totalFills > 0 {
			perf.WinRate = float64(profitableFills) / float64(totalFills)
		}
		if profitableFills > 0 {
			perf.AvgWinSize = sumWin / float64(profitableFills)
		}
		lossFills := totalFills - profitableFills
		if lossFills > 0 {
			perf.AvgLossSize = sumLoss / float64(lossFills)
		}
	}

	// Max drawdown from agent_snapshots
	perf.MaxDrawdown = s.calcMaxDrawdown(publicKey)

	// Sharpe ratio from daily returns
	perf.SharpeRatio = s.calcSharpeRatio(publicKey)

	// Trading frequency
	createdAt := s.getAgentCreatedAt(publicKey)
	if createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			days := time.Since(t).Hours() / 24
			perf.DaysSinceCreation = int(days)
			if days > 0 {
				perf.TradingFrequency = float64(totalFills) / days
			}
		}
	}

	return perf, nil
}

func (s *Store) calcMaxDrawdown(publicKey string) float64 {
	rows, err := s.db.Query(`
		SELECT account_value FROM agent_snapshots
		WHERE public_key = ?
		ORDER BY created_at ASC`, publicKey)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var peak, maxDD float64
	for rows.Next() {
		var val float64
		if err := rows.Scan(&val); err != nil {
			continue
		}
		if val > peak {
			peak = val
		}
		if peak > 0 {
			dd := (peak - val) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

func (s *Store) calcSharpeRatio(publicKey string) float64 {
	// Get daily end-of-day values
	rows, err := s.db.Query(`
		SELECT date(created_at) AS d, account_value
		FROM agent_snapshots
		WHERE public_key = ?
		GROUP BY d
		HAVING created_at = MAX(created_at)
		ORDER BY d ASC`, publicKey)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var d string
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			continue
		}
		values = append(values, v)
	}

	if len(values) < 3 {
		return 0
	}

	// Calculate daily returns
	returns := make([]float64, 0, len(values)-1)
	for i := 1; i < len(values); i++ {
		if values[i-1] > 0 {
			returns = append(returns, (values[i]-values[i-1])/values[i-1])
		}
	}

	if len(returns) < 2 {
		return 0
	}

	// Mean and stddev
	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	stddev := math.Sqrt(variance)

	if stddev == 0 {
		return 0
	}

	return (mean / stddev) * math.Sqrt(365)
}

func (s *Store) getAgentLeaderboard(sortBy string, limit int) ([]AgentLeaderboardItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Get per-agent fill stats
	rows, err := s.db.Query(`
		SELECT f.public_key,
		       IFNULL(a.name, ''),
		       COUNT(1) AS total_fills,
		       IFNULL(SUM(CASE WHEN CAST(json_extract(f.data_json, '$.closedPnl') AS REAL) > 0 THEN 1 ELSE 0 END), 0) AS profitable_fills,
		       IFNULL(SUM(CAST(json_extract(f.data_json, '$.closedPnl') AS REAL)), 0) AS total_closed_pnl,
		       IFNULL(v.account_value, 0),
		       IFNULL(v.initial_capital, 0)
		FROM agent_fills f
		LEFT JOIN agent_accounts a ON a.public_key = f.public_key
		LEFT JOIN agent_vaults v ON lower(v.user_address) = f.public_key
		GROUP BY f.public_key
		HAVING total_fills > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AgentLeaderboardItem
	for rows.Next() {
		var item AgentLeaderboardItem
		var accountValue, initialCapital float64
		if err := rows.Scan(
			&item.PublicKey, &item.Name, &item.TotalFills, &item.ProfitableFills,
			&item.TotalClosedPnl, &accountValue, &initialCapital,
		); err != nil {
			continue
		}
		if item.TotalFills > 0 {
			item.WinRate = float64(item.ProfitableFills) / float64(item.TotalFills)
		}
		if initialCapital > 0 {
			item.ROI = (accountValue - initialCapital) / initialCapital
		}
		item.AccountValue = accountValue
		item.InitialCapital = initialCapital
		items = append(items, item)
	}

	// Sort
	sortLeaderboard(items, sortBy)

	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func sortLeaderboard(items []AgentLeaderboardItem, sortBy string) {
	sort.Slice(items, func(i, j int) bool {
		switch sortBy {
		case "roi":
			return items[i].ROI > items[j].ROI
		case "winRate":
			return items[i].WinRate > items[j].WinRate
		case "trades":
			return items[i].TotalFills > items[j].TotalFills
		default: // "pnl"
			return items[i].TotalClosedPnl > items[j].TotalClosedPnl
		}
	})
}

// --- Helpers ---

func timeAgo(hours int) string {
	return time.Now().UTC().Add(-time.Duration(hours) * time.Hour).Format(time.RFC3339)
}

func periodToCutoff(period string) string {
	switch strings.TrimSpace(period) {
	case "1h":
		return timeAgo(1)
	case "6h":
		return timeAgo(6)
	case "1d":
		return timeAgo(24)
	case "7d":
		return timeAgo(7 * 24)
	case "30d":
		return timeAgo(30 * 24)
	case "90d":
		return timeAgo(90 * 24)
	default:
		return ""
	}
}

func periodToHours(period string) int {
	switch strings.TrimSpace(period) {
	case "1h":
		return 1
	case "6h":
		return 6
	case "1d":
		return 24
	case "7d":
		return 7 * 24
	case "30d":
		return 30 * 24
	case "90d":
		return 90 * 24
	default:
		return 7 * 24
	}
}
