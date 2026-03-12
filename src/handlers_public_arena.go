package main

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var agentArenaUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleAgentArenaSnapshot(c echo.Context) error {
	if s.arenaHub != nil {
		return c.JSON(http.StatusOK, s.arenaHub.snapshotCopy())
	}
	snapshot, err := s.buildAgentArenaSnapshot()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed_to_load_agent_arena"})
	}
	return c.JSON(http.StatusOK, snapshot)
}

func (s *Server) handleAgentArenaWS(c echo.Context) error {
	conn, err := agentArenaUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logWarn("arena", "websocket upgrade failed: %v", err)
		return nil
	}
	defer conn.Close()

	if s.arenaHub == nil {
		snapshot, err := s.buildAgentArenaSnapshot()
		if err != nil {
			return nil
		}
		if err := conn.WriteJSON(AgentArenaWSMessage{
			Type:     "snapshot",
			Snapshot: &snapshot,
		}); err != nil {
			return nil
		}
		return nil
	}

	eventsCh, currentSnapshot, unsubscribe := s.arenaHub.subscribeWithSnapshot(16)
	defer unsubscribe()

	if err := conn.WriteJSON(AgentArenaWSMessage{
		Type:     "snapshot",
		Snapshot: &currentSnapshot,
	}); err != nil {
		return nil
	}

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return nil
			}
		case events, ok := <-eventsCh:
			if !ok {
				return nil
			}
			if len(events) == 0 {
				continue
			}
			if err := conn.WriteJSON(AgentArenaWSMessage{
				Type:   "events",
				Events: events,
			}); err != nil {
				return nil
			}
		}
	}
}

func (s *Server) buildAgentArenaSnapshot() (AgentArenaSnapshotResponse, error) {
	items, err := s.store.listAgentStats("")
	if err != nil {
		return AgentArenaSnapshotResponse{}, err
	}

	latestFills, err := s.store.listLatestAgentFillsForAssignedAgents()
	if err != nil {
		return AgentArenaSnapshotResponse{}, err
	}

	now := time.Now().UTC()
	staleAfterSeconds := s.agentArenaStaleAfterSeconds()
	agents := make([]AgentArenaAgent, 0, len(items))

	for _, item := range items {
		publicKey := strings.ToLower(strings.TrimSpace(item.PublicKey))
		if publicKey == "" {
			continue
		}

		fill := latestFills[publicKey]
		lastSyncedAt, _ := parseArenaRFC3339(item.LastSyncedAt)
		fillAt := arenaFillTime(fill.FillTime)
		status := resolveAgentArenaStatus(item.AgentStatus, now, lastSyncedAt, fillAt, staleAfterSeconds)

		updatedAt := item.LastSyncedAt
		if fillAt.After(lastSyncedAt) {
			updatedAt = fillAt.Format(time.RFC3339)
		}

		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.UserName)
		}
		if name == "" {
			name = shortArenaAgentName(publicKey)
		}

		agent := AgentArenaAgent{
			ID:              publicKey,
			PublicKey:       publicKey,
			Name:            name,
			UserName:        strings.TrimSpace(item.UserName),
			AccountValue:    item.AccountValue,
			TotalPnL:        item.TotalPnL,
			Status:          status,
			LastSyncedAt:    item.LastSyncedAt,
			LastRealizedPnl: fill.Fill.ClosedPnl,
			LastFillID:      fill.ID,
			LastFillTime:    fill.FillTime,
			UpdatedAt:       updatedAt,
		}
		if !fillAt.IsZero() {
			agent.LastRealizedAt = fillAt.Format(time.RFC3339)
			if updatedAt == "" {
				agent.UpdatedAt = agent.LastRealizedAt
			}
		}

		agents = append(agents, agent)
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].PublicKey < agents[j].PublicKey
	})

	return AgentArenaSnapshotResponse{
		GeneratedAt:         nowISO(),
		SyncIntervalSeconds: s.syncIntervalSecs,
		StaleAfterSeconds:   staleAfterSeconds,
		Agents:              agents,
	}, nil
}

func diffAgentArenaSnapshots(prev AgentArenaSnapshotResponse, next AgentArenaSnapshotResponse) []AgentArenaEvent {
	prevByID := make(map[string]AgentArenaAgent, len(prev.Agents))
	nextByID := make(map[string]AgentArenaAgent, len(next.Agents))
	for _, agent := range prev.Agents {
		prevByID[agent.ID] = agent
	}
	for _, agent := range next.Agents {
		nextByID[agent.ID] = agent
	}

	events := make([]AgentArenaEvent, 0)
	for _, nextAgent := range next.Agents {
		prevAgent, existed := prevByID[nextAgent.ID]
		if !existed {
			agentCopy := nextAgent
			events = append(events, AgentArenaEvent{
				Type:      "agent_joined",
				EmittedAt: coalesceArenaTime(nextAgent.UpdatedAt, next.GeneratedAt),
				Agent:     &agentCopy,
			})
			continue
		}

		deathTransition := false
		if prevAgent.Status != "dead" && nextAgent.Status == "dead" {
			events = append(events, AgentArenaEvent{
				Type:      "agent_died",
				AgentID:   nextAgent.ID,
				EmittedAt: coalesceArenaTime(nextAgent.UpdatedAt, next.GeneratedAt),
			})
			deathTransition = true
		} else if prevAgent.Status == "dead" && nextAgent.Status != "dead" {
			events = append(events, AgentArenaEvent{
				Type:      "agent_revived",
				AgentID:   nextAgent.ID,
				EmittedAt: coalesceArenaTime(nextAgent.UpdatedAt, next.GeneratedAt),
			})
			deathTransition = true
		}

		if nextAgent.LastFillID != "" && nextAgent.LastFillID != prevAgent.LastFillID {
			events = append(events, AgentArenaEvent{
				Type:      "agent_pnl_realized",
				AgentID:   nextAgent.ID,
				PnlDelta:  nextAgent.LastRealizedPnl,
				EmittedAt: coalesceArenaTime(nextAgent.LastRealizedAt, next.GeneratedAt),
			})
		}

		if !deathTransition && prevAgent.Status != nextAgent.Status && nextAgent.Status != "dead" {
			events = append(events, AgentArenaEvent{
				Type:      "agent_state_changed",
				AgentID:   nextAgent.ID,
				Status:    nextAgent.Status,
				EmittedAt: coalesceArenaTime(nextAgent.UpdatedAt, next.GeneratedAt),
			})
		}
	}

	for _, prevAgent := range prev.Agents {
		if _, ok := nextByID[prevAgent.ID]; ok {
			continue
		}
		events = append(events, AgentArenaEvent{
			Type:      "agent_removed",
			AgentID:   prevAgent.ID,
			EmittedAt: next.GeneratedAt,
		})
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].EmittedAt == events[j].EmittedAt {
			return events[i].Type < events[j].Type
		}
		return events[i].EmittedAt < events[j].EmittedAt
	})

	return events
}

func (s *Server) reconcileAgentArenaState() {
	if s.arenaHub == nil {
		return
	}
	snapshot, err := s.buildAgentArenaSnapshot()
	if err != nil {
		logWarn("arena", "failed to reconcile arena state: %v", err)
		return
	}
	s.arenaHub.reconcile(snapshot)
}

func (s *Server) agentArenaPollInterval() time.Duration {
	if s.syncIntervalSecs <= 0 {
		return 5 * time.Second
	}
	seconds := s.syncIntervalSecs / 2
	if seconds < 2 {
		seconds = 2
	}
	if seconds > 10 {
		seconds = 10
	}
	return time.Duration(seconds) * time.Second
}

func (s *Server) agentArenaStaleAfterSeconds() int {
	threshold := s.syncIntervalSecs * 3
	if threshold < 180 {
		threshold = 180
	}
	return threshold
}

func resolveAgentArenaStatus(agentStatus string, now time.Time, lastSyncedAt time.Time, fillAt time.Time, staleAfterSeconds int) string {
	if staleAfterSeconds > 0 && !lastSyncedAt.IsZero() && now.Sub(lastSyncedAt) > time.Duration(staleAfterSeconds)*time.Second {
		return "dead"
	}
	switch strings.ToLower(strings.TrimSpace(agentStatus)) {
	case AgentStatusActive:
		if !fillAt.IsZero() && now.Sub(fillAt) <= 90*time.Second {
			return "phone"
		}
		return "running"
	default:
		if lastSyncedAt.IsZero() {
			return "idle"
		}
		if staleAfterSeconds > 0 && now.Sub(lastSyncedAt) > time.Duration(staleAfterSeconds)*time.Second {
			return "dead"
		}
		return "idle"
	}
}

func shortArenaAgentName(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
	if len(publicKey) <= 12 {
		return publicKey
	}
	return publicKey[:6] + "..." + publicKey[len(publicKey)-4:]
}

func coalesceArenaTime(candidates ...string) string {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return nowISO()
}

func parseArenaRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty_time")
	}
	return time.Parse(time.RFC3339, value)
}

func arenaFillTime(fillTime int64) time.Time {
	if fillTime <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(fillTime).UTC()
}
