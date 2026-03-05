<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '../api/client';
import type { DashboardStats } from '../types';

const loading = ref(true);
const stats = ref<DashboardStats | null>(null);
const error = ref('');

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    stats.value = await adminApi.dashboard();
  } catch (err: any) {
    error.value = err?.response?.data?.error || '加载仪表盘数据失败';
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page-scroll">
    <div class="page-header">
      <h1>仪表盘</h1>
      <p class="muted">Agent 账户分配和用户注册概览。</p>
    </div>

    <div v-if="loading" class="table-empty">加载中...</div>
    <div v-else-if="error" class="panel error">{{ error }}</div>
    <div v-else-if="stats" class="stats-grid">
      <article class="panel">
        <h3>总用户数</h3>
        <p>{{ stats.totalUsers }}</p>
      </article>
      <article class="panel">
        <h3>总 Agent 账户</h3>
        <p>{{ stats.totalAgentAccounts }}</p>
      </article>
      <article class="panel">
        <h3>已分配 Agent</h3>
        <p>{{ stats.assignedAgents }}</p>
      </article>
      <article class="panel">
        <h3>未使用 Agent</h3>
        <p>{{ stats.unusedAgents }}</p>
      </article>
      <article class="panel">
        <h3>总邀请码</h3>
        <p>{{ stats.totalInviteCodes }}</p>
      </article>
      <article class="panel">
        <h3>有效邀请码</h3>
        <p>{{ stats.activeInviteCodes }}</p>
      </article>
    </div>
  </section>
</template>
