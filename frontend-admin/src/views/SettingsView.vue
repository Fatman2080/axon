<script setup lang="ts">
import { onMounted, ref, reactive } from "vue";
import { adminApi } from "../api/client";

const loading = ref(true);
const loadError = ref("");

const sync = reactive({ interval: 0, hlConcurrency: 5, saving: false, msg: "", ok: true });
const contracts = reactive({
  rpcURL: "",
  allocatorAddress: "",
  saving: false,
  msg: "",
  ok: true,
});
const dispatch = reactive({
  command: "",
  saving: false,
  msg: "",
  ok: true,
});
const internSlots = reactive({
  total: 100,
  consumed: 0,
  remaining: 100,
  saving: false,
  msg: "",
  ok: true,
});
const tvlOffset = reactive({
  value: 0,
  saving: false,
  msg: "",
  ok: true,
});
const xoauth = reactive({
  clientId: "",
  clientSecret: "",
  scopes: "",
  saving: false,
  msg: "",
  ok: true,
});
const backup = reactive({
  intervalHours: 24,
  retainHourly: 3,
  retainDaily: 3,
  retainWeekly: 3,
  lastBackupAt: "",
  saving: false,
  creating: false,
  msg: "",
  ok: true,
});
const backupList = ref<{ name: string; size: number; createdAt: string }[]>([]);
const backupListLoading = ref(false);
const restoreModal = reactive({
  visible: false,
  step: 1 as 1 | 2,
  name: "",
  password: "",
  running: false,
  msg: "",
  ok: true,
});

const flash = (
  target: { msg: string; ok: boolean },
  message: string,
  isOk: boolean,
) => {
  target.msg = message;
  target.ok = isOk;
  if (isOk)
    setTimeout(() => {
      if (target.msg === message) target.msg = "";
    }, 4000);
};

const load = async () => {
  loading.value = true;
  loadError.value = "";
  try {
    const [syncData, xoauthData, contractsData, internSlotsData, tvlOffsetData, dispatchData, backupData, backupsData] = await Promise.all([
      adminApi.getSyncSettings(),
      adminApi.getXOAuthSettings(),
      adminApi.getContractsSettings(),
      adminApi.getInternSlots(),
      adminApi.getTvlOffset(),
      adminApi.getDispatchSettings(),
      adminApi.getBackupSettings(),
      adminApi.listBackups(),
    ]);
    sync.interval = syncData.intervalSeconds;
    sync.hlConcurrency = syncData.hlConcurrency || 5;
    xoauth.clientId = xoauthData.clientId || "";
    xoauth.clientSecret = xoauthData.clientSecret || "";
    xoauth.scopes = xoauthData.scopes || "";
    contracts.rpcURL = contractsData.rpcURL || "";
    contracts.allocatorAddress = contractsData.allocatorAddress || "";
    dispatch.command = dispatchData.command || "";
    internSlots.total = internSlotsData.total;
    internSlots.consumed = internSlotsData.consumed;
    internSlots.remaining = internSlotsData.remaining;
    tvlOffset.value = tvlOffsetData.tvlOffset || 0;
    backup.intervalHours = backupData.intervalHours;
    backup.retainHourly = backupData.retainHourly;
    backup.retainDaily = backupData.retainDaily;
    backup.retainWeekly = backupData.retainWeekly;
    backup.lastBackupAt = backupData.lastBackupAt || "";
    backupList.value = backupsData || [];
  } catch (err: any) {
    loadError.value = err?.response?.data?.error || "加载设置失败";
  } finally {
    loading.value = false;
  }
};

const saveSyncSettings = async () => {
  sync.saving = true;
  sync.msg = "";
  try {
    const r = await adminApi.updateSyncSettings({
      intervalSeconds: sync.interval,
      hlConcurrency: sync.hlConcurrency,
    });
    sync.interval = r.intervalSeconds;
    sync.hlConcurrency = r.hlConcurrency || 5;
    flash(
      sync,
      r.intervalSeconds > 0
        ? `自动同步已更新（间隔：${r.intervalSeconds}s）`
        : "自动同步已禁用",
      true,
    );
  } catch (err: any) {
    flash(sync, err?.response?.data?.error || "保存失败", false);
  } finally {
    sync.saving = false;
  }
};

const saveContractsSettings = async () => {
  contracts.saving = true;
  contracts.msg = "";
  try {
    await adminApi.updateContractsSettings({
      rpcURL: contracts.rpcURL,
      allocatorAddress: contracts.allocatorAddress,
    });
    flash(contracts, "合约配置已更新", true);
  } catch (err: any) {
    flash(
      contracts,
      err?.response?.data?.error || "保存失败",
      false,
    );
  } finally {
    contracts.saving = false;
  }
};

const saveDispatchSettings = async () => {
  dispatch.saving = true;
  dispatch.msg = "";
  try {
    await adminApi.updateDispatchSettings(dispatch.command);
    flash(dispatch, "配置已更新", true);
  } catch (err: any) {
    flash(dispatch, err?.response?.data?.error || "保存失败", false);
  } finally {
    dispatch.saving = false;
  }
};

const saveInternSlots = async () => {
  internSlots.saving = true;
  internSlots.msg = "";
  try {
    const r = await adminApi.updateInternSlots(internSlots.total);
    internSlots.total = r.total;
    internSlots.consumed = r.consumed;
    internSlots.remaining = r.remaining;
    flash(internSlots, "配置已更新", true);
  } catch (err: any) {
    flash(internSlots, err?.response?.data?.error || "保存失败", false);
  } finally {
    internSlots.saving = false;
  }
};

const saveTvlOffset = async () => {
  tvlOffset.saving = true;
  tvlOffset.msg = "";
  try {
    const r = await adminApi.updateTvlOffset(tvlOffset.value);
    tvlOffset.value = r.tvlOffset;
    flash(tvlOffset, "TVL 偏移量已更新", true);
  } catch (err: any) {
    flash(tvlOffset, err?.response?.data?.error || "保存失败", false);
  } finally {
    tvlOffset.saving = false;
  }
};

const saveXOAuthSettings = async () => {
  xoauth.saving = true;
  xoauth.msg = "";
  try {
    await adminApi.updateXOAuthSettings({
      clientId: xoauth.clientId,
      clientSecret: xoauth.clientSecret,
      scopes: xoauth.scopes,
    });
    flash(xoauth, "配置已更新", true);
  } catch (err: any) {
    flash(xoauth, err?.response?.data?.error || "保存失败", false);
  } finally {
    xoauth.saving = false;
  }
};

const saveBackupSettings = async () => {
  backup.saving = true;
  backup.msg = "";
  try {
    const r = await adminApi.updateBackupSettings({
      intervalHours: backup.intervalHours,
      retainHourly: backup.retainHourly,
      retainDaily: backup.retainDaily,
      retainWeekly: backup.retainWeekly,
    });
    backup.intervalHours = r.intervalHours;
    backup.retainHourly = r.retainHourly;
    backup.retainDaily = r.retainDaily;
    backup.retainWeekly = r.retainWeekly;
    backup.lastBackupAt = r.lastBackupAt || "";
    flash(backup, r.intervalHours > 0 ? `自动备份已更新（间隔：${r.intervalHours}h）` : "自动备份已禁用", true);
    // Refresh list since cleanup may have removed files
    backupList.value = await adminApi.listBackups();
  } catch (err: any) {
    flash(backup, err?.response?.data?.error || "保存失败", false);
  } finally {
    backup.saving = false;
  }
};

const createBackupNow = async () => {
  backup.creating = true;
  backup.msg = "";
  try {
    const info = await adminApi.createBackup();
    backup.lastBackupAt = info.createdAt;
    flash(backup, `备份已创建：${info.name}`, true);
    backupList.value = await adminApi.listBackups();
  } catch (err: any) {
    flash(backup, err?.response?.data?.error || "备份失败", false);
  } finally {
    backup.creating = false;
  }
};

const openRestoreModal = (name: string) => {
  restoreModal.name = name;
  restoreModal.password = "";
  restoreModal.step = 1;
  restoreModal.msg = "";
  restoreModal.running = false;
  restoreModal.visible = true;
};

const closeRestoreModal = () => {
  if (restoreModal.running) return;
  restoreModal.visible = false;
  restoreModal.password = "";
  restoreModal.msg = "";
};

const confirmRestore = async () => {
  if (restoreModal.step === 1) {
    restoreModal.step = 2;
    restoreModal.msg = "";
    return;
  }
  // step 2: execute
  restoreModal.running = true;
  restoreModal.msg = "";
  try {
    await adminApi.restoreBackup(restoreModal.name, restoreModal.password);
    flash(backup, `已从 ${restoreModal.name} 恢复成功`, true);
    restoreModal.visible = false;
    restoreModal.password = "";
    backupList.value = await adminApi.listBackups();
  } catch (err: any) {
    restoreModal.msg = err?.response?.data?.error || "恢复失败";
    restoreModal.ok = false;
  } finally {
    restoreModal.running = false;
  }
};

const formatSize = (bytes: number) => {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / 1024 / 1024).toFixed(1) + " MB";
};

const formatTime = (iso: string) => {
  if (!iso) return "-";
  const d = new Date(iso);
  return d.toLocaleString();
};

onMounted(load);
</script>

<template>
  <section class="page-scroll">
    <div class="page-header">
      <h1>系统配置</h1>
      <p class="muted">管理全局设置，如实习生名额或 Season 配置。</p>
    </div>

    <div v-if="loading" class="table-empty">加载中...</div>
    <div v-else-if="loadError" class="panel error">{{ loadError }}</div>
    <div v-else class="settings-grid">
      <!-- 自动同步 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>自动同步</h3>
          <p class="muted">定时同步已分配 Agent 的链上数据。</p>
        </div>
        <div class="setting-card-body">
          <div class="grid-2">
            <label>
              同步间隔（秒）
              <input
                v-model.number="sync.interval"
                type="number"
                min="0"
                placeholder="0 = 禁用"
              />
            </label>
            <label>
              HL 并发数
              <input
                v-model.number="sync.hlConcurrency"
                type="number"
                min="1"
                max="50"
                placeholder="5"
              />
            </label>
          </div>
        </div>
        <div class="setting-card-foot">
          <span v-if="sync.msg" :class="sync.ok ? 'success' : 'error'">{{
            sync.msg
          }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveSyncSettings"
            :disabled="sync.saving"
          >
            {{ sync.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- 合约配置 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>合约配置</h3>
          <p class="muted">配置 EVM RPC 和 Allocator 合约地址。</p>
        </div>
        <div class="setting-card-body">
          <label>
            RPC URL
            <input
              v-model="contracts.rpcURL"
              placeholder="https://rpc.hyperliquid.xyz/evm"
            />
          </label>
          <label>
            Allocator 地址
            <input v-model="contracts.allocatorAddress" placeholder="0x..." />
          </label>
        </div>
        <div class="setting-card-foot">
          <span
            v-if="contracts.msg"
            :class="contracts.ok ? 'success' : 'error'"
            >{{ contracts.msg }}</span
          >
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveContractsSettings"
            :disabled="contracts.saving"
          >
            {{ contracts.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- 派遣命令 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>派遣命令</h3>
          <p class="muted">配置 Agent 派遣执行的命令模板。</p>
        </div>
        <div class="setting-card-body">
          <label>
            命令模板
            <textarea
              v-model="dispatch.command"
              rows="3"
              placeholder="python bot.py --key #prikey# --addr #pubkey# --vault #agentvaultaddr#"
              style="width:100%;font-family:monospace;resize:vertical"
            ></textarea>
          </label>
          <p class="muted" style="font-size:0.85em;margin-top:4px">
            可用占位符：#prikey#（私钥）、#pubkey#（公钥）、#agentvaultaddr#（AgentVault 地址）
          </p>
        </div>
        <div class="setting-card-foot">
          <span
            v-if="dispatch.msg"
            :class="dispatch.ok ? 'success' : 'error'"
            >{{ dispatch.msg }}</span
          >
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveDispatchSettings"
            :disabled="dispatch.saving"
          >
            {{ dispatch.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- 实习生名额 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>实习生名额</h3>
          <p class="muted">
            已消耗 {{ internSlots.consumed }} / {{ internSlots.total }}，剩余 {{ internSlots.remaining }}
          </p>
        </div>
        <div class="setting-card-body">
          <label>
            总名额
            <input v-model.number="internSlots.total" type="number" min="1" />
          </label>
        </div>
        <div class="setting-card-foot">
          <span v-if="internSlots.msg" :class="internSlots.ok ? 'success' : 'error'">{{
            internSlots.msg
          }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveInternSlots"
            :disabled="internSlots.saving"
          >
            {{ internSlots.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- TVL 偏移量 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>TVL 偏移量</h3>
          <p class="muted">公开 API 输出总 TVL 时加上该偏移，不影响数据库真实数据。</p>
        </div>
        <div class="setting-card-body">
          <label>
            偏移量 (USDC)
            <input v-model.number="tvlOffset.value" type="number" step="any" />
          </label>
        </div>
        <div class="setting-card-foot">
          <span v-if="tvlOffset.msg" :class="tvlOffset.ok ? 'success' : 'error'">{{
            tvlOffset.msg
          }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveTvlOffset"
            :disabled="tvlOffset.saving"
          >
            {{ tvlOffset.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- X OAuth -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>X OAuth</h3>
          <p class="muted">配置 Twitter/X OAuth 2.0 凭据。</p>
        </div>
        <div class="setting-card-body">
          <label>
            Client ID
            <input v-model="xoauth.clientId" placeholder="Client ID" />
          </label>
          <label>
            Client Secret
            <input
              v-model="xoauth.clientSecret"
              type="password"
              placeholder="Client Secret"
            />
          </label>
          <label>
            Scopes
            <input
              v-model="xoauth.scopes"
              placeholder="users.read tweet.read offline.access"
            />
          </label>
        </div>
        <div class="setting-card-foot">
          <span v-if="xoauth.msg" :class="xoauth.ok ? 'success' : 'error'">{{
            xoauth.msg
          }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-primary btn-sm"
            @click="saveXOAuthSettings"
            :disabled="xoauth.saving"
          >
            {{ xoauth.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- 数据库备份 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>数据库备份</h3>
          <p class="muted">
            自动备份间隔与保留策略。上次备份：{{ backup.lastBackupAt ? formatTime(backup.lastBackupAt) : "从未" }}
          </p>
        </div>
        <div class="setting-card-body">
          <label>
            备份间隔（小时）
            <input
              v-model.number="backup.intervalHours"
              type="number"
              min="0"
              placeholder="0 = 禁用"
            />
          </label>
          <p class="muted" style="font-size:0.85em;margin:8px 0 4px">保留策略</p>
          <div class="grid-3">
            <label>
              最近保留数
              <input
                v-model.number="backup.retainHourly"
                type="number"
                min="1"
                placeholder="3"
              />
            </label>
            <label>
              每天保留数
              <input
                v-model.number="backup.retainDaily"
                type="number"
                min="1"
                placeholder="3"
              />
            </label>
            <label>
              每周保留数
              <input
                v-model.number="backup.retainWeekly"
                type="number"
                min="1"
                placeholder="3"
              />
            </label>
          </div>
        </div>
        <div class="setting-card-foot">
          <span v-if="backup.msg" :class="backup.ok ? 'success' : 'error'">{{ backup.msg }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-sm"
            @click="createBackupNow"
            :disabled="backup.creating"
          >
            {{ backup.creating ? "备份中..." : "立即备份" }}
          </button>
          <button
            class="btn btn-primary btn-sm"
            @click="saveBackupSettings"
            :disabled="backup.saving"
          >
            {{ backup.saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>

      <!-- 备份列表 -->
      <div class="setting-card" v-if="backupList.length > 0">
        <div class="setting-card-head">
          <h3>备份列表</h3>
          <p class="muted">共 {{ backupList.length }} 个备份文件</p>
        </div>
        <div class="setting-card-body">
          <div class="backup-table-wrap">
            <table class="backup-table">
              <thead>
                <tr>
                  <th>文件名</th>
                  <th>大小</th>
                  <th>创建时间</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="b in backupList" :key="b.name">
                  <td style="font-family:monospace;font-size:0.85em">{{ b.name }}</td>
                  <td>{{ formatSize(b.size) }}</td>
                  <td>{{ formatTime(b.createdAt) }}</td>
                  <td>
                    <button class="btn btn-danger btn-xs" @click="openRestoreModal(b.name)">恢复</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- 恢复确认弹窗 -->
    <Teleport to="body">
      <div v-if="restoreModal.visible" class="modal-overlay" @click.self="closeRestoreModal">
        <div class="modal-box">
          <!-- step 1: 首次确认 -->
          <template v-if="restoreModal.step === 1">
            <h3>确认恢复备份</h3>
            <p style="margin:12px 0">
              即将从以下备份恢复数据库，当前数据库将被替换：
            </p>
            <p style="font-family:monospace;font-size:0.9em;background:var(--hover-bg,#f3f4f6);padding:8px 12px;border-radius:6px">
              {{ restoreModal.name }}
            </p>
            <p style="margin:12px 0;color:#dc2626;font-weight:500">
              此操作不可撤销（系统会自动创建恢复前安全备份）。
            </p>
            <div class="modal-actions">
              <button class="btn btn-sm" @click="closeRestoreModal">取消</button>
              <button class="btn btn-danger btn-sm" @click="confirmRestore">继续</button>
            </div>
          </template>

          <!-- step 2: 输入密码二次确认 -->
          <template v-else>
            <h3>输入管理员密码</h3>
            <p style="margin:12px 0">请输入您的管理员密码以执行恢复操作：</p>
            <label>
              <input
                v-model="restoreModal.password"
                type="password"
                placeholder="管理员密码"
                style="width:100%"
                @keyup.enter="confirmRestore"
              />
            </label>
            <p v-if="restoreModal.msg" class="error" style="margin-top:8px">{{ restoreModal.msg }}</p>
            <div class="modal-actions">
              <button class="btn btn-sm" @click="closeRestoreModal" :disabled="restoreModal.running">取消</button>
              <button
                class="btn btn-danger btn-sm"
                @click="confirmRestore"
                :disabled="!restoreModal.password || restoreModal.running"
              >
                {{ restoreModal.running ? "恢复中..." : "确认恢复" }}
              </button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.grid-3 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.backup-table-wrap {
  overflow-x: auto;
}
.backup-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9em;
}
.backup-table th,
.backup-table td {
  padding: 6px 10px;
  text-align: left;
  border-bottom: 1px solid var(--border, #e5e7eb);
}
.backup-table tbody tr:hover {
  background: var(--hover-bg, #f9fafb);
}
.btn-danger {
  background: #dc2626;
  color: #fff;
  border-color: #dc2626;
}
.btn-danger:hover:not(:disabled) {
  background: #b91c1c;
}
.btn-danger:disabled {
  opacity: 0.5;
}
.btn-xs {
  padding: 2px 10px;
  font-size: 0.8em;
}
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-box {
  background: var(--card-bg, #fff);
  border-radius: 12px;
  padding: 24px;
  width: 100%;
  max-width: 440px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.2);
}
.modal-box h3 {
  margin: 0;
  font-size: 1.1em;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}
</style>
