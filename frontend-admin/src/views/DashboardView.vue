<script setup lang="ts">
import { onMounted, ref } from "vue";
import { adminApi } from "../api/client";
import type { DashboardStats } from "../types";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const loading = ref(true);
const stats = ref<DashboardStats | null>(null);
const error = ref("");

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    stats.value = await adminApi.dashboard();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("dashboard.loadError");
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page-scroll">
    <div class="page-header">
      <h1>{{ t("dashboard.title") }}</h1>
      <p class="muted">{{ t("dashboard.desc") }}</p>
    </div>

    <div v-if="loading" class="table-empty">{{ t("common.loading") }}</div>
    <div v-else-if="error" class="panel error">{{ error }}</div>
    <div v-else-if="stats" class="stats-grid">
      <article class="panel">
        <h3>{{ t("dashboard.totalUsers") }}</h3>
        <p>{{ stats.totalUsers }}</p>
      </article>
      <article class="panel">
        <h3>{{ t("dashboard.totalAgentAccounts") }}</h3>
        <p>{{ stats.totalAgentAccounts }}</p>
      </article>
      <article class="panel">
        <h3>{{ t("dashboard.assignedAgents") }}</h3>
        <p>{{ stats.assignedAgents }}</p>
      </article>
      <article class="panel">
        <h3>{{ t("dashboard.unusedAgents") }}</h3>
        <p>{{ stats.unusedAgents }}</p>
      </article>
      <article class="panel">
        <h3>{{ t("dashboard.totalInviteCodes") }}</h3>
        <p>{{ stats.totalInviteCodes }}</p>
      </article>
      <article class="panel">
        <h3>{{ t("dashboard.activeInviteCodes") }}</h3>
        <p>{{ stats.activeInviteCodes }}</p>
      </article>
    </div>
  </section>
</template>
