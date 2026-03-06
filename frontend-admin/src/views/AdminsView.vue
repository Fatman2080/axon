<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { adminApi } from "../api/client";
import { useAuthStore } from "../stores/auth";
import type { AdminUser } from "../types";

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
    error.value = err?.response?.data?.error || "加载管理员列表失败";
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
    flash(`管理员已创建：${created.email}`);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "创建管理员失败";
  } finally {
    creating.value = false;
  }
};

const resetPassword = async (admin: AdminUser) => {
  const next = window.prompt(
    `为 ${admin.email} 设置新密码（至少 6 位）：`,
    "",
  );
  if (!next) return;
  if (next.trim().length < 6) {
    error.value = "至少 6 个字符";
    return;
  }
  error.value = "";
  try {
    await adminApi.updateAdminPassword(admin.id, next.trim());
    flash(`密码已更新：${admin.email}`);
  } catch (err: any) {
    error.value = err?.response?.data?.error || "更新密码失败";
  }
};

const deleteAdmin = async (admin: AdminUser) => {
  if (!window.confirm(`确认删除管理员 ${admin.email}？`))
    return;
  error.value = "";
  try {
    await adminApi.deleteAdmin(admin.id);
    flash(`管理员已删除：${admin.email}`);
    await load();
  } catch (err: any) {
    error.value = err?.response?.data?.error || "删除管理员失败";
  }
};

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-header">
      <h1>管理员</h1>
      <p class="muted">管理管理员账户、重置密码和删除。</p>
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
          添加管理员
        </button>
        <span class="toolbar-spacer"></span>
        <span class="toolbar-info">共 {{ admins.length }} 条</span>
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
                <th>邮箱</th>
                <th>名称</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in admins" :key="item.id">
                <td>{{ item.email }}</td>
                <td>{{ item.name }}</td>
                <td class="small">{{ item.createdAt }}</td>
                <td class="actions">
                  <button class="btn btn-sm" @click="resetPassword(item)">
                    重置密码
                  </button>
                  <button
                    class="btn btn-sm btn-danger"
                    :disabled="item.id === currentAdminId"
                    @click="deleteAdmin(item)"
                  >
                    删除
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="admins.length === 0" class="table-empty">
            暂无数据
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
        <h3>添加管理员</h3>
        <div class="form-group">
          <label>邮箱</label>
          <input
            v-model="createForm.email"
            type="email"
            placeholder="admin@example.com"
          />
        </div>
        <div class="form-group">
          <label>名称</label>
          <input
            v-model="createForm.name"
            placeholder="管理员名称"
          />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input
            v-model="createForm.password"
            type="password"
            placeholder="至少 6 个字符"
            @keyup.enter="createAdmin"
          />
        </div>
        <div class="form-actions">
          <button class="btn" @click="showCreateModal = false">
            取消
          </button>
          <button
            class="btn btn-primary"
            @click="createAdmin"
            :disabled="creating"
          >
            {{ creating ? "创建中..." : "创建" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
