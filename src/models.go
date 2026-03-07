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
	ID          string           `json:"id"`
	Code        string           `json:"code"`
	Description string           `json:"description"`
	Status      string           `json:"status"`
	MaxUses     int              `json:"maxUses,omitempty"`
	UsedCount   int              `json:"usedCount"`
	UsedByUsers []InviteCodeUser `json:"usedByUsers,omitempty"`
	CreatedAt   string           `json:"createdAt"`
}

type InviteCodeUser struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	UsedAt   string `json:"usedAt"`
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
	Category       string  `json:"category,omitempty"`
	UserID         string  `json:"userId,omitempty"`
	UserName       string  `json:"userName,omitempty"`
	AccountValue   float64 `json:"accountValue,omitempty"`
	TotalPnL       float64 `json:"totalPnL,omitempty"`
	VaultAddress   string  `json:"vaultAddress,omitempty"`
	EVMBalance     float64 `json:"evmBalance,omitempty"`
	AgentStatus    string  `json:"agentStatus,omitempty"` // "inactive", "active"
	TVL            float64 `json:"tvl,omitempty"`
	LastSyncedAt   string  `json:"lastSyncedAt,omitempty"`
	PerformanceFee float64 `json:"performanceFee"`
	InitialCapital float64 `json:"initialCapital,omitempty"`
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

type VaultRecord struct {
	VaultAddress     string  `json:"vaultAddress"`
	UserAddress      string  `json:"userAddress"`
	EVMBalance       float64 `json:"evmBalance"`
	InitialCapital   float64 `json:"initialCapital"`
	Valid            bool    `json:"valid"`
	AllocatorAddress string  `json:"allocatorAddress"`
	AccountValue     float64 `json:"accountValue"`
	UnrealizedPnl    float64 `json:"unrealizedPnl"`
	LastSyncedAt     string  `json:"lastSyncedAt,omitempty"`
	SyncStatus       string  `json:"syncStatus,omitempty"`
	SyncError        string  `json:"syncError,omitempty"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type VaultPosition struct {
	Coin             string  `json:"coin"`
	Size             float64 `json:"size"`
	EntryPrice       float64 `json:"entryPrice"`
	MarkPrice        float64 `json:"markPrice"`
	UnrealizedPnl    float64 `json:"unrealizedPnl"`
	ReturnOnEquity   float64 `json:"returnOnEquity"`
	PositionValue    float64 `json:"positionValue"`
	Leverage         float64 `json:"leverage"`
	LiquidationPrice float64 `json:"liquidationPrice"`
}

type VaultFill struct {
	Coin          string  `json:"coin"`
	Side          string  `json:"side"`
	Size          float64 `json:"size"`
	Price         float64 `json:"price"`
	Time          int64   `json:"time"`
	Fee           float64 `json:"fee"`
	ClosedPnl     float64 `json:"closedPnl"`
	Hash          string  `json:"hash"`
	StartPosition float64 `json:"startPosition"`
	Direction     string  `json:"direction"`
}

type VaultOverview struct {
	TotalTvl            float64         `json:"totalTvl"`
	TotalEvmBalance     float64         `json:"totalEvmBalance"`
	TotalL1Value        float64         `json:"totalL1Value"`
	AgentCount          int             `json:"agentCount"`
	TotalPnl            float64         `json:"totalPnl"`
	TotalInitialCapital float64         `json:"totalInitialCapital,omitempty"`
	Positions           []VaultPosition `json:"positions"`
	RecentFills         []VaultFill     `json:"recentFills"`
}

type TreasurySnapshot struct {
	ID               string  `json:"id"`
	VaultEvm         float64 `json:"vaultEvm"`
	VaultPerps       float64 `json:"vaultPerps"`
	VaultSpot        float64 `json:"vaultSpot"`
	VaultPnl         float64 `json:"vaultPnl"`
	VaultCapital     float64 `json:"vaultCapital"`
	AllocatorEvm     float64 `json:"allocatorEvm"`
	AllocatorPerps   float64 `json:"allocatorPerps"`
	AllocatorSpot    float64 `json:"allocatorSpot"`
	OwnerEvm         float64 `json:"ownerEvm"`
	OwnerPerps       float64 `json:"ownerPerps"`
	OwnerSpot        float64 `json:"ownerSpot"`
	TotalFunds       float64 `json:"totalFunds"`
	VaultCount       int     `json:"vaultCount"`
	ActiveVaultCount int     `json:"activeVaultCount"`
	AllocatorAddress string  `json:"allocatorAddress"`
	OwnerAddress     string  `json:"ownerAddress"`
	CreatedAt        string  `json:"createdAt"`
}

// Platform-level snapshot — saved each sync round
type PlatformSnapshot struct {
	ID               string  `json:"id"`
	TotalTVL         float64 `json:"totalTvl"`
	TotalPnL         float64 `json:"totalPnl"`
	TotalCapital     float64 `json:"totalCapital"`
	UserCount        int     `json:"userCount"`
	ActiveAgentCount int     `json:"activeAgentCount"`
	TotalAgentCount  int     `json:"totalAgentCount"`
	TotalTrades      int     `json:"totalTrades"`
	CreatedAt        string  `json:"createdAt"`
}

// Enhanced dashboard (backward-compatible + financial/perf/growth/health)
type DashboardStatsEnhanced struct {
	// Original counts
	TotalUsers         int `json:"totalUsers"`
	TotalAgentAccounts int `json:"totalAgentAccounts"`
	AssignedAgents     int `json:"assignedAgents"`
	UnusedAgents       int `json:"unusedAgents"`
	TotalInviteCodes   int `json:"totalInviteCodes"`
	ActiveInviteCodes  int `json:"activeInviteCodes"`
	// Financial
	TotalTVL       float64 `json:"totalTvl"`
	TotalPnL       float64 `json:"totalPnl"`
	TotalCapital   float64 `json:"totalCapital"`
	FundGrowthRate float64 `json:"fundGrowthRate"`
	// Agent performance summary
	AverageROI  float64            `json:"averageRoi"`
	BestAgents  []AgentPerfSummary `json:"bestAgents"`
	WorstAgents []AgentPerfSummary `json:"worstAgents"`
	// User growth
	NewUsersToday  int     `json:"newUsersToday"`
	NewUsersWeek   int     `json:"newUsersWeek"`
	ConversionRate float64 `json:"conversionRate"`
	// Invite codes
	InviteConversionRate float64             `json:"inviteConversionRate"`
	TopInviteCodes       []InviteCodeSummary `json:"topInviteCodes"`
	// System health
	LastSyncAt     string `json:"lastSyncAt"`
	SyncRoundCount int64  `json:"syncRoundCount"`
	DataFreshness  int    `json:"dataFreshness"` // seconds since last platform snapshot
}

type AgentPerfSummary struct {
	PublicKey string  `json:"publicKey"`
	Name      string  `json:"name"`
	PnL       float64 `json:"pnl"`
	ROI       float64 `json:"roi"`
}

type InviteCodeSummary struct {
	Code      string `json:"code"`
	UsedCount int    `json:"usedCount"`
	MaxUses   int    `json:"maxUses"`
}

// Single agent full performance analysis
type AgentPerformance struct {
	PublicKey         string  `json:"publicKey"`
	ROI               float64 `json:"roi"`
	WinRate           float64 `json:"winRate"`
	TotalFills        int     `json:"totalFills"`
	ProfitableFills   int     `json:"profitableFills"`
	MaxDrawdown       float64 `json:"maxDrawdown"`
	SharpeRatio       float64 `json:"sharpeRatio"`
	TradingFrequency  float64 `json:"tradingFrequency"`
	TotalClosedPnl    float64 `json:"totalClosedPnl"`
	AvgWinSize        float64 `json:"avgWinSize"`
	AvgLossSize       float64 `json:"avgLossSize"`
	DaysSinceCreation int     `json:"daysSinceCreation"`
}

// Agent leaderboard item
type AgentLeaderboardItem struct {
	PublicKey       string  `json:"publicKey"`
	Name            string  `json:"name"`
	TotalFills      int     `json:"totalFills"`
	ProfitableFills int     `json:"profitableFills"`
	WinRate         float64 `json:"winRate"`
	TotalClosedPnl  float64 `json:"totalClosedPnl"`
	ROI             float64 `json:"roi"`
	AccountValue    float64 `json:"accountValue"`
	InitialCapital  float64 `json:"initialCapital"`
}
