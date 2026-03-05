<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { adminApi } from "../api/client";
import { useAuthStore } from "../stores/auth";
import type { AdminUser } from "../types";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const auth = useAuthStore();
const loading = ref(true);
const creating = ref(false);
const error = ref("");
const success = ref("");
const admins = ref<AdminUser[]>([]);

const showCreateModal = ref(false);
const createForm = ref({ email: "", name: "", password: "" });

const currentAdminId = computed(() => auth.admin?.id || "");

const flash = (msg: string) => {
  success.value = msg;
  setTimeout(() => {
    if (success.value === msg) success.value = "";
  }, 4000);
};

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    admins.value = await adminApi.listAdmins();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("admins.loadError");
  } finally {
    loading.value = false;
  }
};

const openCreate = () => {
  createForm.value = { email: "", name: "", password: "" };
  showCreateModal.value = true;
};

const createAdmin = async () => {
  if (creating.value) return;
  creating.value = true;
  error.value = "";
  try {
    const created = await adminApi.createAdmin({
      email: createForm.value.email.trim(),
      name: createForm.value.name.trim(),
      password: createForm.value.password,
    });
    showCreateModal.value = false;
    flash(t("admins.createSuccess", { email: created.email }));
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("admins.createError");
  } finally {
    creating.value = false;
  }
};

const resetPassword = async (admin: AdminUser) => {
  const next = window.prompt(
    t("admins.promptNewPassword", { email: admin.email }),
    "",
  );
  if (!next) return;
  if (next.trim().length < 6) {
    error.value = t("admins.passwordLength");
    return;
  }
  error.value = "";
  try {
    await adminApi.updateAdminPassword(admin.id, next.trim());
    flash(t("admins.resetSuccess", { email: admin.email }));
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("admins.resetError");
  }
};

const deleteAdmin = async (admin: AdminUser) => {
  if (!window.confirm(t("admins.confirmDeletePrompt", { email: admin.email })))
    return;
  error.value = "";
  try {
    await adminApi.deleteAdmin(admin.id);
    flash(t("admins.deleteSuccess", { email: admin.email }));
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("admins.deleteError");
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>{{ t("admins.title") }}</h1>
      <p class="muted">{{ t("admins.desc") }}</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">
      {{ success }}
    </div>

    <div class="table-card">
      <div class="table-toolbar">
        <button class="toolbar-btn primary" @click="openCreate">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <path d="M5 12h14" />
            <path d="M12 5v14" />
          </svg>
          {{ t("admins.add") }}
        </button>
        <span class="toolbar-spacer"></span>
        <span class="toolbar-info">{{
          t("common.total", { count: admins.length })
        }}</span>
        <button class="toolbar-icon" @click="load" :title="t('common.refresh')">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
            <path d="M21 3v5h-5" />
            <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
            <path d="M8 16H3v5" />
          </svg>
        </button>
      </div>
      <div class="table-body">
        <div v-if="loading" class="table-empty">{{ t("common.loading") }}</div>
        <div v-else-if="error" class="table-empty error">{{ error }}</div>
        <template v-else>
          <table>
            <thead>
              <tr>
                <th>{{ t("common.email") }}</th>
                <th>{{ t("common.name") }}</th>
                <th>{{ t("common.createdAt") }}</th>
                <th>{{ t("common.actions") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in admins" :key="item.id">
                <td>{{ item.email }}</td>
                <td>{{ item.name }}</td>
                <td class="small">{{ item.createdAt }}</td>
                <td class="actions">
                  <button class="btn btn-sm" @click="resetPassword(item)">
                    {{ t("admins.resetPassword") }}
                  </button>
                  <button
                    class="btn btn-sm btn-danger"
                    :disabled="item.id === currentAdminId"
                    @click="deleteAdmin(item)"
                  >
                    {{ t("common.delete") }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="admins.length === 0" class="table-empty">
            {{ t("common.empty") }}
          </div>
        </template>
      </div>
    </div>

    <div
      v-if="showCreateModal"
      class="modal-overlay"
      @click.self="showCreateModal = false"
    >
      <div class="modal modal-sm">
        <h3>{{ t("admins.createModalTitle") }}</h3>
        <div class="form-group">
          <label>{{ t("common.email") }}</label>
          <input
            v-model="createForm.email"
            type="email"
            :placeholder="t('admins.emailPlaceholder')"
          />
        </div>
        <div class="form-group">
          <label>{{ t("common.name") }}</label>
          <input
            v-model="createForm.name"
            :placeholder="t('admins.namePlaceholder')"
          />
        </div>
        <div class="form-group">
          <label>{{ t("common.password") }}</label>
          <input
            v-model="createForm.password"
            type="password"
            :placeholder="t('admins.passwordLength')"
            @keyup.enter="createAdmin"
          />
        </div>
        <div class="form-actions">
          <button class="btn" @click="showCreateModal = false">
            {{ t("common.cancel") }}
          </button>
          <button
            class="btn btn-primary"
            @click="createAdmin"
            :disabled="creating"
          >
            {{ creating ? t("admins.creatingBtn") : t("admins.createBtn") }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
