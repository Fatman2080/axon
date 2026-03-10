import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { Strategy, AgentMarketItem, VaultPosition, VaultFill } from '../../types';
import { marketApi } from '../../services/api';

interface StrategyState {
  items: Strategy[];
  currentStrategy: Strategy | null;
  currentHistory: number[];
  currentPositions: VaultPosition[];
  currentFills: VaultFill[];
  currentCreatedAt: string;
  loading: boolean;
  error: string | null;
}

const initialState: StrategyState = {
  items: [],
  currentStrategy: null,
  currentHistory: [],
  currentPositions: [],
  currentFills: [],
  currentCreatedAt: '',
  loading: false,
  error: null,
};

function calcRunningDays(startedAt?: string): number {
  if (!startedAt) return 0;
  const ts = new Date(startedAt).getTime();
  if (!Number.isFinite(ts) || ts <= 0) return 0;
  const diff = Date.now() - ts;
  if (diff <= 0) return 0;
  return Math.floor(diff / 86400000);
}

function mapAgentToStrategy(agent: AgentMarketItem): Strategy {
  const name = agent.name || agent.userName || (agent.publicKey ? agent.publicKey.slice(0, 10) + '...' : 'Unknown');

  return {
    id: agent.publicKey,
    name,
    description: agent.description || '',
    category: agent.category || 'trend',
    minInvestment: 0,
    riskLevel: 'medium',
    expectedReturn: 0,
    runningDays: calcRunningDays(agent.startedAt),
    userCount: 0,
    creator: agent.userName || '',
    creatorAvatar: agent.avatar || '',
    ownerUserId: agent.userId || '',
    currentTvl: agent.tvl || agent.accountValue || 0,
    maxTvl: 10000000,
    pnlContribution: agent.totalPnL || 0,
    performanceFee: agent.performanceFee ?? 0.2,
    performance: {
      daily: [],
      monthly: [],
    },
    parameters: {
      stopLoss: 5,
      takeProfit: 10,
      maxPosition: 1000,
    },
    rating: 0,
    reviews: [],
    status: 'active',
    vaultAddress: agent.vaultAddress,
    evmBalance: agent.evmBalance,
    agentStatus: agent.agentStatus,
    initialCapital: agent.initialCapital,
    lastSyncedAt: agent.lastSyncedAt,
    startedAt: agent.startedAt,
  };
}

export const fetchStrategies = createAsyncThunk(
  'strategies/fetchStrategies',
  async () => {
    const agents = await marketApi.getAgentMarket();
    return agents.map(mapAgentToStrategy);
  },
  {
    condition: (_, { getState }) => {
      const { strategies } = getState() as { strategies: StrategyState };
      if (strategies.loading) return false;
    },
  }
);

export const fetchStrategyById = createAsyncThunk(
  'strategies/fetchStrategyById',
  async ({ publicKey, period }: { publicKey: string; period?: string }) => {
    const detail = await marketApi.getAgentDetail(publicKey, period);
    const strategy = mapAgentToStrategy(detail.agent);

    // Use history for chart
    if (detail.history && detail.history.length > 0) {
      strategy.performance = {
        daily: detail.history.slice(-30),
        monthly: detail.history.slice(-12),
      };
    }

    return {
      strategy,
      history: detail.history || [],
      positions: detail.positions || [],
      recentFills: detail.recentFills || [],
      createdAt: detail.createdAt || '',
    };
  }
);

const strategySlice = createSlice({
  name: 'strategies',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchStrategies.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchStrategies.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
      })
      .addCase(fetchStrategies.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch strategies';
      })
      .addCase(fetchStrategyById.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchStrategyById.fulfilled, (state, action) => {
        state.loading = false;
        state.currentStrategy = action.payload.strategy;
        state.currentHistory = action.payload.history;
        state.currentPositions = action.payload.positions;
        state.currentFills = action.payload.recentFills;
        state.currentCreatedAt = action.payload.createdAt;
      })
      .addCase(fetchStrategyById.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch strategy';
      });
  },
});

export default strategySlice.reducer;
