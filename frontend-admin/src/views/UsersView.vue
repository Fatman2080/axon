<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { adminApi } from "../api/client";
import type { User, VaultRecord } from "../types";

const loading = ref(true);
const error = ref("");
const success = ref("");
const users = ref<User[]>([]);
const selectedIds = ref<string[]>([]);
const deleting = ref(false);

const searchQuery = ref("");
const showSearchModal = ref(false);
const searchInput = ref("");

const page = ref(1);
const pageSize = ref(20);
const total = computed(() => users.value.length);
const totalPages = computed(() =>
  Math.max(1, Math.ceil(total.value / pageSize.value)),
);
const pagedItems = computed(() => {
  const s = (page.value - 1) * pageSize.value;
  return users.value.slice(s, s + pageSize.value);
});
watch(total, () => {
  if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value);
});

const allOnPageSelected = computed(
  () =>
    pagedItems.value.length > 0 &&
    pagedItems.value.every((u) => selectedIds.value.includes(u.id)),
);
const toggleSelectAll = () => {
  const ids = pagedItems.value.map((u) => u.id);
  if (allOnPageSelected.value) {
    selectedIds.value = selectedIds.value.filter((id) => !ids.includes(id));
  } else {
    selectedIds.value = [...new Set([...selectedIds.value, ...ids])];
  }
};

const showPasswordModal = ref(false);
const passwordInput = ref("");
const passwordError = ref("");
const revokingUser = ref<string | null>(null);

const flash = (msg: string) => {
  success.value = msg;
  setTimeout(() => {
    if (success.value === msg) success.value = "";
  }, 4000);
};

const shortAddr = (addr: string) => {
  if (!addr || addr.length < 16) return addr || "-";
  return addr.slice(0, 10) + "..." + addr.slice(-6);
};

const copiedAddr = ref("");
const copyAddr = async (addr: string) => {
  if (!addr) return;
  try {
    await navigator.clipboard.writeText(addr);
    flash("已复制到剪贴板");
  } catch {}
};

const vaultMap = ref<Record<string, string>>({});

const loadVaults = async () => {
  try {
    const vaults = await adminApi.listAgentVaults();
    const map: Record<string, string> = {};
    for (const v of vaults) {
      if (v.userAddress) {
        map[v.userAddress.toLowerCase()] = v.vaultAddress;
      }
    }
    vaultMap.value = map;
  } catch {}
};

const getUserVault = (user: User): string => {
  const pk = user.agentPublicKey?.toLowerCase();
  if (!pk) return "";
  return vaultMap.value[pk] || "";
};

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    users.value = await adminApi.listUsers(searchQuery.value);
    const valid = new Set(users.value.map((u) => u.id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
  } catch (err: any) {
    error.value = err?.response?.data?.error || "加载用户失败";
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

const promptDelete = () => {
  if (!selectedIds.value.length) return;
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
    const r = await adminApi.deleteUsers(selectedIds.value, passwordInput.value);
    flash(`已删除 ${r.deleted} 个用户`);
    selectedIds.value = [];
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

const revokeUserAgent = async (userId: string) => {
  if (!confirm("确认撤销该用户的 Agent 分配？")) return;
  revokingUser.value = userId;
  try {
    await adminApi.revokeUserAgent(userId);
    flash("撤销成功");
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "操作失败";
  } finally {
    revokingUser.value = null;
  }
};

const revokeUserInvite = async (userId: string) => {
  if (!confirm("撤销邀请码？这将同时撤销 Agent 分配。")) return;
  revokingUser.value = userId;
  try {
    await adminApi.revokeUserInvite(userId);
    flash("撤销成功");
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "操作失败";
  } finally {
    revokingUser.value = null;
  }
};

onMounted(() => {
  load();
  loadVaults();
});
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>用户</h1>
      <p class="muted">查看注册用户、AgentVault 分配和邀请码使用情况。</p>
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
        <button class="toolbar-btn danger" :disabled="!selectedIds.length || deleting" @click="promptDelete">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M3 6h18" /><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" /><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
          </svg>
          删除{{ selectedIds.length ? ` (${selectedIds.length})` : "" }}
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
                <th>用户名</th>
                <th>邮箱</th>
                <th>AgentVault 账户</th>
                <th>Vault 合约</th>
                <th>邀请码</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in pagedItems" :key="user.id">
                <td><input type="checkbox" :value="user.id" v-model="selectedIds" /></td>
                <td>{{ user.name || "-" }}</td>
                <td>{{ user.email || "-" }}</td>
                <td class="small monospace addr-cell" :title="user.agentPublicKey || ''" @click="copyAddr(user.agentPublicKey || '')">
                  {{ user.agentPublicKey ? shortAddr(user.agentPublicKey) : "-" }}
                </td>
                <td class="small monospace addr-cell" :title="getUserVault(user)" @click="copyAddr(getUserVault(user))">
                  {{ getUserVault(user) ? shortAddr(getUserVault(user)) : "-" }}
                </td>
                <td class="small">{{ user.inviteCodeUsed || "-" }}</td>
                <td class="small">{{ user.createdAt || "-" }}</td>
                <td class="actions">
                  <button
                    v-if="user.agentPublicKey"
                    class="btn btn-sm danger"
                    @click="revokeUserAgent(user.id)"
                    :disabled="revokingUser === user.id"
                  >
                    {{ revokingUser === user.id ? '...' : '撤销 Agent' }}
                  </button>
                  <button
                    v-if="user.inviteCodeUsed"
                    class="btn btn-sm danger"
                    @click="revokeUserInvite(user.id)"
                    :disabled="revokingUser === user.id"
                  >
                    {{ revokingUser === user.id ? '...' : '撤销邀请码' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="users.length === 0" class="table-empty">暂无数据</div>
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
          <input v-model="searchInput" placeholder="用户名搜索..." @keyup.enter="applySearch" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="searchInput = ''; applySearch();">清除</button>
          <button class="btn btn-primary" @click="applySearch">搜索</button>
        </div>
      </div>
    </div>

    <!-- 密码确认弹窗 -->
    <div v-if="showPasswordModal" class="modal-overlay" @click.self="showPasswordModal = false">
      <div class="modal modal-sm">
        <h3>确认删除</h3>
        <p class="muted small">即将删除 {{ selectedIds.length }} 个用户，请输入管理员密码确认。</p>
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
  </section>
</template>
