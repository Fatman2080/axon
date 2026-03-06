import React, { useEffect, useState } from 'react';
import { Line } from 'react-chartjs-2';
import { useLanguage } from '../context/LanguageContext';
import { marketApi } from '../services/api';
import { VaultOverview, TreasurySnapshot } from '../types';
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
  const { t } = useLanguage();
  const [vaultOverview, setVaultOverview] = useState<VaultOverview | null>(null);
  const [treasuryHistory, setTreasuryHistory] = useState<TreasurySnapshot[]>([]);
  const [chartPeriod, setChartPeriod] = useState('1D');

  useEffect(() => {
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
  }, []);

  const periodToApi = (period: string) => {
    switch (period) {
      case '1D': return '1d';
      case '1W': return '7d';
      case '1M': return '30d';
      case 'ALL': return 'ALL';
      default: return '30d';
    }
  };

  useEffect(() => {
    const loadTreasuryHistory = async () => {
      try {
        let data = await marketApi.getTreasuryHistory(periodToApi(chartPeriod));
        if (Array.isArray(data)) {
          if (chartPeriod !== '1D') {
            const filtered = [];
            const seen = new Set();
            data.forEach((item) => {
              const dateStr = new Date(item.createdAt).toDateString();
              if (!seen.has(dateStr)) {
                seen.add(dateStr);
                filtered.push(item);
              }
            });
            data = filtered;
          }
          setTreasuryHistory(data);
        } else {
          setTreasuryHistory([]);
        }
      } catch {
        setTreasuryHistory([]);
      }
    };
    loadTreasuryHistory();
  }, [chartPeriod]);

  const totalTvl = (vaultOverview?.totalL1Value ?? 0) + (vaultOverview?.totalEvmBalance ?? 0);
  const totalPnl = vaultOverview?.totalPnl ?? 0;
  const agentCount = vaultOverview?.agentCount ?? 0;
  const totalInitialCapital = vaultOverview?.totalInitialCapital ?? 0;
  const overallRoi = totalInitialCapital > 0 ? (totalPnl / totalInitialCapital) * 100 : 0;
  const totalFundsHistory = treasuryHistory.map((s) => s.totalFunds || 0);
  const currentTotalFunds = totalFundsHistory.length > 0 ? totalFundsHistory[totalFundsHistory.length - 1] : totalTvl;

  let maxDrawdown = 0;
  let sharpeRatio = 0;
  if (totalFundsHistory.length > 1) {
    let peak = totalFundsHistory[0];
    for (const v of totalFundsHistory) {
      if (v > peak) peak = v;
      const dd = peak > 0 ? ((peak - v) / peak) * 100 : 0;
      if (dd > maxDrawdown) maxDrawdown = dd;
    }
    const returns = totalFundsHistory.slice(1).map((v, i) =>
      totalFundsHistory[i] > 0 ? (v - totalFundsHistory[i]) / totalFundsHistory[i] : 0
    );
    const avgReturn = returns.reduce((a, b) => a + b, 0) / returns.length;
    const stdReturn = Math.sqrt(returns.reduce((a, b) => a + (b - avgReturn) ** 2, 0) / returns.length);
    sharpeRatio = stdReturn > 0 ? (avgReturn / stdReturn) * Math.sqrt(252) : 0;
  }

  const apy = totalInitialCapital > 0 ? overallRoi : (totalTvl > 0 && totalPnl !== 0 ? (totalPnl / (totalTvl - totalPnl)) * 100 : 0);
  const chartSeries = totalFundsHistory;

  const chartData = {
    labels: Array.from({ length: chartSeries.length }, (_, i) => {
      const d = new Date();
      const reverseIndex = chartSeries.length - 1 - i;
      if (chartPeriod === '1D') {
        d.setMinutes(d.getMinutes() - Math.round(reverseIndex * ((24 * 60) / Math.max(chartSeries.length - 1, 1))));
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      }
      if (chartPeriod === '1W') {
        d.setDate(d.getDate() - Math.round(reverseIndex * (6 / Math.max(chartSeries.length - 1, 1))));
        return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
      }
      if (chartPeriod === '1M') {
        d.setDate(d.getDate() - Math.round(reverseIndex * (29 / Math.max(chartSeries.length - 1, 1))));
        return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
      }
      d.setDate(d.getDate() - Math.round(reverseIndex * (90 / Math.max(chartSeries.length - 1, 1))));
      return d.toLocaleDateString([], { year: 'numeric', month: 'short' });
    }),
    datasets: [{
      label: t('vault.totalFunds'),
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
      x: { 
        display: true, 
        grid: { display: false },
        ticks: {
          maxTicksLimit: 7,
          maxRotation: 0,
          font: { family: 'JetBrains Mono, monospace', size: 10 },
          color: '#555B66',
        }
      },
      y: {
        display: true,
        position: 'right' as const,
        grid: { color: 'rgba(255,255,255,0.04)' },
        ticks: {
          callback: (value: number | string) => `$${Number(value).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
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
    { label: t('vault.apy'), value: `${apy > 0 ? '+' : ''}${apy.toFixed(1)}%`, color: apy >= 0 ? 'var(--green)' : 'var(--red)' },
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

        </div>
      </div>

      <div className="max-w-5xl mx-auto">
        {/* Left: Chart + Stats */}
        <div className="space-y-6">
          {/* Chart */}
          <div
            className="h-[380px] rounded p-6"
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
          >
            <div className="mb-4 flex items-center justify-between">
              <div>
                <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                  {t('vault.totalFunds')}
                </div>
                <div className="text-xl font-bold font-mono" style={{ color: 'var(--text-primary)' }}>
                  ${currentTotalFunds.toLocaleString(undefined, { maximumFractionDigits: 2 })}
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
          <div className="grid grid-cols-3 gap-3">
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
      </div>
    </div>
  );
};

export default Agents;
