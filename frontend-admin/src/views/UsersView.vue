<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { adminApi } from '../api/client';
import type { User } from '../types';

const loading = ref(true);
const error = ref('');
const success = ref('');
const users = ref<User[]>([]);
const selectedIds = ref<string[]>([]);
const deleting = ref(false);

// Search
const searchQuery = ref('');
const showSearchModal = ref(false);
const searchInput = ref('');

// Pagination
const page = ref(1);
const pageSize = ref(20);
const total = computed(() => users.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)));
const pagedItems = computed(() => {
  const s = (page.value - 1) * pageSize.value;
  return users.value.slice(s, s + pageSize.value);
});
watch(total, () => { if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value); });

// Select all (current page)
const allOnPageSelected = computed(() => pagedItems.value.length > 0 && pagedItems.value.every(u => selectedIds.value.includes(u.id)));
const toggleSelectAll = () => {
  const ids = pagedItems.value.map(u => u.id);
  if (allOnPageSelected.value) {
    selectedIds.value = selectedIds.value.filter(id => !ids.includes(id));
  } else {
    selectedIds.value = [...new Set([...selectedIds.value, ...ids])];
  }
};

// Password modal
const showPasswordModal = ref(false);
const passwordInput = ref('');
const passwordError = ref('');

const flash = (msg: string) => { success.value = msg; setTimeout(() => { if (success.value === msg) success.value = ''; }, 4000); };

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    users.value = await adminApi.listUsers(searchQuery.value);
    const valid = new Set(users.value.map(u => u.id));
    selectedIds.value = selectedIds.value.filter(id => valid.has(id));
  } catch (err: any) {
    error.value = err?.response?.data?.error || '加载用户列表失败';
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
  searchQuery.value = '';
  searchInput.value = '';
  page.value = 1;
  await load();
};

const promptDelete = () => {
  if (!selectedIds.value.length) return;
  passwordInput.value = '';
  passwordError.value = '';
  showPasswordModal.value = true;
};

const confirmDelete = async () => {
  if (!passwordInput.value.trim()) { passwordError.value = '请输入密码'; return; }
  deleting.value = true;
  passwordError.value = '';
  try {
    const r = await adminApi.deleteUsers(selectedIds.value, passwordInput.value);
    flash(`已删除 ${r.deleted} 个用户`);
    selectedIds.value = [];
    showPasswordModal.value = false;
    passwordInput.value = '';
    await load();
  } catch (err: any) {
    const msg = err?.response?.data?.error || '删除失败';
    passwordError.value = msg === 'invalid_password' ? '密码错误' : msg;
  } finally {
    deleting.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>用户管理</h1>
      <p class="muted">查看已注册的用户及其分配的 Agent 账户。</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">{{ success }}</div>

    <div class="table-card">
      <div class="table-toolbar">
        <input type="checkbox" :checked="allOnPageSelected && pagedItems.length > 0" @change="toggleSelectAll" />
        <button class="toolbar-btn" @click="searchInput = searchQuery; showSearchModal = true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          搜索
        </button>
        <span v-if="searchQuery" class="search-tag">
          {{ searchQuery }}
          <button class="tag-close" @click="clearSearch">&times;</button>
        </span>
        <button class="toolbar-btn danger" :disabled="!selectedIds.length || deleting" @click="promptDelete">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
          删除{{ selectedIds.length ? ` (${selectedIds.length})` : '' }}
        </button>
        <span class="toolbar-spacer"></span>
        <span class="toolbar-info">共 {{ total }} 条</span>
        <button class="toolbar-icon" @click="load" title="刷新">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>
        </button>
      </div>

      <div class="table-body">
        <div v-if="loading" class="table-empty">加载中...</div>
        <div v-else-if="error" class="table-empty error">{{ error }}</div>
        <template v-else>
          <table>
            <thead>
              <tr>
                <th style="width:36px"></th>
                <th>用户名</th>
                <th>邮箱</th>
                <th>钱包地址</th>
                <th>Agent 公钥</th>
                <th>注册时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in pagedItems" :key="user.id">
                <td><input type="checkbox" :value="user.id" v-model="selectedIds" /></td>
                <td>{{ user.name || '-' }}</td>
                <td>{{ user.email || '-' }}</td>
                <td class="small monospace">{{ user.walletAddress || '-' }}</td>
                <td class="small monospace">{{ user.agentPublicKey ? user.agentPublicKey.slice(0, 10) + '...' + user.agentPublicKey.slice(-6) : '-' }}</td>
                <td class="small">{{ user.createdAt || '-' }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="users.length === 0" class="table-empty">暂无用户数据。</div>
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
        <select class="page-nav select" v-model="pageSize" @change="page = 1" style="width:auto;padding:0.2rem 0.3rem;font-size:0.75rem;border-radius:6px">
          <option :value="20">20 条/页</option>
          <option :value="50">50 条/页</option>
          <option :value="100">100 条/页</option>
        </select>
      </div>
    </div>

    <!-- 搜索弹窗 -->
    <div v-if="showSearchModal" class="modal-overlay" @click.self="showSearchModal = false">
      <div class="modal modal-sm">
        <h3>搜索用户</h3>
        <div class="form-group">
          <label>关键词</label>
          <input v-model="searchInput" placeholder="用户名 / 邮箱 / 钱包地址" @keyup.enter="applySearch" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="searchInput = ''; applySearch()">清除</button>
          <button class="btn btn-primary" @click="applySearch">搜索</button>
        </div>
      </div>
    </div>

    <!-- 密码确认弹窗 -->
    <div v-if="showPasswordModal" class="modal-overlay" @click.self="showPasswordModal = false">
      <div class="modal modal-sm">
        <h3>确认删除</h3>
        <p class="muted small">即将删除 {{ selectedIds.length }} 个用户及其关联数据，请输入管理员密码确认。</p>
        <div class="form-group">
          <label>管理员密码</label>
          <input v-model="passwordInput" type="password" placeholder="请输入密码" @keyup.enter="confirmDelete" />
        </div>
        <div v-if="passwordError" class="field-error">{{ passwordError }}</div>
        <div class="form-actions">
          <button class="btn" @click="showPasswordModal = false">取消</button>
          <button class="btn btn-danger" @click="confirmDelete" :disabled="deleting">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
