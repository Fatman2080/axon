<script setup lang="ts">
import { computed, onMounted, ref, reactive, watch } from "vue";
import { adminApi } from "../api/client";
import type { AgentMarketItem } from "../types";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const loading = ref(true);
const error = ref("");
const success = ref("");
const items = ref<AgentMarketItem[]>([]);
const syncingKeys = ref(new Set<string>());
const syncingAll = ref(false);
const selectedKeys = ref<string[]>([]);
const deleting = ref(false);

const searchQuery = ref("");
const showSearchModal = ref(false);
const searchInput = ref("");

const page = ref(1);
const pageSize = ref(20);
const total = computed(() => items.value.length);
const totalPages = computed(() =>
  Math.max(1, Math.ceil(total.value / pageSize.value)),
);
const pagedItems = computed(() => {
  const s = (page.value - 1) * pageSize.value;
  return items.value.slice(s, s + pageSize.value);
});
watch(total, () => {
  if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value);
});

const allOnPageSelected = computed(
  () =>
    pagedItems.value.length > 0 &&
    pagedItems.value.every((i) => selectedKeys.value.includes(i.publicKey)),
);
const toggleSelectAll = () => {
  const keys = pagedItems.value.map((i) => i.publicKey);
  if (allOnPageSelected.value) {
    selectedKeys.value = selectedKeys.value.filter((k) => !keys.includes(k));
  } else {
    selectedKeys.value = [...new Set([...selectedKeys.value, ...keys])];
  }
};

const editingKey = ref("");
const editForm = reactive({ name: "", description: "", performanceFee: 0.2 });
const editSaving = ref(false);

const showPasswordModal = ref(false);
const passwordInput = ref("");
const passwordError = ref("");

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
    items.value = await adminApi.listAgentStats(searchQuery.value || undefined);
    const valid = new Set(items.value.map((i) => i.publicKey));
    selectedKeys.value = selectedKeys.value.filter((k) => valid.has(k));
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("agentStats.loadError");
  } finally {
    loading.value = false;
  }
};

const applySearch = async () => {
  searchQuery.value = searchInput.value.trim();
  showSearchModal.value = false;
  page.value = 1;
  await load();
};
const clearSearch = async () => {
  searchQuery.value = "";
  searchInput.value = "";
  page.value = 1;
  await load();
};

const syncAgent = async (pk: string) => {
  syncingKeys.value.add(pk);
  try {
    await adminApi.syncAgentData(pk);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("agentStats.syncError");
  } finally {
    syncingKeys.value.delete(pk);
  }
};

const syncAll = async () => {
  syncingAll.value = true;
  error.value = "";
  for (const item of items.value) {
    syncingKeys.value.add(item.publicKey);
    try {
      await adminApi.syncAgentData(item.publicKey);
    } catch {
      /* continue */
    } finally {
      syncingKeys.value.delete(item.publicKey);
    }
  }
  syncingAll.value = false;
  await load();
};

const startEdit = (item: AgentMarketItem) => {
  editingKey.value = item.publicKey;
  editForm.name = item.name || "";
  editForm.description = item.description || "";
  editForm.performanceFee = item.performanceFee ?? 0.2;
};

const saveEdit = async () => {
  if (!editingKey.value) return;
  editSaving.value = true;
  try {
    await adminApi.updateAgentProfile(editingKey.value, {
      name: editForm.name,
      description: editForm.description,
      performanceFee: editForm.performanceFee,
    });
    editingKey.value = "";
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || t("agentStats.updateError");
  } finally {
    editSaving.value = false;
  }
};

const promptDelete = () => {
  if (!selectedKeys.value.length) return;
  passwordInput.value = "";
  passwordError.value = "";
  showPasswordModal.value = true;
};

const confirmDelete = async () => {
  if (!passwordInput.value.trim()) {
    passwordError.value = t("common.password");
    return;
  }
  deleting.value = true;
  passwordError.value = "";
  try {
    const r = await adminApi.deleteAgents(
      selectedKeys.value,
      passwordInput.value,
    );
    flash(t("agentPool.deleteSuccess", { count: r.deleted }));
    selectedKeys.value = [];
    showPasswordModal.value = false;
    passwordInput.value = "";
    await load();
  } catch (err: any) {
    const msg = err?.response?.data?.error || t("common.error");
    passwordError.value =
      msg === "invalid_password" ? t("common.password") : msg;
  } finally {
    deleting.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>{{ t("agentStats.title") }}</h1>
      <p class="muted">{{ t("agentStats.desc") }}</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">
      {{ success }}
    </div>

    <div class="table-card">
      <div class="table-toolbar">
        <input
          type="checkbox"
          :checked="allOnPageSelected && pagedItems.length > 0"
          @change="toggleSelectAll"
        />
        <button
          class="toolbar-btn"
          @click="
            searchInput = searchQuery;
            showSearchModal = true;
          "
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          {{ t("common.search") }}
        </button>
        <span v-if="searchQuery" class="search-tag"
          >{{ searchQuery
          }}<button class="tag-close" @click="clearSearch">
            &times;
          </button></span
        >
        <button
          class="toolbar-btn"
          @click="syncAll"
          :disabled="syncingAll || items.length === 0"
        >
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
          {{ syncingAll ? t("common.syncing") : t("common.syncAll") }}
        </button>
        <button
          class="toolbar-btn danger"
          :disabled="!selectedKeys.length || deleting"
          @click="promptDelete"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <path d="M3 6h18" />
            <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
            <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
          </svg>
          {{ t("common.delete")
          }}{{ selectedKeys.length ? ` (${selectedKeys.length})` : "" }}
        </button>
        <span class="toolbar-spacer"></span>
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
                <th style="width: 36px"></th>
                <th>{{ t("agentPool.publicKey") }}</th>
                <th>{{ t("common.name") }}</th>
                <th>{{ t("menu.users") }}</th>
                <th>{{ t("agentStats.accountValue") }}</th>
                <th>{{ t("agentStats.totalPnl") }}</th>
                <th>{{ t("agentStats.lastSynced") }}</th>
                <th>{{ t("common.actions") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.publicKey">
                <td>
                  <input
                    type="checkbox"
                    :value="item.publicKey"
                    v-model="selectedKeys"
                  />
                </td>
                <td class="small monospace">
                  {{ item.publicKey.slice(0, 10) }}...{{
                    item.publicKey.slice(-6)
                  }}
                </td>
                <td>{{ item.name || "-" }}</td>
                <td>{{ item.userName || item.userId || "-" }}</td>
                <td>${{ item.accountValue?.toLocaleString() || "0" }}</td>
                <td
                  :class="{
                    'profit-up': (item.totalPnL || 0) >= 0,
                    'profit-down': (item.totalPnL || 0) < 0,
                  }"
                >
                  {{ (item.totalPnL || 0) >= 0 ? "+" : "" }}${{
                    (item.totalPnL || 0).toLocaleString()
                  }}
                </td>
                <td class="small">
                  {{ item.lastSyncedAt || t("agentStats.neverSynced") }}
                </td>
                <td class="actions">
                  <button
                    class="btn btn-sm"
                    @click="syncAgent(item.publicKey)"
                    :disabled="syncingKeys.has(item.publicKey)"
                  >
                    {{
                      syncingKeys.has(item.publicKey) ? "..." : t("common.sync")
                    }}
                  </button>
                  <button class="btn btn-sm" @click="startEdit(item)">
                    {{ t("common.edit") }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="items.length === 0" class="table-empty">
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

    <!-- 搜索弹窗 -->
    <div
      v-if="showSearchModal"
      class="modal-overlay"
      @click.self="showSearchModal = false"
    >
      <div class="modal modal-sm">
        <h3>{{ t("common.search") }}</h3>
        <div class="form-group">
          <input
            v-model="searchInput"
            placeholder="Search..."
            @keyup.enter="applySearch"
            autofocus
          />
        </div>
        <div class="form-actions">
          <button
            class="btn"
            @click="
              searchInput = '';
              applySearch();
            "
          >
            {{ t("common.clear") }}
          </button>
          <button class="btn btn-primary" @click="applySearch">
            {{ t("common.search") }}
          </button>
        </div>
      </div>
    </div>

    <!-- 编辑资料弹窗 -->
    <div v-if="editingKey" class="modal-overlay" @click.self="editingKey = ''">
      <div class="modal">
        <h3>{{ t("agentStats.editTitle") }}</h3>
        <p class="muted small monospace">{{ editingKey }}</p>
        <div class="form-group" style="margin-top: 0.5rem">
          <label>{{ t("common.name") }}</label>
          <input v-model="editForm.name" />
        </div>
        <div class="form-group">
          <label>{{ t("agentStats.description") }}</label>
          <textarea v-model="editForm.description" rows="3"></textarea>
        </div>
        <div class="form-group">
          <label>{{ t("agentStats.performanceFee") }}</label>
          <input
            v-model.number="editForm.performanceFee"
            type="number"
            step="0.01"
            min="0"
            max="1"
            placeholder="0.2"
          />
        </div>
        <div class="form-actions">
          <button class="btn" @click="editingKey = ''">
            {{ t("common.cancel") }}
          </button>
          <button
            class="btn btn-primary"
            @click="saveEdit"
            :disabled="editSaving"
          >
            {{ editSaving ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </div>
    </div>

    <!-- 密码确认弹窗 -->
    <div
      v-if="showPasswordModal"
      class="modal-overlay"
      @click.self="showPasswordModal = false"
    >
      <div class="modal modal-sm">
        <h3>{{ t("agentPool.deleteModalTitle") }}</h3>
        <p class="muted small">
          {{ t("agentStats.deleteModalDesc", { count: selectedKeys.length }) }}
        </p>
        <div class="form-group">
          <label>{{ t("common.password") }}</label>
          <input
            v-model="passwordInput"
            type="password"
            @keyup.enter="confirmDelete"
          />
        </div>
        <div v-if="passwordError" class="field-error">{{ passwordError }}</div>
        <div class="form-actions">
          <button class="btn" @click="showPasswordModal = false">
            {{ t("common.cancel") }}
          </button>
          <button
            class="btn btn-danger"
            @click="confirmDelete"
            :disabled="deleting"
          >
            {{ deleting ? t("common.deleting") : t("common.confirmDelete") }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
