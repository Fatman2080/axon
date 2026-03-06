<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { adminApi } from "../api/client";
import type { VaultRecord, AgentAccount } from "../types";

const loading = ref(true);
const error = ref("");
const success = ref("");
const deleting = ref(false);
const items = ref<VaultRecord[]>([]);
const accountMap = ref<Record<string, AgentAccount>>({});
const selectedKeys = ref<string[]>([]);

const searchQuery = ref("");
const showSearchModal = ref(false);
const searchInput = ref("");
const showDeleteModal = ref(false);

const page = ref(1);
const pageSize = ref(20);

const filteredItems = computed(() => {
  if (!searchQuery.value) return items.value;
  const q = searchQuery.value.toLowerCase();
  return items.value.filter(
    (i) => {
      const account = accountMap.value[i.userAddress.toLowerCase()];
      return (
      i.vaultAddress.toLowerCase().includes(q) ||
      i.userAddress.toLowerCase().includes(q) ||
      (account?.publicKey || "").toLowerCase().includes(q) ||
      (account?.assignedUserName || "").toLowerCase().includes(q) ||
      (account?.assignedUserId || "").toLowerCase().includes(q)
      );
    },
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
    pagedItems.value.every((i) => selectedKeys.value.includes(i.vaultAddress.toLowerCase())),
);
const deleteTargetCount = computed(() => selectedKeys.value.length);

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    const [vaults, accounts] = await Promise.all([
      adminApi.listAgentVaults(),
      adminApi.listAgentAccounts(),
    ]);
    items.value = vaults;
    const mapped: Record<string, AgentAccount> = {};
    for (const account of accounts) {
      mapped[account.publicKey.toLowerCase()] = account;
    }
    accountMap.value = mapped;
    const valid = new Set(vaults.map((v) => v.vaultAddress.toLowerCase()));
    selectedKeys.value = selectedKeys.value.filter((k) => valid.has(k));
  } catch (err: any) {
    error.value = err?.response?.data?.error || "加载 Vault 数据失败";
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

const getLinkedAccount = (v: VaultRecord) =>
  accountMap.value[v.userAddress.toLowerCase()];
const isSelected = (vaultAddress: string) =>
  selectedKeys.value.includes(vaultAddress.toLowerCase());
const toggleSelect = (vaultAddress: string, checked: boolean) => {
  const key = vaultAddress.toLowerCase();
  if (checked) {
    selectedKeys.value = [...new Set([...selectedKeys.value, key])];
    return;
  }
  selectedKeys.value = selectedKeys.value.filter((v) => v !== key);
};
const onSelectChange = (vaultAddress: string, event: Event) => {
  const checked = (event.target as HTMLInputElement | null)?.checked === true;
  toggleSelect(vaultAddress, checked);
};
const toggleSelectAll = () => {
  const keys = pagedItems.value.map((i) => i.vaultAddress.toLowerCase());
  if (allOnPageSelected.value) {
    selectedKeys.value = selectedKeys.value.filter((k) => !keys.includes(k));
    return;
  }
  selectedKeys.value = [...new Set([...selectedKeys.value, ...keys])];
};

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

const copyAddr = async (addr: string) => {
  if (!addr) return;
  try {
    await navigator.clipboard.writeText(addr);
    success.value = "已复制到剪贴板";
    setTimeout(() => { success.value = ""; }, 2000);
  } catch {}
};

const openDeleteSelectedModal = () => {
  if (!selectedKeys.value.length || deleting.value) return;
  showDeleteModal.value = true;
};

const confirmDelete = async () => {
  if (deleteTargetCount.value <= 0 || deleting.value) return;
  deleting.value = true;
  error.value = "";
  try {
    const r = await adminApi.deleteAgentVaults({
      vaultAddresses: selectedKeys.value,
    });
    success.value = `已删除 ${r.deleted} 条 AgentVault 记录`;
    setTimeout(() => { success.value = ""; }, 2500);
    selectedKeys.value = [];
    showDeleteModal.value = false;
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "删除失败";
  } finally {
    deleting.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>AgentVault 合约</h1>
      <p class="muted">查看从 Allocator 合约发现的所有 AgentVault 及其链上数据。</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">{{ success }}</div>

    <div class="table-card">
      <div class="table-toolbar">
        <input type="checkbox" :checked="allOnPageSelected && pagedItems.length > 0" @change="toggleSelectAll" />
        <button
          class="toolbar-btn"
          @click="searchInput = searchQuery; showSearchModal = true;"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
          </svg>
          搜索
        </button>
        <span v-if="searchQuery" class="search-tag"
          >{{ searchQuery }}<button class="tag-close" @click="clearSearch">&times;</button></span
        >
        <button class="toolbar-btn danger" :disabled="!selectedKeys.length || deleting" @click="openDeleteSelectedModal">
          删除选中{{ selectedKeys.length ? ` (${selectedKeys.length})` : "" }}
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
                <th style="width: 44px"></th>
                <th>Vault 地址</th>
                <th>User 地址</th>
                <th>EVM 余额</th>
                <th>初始资金</th>
                <th>账户价值</th>
                <th>未实现盈亏</th>
                <th>有效</th>
                <th>AgentVault 帐号</th>
                <th>用户</th>
                <th>最后同步</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.vaultAddress">
                <td>
                  <input
                    type="checkbox"
                    :checked="isSelected(item.vaultAddress)"
                    @change="onSelectChange(item.vaultAddress, $event)"
                  />
                </td>
                <td class="small monospace addr-cell" :title="item.vaultAddress" @click="copyAddr(item.vaultAddress)">
                  {{ shortAddr(item.vaultAddress) }}
                </td>
                <td class="small monospace addr-cell" :title="item.userAddress" @click="copyAddr(item.userAddress)">
                  {{ shortAddr(item.userAddress) }}
                </td>
                <td>${{ item.evmBalance?.toLocaleString() || "0" }}</td>
                <td>
                  <template v-if="item.initialCapital && item.initialCapital > 0">${{ item.initialCapital.toLocaleString() }}</template>
                  <span v-else class="muted">-</span>
                </td>
                <td>${{ item.accountValue?.toLocaleString() || "0" }}</td>
                <td :class="{ 'profit-up': (item.unrealizedPnl || 0) >= 0, 'profit-down': (item.unrealizedPnl || 0) < 0 }">
                  {{ (item.unrealizedPnl || 0) >= 0 ? "+" : "" }}${{ (item.unrealizedPnl || 0).toLocaleString() }}
                </td>
                <td>
                  <span :class="item.valid ? 'badge badge-success' : 'badge badge-disabled'">{{ item.valid ? "有效" : "无效" }}</span>
                </td>
                <td class="small monospace">
                  <template v-if="getLinkedAccount(item)">
                    {{ shortAddr(getLinkedAccount(item)!.publicKey) }}
                  </template>
                  <span v-else class="muted"></span>
                </td>
                <td>
                  <template v-if="getLinkedAccount(item)">
                    {{ getLinkedAccount(item)!.assignedUserName || getLinkedAccount(item)!.assignedUserId || "" }}
                  </template>
                  <span v-else class="muted"></span>
                </td>
                <td class="small">
                  <span
                    v-if="item.lastSyncedAt"
                    :class="{ 'sync-error': item.syncStatus === 'error' }"
                    :title="item.syncStatus === 'error' ? item.syncError : ''"
                  >{{ formatLocalDateTime(item.lastSyncedAt) }}</span>
                  <span v-else class="muted">从未</span>
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
          <input v-model="searchInput" placeholder="地址搜索..." @keyup.enter="applySearch" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="searchInput = ''; applySearch();">清除</button>
          <button class="btn btn-primary" @click="applySearch">搜索</button>
        </div>
      </div>
    </div>

    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal modal-sm">
        <h3>确认删除</h3>
        <p class="muted small">即将删除 {{ deleteTargetCount }} 条 AgentVault 合约记录。</p>
        <div class="form-actions">
          <button class="btn" @click="showDeleteModal = false">取消</button>
          <button class="btn btn-danger" @click="confirmDelete" :disabled="deleting || deleteTargetCount <= 0">
            {{ deleting ? "删除中..." : "确认删除" }}
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
