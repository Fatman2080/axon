import axios from 'axios';
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
    async login(email, password) {
        const { data } = await http.post('/admin/api/login', { email, password });
        return data;
    },
    async me() {
        const { data } = await http.get('/admin/api/me');
        return data;
    },
    async listAdmins() {
        const { data } = await http.get('/admin/api/admins');
        return data;
    },
    async createAdmin(payload) {
        const { data } = await http.post('/admin/api/admins', payload);
        return data;
    },
    async updateAdminPassword(id, password) {
        const { data } = await http.patch(`/admin/api/admins/${id}/password`, { password });
        return data;
    },
    async deleteAdmin(id) {
        const { data } = await http.delete(`/admin/api/admins/${id}`);
        return data;
    },
    async dashboard() {
        const { data } = await http.get('/admin/api/dashboard');
        return data;
    },
    async listUsers(search = '') {
        const { data } = await http.get('/admin/api/users', { params: { search } });
        return data;
    },
    async listAgentAccounts(status = '') {
        const { data } = await http.get('/admin/api/agent-accounts', { params: { status } });
        return data;
    },
    async importAgentAccounts(encryptedPayload) {
        const { data } = await http.post('/admin/api/agent-accounts/import', { encryptedPayload });
        return data;
    },
    async listAgentVaults() {
        const { data } = await http.get('/admin/api/agent-vaults');
        return data;
    },
    async deleteAgentVaults(payload) {
        const { data } = await http.delete('/admin/api/agent-vaults', { data: payload });
        return data;
    },
    async updateAgentProfile(publicKey, payload) {
        const { data } = await http.patch(`/admin/api/agent-accounts/${publicKey}/profile`, payload);
        return data;
    },
    async listInviteCodes(status) {
        const { data } = await http.get('/admin/api/invite-codes', { params: status ? { status } : {} });
        return data;
    },
    async generateInviteCodes(count) {
        const { data } = await http.post('/admin/api/invite-codes/batch', { count });
        return data;
    },
    async createInviteCode(payload) {
        const { data } = await http.post('/admin/api/invite-codes', payload);
        return data;
    },
    async createInviteCodeBatch(payload) {
        const { data } = await http.post('/admin/api/invite-codes/batch', payload);
        return data;
    },
    async updateInviteCode(id, payload) {
        const { data } = await http.patch(`/admin/api/invite-codes/${id}`, payload);
        return data;
    },
    async deleteInviteCodes(ids) {
        const { data } = await http.delete('/admin/api/invite-codes', { data: { ids } });
        return data;
    },
    async deleteUsers(ids, password) {
        const { data } = await http.delete('/admin/api/users', { data: { ids, password } });
        return data;
    },
    async deleteAgentPool(publicKeys, password) {
        const { data } = await http.delete('/admin/api/agent-accounts', { data: { publicKeys, password } });
        return data;
    },
    async exportUnusedInviteCodes(format = 'json') {
        const response = await http.get('/admin/api/invite-codes/unused/export', {
            params: { format },
            responseType: format === 'csv' ? 'text' : 'json',
        });
        return response.data;
    },
    async updateInviteCodeStatus(id, status) {
        const { data } = await http.patch(`/admin/api/invite-codes/${id}`, { status });
        return data;
    },
    async getSyncSettings() {
        const { data } = await http.get('/admin/api/settings/sync');
        return data;
    },
    async updateSyncSettings(payload) {
        const { data } = await http.patch('/admin/api/settings/sync', payload);
        return data;
    },
    async getXOAuthSettings() {
        const { data } = await http.get('/admin/api/settings/xoauth');
        return data;
    },
    async updateXOAuthSettings(payload) {
        const { data } = await http.patch('/admin/api/settings/xoauth', payload);
        return data;
    },
    async getDailySlotsSettings() {
        const { data } = await http.get('/admin/api/settings/daily-slots');
        return data;
    },
    async updateDailySlotsSettings(payload) {
        const { data } = await http.patch('/admin/api/settings/daily-slots', payload);
        return data;
    },
    async getContractsSettings() {
        const { data } = await http.get('/admin/api/settings/contracts');
        return data;
    },
    async updateContractsSettings(payload) {
        const { data } = await http.patch('/admin/api/settings/contracts', payload);
        return data;
    },
    async getDispatchSettings() {
        const { data } = await http.get('/admin/api/settings/dispatch');
        return data;
    },
    async updateDispatchSettings(command) {
        const { data } = await http.patch('/admin/api/settings/dispatch', { command });
        return data;
    },
    async dispatchAgent(publicKey) {
        const { data } = await http.post(`/admin/api/agent-accounts/${publicKey}/dispatch`);
        return data;
    },
    async getAgentPrivateKey(publicKey, password) {
        const { data } = await http.post(`/admin/api/agent-accounts/${publicKey}/privatekey`, { password });
        return data;
    },
    async revokeAgent(publicKey) {
        const { data } = await http.post(`/admin/api/agent-accounts/${publicKey}/revoke`);
        return data;
    },
    async reassignAgent(publicKey, userName) {
        const { data } = await http.post(`/admin/api/agent-accounts/${publicKey}/reassign`, { userName });
        return data;
    },
    async revokeUserInvite(userId) {
        const { data } = await http.post(`/admin/api/users/${userId}/revoke-invite`);
        return data;
    },
    async revokeUserAgent(userId) {
        const { data } = await http.post(`/admin/api/users/${userId}/revoke-agent`);
        return data;
    },
    async treasury() {
        const { data } = await http.get('/admin/api/treasury');
        return data;
    },
    async treasuryHistory(period = '7d', limit = 200) {
        const { data } = await http.get('/admin/api/treasury/history', { params: { period, limit } });
        return data;
    },
    async dashboardTrends(period = '7d') {
        const { data } = await http.get('/admin/api/dashboard/trends', { params: { period } });
        return data;
    },
    async agentPerformance(publicKey) {
        const { data } = await http.get(`/admin/api/agents/${publicKey}/performance`);
        return data;
    },
    async agentLeaderboard(sortBy = 'pnl', limit = 10) {
        const { data } = await http.get('/admin/api/agents/leaderboard', { params: { sortBy, limit } });
        return data;
    },
    async getBackupSettings() {
        const { data } = await http.get('/admin/api/settings/backup');
        return data;
    },
    async updateBackupSettings(payload) {
        const { data } = await http.patch('/admin/api/settings/backup', payload);
        return data;
    },
    async listBackups() {
        const { data } = await http.get('/admin/api/backups');
        return data;
    },
    async createBackup() {
        const { data } = await http.post('/admin/api/backups');
        return data;
    },
    async restoreBackup(name, password) {
        const { data } = await http.post('/admin/api/backups/restore', { name, password });
        return data;
    },
};
export default http;
