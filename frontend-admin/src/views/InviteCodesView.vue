<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { adminApi } from '../api/client';
import type { InviteCode } from '../types';

const loading = ref(true);
const error = ref('');
const success = ref('');
const codes = ref<InviteCode[]>([]);
const selectedIds = ref<string[]>([]);
const generating = ref(false);
const deleting = ref(false);
const exporting = ref(false);

// Search (client-side)
const searchQuery = ref('');
const showSearchModal = ref(false);
const searchInput = ref('');

// Generate modal
const showGenerateModal = ref(false);
const batchForm = ref({ prefix: 'INV', length: 12, count: 20, maxUses: 1, description: '' });

// Export modal
const showExportModal = ref(false);

// Pagination
const page = ref(1);
const pageSize = ref(20);

const filteredCodes = computed(() => {
  if (!searchQuery.value) return codes.value;
  const q = searchQuery.value.toLowerCase();
  return codes.value.filter(c => c.code.toLowerCase().includes(q) || (c.description || '').toLowerCase().includes(q));
});

const total = computed(() => filteredCodes.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)));
const pagedItems = computed(() => {
  const s = (page.value - 1) * pageSize.value;
  return filteredCodes.value.slice(s, s + pageSize.value);
});
watch(total, () => { if (page.value > totalPages.value) page.value = Math.max(1, totalPages.value); });

const allOnPageSelected = computed(() => pagedItems.value.length > 0 && pagedItems.value.every(c => selectedIds.value.includes(c.id)));
const toggleSelectAll = () => {
  const ids = pagedItems.value.map(c => c.id);
  if (allOnPageSelected.value) {
    selectedIds.value = selectedIds.value.filter(id => !ids.includes(id));
  } else {
    selectedIds.value = [...new Set([...selectedIds.value, ...ids])];
  }
};

const flash = (msg: string) => { success.value = msg; setTimeout(() => { if (success.value === msg) success.value = ''; }, 4000); };

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    codes.value = await adminApi.listInviteCodes();
    const valid = new Set(codes.value.map(c => c.id));
    selectedIds.value = selectedIds.value.filter(id => valid.has(id));
  } catch (err: any) {
    error.value = err?.response?.data?.error || '加载邀请码列表失败';
  } finally {
    loading.value = false;
  }
};

const applySearch = () => { searchQuery.value = searchInput.value.trim(); showSearchModal.value = false; page.value = 1; };
const clearSearch = () => { searchQuery.value = ''; searchInput.value = ''; page.value = 1; };

const createBatch = async () => {
  if (generating.value) return;
  generating.value = true;
  error.value = '';
  try {
    const created = await adminApi.createInviteCodeBatch({
      prefix: batchForm.value.prefix.trim(),
      length: batchForm.value.length,
      count: batchForm.value.count,
      maxUses: batchForm.value.maxUses,
      description: batchForm.value.description.trim(),
    });
    showGenerateModal.value = false;
    flash(`已生成 ${created.length} 个邀请码`);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || '生成邀请码失败';
  } finally {
    generating.value = false;
  }
};

const deleteSelected = async () => {
  if (!selectedIds.value.length || deleting.value) return;
  if (!window.confirm(`确认删除 ${selectedIds.value.length} 个邀请码？`)) return;
  deleting.value = true;
  error.value = '';
  try {
    const r = await adminApi.deleteInviteCodes(selectedIds.value);
    flash(`已删除 ${r.deleted} 个邀请码`);
    selectedIds.value = [];
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || '删除邀请码失败';
  } finally {
    deleting.value = false;
  }
};

const toggleStatus = async (id: string, currentStatus: string) => {
  const newStatus = currentStatus === 'active' ? 'disabled' : 'active';
  error.value = '';
  try {
    await adminApi.updateInviteCode(id, { status: newStatus });
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || '更新邀请码状态失败';
  }
};

const exportUnused = async (format: 'json' | 'csv') => {
  exporting.value = true;
  error.value = '';
  showExportModal.value = false;
  try {
    const result = await adminApi.exportUnusedInviteCodes(format);
    if (format === 'csv') {
      downloadText(typeof result === 'string' ? result : '', 'unused-invite-codes.csv', 'text/csv;charset=utf-8');
    } else {
      downloadText(typeof result === 'string' ? result : JSON.stringify(result, null, 2), 'unused-invite-codes.json', 'application/json;charset=utf-8');
    }
    flash('未使用的邀请码已导出');
  } catch (err: any) {
    error.value = err?.response?.data?.error || '导出邀请码失败';
  } finally {
    exporting.value = false;
  }
};

const downloadText = (content: string, filename: string, contentType: string) => {
  const blob = new Blob([content], { type: contentType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>邀请码管理</h1>
      <p class="muted">批量生成邀请码，支持批量删除和导出。</p>
    </div>

    <div v-if="success" class="alert alert-success" @click="success = ''">{{ success }}</div>

    <div class="table-card">
      <div class="table-toolbar">
        <input type="checkbox" :checked="allOnPageSelected && pagedItems.length > 0" @change="toggleSelectAll" />
        <button class="toolbar-btn" @click="searchInput = searchQuery; showSearchModal = true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          搜索
        </button>
        <span v-if="searchQuery" class="search-tag">{{ searchQuery }}<button class="tag-close" @click="clearSearch">&times;</button></span>
        <button class="toolbar-btn primary" @click="showGenerateModal = true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
          生成
        </button>
        <button class="toolbar-btn" @click="showExportModal = true" :disabled="exporting">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>
          导出
        </button>
        <button class="toolbar-btn danger" :disabled="!selectedIds.length || deleting" @click="deleteSelected">
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
                <th>邀请码</th>
                <th>描述</th>
                <th>状态</th>
                <th>最大次数</th>
                <th>已使用</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="code in pagedItems" :key="code.id">
                <td><input type="checkbox" :value="code.id" v-model="selectedIds" /></td>
                <td class="monospace" style="font-weight:600;font-size:0.82rem">{{ code.code }}</td>
                <td class="small">{{ code.description || '-' }}</td>
                <td>
                  <span class="badge" :class="{ 'badge-success': code.status === 'active', 'badge-disabled': code.status === 'disabled' }">
                    {{ code.status === 'active' ? '有效' : '已禁用' }}
                  </span>
                </td>
                <td>{{ code.maxUses || '无限' }}</td>
                <td>{{ code.usedCount }}</td>
                <td class="small">{{ code.createdAt }}</td>
                <td>
                  <button class="btn btn-sm" @click="toggleStatus(code.id, code.status)">
                    {{ code.status === 'active' ? '禁用' : '启用' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="filteredCodes.length === 0" class="table-empty">暂无邀请码数据。</div>
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
          <option :value="20">20 条/页</option>
          <option :value="50">50 条/页</option>
          <option :value="100">100 条/页</option>
        </select>
      </div>
    </div>

    <!-- 搜索弹窗 -->
    <div v-if="showSearchModal" class="modal-overlay" @click.self="showSearchModal = false">
      <div class="modal modal-sm">
        <h3>搜索邀请码</h3>
        <div class="form-group">
          <label>关键词</label>
          <input v-model="searchInput" placeholder="邀请码 / 描述" @keyup.enter="applySearch" autofocus />
        </div>
        <div class="form-actions">
          <button class="btn" @click="searchInput = ''; applySearch()">清除</button>
          <button class="btn btn-primary" @click="applySearch">搜索</button>
        </div>
      </div>
    </div>

    <!-- 批量生成弹窗 -->
    <div v-if="showGenerateModal" class="modal-overlay" @click.self="showGenerateModal = false">
      <div class="modal">
        <h3>批量生成邀请码</h3>
        <div class="grid-2" style="margin-top:0.6rem">
          <div class="form-group">
            <label>前缀</label>
            <input v-model="batchForm.prefix" placeholder="INV" />
          </div>
          <div class="form-group">
            <label>码长度</label>
            <input v-model.number="batchForm.length" type="number" min="6" max="64" />
          </div>
          <div class="form-group">
            <label>数量</label>
            <input v-model.number="batchForm.count" type="number" min="1" max="5000" />
          </div>
          <div class="form-group">
            <label>最大使用次数</label>
            <input v-model.number="batchForm.maxUses" type="number" min="0" placeholder="0 = 无限" />
          </div>
        </div>
        <div class="form-group">
          <label>描述</label>
          <input v-model="batchForm.description" placeholder="活动 / 来源 / 备注" />
        </div>
        <div class="form-actions">
          <button class="btn" @click="showGenerateModal = false">取消</button>
          <button class="btn btn-primary" @click="createBatch" :disabled="generating">
            {{ generating ? '生成中...' : '批量生成' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 导出弹窗 -->
    <div v-if="showExportModal" class="modal-overlay" @click.self="showExportModal = false">
      <div class="modal modal-sm">
        <h3>导出未使用邀请码</h3>
        <p class="muted small">选择导出格式，将导出所有未使用的邀请码。</p>
        <div class="form-actions" style="margin-top:0.8rem">
          <button class="btn" @click="exportUnused('csv')">CSV 格式</button>
          <button class="btn btn-primary" @click="exportUnused('json')">JSON 格式</button>
        </div>
      </div>
    </div>
  </section>
</template>
