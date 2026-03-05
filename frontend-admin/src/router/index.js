import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import LoginView from '../views/LoginView.vue';
import DashboardView from '../views/DashboardView.vue';
import UsersView from '../views/UsersView.vue';
import AdminsView from '../views/AdminsView.vue';
import AgentPoolView from '../views/AgentPoolView.vue';
import AgentStatsView from '../views/AgentStatsView.vue';
import InviteCodesView from '../views/InviteCodesView.vue';
import SettingsView from '../views/SettingsView.vue';
const router = createRouter({
    history: createWebHistory('/admin/'),
    routes: [
        { path: '/login', name: 'login', component: LoginView },
        { path: '/', redirect: '/dashboard' },
        { path: '/dashboard', name: 'dashboard', component: DashboardView },
        { path: '/admins', name: 'admins', component: AdminsView },
        { path: '/users', name: 'users', component: UsersView },
        { path: '/agent-pool', name: 'agent-pool', component: AgentPoolView },
        { path: '/agent-stats', name: 'agent-stats', component: AgentStatsView },
        { path: '/invite-codes', name: 'invite-codes', component: InviteCodesView },
        { path: '/settings', name: 'settings', component: SettingsView },
    ],
});
router.beforeEach(async (to) => {
    const auth = useAuthStore();
    if (auth.token && !auth.admin) {
        await auth.fetchMe();
    }
    if (to.path !== '/login' && !auth.isLoggedIn) {
        return '/login';
    }
    if (to.path === '/login' && auth.isLoggedIn) {
        return '/dashboard';
    }
    return true;
});
export default router;
