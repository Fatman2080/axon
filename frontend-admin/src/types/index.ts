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
}

export interface User {
  id: string;
  email?: string;
  walletAddress?: string;
  name?: string;
  agentPublicKey?: string;
  createdAt: string;
}

export interface InviteCode {
  id: string;
  code: string;
  description: string;
  status: 'active' | 'disabled';
  maxUses?: number;
  usedCount: number;
  createdAt: string;
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
}

export interface AgentImportResult {
  imported: number;
  duplicates: number;
  invalid: number;
  publicKeys: string[];
}
