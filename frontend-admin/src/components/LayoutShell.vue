<script setup lang="ts">
import { useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";
import { useI18n } from "vue-i18n";

const { t, locale } = useI18n();
const router = useRouter();
const auth = useAuthStore();

const toggleLanguage = () => {
  locale.value = locale.value === "zh" ? "en" : "zh";
};

const logout = async () => {
  auth.logout();
  await router.push("/login");
};
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <h2>{{ t("layout.title") }}</h2>
      </div>
      <nav>
        <RouterLink to="/dashboard" class="menu-item">{{
          t("menu.dashboard")
        }}</RouterLink>
        <RouterLink to="/admins" class="menu-item">{{
          t("menu.admins")
        }}</RouterLink>
        <RouterLink to="/users" class="menu-item">{{
          t("menu.users")
        }}</RouterLink>
        <RouterLink to="/agent-pool" class="menu-item">{{
          t("menu.agentPool")
        }}</RouterLink>
        <RouterLink to="/agent-stats" class="menu-item">{{
          t("menu.agentStats")
        }}</RouterLink>
        <RouterLink to="/invite-codes" class="menu-item">{{
          t("menu.inviteCodes")
        }}</RouterLink>
        <RouterLink to="/settings" class="menu-item">{{
          t("menu.settings")
        }}</RouterLink>
      </nav>
      <div class="sidebar-footer">
        <button
          class="btn btn-sm"
          style="margin-bottom: 0.5rem"
          @click="toggleLanguage"
        >
          {{ locale === "zh" ? "EN" : "中" }}
        </button>
        <p class="small muted">{{ auth.admin?.email }}</p>
        <button class="btn" @click="logout">{{ t("layout.logout") }}</button>
      </div>
    </aside>
    <main class="content">
      <slot />
    </main>
  </div>
</template>
