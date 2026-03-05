<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { adminApi } from "../api/client";
import type { InviteCode } from "../types";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const loading = ref(true);
const error = ref("");
const success = ref("");
const codes = ref<InviteCode[]>([]);

const showGenModal = ref(false);
const genCount = ref(10);
const generating = ref(false);

const statusFilter = ref("");

const page = ref(1);
const pageSize = ref(20);

const filteredItems = computed(() => {
  if (!statusFilter.value) return codes.value;
  return codes.value.filter((c) => c.status === statusFilter.value);
});
const total = computed(() => filteredItems.value.length);
const totalPages = computed(() =>
  Math.max(1, Math.ceil(total.value / pageSize.value)),
);

const pagedItems = computed(() => {
  const s = (page.value - 1) * pageSize.value;
  return filteredItems.value.slice(s, s + pageSize.value);
});

watch(total, () => {
  if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value);
});

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
    codes.value = await adminApi.listInviteCodes(statusFilter.value);
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("inviteCodes.loadError");
  } finally {
    loading.value = false;
  }
};

const generate = async () => {
  if (generating.value || genCount.value < 1) return;
  generating.value = true;
  error.value = "";
  try {
    const r = await adminApi.generateInviteCodes(genCount.value);
    showGenModal.value = false;
    flash(t("inviteCodes.generateSuccess", { count: r.length }));
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("inviteCodes.generateError");
  } finally {
    generating.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>{{ t("inviteCodes.title") }}</h1>
      <p class="muted">{{ t("inviteCodes.desc") }}</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">
      {{ success }}
    </div>

    <div class="table-card">
      <div class="table-toolbar">
        <button
          class="toolbar-btn primary"
          @click="
            showGenModal = true;
            genCount = 10;
          "
        >
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
          {{ t("inviteCodes.generate") }}
        </button>
        <span class="toolbar-spacer"></span>
        <select
          class="toolbar-select"
          v-model="statusFilter"
          @change="
            page = 1;
            load();
          "
        >
          <option value="">{{ t("inviteCodes.allStatus") }}</option>
          <option value="unused">{{ t("inviteCodes.unused") }}</option>
          <option value="used">{{ t("inviteCodes.used") }}</option>
        </select>
        <span class="toolbar-info">{{
          t("common.total", { count: total })
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
                <th>{{ t("inviteCodes.code") }}</th>
                <th>{{ t("inviteCodes.status") }}</th>
                <th>{{ t("inviteCodes.usedBy") }}</th>
                <th>{{ t("inviteCodes.usedAt") }}</th>
                <th>{{ t("common.createdAt") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.code">
                <td class="monospace font-bold">{{ item.code }}</td>
                <td>
                  <span
                    class="badge"
                    :class="{
                      'badge-success': item.status === 'unused',
                      'badge-disabled': item.status === 'used',
                    }"
                  >
                    {{
                      item.status === "unused"
                        ? t("inviteCodes.unused")
                        : t("inviteCodes.used")
                    }}
                  </span>
                </td>
                <td>{{ item.usedByEmail || item.usedByUserId || "-" }}</td>
                <td class="small">{{ item.usedAt || "-" }}</td>
                <td class="small">{{ item.createdAt }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="filteredItems.length === 0" class="table-empty">
            {{ t("common.empty") }}
          </div>
        </template>
      </div>

      <div class="table-footer">
        <span>{{ t("common.total", { count: total }) }}</span>
        <div class="page-nav">
          <button :disabled="page <= 1" @click="page = 1">&laquo;</button>
          <button :disabled="page <= 1" @click="page--">&lsaquo;</button>
          <span style="font-size: 0.78rem; padding: 0 0.3rem"
            >{{ page }} / {{ totalPages }}</span
          >
          <button :disabled="page >= totalPages" @click="page++">
            &rsaquo;
          </button>
          <button :disabled="page >= totalPages" @click="page = totalPages">
            &raquo;
          </button>
        </div>
        <select
          v-model="pageSize"
          @change="page = 1"
          style="
            width: auto;
            padding: 0.2rem 0.3rem;
            font-size: 0.75rem;
            border-radius: 6px;
          "
        >
          <option :value="20">20 / {{ t("common.page") }}</option>
          <option :value="50">50 / {{ t("common.page") }}</option>
          <option :value="100">100 / {{ t("common.page") }}</option>
        </select>
      </div>
    </div>

    <!-- 生成邀请码弹窗 -->
    <div
      v-if="showGenModal"
      class="modal-overlay"
      @click.self="showGenModal = false"
    >
      <div class="modal modal-sm">
        <h3>{{ t("inviteCodes.generateModal") }}</h3>
        <p class="muted small">{{ t("inviteCodes.generateDesc") }}</p>
        <div class="form-group" style="margin-top: 0.5rem">
          <label>{{ t("inviteCodes.count") }}</label>
          <input v-model.number="genCount" type="number" min="1" max="100" />
        </div>
        <div class="form-actions">
          <button class="btn" @click="showGenModal = false">
            {{ t("common.cancel") }}
          </button>
          <button
            class="btn btn-primary"
            @click="generate"
            :disabled="generating"
          >
            {{
              generating
                ? t("inviteCodes.generatingBtn")
                : t("inviteCodes.generateBtn")
            }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
