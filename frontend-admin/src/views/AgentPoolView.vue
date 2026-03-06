<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { adminApi } from "../api/client";
import type { AgentAccount, AgentImportResult, VaultRecord } from "../types";

const loading = ref(true);
const error = ref("");
const success = ref("");
const accounts = ref<AgentAccount[]>([]);
const vaultMap = ref<Record<string, VaultRecord>>({});
const statusFilter = ref("");
const selectedKeys = ref<string[]>([]);
const deleting = ref(false);

const searchQuery = ref("");
const showSearchModal = ref(false);
const searchInput = ref("");

const showImportModal = ref(false);
const encryptedPayload = ref("");
const importResult = ref<AgentImportResult | null>(null);
const importing = ref(false);

const showPasswordModal = ref(false);
const passwordInput = ref("");
const passwordError = ref("");

const showReassignModal = ref(false);
const reassignTarget = ref("");
const reassignUserName = ref("");
const reassigning = ref(false);
const revoking = ref<string | null>(null);

const page = ref(1);
const pageSize = ref(20);

const filteredItems = computed(() => {
  if (!searchQuery.value) return accounts.value;
  const q = searchQuery.value.toLowerCase();
  return accounts.value.filter(
    (a) =>
      a.publicKey.toLowerCase().includes(q) ||
      (a.assignedUserName || "").toLowerCase().includes(q) ||
      (a.assignedUserId || "").toLowerCase().includes(q),
  );
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

const allOnPageSelected = computed(
  () =>
    pagedItems.value.length > 0 &&
    pagedItems.value.every((a) => selectedKeys.value.includes(a.publicKey)),
);
const toggleSelectAll = () => {
  const keys = pagedItems.value.map((a) => a.publicKey);
  if (allOnPageSelected.value) {
    selectedKeys.value = selectedKeys.value.filter((k) => !keys.includes(k));
  } else {
    selectedKeys.value = [...new Set([...selectedKeys.value, ...keys])];
  }
};

const flash = (msg: string) => {
  success.value = msg;
  setTimeout(() => {
    if (success.value === msg) success.value = "";
  }, 4000);
};

const getVault = (pk: string): VaultRecord | undefined =>
  vaultMap.value[pk.toLowerCase()];

const shortAddr = (addr: string) => {
  if (!addr || addr.length < 16) return addr || "-";
  return addr.slice(0, 10) + "..." + addr.slice(-6);
};

const formatLocalDateTime = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

const copiedAddr = ref("");
const copyAddr = async (addr: string) => {
  if (!addr) return;
  try {
    await navigator.clipboard.writeText(addr);
    flash("已复制到剪贴板");
  } catch {}
};

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    const [accts, vaults] = await Promise.all([
      adminApi.listAgentAccounts(statusFilter.value),
      adminApi.listAgentVaults(),
    ]);
    accounts.value = accts;
    const map: Record<string, VaultRecord> = {};
    for (const v of vaults) {
      if (v.userAddress) map[v.userAddress.toLowerCase()] = v;
    }
    vaultMap.value = map;
    const valid = new Set(accts.map((a: AgentAccount) => a.publicKey));
    selectedKeys.value = selectedKeys.value.filter((k) => valid.has(k));
  } catch (err: any) {
    error.value = err?.response?.data?.error || "加载 Agent 账户失败";
  } finally {
    loading.value = false;
  }
};

const applySearch = () => {
  searchQuery.value = searchInput.value.trim();
  showSearchModal.value = false;
  page.value = 1;
};
const clearSearch = () => {
  searchQuery.value = "";
  searchInput.value = "";
  page.value = 1;
};

const importAccounts = async () => {
  if (!encryptedPayload.value.trim() || importing.value) return;
  importing.value = true;
  error.value = "";
  importResult.value = null;
  try {
    importResult.value = await adminApi.importAgentAccounts(
      encryptedPayload.value.trim(),
    );
    encryptedPayload.value = "";
    flash(`已导入 ${importResult.value.imported} 个账户`);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "导入账户失败";
  } finally {
    importing.value = false;
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
    passwordError.value = "请输入密码";
    return;
  }
  deleting.value = true;
  passwordError.value = "";
  try {
    const r = await adminApi.deleteAgentPool(selectedKeys.value, passwordInput.value);
    flash(`已删除 ${r.deleted} 个账户`);
    selectedKeys.value = [];
    showPasswordModal.value = false;
    passwordInput.value = "";
    await load();
  } catch (err: any) {
    const msg = err?.response?.data?.error || "操作失败";
    passwordError.value = msg === "invalid_password" ? "密码错误" : msg;
  } finally {
    deleting.value = false;
  }
};

const revokeAgent = async (pk: string) => {
  if (!confirm("确认撤销该 Agent 的用户分配？")) return;
  revoking.value = pk;
  try {
    await adminApi.revokeAgent(pk);
    flash("Agent 已撤销");
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "操作失败";
  } finally {
    revoking.value = null;
  }
};

const openReassign = (pk: string) => {
  reassignTarget.value = pk;
  reassignUserName.value = "";
  showReassignModal.value = true;
};

const confirmReassign = async () => {
  if (!reassignUserName.value.trim()) return;
  reassigning.value = true;
  try {
    await adminApi.reassignAgent(reassignTarget.value, reassignUserName.value.trim());
    flash("Agent 已分配");
    showReassignModal.value = false;
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "操作失败";
  } finally {
    reassigning.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>AgentVault 用户</h1>
      <p class="muted">管理 AgentVault 用户帐号分配，导入加密密钥。</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">{{ success }}</div>

    <div class="table-card">
      <div class="table-toolbar">
        <input type="checkbox" :checked="allOnPageSelected && pagedItems.length > 0" @change="toggleSelectAll" />
        <button class="toolbar-btn" @click="searchInput = searchQuery; showSearchModal = true;">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
          </svg>
          搜索
        </button>
        <span v-if="searchQuery" class="search-tag">{{ searchQuery }}<button class="tag-close" @click="clearSearch">&times;</button></span>
        <select class="toolbar-select" v-model="statusFilter" @change="page = 1; load();">
          <option value="">全部状态</option>
          <option value="unused">未使用</option>
          <option value="assigned">已分配</option>
        </select>
        <button class="toolbar-btn" @click="showImportModal = true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="17 8 12 3 7 8" /><line x1="12" x2="12" y1="3" y2="15" />
          </svg>
          导入
        </button>
        <button class="toolbar-btn danger" :disabled="!selectedKeys.length || deleting" @click="promptDelete">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M3 6h18" /><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" /><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
          </svg>
          删除{{ selectedKeys.length ? ` (${selectedKeys.length})` : "" }}
        </button>
        <span class="toolbar-spacer"></span>
        <span class="toolbar-info">共 {{ total }} 条</span>
        <button class="toolbar-icon" @click="load" title="刷新">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" /><path d="M21 3v5h-5" />
            <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" /><path d="M8 16H3v5" />
          </svg>
        </button>
      </div>

      <div class="table-body">
        <div v-if="loading" class="table-empty">加载中...</div>
        <div v-else-if="error" class="table-empty error">{{ error }}</div>
        <template v-else>
          <table>
            <thead>
              <tr>
                <th style="width: 36px"></th>
                <th>公钥</th>
                <th>状态</th>
                <th>分配用户</th>
                <th>Vault 地址</th>
                <th>Vault 有效</th>
                <th>最后同步</th>
                <th>分配时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.id">
                <td><input type="checkbox" :value="item.publicKey" v-model="selectedKeys" /></td>
                <td class="small monospace addr-cell" :title="item.publicKey" @click="copyAddr(item.publicKey)">
                  {{ shortAddr(item.publicKey) }}
                </td>
                <td>
                  <span class="badge" :class="{ 'badge-success': item.status === 'assigned', 'badge-disabled': item.status === 'unused' }">
                    {{ item.status === "assigned" ? "已分配" : "未使用" }}
                  </span>
                </td>
                <td>{{ item.assignedUserName || item.assignedUserId || "-" }}</td>
                <td class="small monospace addr-cell" :title="getVault(item.publicKey)?.vaultAddress || ''" @click="copyAddr(getVault(item.publicKey)?.vaultAddress || '')">
                  {{ getVault(item.publicKey)?.vaultAddress ? shortAddr(getVault(item.publicKey)!.vaultAddress) : "-" }}
                </td>
                <td>
                  <template v-if="getVault(item.publicKey)">
                    <span :class="getVault(item.publicKey)!.valid ? 'badge badge-success' : 'badge badge-disabled'">
                      {{ getVault(item.publicKey)!.valid ? "有效" : "无效" }}
                    </span>
                  </template>
                  <span v-else class="muted">-</span>
                </td>
                <td class="small">
                  <template v-if="getVault(item.publicKey)?.lastSyncedAt">
                    <span
                      :class="{ 'sync-error': getVault(item.publicKey)?.syncStatus === 'error' }"
                      :title="getVault(item.publicKey)?.syncStatus === 'error' ? getVault(item.publicKey)?.syncError : ''"
                    >
                      {{ formatLocalDateTime(getVault(item.publicKey)!.lastSyncedAt) }}
                    </span>
                  </template>
                  <span v-else class="muted">-</span>
                </td>
                <td class="small">{{ item.assignedAt || "-" }}</td>
                <td class="actions">
                  <button v-if="item.status === 'assigned'" class="btn btn-sm danger" @click="revokeAgent(item.publicKey)" :disabled="revoking === item.publicKey">
                    {{ revoking === item.publicKey ? '...' : '撤销' }}
                  </button>
                  <button v-if="item.status === 'unused'" class="btn btn-sm" @click="openReassign(item.publicKey)">分配</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="filteredItems.length === 0" class="table-empty">暂无数据</div>
        </template>
      </div>

      <div class="table-footer">
        <span>共 {{ total }} 条</span>
        <div class="page-nav">
          <button :disabled="page <= 1" @click="page = 1">&laquo;</button>
          <button :disabled="page <= 1" @click="page--">&lsaquo;</button>
          <span style="font-size:0.78rem;padding:0 0.3rem">{{ page }} / {{ totalPages }}</span>
          <button :disabled="page >= totalPages" @click="page++">&rsaquo;</button>
          <button :disabled="page >= totalPages" @click="page = totalPages">&raquo;</button>
        </div>
        <select v-model="pageSize" @change="page = 1" style="width:auto;padding:0.2rem 0.3rem;font-size:0.75rem;border-radius:6px">
          <option :value="20">20 / 页</option>
          <option :value="50">50 / 页</option>
          <option :value="100">100 / 页</option>
        </select>
      </div>
    </div>

    <!-- 搜索弹窗 -->
    <div v-if="showSearchModal" class="modal-overlay" @click.self="showSearchModal = false">
      <div class="modal modal-sm">
        <h3>搜索</h3>
        <div class="form-group">
          <input v-model="searchInput" placeholder="公钥或用户名..." @keyup.enter="applySearch" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="searchInput = ''; applySearch();">清除</button>
          <button class="btn btn-primary" @click="applySearch">搜索</button>
        </div>
      </div>
    </div>

    <!-- 导入弹窗 -->
    <div v-if="showImportModal" class="modal-overlay" @click.self="showImportModal = false">
      <div class="modal">
        <h3>导入加密密钥</h3>
        <p class="muted small">粘贴加密的 JSON 数据导入 Agent 账户。</p>
        <div class="form-group" style="margin-top: 0.5rem">
          <textarea v-model="encryptedPayload" rows="7" placeholder='{"status":"ok","format":"AES-GCM-256","encrypted_data":"...","count":12}'></textarea>
        </div>
        <p v-if="importResult" class="small muted">
          已导入 {{ importResult.imported }}，重复 {{ importResult.duplicates }}，无效 {{ importResult.invalid }}
        </p>
        <div class="form-actions">
          <button class="btn" @click="showImportModal = false">关闭</button>
          <button class="btn btn-primary" @click="importAccounts" :disabled="importing || !encryptedPayload.trim()">
            {{ importing ? "加载中..." : "导入" }}
          </button>
        </div>
      </div>
    </div>

    <!-- 密码确认弹窗 -->
    <div v-if="showPasswordModal" class="modal-overlay" @click.self="showPasswordModal = false">
      <div class="modal modal-sm">
        <h3>确认删除</h3>
        <p class="muted small">即将删除 {{ selectedKeys.length }} 个 Agent 账户及其关联数据，请输入管理员密码确认。</p>
        <div class="form-group">
          <label>密码</label>
          <input v-model="passwordInput" type="password" @keyup.enter="confirmDelete" />
        </div>
        <div v-if="passwordError" class="field-error">{{ passwordError }}</div>
        <div class="form-actions">
          <button class="btn" @click="showPasswordModal = false">取消</button>
          <button class="btn btn-danger" @click="confirmDelete" :disabled="deleting">
            {{ deleting ? "删除中..." : "确认删除" }}
          </button>
        </div>
      </div>
    </div>

    <!-- 分配弹窗 -->
    <div v-if="showReassignModal" class="modal-overlay" @click.self="showReassignModal = false">
      <div class="modal modal-sm">
        <h3>分配 Agent</h3>
        <p class="muted small monospace">{{ reassignTarget }}</p>
        <div class="form-group" style="margin-top: 0.5rem">
          <label>用户名</label>
          <input v-model="reassignUserName" placeholder="输入用户名..." @keyup.enter="confirmReassign" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="showReassignModal = false">取消</button>
          <button class="btn btn-primary" @click="confirmReassign" :disabled="reassigning || !reassignUserName.trim()">
            {{ reassigning ? "加载中..." : "确认" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.sync-error {
  color: #b42318;
  cursor: help;
  border-bottom: 1px dashed #b42318;
}
</style>
