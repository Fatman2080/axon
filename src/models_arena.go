package main

type AgentArenaLatestFill struct {
	ID        string    `json:"id"`
	PublicKey string    `json:"publicKey"`
	FillTime  int64     `json:"fillTime"`
	Fill      VaultFill `json:"fill"`
}

type AgentArenaAgent struct {
	ID              string  `json:"id"`
	PublicKey       string  `json:"publicKey"`
	Name            string  `json:"name"`
	UserName        string  `json:"userName,omitempty"`
	AccountValue    float64 `json:"accountValue"`
	TotalPnL        float64 `json:"totalPnl"`
	Status          string  `json:"status"`
	LastSyncedAt    string  `json:"lastSyncedAt,omitempty"`
	LastRealizedPnl float64 `json:"lastRealizedPnl"`
	LastRealizedAt  string  `json:"lastRealizedAt,omitempty"`
	LastFillID      string  `json:"lastFillId,omitempty"`
	LastFillTime    int64   `json:"lastFillTime,omitempty"`
	UpdatedAt       string  `json:"updatedAt,omitempty"`
}

type AgentArenaSnapshotResponse struct {
	GeneratedAt         string            `json:"generatedAt"`
	SyncIntervalSeconds int               `json:"syncIntervalSeconds"`
	StaleAfterSeconds   int               `json:"staleAfterSeconds"`
	Agents              []AgentArenaAgent `json:"agents"`
}

type AgentArenaEvent struct {
	Type      string           `json:"type"`
	EmittedAt string           `json:"emittedAt"`
	AgentID   string           `json:"agentId,omitempty"`
	Agent     *AgentArenaAgent `json:"agent,omitempty"`
	PnlDelta  float64          `json:"pnlDelta,omitempty"`
	Status    string           `json:"status,omitempty"`
}

type AgentArenaWSMessage struct {
	Type     string                      `json:"type"`
	Snapshot *AgentArenaSnapshotResponse `json:"snapshot,omitempty"`
	Events   []AgentArenaEvent           `json:"events,omitempty"`
}
