import { defineStore } from 'pinia';
import { adminApi } from '../api/client';
export const useAuthStore = defineStore('auth', {
    state: () => ({
        token: localStorage.getItem('admin_token') || '',
        admin: null,
        loading: false,
    }),
    getters: {
        isLoggedIn: (state) => Boolean(state.token),
    },
    actions: {
        async login(email, password) {
            this.loading = true;
            try {
                const response = await adminApi.login(email, password);
                this.token = response.token;
                this.admin = response.admin;
                localStorage.setItem('admin_token', response.token);
            }
            finally {
                this.loading = false;
            }
        },
        async fetchMe() {
            if (!this.token)
                return;
            this.loading = true;
            try {
                this.admin = await adminApi.me();
            }
            catch {
                this.logout();
            }
            finally {
                this.loading = false;
            }
        },
        logout() {
            this.token = '';
            this.admin = null;
            localStorage.removeItem('admin_token');
        },
    },
});
