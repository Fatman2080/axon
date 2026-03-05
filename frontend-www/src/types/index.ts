export interface Vault {
  id: string;
  name: string;
  totalTvl: number; // 全球LP总资金池
  sharePrice: number; // HLP share price
  apy: number; // 预期年化
  totalProfit: number;
  agentCount: number; // 正在工作的Agent数量
  depositorCount: number;
  performanceHistory: number[]; // 每日收益率历史
  // Equity Breakdown
  chainBalance: number;    // On-chain Wallet
  clusterBalance: number;  // Address Cluster
  hypeBalance: number;     // Hyperliquid Exchange
  agentBalance: number;    // Agent Cluster Account
}

export interface Strategy {
  id: string;
  name: string;
  description: string;
  category: 'arbitrage' | 'trend' | 'grid' | 'martingale';
  minInvestment: number; // 这里的投资概念变为"最低分配额度"
  riskLevel: 'low' | 'medium' | 'high';
  expectedReturn: number; // 预期年化收益率
  runningDays: number;
  userCount: number; // 支持者数量 (Stakers/Voters)
  creator: string;
  // Agent TVL管理
  currentTvl: number; // 从大池子分配到的管理资金
  maxTvl: number; // 最大资金管理上限
  pnlContribution: number; // 为大池子贡献的总利润
  performanceFee: number; // 绩效费率 (e.g. 0.2 = 20%)
  performance: {
    daily: number[]; // 最近30天收益
    monthly: number[]; // 最近12月收益
  };
  parameters: {
    stopLoss: number;
    takeProfit: number;
    maxPosition: number;
  };
  rating: number;
  reviews: Review[];
  // Agent 提交与审核状态
  status: 'pending' | 'approved' | 'rejected' | 'active'; 
  codeType?: 'python' | 'langgraph' | 'api';
  backtestMetrics?: {
    sharpeRatio: number;
    maxDrawdown: number;
  };
  social?: {
    twitter?: string;
    discord?: string;
    website?: string;
  };
  vaultAddress?: string;
  evmBalance?: number;
  agentStatus?: 'inactive' | 'active' | 'revoked';
}

export interface Review {
  id: string;
  userId: string;
  rating: number;
  comment: string;
  createdAt: string; // Changed to string for serialization
}

export interface Agent {
  id: string;
  name: string;
  strategyId: string;
  userId: string; // Owner/Developer
  publicKey?: string;
  status: 'running' | 'stopped' | 'error';
  investment: number; // 这里的 investment 实际是指分配到的 TVL
  currentValue: number; // 当前 TVL + 浮盈
  totalProfit: number; // 累计贡献利润
  todayProfit: number;
  createdAt: string; // Changed to string for serialization
  lastTrade: string; // Changed to string for serialization
  trades: Trade[];
  config: {
    tradingPairs: string[];
    riskLevel: number;
    autoReinvest: boolean;
  };
}

export interface Trade {
  id: string;
  pair: string;
  type: 'buy' | 'sell';
  amount: number;
  price: number;
  profit: number;
  timestamp: string; // Changed to string for serialization
}

export interface User {
  id: string;
  email: string;
  walletAddress: string;
  name: string;
  xId?: string;
  xUsername?: string;
  avatar?: string;
  level: 'basic' | 'premium' | 'vip';
  // LP 相关
  lpShares: number; // 持有的 HLP 份额
  lpValue: number; // 当前 LP 价值
  totalInvestment: number; // 总投入本金
  totalProfit: number; // 总收益 (LP分红)
  agentCount: number; // 拥有的 Agent (作为开发者)
  joinedAt: string; // Changed to string for serialization
  inviteCodeUsed?: string;
  agentPublicKey?: string;
  agentAssignedAt?: string;
  preferences: {
    notifications: boolean;
    autoUpdate: boolean;
    currency: 'USD' | 'CNY' | 'BTC';
  };
  agents?: Agent[]; // Fetched from backend
  strategies?: Strategy[]; // Fetched from backend
}

export interface ProfitStats {
  totalProfit: number;
  todayProfit: number;
  monthlyProfit: number[];
  profitTrend: number[];
}

// Backend API types
export interface AgentMarketItem {
  publicKey: string;
  name?: string;
  description?: string;
  userId?: string;
  userName?: string;
  accountValue?: number;
  totalPnL?: number;
  vaultAddress?: string;
  evmBalance?: number;
  agentStatus?: 'inactive' | 'active' | 'revoked';
  tvl?: number;
  lastSyncedAt?: string;
  performanceFee?: number;
}

export interface VaultStats {
  totalTvl: number;
  totalEvmBalance: number;
  totalL1Value: number;
  agentCount: number;
}

export interface AgentMarketDetail {
  agent: AgentMarketItem;
  history: number[];
  positions: VaultPosition[];
  recentFills: VaultFill[];
  createdAt: string;
}

export interface DailySlotsResponse {
  total: number;
  consumed: number;
  remaining: number;
  resetHour: number;
  resetsAt: string;
}

export interface VaultPosition {
  coin: string;
  size: number;
  entryPrice: number;
  markPrice: number;
  unrealizedPnl: number;
  returnOnEquity: number;
  positionValue: number;
  leverage: number;
  liquidationPrice: number;
}

export interface VaultFill {
  coin: string;
  side: string;
  size: number;
  price: number;
  time: number;
  fee: number;
  closedPnl: number;
  hash: string;
  startPosition: number;
  direction: string;
}

export interface VaultOverview {
  totalTvl: number;
  totalEvmBalance: number;
  totalL1Value: number;
  agentCount: number;
  totalPnl: number;
  positions: VaultPosition[];
  recentFills: VaultFill[];
}

export interface UserAgentStats {
  publicKey: string;
  accountValue: number;
  totalPnl: number;
  positions: VaultPosition[];
  recentFills: VaultFill[];
}

