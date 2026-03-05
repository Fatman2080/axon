import axios from 'axios';
import type { AdminUser, AgentAccount, AgentImportResult, AgentMarketItem, DashboardStats, InviteCode, User } from '../types';

const BASE_URL = import.meta.env.VITE_API_URL || '';

const http = axios.create({
  baseURL: BASE_URL,
});

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('admin_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const adminApi = {
  async login(email: string, password: string): Promise<{ token: string; admin: AdminUser }> {
    const { data } = await http.post('/admin/api/login', { email, password });
    return data;
  },

  async me(): Promise<AdminUser> {
    const { data } = await http.get('/admin/api/me');
    return data;
  },

  async listAdmins(): Promise<AdminUser[]> {
    const { data } = await http.get('/admin/api/admins');
    return data;
  },

  async createAdmin(payload: { email: string; name?: string; password: string }): Promise<AdminUser> {
    const { data } = await http.post('/admin/api/admins', payload);
    return data;
  },

  async updateAdminPassword(id: string, password: string): Promise<{ success: boolean }> {
    const { data } = await http.patch(`/admin/api/admins/${id}/password`, { password });
    return data;
  },

  async deleteAdmin(id: string): Promise<{ success: boolean }> {
    const { data } = await http.delete(`/admin/api/admins/${id}`);
    return data;
  },

  async dashboard(): Promise<DashboardStats> {
    const { data } = await http.get('/admin/api/dashboard');
    return data;
  },

  async listUsers(search = ''): Promise<User[]> {
    const { data } = await http.get('/admin/api/users', { params: { search } });
    return data;
  },

  async listAgentAccounts(status = ''): Promise<AgentAccount[]> {
    const { data } = await http.get('/admin/api/agent-accounts', { params: { status } });
    return data;
  },

  async importAgentAccounts(encryptedPayload: string): Promise<AgentImportResult> {
    const { data } = await http.post('/admin/api/agent-accounts/import', { encryptedPayload });
    return data;
  },

  async listAgentStats(search?: string): Promise<AgentMarketItem[]> {
    const { data } = await http.get('/admin/api/agent-stats', { params: { search } });
    return data;
  },

  async syncAgentData(publicKey: string): Promise<void> {
    await http.post(`/admin/api/agent-stats/${publicKey}/sync`);
  },

  async updateAgentProfile(publicKey: string, payload: { name?: string; description?: string; performanceFee?: number }): Promise<{ success: boolean }> {
    const { data } = await http.patch(`/admin/api/agent-stats/${publicKey}/profile`, payload);
    return data;
  },

  async listInviteCodes(): Promise<InviteCode[]> {
    const { data } = await http.get('/admin/api/invite-codes');
    return data;
  },

  async createInviteCode(payload: { code: string; description?: string; maxUses?: number; status?: string }): Promise<InviteCode> {
    const { data } = await http.post('/admin/api/invite-codes', payload);
    return data;
  },

  async createInviteCodeBatch(payload: {
    prefix?: string;
    length?: number;
    count: number;
    maxUses?: number;
    description?: string;
  }): Promise<InviteCode[]> {
    const { data } = await http.post('/admin/api/invite-codes/batch', payload);
    return data;
  },

  async updateInviteCode(id: string, payload: { description?: string; maxUses?: number; status?: string }): Promise<InviteCode> {
    const { data } = await http.patch(`/admin/api/invite-codes/${id}`, payload);
    return data;
  },

  async deleteInviteCodes(ids: string[]): Promise<{ deleted: number }> {
    const { data } = await http.delete('/admin/api/invite-codes', { data: { ids } });
    return data;
  },

  async deleteUsers(ids: string[], password: string): Promise<{ deleted: number }> {
    const { data } = await http.delete('/admin/api/users', { data: { ids, password } });
    return data;
  },

  async deleteAgents(publicKeys: string[], password: string): Promise<{ deleted: number }> {
    const { data } = await http.delete('/admin/api/agent-stats', { data: { publicKeys, password } });
    return data;
  },

  async deleteAgentPool(publicKeys: string[], password: string): Promise<{ deleted: number }> {
    const { data } = await http.delete('/admin/api/agent-accounts', { data: { publicKeys, password } });
    return data;
  },

  async exportUnusedInviteCodes(format: 'json' | 'csv' = 'json'): Promise<{ codes: string[] } | string> {
    const response = await http.get('/admin/api/invite-codes/unused/export', {
      params: { format },
      responseType: format === 'csv' ? 'text' : 'json',
    });
    return response.data;
  },

  async updateInviteCodeStatus(id: string, status: string): Promise<InviteCode> {
    const { data } = await http.patch(`/admin/api/invite-codes/${id}`, { status });
    return data;
  },

  async getSyncSettings(): Promise<{ intervalSeconds: number }> {
    const { data } = await http.get('/admin/api/settings/sync');
    return data;
  },

  async updateSyncSettings(intervalSeconds: number): Promise<{ intervalSeconds: number }> {
    const { data } = await http.patch('/admin/api/settings/sync', { intervalSeconds });
    return data;
  },

  async getXOAuthSettings(): Promise<{ clientId: string; clientSecret: string; scopes: string }> {
    const { data } = await http.get('/admin/api/settings/xoauth');
    return data;
  },

  async updateXOAuthSettings(payload: { clientId: string; clientSecret: string; scopes: string }): Promise<{ success: boolean }> {
    const { data } = await http.patch('/admin/api/settings/xoauth', payload);
    return data;
  },

  async getDailySlotsSettings(): Promise<{ total: number; consumed: number; remaining: number; resetHour: number; resetsAt: string }> {
    const { data } = await http.get('/admin/api/settings/daily-slots');
    return data;
  },

  async updateDailySlotsSettings(payload: { total?: number; resetHour?: number; resetConsumed?: boolean }): Promise<{ total: number; consumed: number; remaining: number; resetHour: number; resetsAt: string }> {
    const { data } = await http.patch('/admin/api/settings/daily-slots', payload);
    return data;
  },

  async getContractsSettings(): Promise<{ rpcURL: string; allocatorAddress: string }> {
    const { data } = await http.get('/admin/api/settings/contracts');
    return data;
  },

  async updateContractsSettings(payload: { rpcURL: string; allocatorAddress: string }): Promise<{ success: boolean }> {
    const { data } = await http.patch('/admin/api/settings/contracts', payload);
    return data;
  },
};

export default http;
