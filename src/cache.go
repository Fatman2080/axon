package main

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// apiCache holds pre-serialized JSON responses for public API endpoints.
// Data is refreshed after each sync round and served directly from memory.
type apiCache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func newAPICache() *apiCache {
	return &apiCache{items: make(map[string][]byte)}
}

// get returns the cached JSON blob for a key. Thread-safe.
func (c *apiCache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.items[key]
	return data, ok
}

// set stores a single cached entry. Thread-safe.
func (c *apiCache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = data
}

// replace atomically swaps the entire cache contents.
func (c *apiCache) replace(items map[string][]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = items
}

// refreshPublicCache pre-computes all public API JSON responses and atomically
// swaps the cache. Called at the end of each sync round.
func (s *Server) refreshPublicCache() {
	start := time.Now()
	m := make(map[string][]byte)

	marshal := func(key string, v interface{}) {
		data, err := json.Marshal(v)
		if err != nil {
			logWarn("cache", "failed to marshal %s: %v", key, err)
			return
		}
		m[key] = data
	}

	// --- agent list (used by vault_stats, vault_overview, agent_market) ---
	items, err := s.store.listAgentStats("")
	if err != nil {
		logWarn("cache", "listAgentStats failed, skipping refresh: %v", err)
		return
	}

	// vault_stats
	var totalTvl, totalL1Value, totalEvmBalance, totalInitialCapital float64
	var activeCount int
	for _, item := range items {
		if item.AgentStatus != AgentStatusActive {
			continue
		}
		totalTvl += item.TVL
		totalL1Value += item.AccountValue
		totalEvmBalance += item.EVMBalance
		totalInitialCapital += item.InitialCapital
		activeCount++
	}
	marshal("vault_stats", echo.Map{
		"totalTvl":            totalTvl,
		"totalEvmBalance":     totalEvmBalance,
		"totalL1Value":        totalL1Value,
		"agentCount":          activeCount,
		"totalInitialCapital": totalInitialCapital,
		"treasuryTotal":       s.getTreasuryTotal(),
	})

	// vault_overview
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
		overview.TotalInitialCapital += item.InitialCapital
		overview.AgentCount++
	}
	if fills, err := s.store.listRecentFillsForActiveAgents(50); err == nil {
		overview.RecentFills = fills
	}
	marshal("vault_overview", overview)

	// agent_market (no search filter)
	marshal("agent_market", items)

	// --- agent_detail:{publicKey} for each assigned agent ---
	assignedKeys, _ := s.store.listAssignedPublicKeys()
	for _, pk := range assignedKeys {
		pk = strings.ToLower(strings.TrimSpace(pk))
		if pk == "" {
			continue
		}
		agent, err := s.store.getAgentStats(pk)
		if err != nil {
			continue
		}
		snapshots, err := s.store.listAgentSnapshots(pk, 120, "")
		if err != nil {
			snapshots = nil
		}
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i].CreatedAt < snapshots[j].CreatedAt
		})
		history := make([]float64, 0, len(snapshots))
		for _, snap := range snapshots {
			history = append(history, snap.AccountValue)
		}
		recentFills, err := s.store.listAgentFills(pk, 50)
		if err != nil {
			recentFills = make([]VaultFill, 0)
		}
		createdAt := s.store.getAgentCreatedAt(pk)
		perf, _ := s.store.getAgentPerformance(pk)

		marshal("agent_detail:"+pk, echo.Map{
			"agent":       agent,
			"history":     history,
			"positions":   make([]VaultPosition, 0),
			"recentFills": recentFills,
			"createdAt":   createdAt,
			"performance": perf,
		})
	}

	// --- treasury ---
	if snap, err := s.store.getLatestTreasurySnapshot(); err == nil && snap != nil {
		marshal("treasury", snap)
	}

	// --- treasury_history:{period} ---
	for _, period := range []string{"24h", "7d", "30d", "ALL"} {
		if hist, err := s.store.listTreasurySnapshots(200, period); err == nil {
			marshal("treasury_history:"+period, hist)
		}
	}

	// --- platform_stats ---
	latest, err := s.store.getLatestPlatformSnapshot()
	if err == nil && latest != nil {
		growth := s.store.getPlatformGrowth("7d")
		marshal("platform_stats", echo.Map{
			"totalTvl":        latest.TotalTVL,
			"totalPnl":        latest.TotalPnL,
			"totalCapital":    latest.TotalCapital,
			"agentCount":      latest.ActiveAgentCount,
			"totalAgentCount": latest.TotalAgentCount,
			"userCount":       latest.UserCount,
			"totalTrades":     latest.TotalTrades,
			"growthRate7d":    growth,
			"lastUpdated":     latest.CreatedAt,
		})
	}

	// --- platform_history:{period} ---
	for _, period := range []string{"24h", "7d", "30d", "ALL"} {
		if hist, err := s.store.listPlatformSnapshots(200, period); err == nil {
			marshal("platform_history:"+period, hist)
		}
	}

	// --- daily_slots ---
	if slots, err := s.store.getDailySlots(); err == nil {
		marshal("daily_slots", slots)
	}

	s.cache.replace(m)
	logInfo("cache", "refreshed %d keys in %.0fms", len(m), float64(time.Since(start).Milliseconds()))
}
