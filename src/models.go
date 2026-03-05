package main

import "time"

type User struct {
	ID              string `json:"id"`
	Email           string `json:"email,omitempty"`
	Name            string `json:"name,omitempty"`
	XID             string `json:"xId,omitempty"`
	XUsername       string `json:"xUsername,omitempty"`
	Avatar          string `json:"avatar,omitempty"`
	InviteCodeUsed  string `json:"inviteCodeUsed,omitempty"`
	AgentPublicKey  string `json:"agentPublicKey,omitempty"`
	AgentAssignedAt string `json:"agentAssignedAt,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type InviteCode struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      string `json:"status"`
	MaxUses     int    `json:"maxUses,omitempty"`
	UsedCount   int    `json:"usedCount"`
	CreatedAt   string `json:"createdAt"`
}

type AgentAccount struct {
	ID               string `json:"id"`
	PublicKey        string `json:"publicKey"`
	Status           string `json:"status"`
	AssignedUserID   string `json:"assignedUserId,omitempty"`
	AssignedUserName string `json:"assignedUserName,omitempty"`
	AssignedAt       string `json:"assignedAt,omitempty"`
	CreatedAt        string `json:"createdAt"`
}

type AgentSnapshot struct {
	ID            string  `json:"id"`
	PublicKey     string  `json:"publicKey"`
	AccountValue  float64 `json:"accountValue"`
	UnrealizedPNL float64 `json:"unrealizedPnl"`
	Source        string  `json:"source"`
	CreatedAt     string  `json:"createdAt"`
}

type AgentMarketItem struct {
	PublicKey      string  `json:"publicKey"`
	Name           string  `json:"name,omitempty"`
	Description    string  `json:"description,omitempty"`
	UserID         string  `json:"userId,omitempty"`
	UserName       string  `json:"userName,omitempty"`
	AccountValue   float64 `json:"accountValue,omitempty"`
	TotalPnL       float64 `json:"totalPnL,omitempty"`
	VaultAddress   string  `json:"vaultAddress,omitempty"`
	EVMBalance     float64 `json:"evmBalance,omitempty"`
	AgentStatus    string  `json:"agentStatus,omitempty"` // "inactive", "active", "revoked"
	TVL            float64 `json:"tvl,omitempty"`
	LastSyncedAt   string  `json:"lastSyncedAt,omitempty"`
	PerformanceFee float64 `json:"performanceFee"`
}

type AgentImportResult struct {
	Imported   int      `json:"imported"`
	Duplicates int      `json:"duplicates"`
	Invalid    int      `json:"invalid"`
	PublicKeys []string `json:"publicKeys"`
}

type OAuthState struct {
	Provider     string
	State        string
	CodeVerifier string
	InviteCode   string
	NextURL      string
	CreatedAt    string
}

type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type DashboardStats struct {
	TotalUsers         int `json:"totalUsers"`
	TotalAgentAccounts int `json:"totalAgentAccounts"`
	AssignedAgents     int `json:"assignedAgents"`
	UnusedAgents       int `json:"unusedAgents"`
	TotalInviteCodes   int `json:"totalInviteCodes"`
	ActiveInviteCodes  int `json:"activeInviteCodes"`
}

type DailySlots struct {
	Total     int       `json:"total"`
	Consumed  int       `json:"consumed"`
	Remaining int       `json:"remaining"`
	ResetHour int       `json:"resetHour"`
	ResetsAt  time.Time `json:"resetsAt"`
}

type VaultPosition struct {
	Coin            string  `json:"coin"`
	Size            float64 `json:"size"`
	EntryPrice      float64 `json:"entryPrice"`
	MarkPrice       float64 `json:"markPrice"`
	UnrealizedPnl   float64 `json:"unrealizedPnl"`
	ReturnOnEquity  float64 `json:"returnOnEquity"`
	PositionValue   float64 `json:"positionValue"`
	Leverage        float64 `json:"leverage"`
	LiquidationPrice float64 `json:"liquidationPrice"`
}

type VaultFill struct {
	Coin      string  `json:"coin"`
	Side      string  `json:"side"`
	Size      float64 `json:"size"`
	Price     float64 `json:"price"`
	Time      int64   `json:"time"`
	Fee       float64 `json:"fee"`
	ClosedPnl float64 `json:"closedPnl"`
	Hash      string  `json:"hash"`
	StartPosition float64 `json:"startPosition"`
	Direction string  `json:"direction"`
}

type VaultOverview struct {
	TotalTvl       float64         `json:"totalTvl"`
	TotalEvmBalance float64        `json:"totalEvmBalance"`
	TotalL1Value   float64         `json:"totalL1Value"`
	AgentCount     int             `json:"agentCount"`
	TotalPnl       float64         `json:"totalPnl"`
	Positions      []VaultPosition `json:"positions"`
	RecentFills    []VaultFill     `json:"recentFills"`
}
