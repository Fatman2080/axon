import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchAgents } from '../store/slices/agentSlice';
import { fetchUser } from '../store/slices/userSlice';
import { Info } from 'lucide-react';
import { Line } from 'react-chartjs-2';
import { useLanguage } from '../context/LanguageContext';
import { authApi, marketApi } from '../services/api';
import { VaultOverview } from '../types';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler);

const Agents = () => {
  const dispatch = useAppDispatch();
  const { currentUser: user } = useAppSelector((state) => state.user);
  const { t } = useLanguage();
  const [actionTab, setActionTab] = useState('deposit');
  const [amount, setAmount] = useState('');
  const [vaultOverview, setVaultOverview] = useState<VaultOverview | null>(null);
  const [agentHistory, setAgentHistory] = useState<number[]>([]);
  const [chartPeriod, setChartPeriod] = useState('1M');

  useEffect(() => {
    dispatch(fetchAgents());
    dispatch(fetchUser());
    marketApi.getVaultOverview().then(setVaultOverview).catch(() => {
      marketApi.getVaultStats().then(stats => {
        setVaultOverview({
          totalTvl: stats.totalTvl,
          totalEvmBalance: stats.totalEvmBalance,
          totalL1Value: stats.totalL1Value,
          agentCount: stats.agentCount,
          totalPnl: 0,
          positions: [],
          recentFills: [],
        });
      }).catch(() => {});
    });
  }, [dispatch]);

  useEffect(() => {
    if (!user) return;
    const loadAgentData = async () => {
      try {
        const data = await authApi.getAgentHistory(chartPeriod);
        setAgentHistory(Array.isArray(data.history) ? data.history : []);
      } catch {
        setAgentHistory([]);
      }
    };
    loadAgentData();
  }, [user, chartPeriod]);

  const totalTvl = vaultOverview?.totalTvl ?? 0;
  const totalPnl = vaultOverview?.totalPnl ?? 0;
  const agentCount = vaultOverview?.agentCount ?? 0;

  const sharePrice = agentHistory.length > 0
    ? agentHistory[agentHistory.length - 1]
    : totalTvl > 0 ? 1 + totalTvl / 1000000 * 0.2 : 1.0;

  let maxDrawdown = 0;
  let sharpeRatio = 0;
  if (agentHistory.length > 1) {
    let peak = agentHistory[0];
    for (const v of agentHistory) {
      if (v > peak) peak = v;
      const dd = peak > 0 ? ((peak - v) / peak) * 100 : 0;
      if (dd > maxDrawdown) maxDrawdown = dd;
    }
    const returns = agentHistory.slice(1).map((v, i) => (v - agentHistory[i]) / agentHistory[i]);
    const avgReturn = returns.reduce((a, b) => a + b, 0) / returns.length;
    const stdReturn = Math.sqrt(returns.reduce((a, b) => a + (b - avgReturn) ** 2, 0) / returns.length);
    sharpeRatio = stdReturn > 0 ? (avgReturn / stdReturn) * Math.sqrt(252) : 0;
  }

  const apy = totalTvl > 0 && totalPnl !== 0 ? (totalPnl / (totalTvl - totalPnl)) * 100 : 0;
  const chartSeries = agentHistory.length > 0 ? agentHistory : [];

  const chartData = {
    labels: Array.from({ length: chartSeries.length }, (_, i) => `Day ${i + 1}`),
    datasets: [{
      label: t('vault.sharePrice'),
      data: chartSeries,
      borderColor: '#00FF41',
      backgroundColor: 'rgba(0, 240, 255, 0.06)',
      tension: 0.4,
      fill: true,
      pointRadius: 0,
      pointHoverRadius: 4,
    }],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
        backgroundColor: '#16181A',
        titleColor: '#8E929B',
        bodyColor: '#E8EAED',
        borderColor: 'rgba(255,255,255,0.07)',
        borderWidth: 1,
      },
    },
    scales: {
      x: { display: false, grid: { display: false } },
      y: {
        display: true,
        position: 'right' as const,
        grid: { color: 'rgba(255,255,255,0.04)' },
        ticks: {
          callback: (value: number | string) => `$${Number(value).toFixed(2)}`,
          font: { family: 'JetBrains Mono, monospace' },
          color: '#555B66',
        },
      },
    },
    interaction: { mode: 'nearest' as const, axis: 'x' as const, intersect: false },
  };

  const metricCards = [
    { label: t('vault.maxDrawdown'), value: `-${maxDrawdown.toFixed(2)}%`, color: 'var(--red)' },
    { label: t('vault.sharpeRatio'), value: sharpeRatio.toFixed(2), color: 'var(--text-primary)' },
    { label: t('vault.depositors'), value: agentCount > 0 ? `${agentCount}` : '0', color: 'var(--text-primary)' },
    { label: t('vault.managerFee'), value: '20%', color: 'var(--text-primary)' },
  ];

  return (
    <div className="pb-20 animate-fade-in-up">
      {/* Header */}
      <div className="mb-8">
        <div className="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-2xl font-bold tracking-tight" style={{ color: 'var(--text-primary)' }}>
                {t('vault.title')}
              </h1>
              <span
                className="font-mono text-xs px-2 py-0.5 rounded"
                style={{
                  background: 'rgba(255,255,255,0.04)',
                  border: '1px solid var(--border)',
                  color: 'var(--text-secondary)',
                }}
              >
                {agentCount > 0 ? `${agentCount} agents` : 'No agents'}
              </span>
            </div>
            <div className="flex items-center gap-2 text-sm" style={{ color: 'var(--text-tertiary)' }}>
              <span className="flex items-center gap-1.5">
                <span className="status-dot" />
                {t('vault.status')}
              </span>
              <span>•</span>
              <span>{t('vault.createdBy')}</span>
            </div>
          </div>

          <div className="flex gap-8">
            {/* TVL */}
            <div className="relative group cursor-help">
              <div className="flex items-center gap-1.5 text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                {t('vault.equity')} <Info size={12} style={{ color: 'var(--text-tertiary)' }} />
              </div>
              <div className="text-2xl font-bold font-mono" style={{ color: 'var(--neon-green)' }}>
                ${totalTvl.toLocaleString(undefined, { maximumFractionDigits: 0 })}
              </div>

              {/* Hover Breakdown */}
              <div
                className="absolute top-full right-0 mt-2 w-72 rounded p-4 z-50 invisible group-hover:visible opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0 transition-all duration-150 shadow-2xl"
                style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
              >
                <div className="text-xs font-bold font-mono uppercase tracking-widest mb-3 pb-2" style={{ color: 'var(--text-tertiary)', borderBottom: '1px solid var(--border)' }}>
                  {t('vault.breakdown.title')}
                </div>
                <div className="space-y-3">
                  {[
                    { label: t('vault.breakdown.chain'), val: vaultOverview?.totalEvmBalance ?? 0, color: 'var(--tier-analyst)' },
                    { label: t('vault.breakdown.hype'), val: vaultOverview?.totalL1Value ?? 0, color: 'var(--green)' },
                  ].map(b => (
                    <div key={b.label} className="flex justify-between text-xs items-center">
                      <span className="flex items-center gap-2" style={{ color: 'var(--text-secondary)' }}>
                        <span className="w-2 h-2 rounded-full" style={{ background: b.color }} />
                        {b.label}
                      </span>
                      <span className="font-mono font-medium" style={{ color: 'var(--text-primary)' }}>
                        ${b.val.toLocaleString(undefined, { maximumFractionDigits: 2 })}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* APY */}
            <div>
              <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                {t('vault.apy')}
              </div>
              <div className="text-2xl font-bold font-mono" style={{ color: 'var(--green)' }}>
                {apy.toFixed(1)}%
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left: Chart + Stats */}
        <div className="lg:col-span-2 space-y-6">
          {/* Chart */}
          <div
            className="h-[380px] rounded p-6"
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
          >
            <div className="mb-4 flex items-center justify-between">
              <div>
                <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                  {t('vault.sharePrice')}
                </div>
                <div className="text-xl font-bold font-mono" style={{ color: 'var(--text-primary)' }}>
                  ${sharePrice.toFixed(4)}
                </div>
              </div>
              <div className="flex gap-1">
                {['1D', '1W', '1M', 'ALL'].map(p => (
                  <button
                    key={p}
                    onClick={() => setChartPeriod(p)}
                    className="px-3 py-1 text-xs font-mono font-medium rounded transition-all"
                    style={{
                      background: chartPeriod === p ? 'rgba(0,240,255,0.12)' : 'transparent',
                      color: chartPeriod === p ? 'var(--neon-green)' : 'var(--text-tertiary)',
                      border: chartPeriod === p ? '1px solid rgba(0,240,255,0.2)' : '1px solid var(--border)',
                    }}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
            <div className="h-[280px] w-full">
              {chartSeries.length > 0 ? (
                <Line data={chartData} options={chartOptions} />
              ) : (
                <div className="h-full w-full flex items-center justify-center text-sm font-mono" style={{ color: 'var(--text-tertiary)' }}>
                  // NO_DATA — awaiting agent telemetry
                </div>
              )}
            </div>
          </div>

          {/* Metrics */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {metricCards.map(m => (
              <div key={m.label} className="rounded p-4" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
                <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                  {m.label}
                </div>
                <div className="text-lg font-bold font-mono" style={{ color: m.color }}>
                  {m.value}
                </div>
              </div>
            ))}
          </div>

          {/* Description */}
          <div className="rounded p-6" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
              {t('vault.description.title')}
            </h3>
            <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
              {t('vault.description.content')}
            </p>
          </div>
        </div>

        {/* Right: Action Panel */}
        <div className="lg:col-span-1">
          <div className="rounded overflow-hidden sticky top-20" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
            {/* Tab buttons */}
            <div className="flex" style={{ borderBottom: '1px solid var(--border)' }}>
              {['deposit', 'withdraw'].map(tab => (
                <button
                  key={tab}
                  onClick={() => setActionTab(tab)}
                  className="flex-1 py-3.5 text-sm font-bold font-mono uppercase tracking-wider transition-all"
                  style={{
                    background: actionTab === tab ? 'rgba(0,240,255,0.06)' : 'transparent',
                    color: actionTab === tab ? 'var(--neon-green)' : 'var(--text-tertiary)',
                    borderBottom: actionTab === tab ? '2px solid var(--neon-green)' : '2px solid transparent',
                  }}
                >
                  {tab === 'deposit' ? t('vault.actions.deposit') : t('vault.actions.withdraw')}
                </button>
              ))}
            </div>

            <div className="p-6 space-y-5">
              <div>
                <div className="flex justify-between text-xs font-mono mb-2" style={{ color: 'var(--text-tertiary)' }}>
                  <span>{t('vault.actions.asset')}</span>
                  <span>{t('vault.actions.balance')}: {user?.lpValue ? user.lpValue.toLocaleString(undefined, { maximumFractionDigits: 2 }) : '0.00'} USDC</span>
                </div>
                <div className="relative">
                  <input
                    type="number"
                    placeholder="0.00"
                    disabled
                    className="w-full px-4 py-3 pr-16 text-lg font-mono cursor-not-allowed opacity-50"
                    style={{
                      background: 'var(--bg-input)',
                      border: '1px solid var(--border)',
                      color: 'var(--text-primary)',
                      borderRadius: '4px',
                    }}
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                  />
                  <div className="absolute right-4 top-1/2 -translate-y-1/2 text-sm font-mono" style={{ color: 'var(--text-tertiary)' }}>
                    USDC
                  </div>
                </div>
                <div className="flex gap-1.5 mt-2">
                  {[25, 50, 75, 100].map(pct => (
                    <button
                      key={pct}
                      disabled
                      className="flex-1 py-1 text-xs font-mono rounded cursor-not-allowed opacity-40"
                      style={{ background: 'var(--bg-input)', border: '1px solid var(--border)', color: 'var(--text-tertiary)' }}
                    >
                      {pct}%
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-2.5 py-4" style={{ borderTop: '1px solid var(--border)', borderBottom: '1px solid var(--border)' }}>
                {[
                  { label: t('vault.actions.exchangeRate'), value: `1 HLP = ${sharePrice.toFixed(4)} USDC` },
                  { label: t('vault.actions.estReceive'), value: '0.00 HLP' },
                ].map(row => (
                  <div key={row.label} className="flex justify-between text-sm">
                    <span style={{ color: 'var(--text-secondary)' }}>{row.label}</span>
                    <span className="font-mono" style={{ color: 'var(--text-primary)' }}>{row.value}</span>
                  </div>
                ))}
              </div>

              <button
                disabled
                className="w-full py-3.5 text-sm font-bold font-mono uppercase tracking-wider rounded cursor-not-allowed opacity-40"
                style={{ background: 'var(--bg-input)', border: '1px solid var(--border)', color: 'var(--text-secondary)' }}
              >
                {t('vault.actions.comingSoon')}
              </button>

              <div className="text-xs text-center font-mono" style={{ color: 'var(--text-tertiary)' }}>
                {t('vault.actions.terms')}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Agents;
