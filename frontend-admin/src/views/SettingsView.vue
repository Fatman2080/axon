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
const slots = reactive({
  total: 1000,
  resetHour: 0,
  consumed: 0,
  remaining: 1000,
  saving: false,
  resetting: false,
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
    const [syncData, xoauthData, contractsData, slotsData, dispatchData] = await Promise.all([
      adminApi.getSyncSettings(),
      adminApi.getXOAuthSettings(),
      adminApi.getContractsSettings(),
      adminApi.getDailySlotsSettings(),
      adminApi.getDispatchSettings(),
    ]);
    sync.interval = syncData.intervalSeconds;
    sync.hlConcurrency = syncData.hlConcurrency || 5;
    xoauth.clientId = xoauthData.clientId || "";
    xoauth.clientSecret = xoauthData.clientSecret || "";
    xoauth.scopes = xoauthData.scopes || "";
    contracts.rpcURL = contractsData.rpcURL || "";
    contracts.allocatorAddress = contractsData.allocatorAddress || "";
    dispatch.command = dispatchData.command || "";
    slots.total = slotsData.total;
    slots.resetHour = slotsData.resetHour;
    slots.consumed = slotsData.consumed;
    slots.remaining = slotsData.remaining;
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

const saveDailySlotsSettings = async () => {
  slots.saving = true;
  slots.msg = "";
  try {
    const r = await adminApi.updateDailySlotsSettings({
      total: slots.total,
      resetHour: slots.resetHour,
    });
    slots.total = r.total;
    slots.resetHour = r.resetHour;
    slots.consumed = r.consumed;
    slots.remaining = r.remaining;
    flash(slots, "配置已更新", true);
  } catch (err: any) {
    flash(slots, err?.response?.data?.error || "保存失败", false);
  } finally {
    slots.saving = false;
  }
};

const resetDailySlotsConsumed = async () => {
  slots.resetting = true;
  slots.msg = "";
  try {
    const r = await adminApi.updateDailySlotsSettings({ resetConsumed: true });
    slots.consumed = r.consumed;
    slots.remaining = r.remaining;
    flash(slots, "今日消耗已重置", true);
  } catch (err: any) {
    flash(
      slots,
      err?.response?.data?.error || "重置失败",
      false,
    );
  } finally {
    slots.resetting = false;
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

      <!-- 每日名额 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>每日名额</h3>
          <p class="muted">
            已消耗 {{ slots.consumed }} / {{ slots.total }}，剩余 {{ slots.remaining }}
          </p>
        </div>
        <div class="setting-card-body">
          <div class="grid-2">
            <label>
              每日总名额
              <input v-model.number="slots.total" type="number" min="1" />
            </label>
            <label>
              重置小时 (UTC)
              <input
                v-model.number="slots.resetHour"
                type="number"
                min="0"
                max="23"
              />
            </label>
          </div>
        </div>
        <div class="setting-card-foot">
          <span v-if="slots.msg" :class="slots.ok ? 'success' : 'error'">{{
            slots.msg
          }}</span>
          <span class="toolbar-spacer"></span>
          <button
            class="btn btn-sm"
            @click="resetDailySlotsConsumed"
            :disabled="slots.resetting"
          >
            {{ slots.resetting ? "..." : "重置今日消耗" }}
          </button>
          <button
            class="btn btn-primary btn-sm"
            @click="saveDailySlotsSettings"
            :disabled="slots.saving"
          >
            {{ slots.saving ? "保存中..." : "保存" }}
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
    </div>
  </section>
</template>
