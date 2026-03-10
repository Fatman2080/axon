import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategyById } from '../store/slices/strategySlice';
import {
  ArrowLeft, Shield, AlertTriangle, Clock, ArrowRight, Twitter
} from 'lucide-react';
import { Line } from 'react-chartjs-2';
import { useLanguage } from '../context/LanguageContext';

const StrategyDetail = () => {
  const { id } = useParams<{ id: string }>();
  const dispatch = useAppDispatch();
  const { currentStrategy: strategy, currentHistory, currentPositions, currentFills, currentCreatedAt, loading } = useAppSelector((state) => state.strategies);
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState('overview');
  const [period] = useState('ALL');

  useEffect(() => {
    if (id) {
      dispatch(fetchStrategyById({ publicKey: id, period }));
    }
  }, [dispatch, id, period]);

  if (loading || !strategy) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div
          className="h-8 w-8 animate-spin rounded-full border-2"
          style={{ borderColor: 'var(--border)', borderTopColor: 'var(--neon-green)' }}
        />
      </div>
    );
  }

  const historyData = currentHistory && currentHistory.length > 0 ? currentHistory : [];

  const chartData = {
    labels: historyData.map((_, i) => `${t('common.point')} ${i + 1}`),
    datasets: [{
      label: t('strategyDetail.accountValue'),
      data: historyData,
      borderColor: '#00FF41',
      backgroundColor: 'rgba(0, 240, 255, 0.05)',
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
        callbacks: {
          label: (context: any) => `$${context.raw.toFixed(2)}`
        },
      },
    },
    scales: {
      x: { display: false },
      y: { display: false },
    },
    interaction: { mode: 'nearest' as const, axis: 'x' as const, intersect: false },
  };

  const runningDays = currentCreatedAt
    ? Math.max(0, Math.floor((Date.now() - new Date(currentCreatedAt).getTime()) / 86400000))
    : 0;

  const totalTrades = currentFills.length;
  const winningTrades = currentFills.filter(f => f.closedPnl > 0).length;
  const winRate = totalTrades > 0 ? (winningTrades / totalTrades) * 100 : 0;
  const totalWinPnl = currentFills.filter(f => f.closedPnl > 0).reduce((sum, f) => sum + f.closedPnl, 0);
  const totalLossPnl = Math.abs(currentFills.filter(f => f.closedPnl < 0).reduce((sum, f) => sum + f.closedPnl, 0));
  const profitFactor = totalLossPnl > 0 ? totalWinPnl / totalLossPnl : totalWinPnl > 0 ? Infinity : 0;
  const avgTradePnl = totalTrades > 0 ? currentFills.reduce((sum, f) => sum + f.closedPnl, 0) / totalTrades : 0;

  const tradedAssets = currentPositions.length > 0
    ? [...new Set(currentPositions.map(p => p.coin))]
    : currentFills.length > 0
      ? [...new Set(currentFills.map(f => f.coin))]
      : [];
  const resolveAgentStatus = (status: string) => {
    const key = `strategyDetail.status.${status}`;
    const translated = t(key);
    return translated === key ? status.toUpperCase() : translated;
  };

  const pnlPositive = strategy.pnlContribution >= 0;

  return (
    <div className="mx-auto max-w-5xl pb-20 animate-fade-in-up">
      {/* Back link */}
      <Link
        to="/strategies"
        className="mb-6 inline-flex items-center gap-2 text-sm font-mono transition-colors"
        style={{ color: 'var(--text-tertiary)' }}
        onMouseEnter={e => (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)'}
        onMouseLeave={e => (e.currentTarget as HTMLElement).style.color = 'var(--text-tertiary)'}
      >
        <ArrowLeft size={14} />
        {t('strategyDetail.backToMarket')}
      </Link>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Left: Main Info */}
        <div className="lg:col-span-2 space-y-5">
          {/* Header Card */}
          <div className="rounded p-6" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
            <div className="mb-5 flex items-start justify-between">
              <div className="flex gap-4">
                <div className="h-14 w-14 overflow-hidden rounded" style={{ border: '1px solid var(--border)' }}>
                  <img
                    src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${strategy.id}`}
                    alt={strategy.name}
                    className="h-full w-full object-cover"
                  />
                </div>
                <div>
                  <div className="flex flex-wrap items-center gap-2 mb-1">
                    <h1 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>
                      {strategy.name}
                    </h1>
                    <span
                      className="inline-flex items-center gap-1 text-[10px] font-mono font-bold uppercase tracking-widest px-2 py-0.5 rounded"
                      style={{ background: 'var(--green-dim)', color: 'var(--green)', border: '1px solid rgba(0,255,102,0.2)' }}
                    >
                      <Shield size={10} />
                      {t('strategyDetail.verifiedRisk')}
                    </span>
                    {strategy.agentStatus && (
                      <span
                        className="text-[10px] font-mono font-bold uppercase tracking-widest px-2 py-0.5 rounded"
                        style={{
                          background: strategy.agentStatus === 'active' ? 'var(--green-dim)' :
                            strategy.agentStatus === 'revoked' ? 'var(--red-dim)' : 'rgba(255,255,255,0.04)',
                          color: strategy.agentStatus === 'active' ? 'var(--green)' :
                            strategy.agentStatus === 'revoked' ? 'var(--red)' : 'var(--text-tertiary)',
                          border: strategy.agentStatus === 'active' ? '1px solid rgba(0,255,102,0.2)' :
                            strategy.agentStatus === 'revoked' ? '1px solid rgba(255,42,42,0.2)' : '1px solid var(--border)',
                        }}
                      >
                        {resolveAgentStatus(strategy.agentStatus)}
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-3 text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>
                    <span className="capitalize">{strategy.category}</span>
                    {strategy.creator && (
                      <a
                        href={`https://x.com/${strategy.creator}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 transition-colors hover:text-white"
                      >
                        <Twitter size={12} />
                        @{strategy.creator}
                      </a>
                    )}
                  </div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                  {t('strategyDetail.tvl')}
                </div>
                <div className="text-xl font-bold font-mono" style={{ color: 'var(--neon-green)' }}>
                  ${strategy.currentTvl >= 1000000
                    ? (strategy.currentTvl / 1000000).toFixed(2) + 'M'
                    : strategy.currentTvl.toLocaleString()}
                </div>
              </div>
            </div>

            {/* Chart */}
            <div className="mb-6 h-[260px] w-full">
              {historyData.length > 0 ? (
                <Line data={chartData} options={chartOptions} />
              ) : (
                <div className="h-full w-full flex items-center justify-center text-sm font-mono" style={{ color: 'var(--text-tertiary)' }}>
                  {t('strategyDetail.noHistoryTelemetry')}
                </div>
              )}
            </div>

            {/* Key Metrics Row */}
            <div
              className="grid grid-cols-3 gap-4 pt-5"
              style={{ borderTop: '1px solid var(--border)' }}
            >
              {[
                { label: t('strategyDetail.pnl'), value: `${pnlPositive ? '+' : ''}$${strategy.pnlContribution.toFixed(2)}`, color: pnlPositive ? 'var(--green)' : 'var(--red)' },
                { label: t('strategyDetail.sharpe'), value: strategy.backtestMetrics?.sharpeRatio?.toFixed(2) ?? '-', color: 'var(--text-primary)' },
                { label: t('strategyDetail.drawdown'), value: strategy.backtestMetrics?.maxDrawdown ? `-${strategy.backtestMetrics.maxDrawdown.toFixed(2)}%` : '-', color: 'var(--red)' },
              ].map(m => (
                <div key={m.label}>
                  <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                    {m.label}
                  </div>
                  <div className="text-lg font-bold font-mono" style={{ color: m.color }}>
                    {m.value}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Tabs Card */}
          <div className="rounded overflow-hidden" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
            <div className="flex" style={{ borderBottom: '1px solid var(--border)' }}>
              {['overview', 'metrics'].map(tab => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className="flex-1 py-3.5 text-sm font-bold font-mono uppercase tracking-wider transition-all"
                  style={{
                    background: activeTab === tab ? 'rgba(0,240,255,0.04)' : 'transparent',
                    color: activeTab === tab ? 'var(--neon-green)' : 'var(--text-tertiary)',
                    borderBottom: activeTab === tab ? '2px solid var(--neon-green)' : '2px solid transparent',
                  }}
                >
                  {tab === 'overview' ? t('strategyDetail.tabs.overview') : t('strategyDetail.tabs.metrics')}
                </button>
              ))}
            </div>

            <div className="p-6">
              {activeTab === 'overview' && (
                <div className="space-y-5">
                  <div>
                    <h3 className="font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
                      {t('strategyDetail.description')}
                    </h3>
                    <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
                      {strategy.description || t('strategyDetail.noDescription')}
                    </p>
                  </div>

                  {strategy.vaultAddress && (
                    <div
                      className="grid grid-cols-2 gap-4 p-4 rounded"
                      style={{ background: 'var(--bg-input)', border: '1px solid var(--border)' }}
                    >
                      <div>
                        <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                          {t('strategyDetail.vaultAddress')}
                        </div>
                        <a
                          href={`https://testnet.purrsec.com/address/${strategy.vaultAddress}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-mono text-xs break-all"
                          style={{ color: 'var(--neon-green)' }}
                        >
                          {strategy.vaultAddress.slice(0, 10)}...{strategy.vaultAddress.slice(-8)}
                        </a>
                      </div>
                      <div>
                        <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                          {t('strategyDetail.evmBalanceUsdc')}
                        </div>
                        <div className="font-mono text-sm font-bold" style={{ color: 'var(--text-primary)' }}>
                          ${(strategy.evmBalance || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}
                        </div>
                      </div>
                    </div>
                  )}

                  <div className="flex flex-wrap gap-2">
                    <span
                      className="rounded px-3 py-1 text-xs font-mono capitalize"
                      style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', color: 'var(--text-secondary)' }}
                    >
                      {strategy.category}
                    </span>
                    {tradedAssets.map(asset => (
                      <span
                        key={asset}
                        className="rounded px-3 py-1 text-xs font-mono"
                        style={{ background: 'var(--neon-green-dim)', border: '1px solid rgba(0,240,255,0.15)', color: 'var(--neon-green)' }}
                      >
                        {asset}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {activeTab === 'metrics' && (
                <div className="grid grid-cols-2 gap-3">
                  {[
                    { label: t('strategyDetail.winRate'), value: totalTrades > 0 ? `${winRate.toFixed(1)}%` : '-', color: 'var(--text-primary)' },
                    { label: t('strategyDetail.trades'), value: totalTrades > 0 ? totalTrades.toLocaleString() : '-', color: 'var(--text-primary)' },
                    { label: t('strategyDetail.profitFactor'), value: totalTrades > 0 ? (profitFactor === Infinity ? '∞' : profitFactor.toFixed(2)) : '-', color: 'var(--green)' },
                    { label: t('strategyDetail.avgTrade'), value: totalTrades > 0 ? `$${avgTradePnl.toFixed(2)}` : '-', color: avgTradePnl >= 0 ? 'var(--green)' : 'var(--red)' },
                  ].map(m => (
                    <div
                      key={m.label}
                      className="p-4 rounded"
                      style={{ background: 'var(--bg-input)', border: '1px solid var(--border)' }}
                    >
                      <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
                        {m.label}
                      </div>
                      <div className="text-lg font-bold font-mono" style={{ color: m.color }}>{m.value}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right: Actions Sidebar */}
        <div className="space-y-5">
          <div className="sticky top-20 space-y-4">
            {/* Risk Card */}
            <div className="rounded p-5" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
              <h3
                className="font-bold mb-4 text-xs font-mono uppercase tracking-widest"
                style={{ color: 'var(--text-tertiary)' }}
              >
                {t('strategyDetail.riskAnalysis')}
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('strategyDetail.riskLevel')}</span>
                  <span className="flex items-center gap-1 text-sm font-mono font-bold" style={{ color: 'var(--tier-partner)' }}>
                    <AlertTriangle size={12} />
                    {strategy.riskLevel === 'high'
                      ? t('strategyDetail.riskHigh')
                      : strategy.riskLevel === 'low'
                        ? t('strategyDetail.riskLow')
                        : t('strategyDetail.riskMedium')}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('strategyDetail.runningDays')}</span>
                  <span className="flex items-center gap-1 text-sm font-mono font-bold" style={{ color: 'var(--text-primary)' }}>
                    <Clock size={12} />
                    {runningDays > 0 ? `${runningDays}${t('strategyDetail.runningDaysUnit')}` : '-'}
                  </span>
                </div>
              </div>

              <div className="mt-4 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
                <div
                  className="rounded p-3 text-xs font-mono leading-relaxed"
                  style={{ background: 'rgba(255,184,0,0.06)', border: '1px solid rgba(255,184,0,0.15)', color: 'var(--tier-partner)' }}
                >
                  ⚠ {t('strategyDetail.riskWarning')}
                </div>
              </div>
            </div>

            {/* Hiring Callout */}
            <div
              className="rounded p-5 text-center"
              style={{ background: 'var(--bg-card)', border: '1px dashed rgba(255,255,255,0.1)' }}
            >
              <h4 className="font-bold mb-2 text-sm" style={{ color: 'var(--text-primary)' }}>
                {t('strategyDetail.hiring.title')}
              </h4>
              <p className="text-xs mb-4" style={{ color: 'var(--text-secondary)' }}>
                {t('strategyDetail.hiring.desc')}
              </p>
              <Link
                to="/submit-agent"
                className="inline-flex items-center gap-2 text-sm font-bold font-mono transition-colors"
                style={{ color: 'var(--neon-green)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.color = 'var(--green)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.color = 'var(--neon-green)'}
              >
                {t('strategyDetail.hiring.button')}
                <ArrowRight size={12} />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StrategyDetail;
