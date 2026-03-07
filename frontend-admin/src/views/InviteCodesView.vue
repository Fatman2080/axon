<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { adminApi } from "../api/client";
import type { InviteCode } from "../types";

const loading = ref(true);
const error = ref("");
const success = ref("");
const codes = ref<InviteCode[]>([]);

const showGenModal = ref(false);
const genPrefix = ref("");
const genLength = ref(8);
const genCount = ref(10);
const genDescription = ref("");
const generating = ref(false);

const statusFilter = ref("");

const page = ref(1);
const pageSize = ref(20);

const isUsed = (c: InviteCode) => c.usedCount > 0;
const filteredItems = computed(() => {
  if (!statusFilter.value) return codes.value;
  if (statusFilter.value === "unused")
    return codes.value.filter((c) => !isUsed(c));
  if (statusFilter.value === "used")
    return codes.value.filter((c) => isUsed(c));
  return codes.value;
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
    error.value = err?.response?.data?.error || "加载邀请码失败";
  } finally {
    loading.value = false;
  }
};

const generate = async () => {
  if (generating.value || genCount.value < 1) return;
  generating.value = true;
  error.value = "";
  try {
    const r = await adminApi.createInviteCodeBatch({
      prefix: genPrefix.value || undefined,
      length: genLength.value,
      count: genCount.value,
      description: genDescription.value || undefined,
    });
    showGenModal.value = false;
    flash(`成功生成 ${r.length} 个邀请码`);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "生成邀请码失败";
  } finally {
    generating.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>邀请码</h1>
      <p class="muted">生成与管理一次性邀请码。</p>
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
            genPrefix = '';
            genLength = 8;
            genCount = 10;
            genDescription = '';
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
          生成邀请码
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
          <option value="">全部状态</option>
          <option value="unused">未使用</option>
          <option value="used">已使用</option>
        </select>
        <span class="toolbar-info">共 {{ total }} 条</span>
        <button class="toolbar-icon" @click="load" title="刷新">
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
        <div v-if="loading" class="table-empty">加载中...</div>
        <div v-else-if="error" class="table-empty error">{{ error }}</div>
        <template v-else>
          <table>
            <thead>
              <tr>
                <th>代码</th>
                <th>状态</th>
                <th>使用者</th>
                <th>描述</th>
                <th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.code">
                <td class="monospace font-bold">{{ item.code }}</td>
                <td>
                  <span
                    class="badge"
                    :class="{
                      'badge-success': item.usedCount === 0,
                      'badge-disabled': item.usedCount > 0,
                    }"
                  >
                    {{ item.usedCount === 0 ? "未使用" : "已使用" }}
                  </span>
                </td>
                <td class="small">
                  <template v-if="item.usedByUsers && item.usedByUsers.length">
                    <span v-for="(u, i) in item.usedByUsers" :key="u.userId">
                      <span class="user-tag">{{
                        u.userName || u.userId.slice(0, 10)
                      }}</span>
                      <template v-if="i < item.usedByUsers.length - 1"
                        >,
                      </template>
                    </span>
                  </template>
                  <span v-else class="muted">-</span>
                </td>
                <td class="small">{{ item.description || "-" }}</td>
                <td class="small">{{ item.createdAt }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="filteredItems.length === 0" class="table-empty">
            暂无数据
          </div>
        </template>
      </div>

      <div class="table-footer">
        <span>共 {{ total }} 条</span>
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
          <option :value="20">20 / 页</option>
          <option :value="50">50 / 页</option>
          <option :value="100">100 / 页</option>
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
        <h3>生成邀请码</h3>
        <p class="muted small">生成一次性邀请码。</p>
        <div class="form-group" style="margin-top: 0.5rem">
          <label>前缀 (可选)</label>
          <input v-model="genPrefix" placeholder="如 OPENFI-" />
        </div>
        <div class="form-group">
          <label>随机部分长度</label>
          <input v-model.number="genLength" type="number" min="4" max="32" />
        </div>
        <div class="form-group">
          <label>生成数量 (1-100)</label>
          <input v-model.number="genCount" type="number" min="1" max="100" />
        </div>
        <div class="form-group">
          <label>描述 (可选)</label>
          <input v-model="genDescription" placeholder="批次备注..." />
        </div>
        <div v-if="error" class="field-error">{{ error }}</div>
        <div class="form-actions">
          <button class="btn" @click="showGenModal = false">取消</button>
          <button
            class="btn btn-primary"
            @click="generate"
            :disabled="generating"
          >
            {{ generating ? "生成中..." : "生成" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.user-tag {
  display: inline-block;
  background: #eff6ff;
  color: #1d4ed8;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  font-size: 0.72rem;
  white-space: nowrap;
}
</style>
