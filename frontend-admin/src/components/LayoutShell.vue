<script setup lang="ts">
import { useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";

const router = useRouter();
const auth = useAuthStore();

const logout = async () => {
  auth.logout();
  await router.push("/login");
};
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <h2>OpenFi 管理后台</h2>
      </div>
      <nav>
        <RouterLink to="/dashboard" class="menu-item">仪表盘</RouterLink>
        <RouterLink to="/admins" class="menu-item">管理员</RouterLink>
        <RouterLink to="/users" class="menu-item">用户</RouterLink>
        <RouterLink to="/agent-accounts" class="menu-item">AgentVault 用户</RouterLink>
        <RouterLink to="/agent-vaults" class="menu-item">AgentVault 合约</RouterLink>
        <RouterLink to="/invite-codes" class="menu-item">邀请码</RouterLink>
        <RouterLink to="/settings" class="menu-item">系统配置</RouterLink>
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
