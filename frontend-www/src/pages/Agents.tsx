import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchAgents } from '../store/slices/agentSlice';
import { fetchUser } from '../store/slices/userSlice';
import {
  Info
} from 'lucide-react';
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

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

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
      // Fallback to basic stats
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

  // Compute share price from history
  const sharePrice = agentHistory.length > 0
    ? agentHistory[agentHistory.length - 1]
    : totalTvl > 0 ? 1 + totalTvl / 1000000 * 0.2 : 1.0;

  // Compute stats from history
  let maxDrawdown = 0;
  let sharpeRatio = 0;
  if (agentHistory.length > 1) {
    let peak = agentHistory[0];
    for (const v of agentHistory) {
      if (v > peak) peak = v;
      const dd = peak > 0 ? ((peak - v) / peak) * 100 : 0;
      if (dd > maxDrawdown) maxDrawdown = dd;
    }
    // Simple Sharpe approximation from returns
    const returns = agentHistory.slice(1).map((v, i) => (v - agentHistory[i]) / agentHistory[i]);
    const avgReturn = returns.reduce((a, b) => a + b, 0) / returns.length;
    const stdReturn = Math.sqrt(returns.reduce((a, b) => a + (b - avgReturn) ** 2, 0) / returns.length);
    sharpeRatio = stdReturn > 0 ? (avgReturn / stdReturn) * Math.sqrt(252) : 0;
  }

  // Compute APY from total PnL and TVL
  const apy = totalTvl > 0 && totalPnl !== 0 ? (totalPnl / (totalTvl - totalPnl)) * 100 : 0;

  const chartSeries = agentHistory.length > 0
    ? agentHistory
    : [];

  const chartData = {
    labels: Array.from({ length: chartSeries.length }, (_, i) => `Day ${i + 1}`),
    datasets: [
      {
        label: t('vault.sharePrice'),
        data: chartSeries,
        borderColor: '#10b981',
        backgroundColor: 'rgba(16, 185, 129, 0.1)',
        tension: 0.4,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 4,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
        backgroundColor: '#18181b',
        titleColor: '#fff',
        bodyColor: '#fff',
        borderColor: '#27272a',
        borderWidth: 1,
      },
    },
    scales: {
      x: {
        display: false,
        grid: { display: false }
      },
      y: {
        display: true,
        position: 'right' as const,
        grid: { color: '#f4f4f5' },
        ticks: {
          callback: (value: number | string) => `$${Number(value).toFixed(2)}`,
          font: { family: 'monospace' },
        }
      },
    },
    interaction: {
      mode: 'nearest' as const,
      axis: 'x' as const,
      intersect: false,
    },
  };

  return (
    <div className="min-h-screen pb-20">
      {/* Header Info */}
      <div className="mb-8">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-bold tracking-tight text-zinc-900">{t('vault.title')}</h1>
              <span className="rounded-md border border-zinc-200 bg-zinc-50 px-2 py-1 text-xs font-medium text-zinc-500 font-mono">
                {agentCount > 0 ? `${agentCount} agents` : 'No agents'}
              </span>
            </div>
            <div className="mt-2 flex items-center gap-2 text-sm text-zinc-500">
              <span className="flex items-center gap-1.5">
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                </span>
                {t('vault.status')}
              </span>
              <span>•</span>
              <span>{t('vault.createdBy')}</span>
            </div>
          </div>
          <div className="flex gap-8">
            <div className="relative group cursor-help">
              <div className="flex items-center gap-1.5 text-sm font-medium text-zinc-500 mb-1">
                {t('vault.equity')} <Info size={14} className="text-zinc-400" />
              </div>
              <div className="text-2xl font-bold text-zinc-900 font-mono">
                ${totalTvl.toLocaleString(undefined, { maximumFractionDigits: 0 })}
              </div>

              {/* Hover Breakdown */}
              <div className="absolute top-full left-0 mt-2 w-72 bg-white border border-zinc-200 rounded-lg shadow-xl p-4 z-50 invisible group-hover:visible transition-all opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0">
                <div className="text-xs font-bold text-zinc-900 mb-3 uppercase tracking-wider border-b border-zinc-100 pb-2">
                  {t('vault.breakdown.title')}
                </div>
                <div className="space-y-3">
                  <div className="flex justify-between text-xs items-center">
                    <span className="text-zinc-500 flex items-center gap-1.5">
                      <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                      {t('vault.breakdown.chain')}
                    </span>
                    <span className="font-mono font-medium text-zinc-900">${(vaultOverview?.totalEvmBalance ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}</span>
                  </div>
                  <div className="flex justify-between text-xs items-center">
                    <span className="text-zinc-500 flex items-center gap-1.5">
                      <div className="w-2 h-2 rounded-full bg-emerald-500"></div>
                      {t('vault.breakdown.hype')}
                    </span>
                    <span className="font-mono font-medium text-zinc-900">${(vaultOverview?.totalL1Value ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}</span>
                  </div>
                </div>
              </div>
            </div>
            <div>
              <div className="text-sm font-medium text-zinc-500 mb-1">{t('vault.apy')}</div>
              <div className="text-2xl font-bold text-emerald-600 font-mono">{apy.toFixed(1)}%</div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column: Chart & Overview */}
        <div className="lg:col-span-2 space-y-8">
          {/* Main Chart Card */}
          <div className="h-[400px] rounded-xl border border-zinc-200 bg-white p-6">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-zinc-500">{t('vault.sharePrice')}</div>
                <div className="text-xl font-bold text-zinc-900 font-mono">${sharePrice.toFixed(4)}</div>
              </div>
              <div className="flex gap-2">
                {['1D', '1W', '1M', 'ALL'].map(p => (
                  <button
                    key={p}
                    onClick={() => setChartPeriod(p)}
                    className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                      chartPeriod === p ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-100'
                    }`}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
            <div className="h-[300px] w-full">
              {chartSeries.length > 0 ? (
                <Line data={chartData} options={chartOptions} />
              ) : (
                <div className="h-full w-full flex items-center justify-center text-zinc-400 text-sm">
                  No performance data available yet
                </div>
              )}
            </div>
          </div>

          {/* Detailed Stats Grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="text-xs text-zinc-500 mb-1">{t('vault.maxDrawdown')}</div>
              <div className="text-lg font-bold text-red-600 font-mono">-{maxDrawdown.toFixed(2)}%</div>
            </div>
            <div className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="text-xs text-zinc-500 mb-1">{t('vault.sharpeRatio')}</div>
              <div className="text-lg font-bold text-zinc-900 font-mono">{sharpeRatio.toFixed(2)}</div>
            </div>
            <div className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="text-xs text-zinc-500 mb-1">{t('vault.depositors')}</div>
              <div className="text-lg font-bold text-zinc-900 font-mono">{agentCount > 0 ? agentCount : '0'}</div>
            </div>
            <div className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="text-xs text-zinc-500 mb-1">{t('vault.managerFee')}</div>
              <div className="text-lg font-bold text-zinc-900 font-mono">20%</div>
            </div>
          </div>

          {/* Overview Content (no tabs) */}
          <div className="rounded-xl border border-zinc-200 bg-white overflow-hidden">
            <div className="p-6">
              <div>
                <h3 className="text-lg font-bold text-zinc-900 mb-2">{t('vault.description.title')}</h3>
                <p className="text-zinc-600 leading-relaxed">
                  {t('vault.description.content')}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column: Action Panel */}
        <div className="lg:col-span-1 space-y-6">
          <div className="rounded-xl border border-zinc-200 bg-white overflow-hidden sticky top-6">
            <div className="flex border-b border-zinc-200">
              <button
                onClick={() => setActionTab('deposit')}
                className={`flex-1 py-4 text-sm font-bold transition-colors ${
                  actionTab === 'deposit' ? 'bg-black text-white' : 'bg-white text-zinc-500 hover:text-black'
                }`}
              >
                {t('vault.actions.deposit')}
              </button>
              <button
                onClick={() => setActionTab('withdraw')}
                className={`flex-1 py-4 text-sm font-bold transition-colors ${
                  actionTab === 'withdraw' ? 'bg-black text-white' : 'bg-white text-zinc-500 hover:text-black'
                }`}
              >
                {t('vault.actions.withdraw')}
              </button>
            </div>

            <div className="p-6 space-y-6">
              <div>
                <div className="flex justify-between text-xs text-zinc-500 mb-2">
                  <span>{t('vault.actions.asset')}</span>
                  <span>{t('vault.actions.balance')}: {user?.lpValue ? user.lpValue.toLocaleString(undefined, { maximumFractionDigits: 2 }) : '0.00'} USDC</span>
                </div>
                <div className="relative">
                  <input
                    type="number"
                    placeholder="0.00"
                    disabled
                    className="w-full rounded-md border border-zinc-200 px-4 py-3 pr-16 text-lg font-mono focus:border-black focus:outline-none focus:ring-1 focus:ring-black transition-colors bg-zinc-50 cursor-not-allowed opacity-60"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                  />
                  <div className="absolute right-4 top-1/2 -translate-y-1/2 text-sm font-medium text-zinc-400">
                    USDC
                  </div>
                </div>
                <div className="flex gap-2 mt-2">
                  {[25, 50, 75, 100].map(pct => (
                    <button key={pct} disabled className="flex-1 rounded-sm bg-zinc-100 py-1 text-xs font-medium text-zinc-400 cursor-not-allowed">
                      {pct}%
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-3 py-4 border-y border-zinc-100">
                <div className="flex justify-between text-sm">
                  <span className="text-zinc-500">{t('vault.actions.exchangeRate')}</span>
                  <span className="font-mono font-medium">1 HLP = {sharePrice.toFixed(4)} USDC</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-zinc-500">{t('vault.actions.estReceive')}</span>
                  <span className="font-mono font-medium text-zinc-900">0.00 HLP</span>
                </div>
              </div>

              <button disabled className="w-full rounded-md bg-zinc-200 py-3.5 text-sm font-bold text-zinc-500 cursor-not-allowed">
                {t('vault.actions.comingSoon')}
              </button>

              <div className="text-xs text-center text-zinc-400">
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
