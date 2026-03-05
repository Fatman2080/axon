<script setup lang="ts">
import { onMounted, ref, reactive } from "vue";
import { adminApi } from "../api/client";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const loading = ref(true);
const loadError = ref("");

const sync = reactive({ interval: 0, saving: false, msg: "", ok: true });
const contracts = reactive({
  rpcURL: "",
  allocatorAddress: "",
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
    const [syncData, xoauthData, contractsData, slotsData] = await Promise.all([
      adminApi.getSyncSettings(),
      adminApi.getXOAuthSettings(),
      adminApi.getContractsSettings(),
      adminApi.getDailySlotsSettings(),
    ]);
    sync.interval = syncData.intervalSeconds;
    xoauth.clientId = xoauthData.clientId || "";
    xoauth.clientSecret = xoauthData.clientSecret || "";
    xoauth.scopes = xoauthData.scopes || "";
    contracts.rpcURL = contractsData.rpcURL || "";
    contracts.allocatorAddress = contractsData.allocatorAddress || "";
    slots.total = slotsData.total;
    slots.resetHour = slotsData.resetHour;
    slots.consumed = slotsData.consumed;
    slots.remaining = slotsData.remaining;
  } catch (err: any) {
    loadError.value = err?.response?.data?.error || t("settings.loadError");
  } finally {
    loading.value = false;
  }
};

const saveSyncSettings = async () => {
  sync.saving = true;
  sync.msg = "";
  try {
    const r = await adminApi.updateSyncSettings(sync.interval);
    sync.interval = r.intervalSeconds;
    flash(
      sync,
      r.intervalSeconds > 0
        ? t("settings.syncIntervalSuccess", { interval: r.intervalSeconds })
        : t("settings.syncDisabled"),
      true,
    );
  } catch (err: any) {
    flash(sync, err?.response?.data?.error || t("common.saveError"), false);
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
    flash(contracts, t("settings.contractsSuccess"), true);
  } catch (err: any) {
    flash(
      contracts,
      err?.response?.data?.error || t("common.saveError"),
      false,
    );
  } finally {
    contracts.saving = false;
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
    flash(slots, t("settings.configUpdated"), true);
  } catch (err: any) {
    flash(slots, err?.response?.data?.error || t("common.saveError"), false);
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
    flash(slots, t("settings.slotsResetSuccess"), true);
  } catch (err: any) {
    flash(
      slots,
      err?.response?.data?.error || t("settings.slotsResetError"),
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
    flash(xoauth, t("settings.configUpdated"), true);
  } catch (err: any) {
    flash(xoauth, err?.response?.data?.error || t("common.saveError"), false);
  } finally {
    xoauth.saving = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page-scroll">
    <div class="page-header">
      <h1>{{ t("settings.title") }}</h1>
      <p class="muted">{{ t("settings.desc") }}</p>
    </div>

    <div v-if="loading" class="table-empty">{{ t("common.loading") }}</div>
    <div v-else-if="loadError" class="panel error">{{ loadError }}</div>
    <div v-else class="settings-grid">
      <!-- 自动同步 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>{{ t("settings.syncTitle") }}</h3>
          <p class="muted">{{ t("settings.syncDesc") }}</p>
        </div>
        <div class="setting-card-body">
          <label>
            {{ t("settings.syncInterval") }}
            <input
              v-model.number="sync.interval"
              type="number"
              min="0"
              placeholder="0 = 禁用"
            />
          </label>
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
            {{ sync.saving ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </div>

      <!-- 合约配置 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>{{ t("settings.contractsTitle") }}</h3>
          <p class="muted">{{ t("settings.contractsDesc") }}</p>
        </div>
        <div class="setting-card-body">
          <label>
            {{ t("settings.rpcURL") }}
            <input
              v-model="contracts.rpcURL"
              placeholder="https://rpc.hyperliquid.xyz/evm"
            />
          </label>
          <label>
            {{ t("settings.allocatorAddress") }}
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
            {{ contracts.saving ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </div>

      <!-- 每日名额 -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>{{ t("settings.slotsTitle") }}</h3>
          <p class="muted">
            {{
              t("settings.slotsDesc", {
                consumed: slots.consumed,
                total: slots.total,
                remaining: slots.remaining,
              })
            }}
          </p>
        </div>
        <div class="setting-card-body">
          <div class="grid-2">
            <label>
              {{ t("settings.slotsTotal") }}
              <input v-model.number="slots.total" type="number" min="1" />
            </label>
            <label>
              {{ t("settings.slotsResetHour") }}
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
            {{ slots.resetting ? "..." : t("settings.slotsResetBtn") }}
          </button>
          <button
            class="btn btn-primary btn-sm"
            @click="saveDailySlotsSettings"
            :disabled="slots.saving"
          >
            {{ slots.saving ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </div>

      <!-- X OAuth -->
      <div class="setting-card">
        <div class="setting-card-head">
          <h3>{{ t("settings.xoauthTitle") }}</h3>
          <p class="muted">{{ t("settings.xoauthDesc") }}</p>
        </div>
        <div class="setting-card-body">
          <label>
            {{ t("settings.clientId") }}
            <input v-model="xoauth.clientId" placeholder="Client ID" />
          </label>
          <label>
            {{ t("settings.clientSecret") }}
            <input
              v-model="xoauth.clientSecret"
              type="password"
              placeholder="Client Secret"
            />
          </label>
          <label>
            {{ t("settings.scopes") }}
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
            {{ xoauth.saving ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
