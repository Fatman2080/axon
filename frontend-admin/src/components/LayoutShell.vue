<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const router = useRouter();
const auth = useAuthStore();

const menu = [
  { path: '/dashboard', label: '仪表盘' },
  { path: '/admins', label: '管理员' },
  { path: '/users', label: '用户' },
  { path: '/agent-pool', label: 'Agent 池' },
  { path: '/agent-stats', label: 'Agent 统计' },
  { path: '/invite-codes', label: '邀请码' },
  { path: '/settings', label: '系统配置' },
];

const logout = async () => {
  auth.logout();
  await router.push('/login');
};
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <h2>OpenFi 管理后台</h2>
      </div>
      <nav>
        <RouterLink v-for="item in menu" :key="item.path" :to="item.path" class="menu-item">
          {{ item.label }}
        </RouterLink>
      </nav>
      <div class="sidebar-footer">
        <p class="small muted">{{ auth.admin?.email }}</p>
        <button class="btn" @click="logout">退出登录</button>
      </div>
    </aside>
    <main class="content">
      <slot />
    </main>
  </div>
</template>
