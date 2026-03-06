export interface AdminUser {
  id: string;
  email: string;
  name: string;
  createdAt: string;
}

export interface DashboardStats {
  totalUsers: number;
  totalAgentAccounts: number;
  assignedAgents: number;
  unusedAgents: number;
  totalInviteCodes: number;
  activeInviteCodes: number;
  // Financial
  totalTvl: number;
  totalPnl: number;
  totalCapital: number;
  fundGrowthRate: number;
  // Agent performance summary
  averageRoi: number;
  bestAgents: AgentPerfSummary[];
  worstAgents: AgentPerfSummary[];
  // User growth
  newUsersToday: number;
  newUsersWeek: number;
  conversionRate: number;
  // Invite codes
  inviteConversionRate: number;
  topInviteCodes: InviteCodeSummary[];
  // System health
  lastSyncAt: string;
  syncRoundCount: number;
  dataFreshness: number;
}

export interface AgentPerfSummary {
  publicKey: string;
  name: string;
  pnl: number;
  roi: number;
}

export interface InviteCodeSummary {
  code: string;
  usedCount: number;
}

export interface User {
  id: string;
  email?: string;
  walletAddress?: string;
  name?: string;
  agentPublicKey?: string;
  inviteCodeUsed?: string;
  createdAt: string;
}

export interface InviteCode {
  id: string;
  code: string;
  description: string;
  status: string;
  maxUses?: number;
  usedCount: number;
  usedByUsers?: InviteCodeUser[];
  createdAt: string;
}

export interface InviteCodeUser {
  userId: string;
  userName: string;
  usedAt: string;
}

export interface AgentAccount {
  id: string;
  publicKey: string;
  status: 'unused' | 'assigned';
  assignedUserId?: string;
  assignedUserName?: string;
  assignedAt?: string;
  createdAt: string;
}

export interface AgentMarketItem {
  publicKey: string;
  name?: string;
  description?: string;
  userId?: string;
  userName?: string;
  accountValue?: number;
  totalPnL?: number;
  lastSyncedAt?: string;
  performanceFee?: number;
  initialCapital?: number;
}

export interface VaultRecord {
  vaultAddress: string;
  userAddress: string;
  evmBalance: number;
  initialCapital: number;
  valid: boolean;
  allocatorAddress: string;
  accountValue: number;
  unrealizedPnl: number;
  lastSyncedAt?: string;
  syncStatus?: string;
  syncError?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AgentImportResult {
  imported: number;
  duplicates: number;
  invalid: number;
  publicKeys: string[];
}

export interface TreasurySnapshot {
  id: string;
  vaultEvm: number;
  vaultPerps: number;
  vaultSpot: number;
  vaultPnl: number;
  vaultCapital: number;
  allocatorEvm: number;
  allocatorPerps: number;
  allocatorSpot: number;
  ownerEvm: number;
  ownerPerps: number;
  ownerSpot: number;
  totalFunds: number;
  vaultCount: number;
  activeVaultCount: number;
  allocatorAddress: string;
  ownerAddress: string;
  createdAt: string;
}

export interface AgentPerformance {
  publicKey: string;
  roi: number;
  winRate: number;
  totalFills: number;
  profitableFills: number;
  maxDrawdown: number;
  sharpeRatio: number;
  tradingFrequency: number;
  totalClosedPnl: number;
  avgWinSize: number;
  avgLossSize: number;
  daysSinceCreation: number;
}

export interface PlatformSnapshot {
  id: string;
  totalTvl: number;
  totalPnl: number;
  totalCapital: number;
  userCount: number;
  activeAgentCount: number;
  totalAgentCount: number;
  totalTrades: number;
  createdAt: string;
}

export interface AgentLeaderboardItem {
  publicKey: string;
  name: string;
  totalFills: number;
  profitableFills: number;
  winRate: number;
  totalClosedPnl: number;
  roi: number;
  accountValue: number;
  initialCapital: number;
}
