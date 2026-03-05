import axios from 'axios';
import type { AgentMarketItem, AgentMarketDetail, VaultStats, DailySlotsResponse, VaultOverview, UserAgentStats } from '../types';

const API_URL = '';

const api = axios.create({
  baseURL: '/api',
});

// Add token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const authApi = {
  getXOAuthStartUrl: (inviteCode?: string, nextPath?: string) => {
    const params = new URLSearchParams();
    if (inviteCode?.trim()) {
      params.set('inviteCode', inviteCode.trim());
    }
    if (nextPath?.trim()) {
      params.set('next', nextPath.trim());
    }
    const query = params.toString();
    return `/api/auth/x/start${query ? `?${query}` : ''}`;
  },

  logout: () => {
    localStorage.removeItem('token');
  },

  getMe: async () => {
    const response = await api.get('/user/me');
    return response.data;
  },

  consumeInviteCode: async (code: string) => {
    const response = await api.post('/invite-codes/consume', { code });
    return response.data;
  },

  getAgentHistory: async (period?: string) => {
    const response = await api.get('/user/agent/history', { params: { period } });
    return response.data;
  },

  getAgentStats: async (): Promise<UserAgentStats> => {
    const response = await api.get('/user/agent/stats');
    return response.data;
  }
};

export const marketApi = {
  getAgentMarket: async (search?: string): Promise<AgentMarketItem[]> => {
    const response = await api.get('/agent-market', { params: { search } });
    return response.data;
  },

  getAgentDetail: async (publicKey: string, period?: string): Promise<AgentMarketDetail> => {
    const response = await api.get(`/agent-market/${publicKey}`, { params: { period } });
    return response.data;
  },

  getVaultStats: async (): Promise<VaultStats> => {
    const response = await api.get('/vault/stats');
    return response.data;
  },

  getDailySlots: async (): Promise<DailySlotsResponse> => {
    const response = await api.get('/daily-slots');
    return response.data;
  },

  getVaultOverview: async (): Promise<VaultOverview> => {
    const response = await api.get('/vault/overview');
    return response.data;
  },
};

export default api;
