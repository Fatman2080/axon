import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { Agent, AgentMarketItem } from '../../types';
import { marketApi } from '../../services/api';

interface AgentState {
  items: Agent[];
  loading: boolean;
  error: string | null;
}

const initialState: AgentState = {
  items: [],
  loading: false,
  error: null,
};

function mapMarketItemToAgent(item: AgentMarketItem): Agent {
  return {
    id: item.publicKey,
    name: item.name || item.publicKey.slice(0, 10) + '...',
    strategyId: '',
    userId: item.userId || '',
    publicKey: item.publicKey,
    status: item.agentStatus === 'active' ? 'running' : 'stopped',
    investment: item.tvl || item.accountValue || 0,
    currentValue: item.accountValue || 0,
    totalProfit: item.totalPnL || 0,
    todayProfit: 0,
    createdAt: '',
    lastTrade: item.lastSyncedAt || '',
    trades: [],
    config: {
      tradingPairs: [],
      riskLevel: 5,
      autoReinvest: false,
    },
  };
}

export const fetchAgents = createAsyncThunk(
  'agents/fetchAgents',
  async () => {
    const items = await marketApi.getAgentMarket();
    return items.map(mapMarketItemToAgent);
  }
);

const agentSlice = createSlice({
  name: 'agents',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchAgents.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchAgents.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
      })
      .addCase(fetchAgents.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch agents';
      });
  },
});

export default agentSlice.reducer;
