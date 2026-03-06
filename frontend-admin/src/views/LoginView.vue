<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";

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
    error.value = err?.response?.data?.error || "登录失败，请检查凭据";
  }
};
</script>

<template>
  <div class="login-page">
    <form class="panel login-panel" @submit.prevent="submit">
      <h1>管理后台登录</h1>

      <label>邮箱</label>
      <input v-model="email" type="email" required />

      <label>密码</label>
      <input v-model="password" type="password" required />

      <p v-if="error" class="error">{{ error }}</p>
      <button class="btn btn-primary" :disabled="auth.loading">
        {{ auth.loading ? "登录中..." : "进入控制台" }}
      </button>
    </form>
  </div>
</template>
