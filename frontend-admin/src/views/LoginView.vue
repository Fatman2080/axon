<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const router = useRouter();
const auth = useAuthStore();

const email = ref("admin@openfi.local");
const password = ref("");
const error = ref("");

const submit = async () => {
  error.value = "";
  try {
    await auth.login(email.value, password.value);
    await router.push("/dashboard");
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("login.error");
  }
};
</script>

<template>
  <div class="login-page">
    <form class="panel login-panel" @submit.prevent="submit">
      <h1>{{ t("login.title") }}</h1>
      <p class="muted">{{ t("login.desc") }}</p>

      <label>{{ t("common.email") }}</label>
      <input v-model="email" type="email" required />

      <label>{{ t("common.password") }}</label>
      <input v-model="password" type="password" required />

      <p v-if="error" class="error">{{ error }}</p>
      <button class="btn btn-primary" :disabled="auth.loading">
        {{ auth.loading ? t("login.loading") : t("login.submit") }}
      </button>
    </form>
  </div>
</template>
