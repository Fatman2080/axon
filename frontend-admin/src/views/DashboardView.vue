<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { adminApi } from "../api/client";
import type { DashboardStats, TreasurySnapshot } from "../types";

const loading = ref(true);
const stats = ref<DashboardStats | null>(null);
const treasury = ref<TreasurySnapshot | null>(null);
const error = ref("");

const fmtUsd = (v: number) => {
  if (v === undefined || v === null) return "$0.00";
  return "$" + v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
};
const fmtPct = (v: number) => {
  if (v === undefined || v === null) return "0.00%";
  return (v * 100).toFixed(2) + "%";
};
const pnlCls = (v: number) => (v >= 0 ? "up" : "down");
const pnlSign = (v: number) => (v >= 0 ? "+" : "");
const formatLocalDateTime = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

// 按资金类型汇总（EVM / Perps / Spot，跨三个来源）
const totalEvm = computed(() => treasury.value ? treasury.value.vaultEvm + treasury.value.allocatorEvm + treasury.value.ownerEvm : 0);
const totalPerps = computed(() => treasury.value ? treasury.value.vaultPerps + treasury.value.allocatorPerps + treasury.value.ownerPerps : 0);
const totalSpot = computed(() => treasury.value ? treasury.value.vaultSpot + treasury.value.allocatorSpot + treasury.value.ownerSpot : 0);

const load = async () => {
  loading.value = true;
  error.value = "";
  try {
    const [s, t] = await Promise.all([
      adminApi.dashboard(),
      adminApi.treasury().catch(() => null),
    ]);
    stats.value = s;
    treasury.value = t;
  } catch (err: any) {
    error.value = err?.response?.data?.error || "加载仪表盘数据失败";
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>

<template>
  <section class="page-scroll">
    <div class="page-header">
      <h1>仪表盘</h1>
      <p class="muted">平台总览与关键指标。</p>
    </div>

    <div v-if="loading" class="table-empty">加载中...</div>
    <div v-else-if="error" class="panel error">{{ error }}</div>
    <template v-else-if="stats">

      <!-- 资金概览：总资产 + EVM/Perps/Spot 分类 + 盈亏 -->
      <div v-if="treasury && treasury.totalFunds !== undefined" class="section">
        <h2>资金概览</h2>
        <div class="grid-5">
          <div class="stat-card accent">
            <span class="stat-label">总资产</span>
            <span class="stat-num lg">{{ fmtUsd(treasury.totalFunds) }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">EVM 总额</span>
            <span class="stat-num">{{ fmtUsd(totalEvm) }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">Perps 总额</span>
            <span class="stat-num">{{ fmtUsd(totalPerps) }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">Spot 总额</span>
            <span class="stat-num">{{ fmtUsd(totalSpot) }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">总盈亏</span>
            <span class="stat-num" :class="pnlCls(treasury.vaultPnl)">{{ pnlSign(treasury.vaultPnl) }}{{ fmtUsd(treasury.vaultPnl) }}</span>
          </div>
        </div>
      </div>

      <!-- 平台统计 -->
      <div class="section">
        <h2>平台统计</h2>
        <div class="grid-4">
          <div class="stat-card">
            <span class="stat-label">总用户数</span>
            <span class="stat-num">{{ stats.totalUsers }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">Agent 总量</span>
            <span class="stat-num">{{ stats.totalAgentAccounts }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">已分配 Agent</span>
            <span class="stat-num">{{ stats.assignedAgents }}</span>
          </div>
          <div class="stat-card">
            <span class="stat-label">未使用 Agent</span>
            <span class="stat-num">{{ stats.unusedAgents }}</span>
          </div>
        </div>
      </div>

      <!-- 用户增长 + 邀请码 + 系统同步 -->
      <div class="section">
        <div class="grid-3-block">
          <div class="block-card">
            <h3>用户增长</h3>
            <div class="kv-list">
              <div class="kv"><span>今日新增</span><b>{{ stats.newUsersToday }}</b></div>
              <div class="kv"><span>本周新增</span><b>{{ stats.newUsersWeek }}</b></div>
              <div class="kv"><span>Agent 转化率</span><b>{{ fmtPct(stats.conversionRate) }}</b></div>
            </div>
          </div>
          <div class="block-card">
            <h3>邀请码</h3>
            <div class="kv-list">
              <div class="kv"><span>总量</span><b>{{ stats.totalInviteCodes }}</b></div>
              <div class="kv"><span>有效</span><b>{{ stats.activeInviteCodes }}</b></div>
              <div class="kv"><span>已使用率</span><b>{{ fmtPct(stats.inviteConversionRate) }}</b></div>
            </div>
          </div>
          <div class="block-card">
            <h3>系统同步</h3>
            <div class="kv-list">
              <div class="kv"><span>同步轮次</span><b>{{ stats.syncRoundCount }}</b></div>
              <div class="kv"><span>数据新鲜度</span><b>{{ stats.dataFreshness > 0 ? Math.round(stats.dataFreshness / 60) + ' 分钟前' : '-' }}</b></div>
              <div class="kv"><span>最后同步</span><b class="mono">{{ formatLocalDateTime(stats.lastSyncAt) }}</b></div>
            </div>
          </div>
        </div>
      </div>

      <!-- 国库明细：按来源 × 资金类型 -->
      <div v-if="treasury && treasury.totalFunds !== undefined" class="section">
        <h2>国库明细 <span v-if="treasury.createdAt" class="muted small">{{ formatLocalDateTime(treasury.createdAt) }}</span></h2>
        <div class="treasury-sources">
          <!-- AgentVaults -->
          <div class="block-card">
            <h3>AgentVaults</h3>
            <div class="kv-list">
              <div class="kv"><span>EVM</span><b>{{ fmtUsd(treasury.vaultEvm) }}</b></div>
              <div class="kv"><span>Perps</span><b>{{ fmtUsd(treasury.vaultPerps) }}</b></div>
              <div class="kv"><span>Spot</span><b>{{ fmtUsd(treasury.vaultSpot) }}</b></div>
              <div class="kv kv-total"><span>小计</span><b>{{ fmtUsd(treasury.vaultEvm + treasury.vaultPerps + treasury.vaultSpot) }}</b></div>
            </div>
            <div class="block-meta">
              <span>活跃 {{ treasury.activeVaultCount }} / 总计 {{ treasury.vaultCount }}</span>
              <span :class="pnlCls(treasury.vaultPnl)">盈亏 {{ pnlSign(treasury.vaultPnl) }}{{ fmtUsd(treasury.vaultPnl) }}</span>
            </div>
          </div>
          <!-- Allocator -->
          <div class="block-card">
            <h3>Allocator</h3>
            <div class="kv-list">
              <div class="kv"><span>EVM</span><b>{{ fmtUsd(treasury.allocatorEvm) }}</b></div>
              <div class="kv"><span>Perps</span><b>{{ fmtUsd(treasury.allocatorPerps) }}</b></div>
              <div class="kv"><span>Spot</span><b>{{ fmtUsd(treasury.allocatorSpot) }}</b></div>
              <div class="kv kv-total"><span>小计</span><b>{{ fmtUsd(treasury.allocatorEvm + treasury.allocatorPerps + treasury.allocatorSpot) }}</b></div>
            </div>
          </div>
          <!-- Owner -->
          <div class="block-card">
            <h3>Owner</h3>
            <div class="kv-list">
              <div class="kv"><span>EVM</span><b>{{ fmtUsd(treasury.ownerEvm) }}</b></div>
              <div class="kv"><span>Perps</span><b>{{ fmtUsd(treasury.ownerPerps) }}</b></div>
              <div class="kv"><span>Spot</span><b>{{ fmtUsd(treasury.ownerSpot) }}</b></div>
              <div class="kv kv-total"><span>小计</span><b>{{ fmtUsd(treasury.ownerEvm + treasury.ownerPerps + treasury.ownerSpot) }}</b></div>
            </div>
          </div>
        </div>
      </div>

    </template>
  </section>
</template>

<style scoped>
.section { margin-bottom: 1rem; }
.section h2 {
  font-size: 0.95rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

/* ---- grids ---- */
.grid-5 {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 0.6rem;
}
.grid-4 {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.6rem;
}
.grid-3-block {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.6rem;
}
.treasury-sources {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.6rem;
}
@media (max-width: 1100px) {
  .grid-5 { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 900px) {
  .grid-5 { grid-template-columns: repeat(2, 1fr); }
  .grid-4 { grid-template-columns: repeat(2, 1fr); }
  .grid-3-block { grid-template-columns: 1fr; }
  .treasury-sources { grid-template-columns: 1fr; }
}

/* ---- stat card ---- */
.stat-card {
  background: #fff;
  border: 1px solid #eaecf0;
  border-radius: 10px;
  padding: 0.65rem 0.8rem;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}
.stat-card.accent {
  border-left: 3px solid #22c55e;
}
.stat-label {
  font-size: 0.72rem;
  color: #667085;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  font-weight: 500;
}
.stat-num {
  font-size: 1.15rem;
  font-weight: 700;
  color: #101828;
  line-height: 1.25;
}
.stat-num.lg {
  font-size: 1.4rem;
}

/* ---- block card ---- */
.block-card {
  background: #fff;
  border: 1px solid #eaecf0;
  border-radius: 10px;
  padding: 0.7rem 0.85rem;
}
.block-card h3 {
  font-size: 0.82rem;
  font-weight: 600;
  color: #344054;
  margin-bottom: 0.45rem;
}
.block-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 0.4rem;
  padding-top: 0.35rem;
  border-top: 1px solid #f2f4f7;
  font-size: 0.72rem;
  color: #667085;
}

/* ---- kv list ---- */
.kv-list { display: flex; flex-direction: column; }
.kv {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.8rem;
  color: #344054;
  padding: 0.25rem 0;
  border-bottom: 1px solid #f2f4f7;
}
.kv:last-child { border-bottom: none; }
.kv span { color: #667085; }
.kv b { font-weight: 600; color: #101828; }
.kv-total {
  border-top: 1px solid #eaecf0;
  border-bottom: none;
  padding-top: 0.3rem;
  margin-top: 0.1rem;
}
.kv-total span { font-weight: 500; color: #344054; }

/* ---- colors ---- */
.up { color: #067647; }
.down { color: #b42318; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 0.75rem; }
</style>
