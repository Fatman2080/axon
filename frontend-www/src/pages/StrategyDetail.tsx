import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategyById } from '../store/slices/strategySlice';
import {
  ArrowLeft, Shield,
  AlertTriangle, Clock, ArrowRight,
  Twitter
} from 'lucide-react';
import { Line } from 'react-chartjs-2';
import { useLanguage } from '../context/LanguageContext';

const StrategyDetail = () => {
  const { id } = useParams<{ id: string }>();
  const dispatch = useAppDispatch();
  const { currentStrategy: strategy, currentHistory, currentPositions, currentFills, currentCreatedAt, loading } = useAppSelector((state) => state.strategies);
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState('overview');
  const [period, setPeriod] = useState('ALL');

  useEffect(() => {
    if (id) {
      dispatch(fetchStrategyById({ publicKey: id, period }));
    }
  }, [dispatch, id, period]);

  if (loading || !strategy) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-zinc-200 border-t-black"></div>
      </div>
    );
  }

  // Chart data from real history or fallback
  const historyData = currentHistory && currentHistory.length > 0
    ? currentHistory
    : [];

  const chartData = {
    labels: historyData.map((_, i) => `Point ${i + 1}`),
    datasets: [
      {
        label: 'Account Value',
        data: historyData,
        borderColor: '#000',
        backgroundColor: 'rgba(0, 0, 0, 0.05)',
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
        callbacks: {
          label: (context: any) => {
            return `$${context.raw.toFixed(2)}`;
          }
        }
      },
    },
    scales: {
      x: { display: false },
      y: { display: false },
    },
    interaction: {
      mode: 'nearest' as const,
      axis: 'x' as const,
      intersect: false,
    },
  };

  // Compute running days from creation date
  const runningDays = currentCreatedAt
    ? Math.max(0, Math.floor((Date.now() - new Date(currentCreatedAt).getTime()) / 86400000))
    : 0;

  // Compute metrics from fills
  const totalTrades = currentFills.length;
  const winningTrades = currentFills.filter(f => f.closedPnl > 0).length;
  const losingTrades = currentFills.filter(f => f.closedPnl < 0).length;
  const winRate = totalTrades > 0 ? (winningTrades / totalTrades) * 100 : 0;
  const totalWinPnl = currentFills.filter(f => f.closedPnl > 0).reduce((sum, f) => sum + f.closedPnl, 0);
  const totalLossPnl = Math.abs(currentFills.filter(f => f.closedPnl < 0).reduce((sum, f) => sum + f.closedPnl, 0));
  const profitFactor = totalLossPnl > 0 ? totalWinPnl / totalLossPnl : totalWinPnl > 0 ? Infinity : 0;
  const avgTradePnl = totalTrades > 0 ? currentFills.reduce((sum, f) => sum + f.closedPnl, 0) / totalTrades : 0;

  // Derive traded assets from positions
  const tradedAssets = currentPositions.length > 0
    ? [...new Set(currentPositions.map(p => p.coin))]
    : currentFills.length > 0
      ? [...new Set(currentFills.map(f => f.coin))]
      : [];

  return (
    <div className="mx-auto max-w-5xl pb-20">
      <Link to="/strategies" className="mb-6 inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-black transition-colors">
        <ArrowLeft size={16} />
        {t('strategyDetail.backToMarket')}
      </Link>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Left Column: Main Info */}
        <div className="lg:col-span-2 space-y-8">
          {/* Header Card */}
          <div className="rounded-xl border border-zinc-200 bg-white p-6">
            <div className="mb-6 flex items-start justify-between">
              <div className="flex gap-4">
                <div className="h-16 w-16 overflow-hidden rounded-xl bg-zinc-100">
                  <img
                    src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${strategy.id}`}
                    alt={strategy.name}
                    className="h-full w-full object-cover"
                  />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h1 className="text-2xl font-bold text-zinc-900">{strategy.name}</h1>
                    <Shield size={16} className="text-emerald-500" />
                    <span className="text-xs font-medium text-emerald-600 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-100">
                      {t('strategyDetail.verifiedRisk')}
                    </span>
                    {strategy.agentStatus && (
                      <span className={`text-xs font-medium px-2 py-0.5 rounded-full border ${
                        strategy.agentStatus === 'active' ? 'bg-emerald-50 text-emerald-700 border-emerald-200' :
                        strategy.agentStatus === 'revoked' ? 'bg-red-50 text-red-700 border-red-200' :
                        'bg-zinc-100 text-zinc-500 border-zinc-200'
                      }`}>
                        {strategy.agentStatus}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 flex items-center gap-4 text-sm text-zinc-500">
                    <span className="flex items-center gap-1 capitalize">{strategy.category}</span>
                  </div>
                  <div className="mt-3 flex gap-2">
                    {strategy.creator && (
                      <a
                        href={`https://x.com/${strategy.creator}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="p-1.5 rounded-full bg-zinc-100 text-zinc-500 hover:bg-black hover:text-white transition-colors"
                      >
                        <Twitter size={14} />
                      </a>
                    )}
                  </div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-sm text-zinc-500">{t('strategyDetail.tvl')}</div>
                <div className="text-2xl font-bold font-mono text-zinc-900">
                  ${strategy.currentTvl >= 1000000
                    ? (strategy.currentTvl / 1000000).toFixed(2) + 'M'
                    : strategy.currentTvl.toLocaleString()}
                </div>
              </div>
            </div>

            <div className="mb-8 h-[300px] w-full">
              {historyData.length > 0 ? (
                <Line data={chartData} options={chartOptions} />
              ) : (
                <div className="h-full w-full flex items-center justify-center text-zinc-400 text-sm">
                  No performance data available yet
                </div>
              )}
            </div>

            <div className="grid grid-cols-3 gap-4 border-t border-zinc-100 pt-6">
              <div>
                <div className="text-xs text-zinc-500 mb-1">PnL</div>
                <div className={`text-xl font-bold font-mono ${strategy.pnlContribution >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                  {strategy.pnlContribution >= 0 ? '+' : ''}${strategy.pnlContribution.toFixed(2)}
                </div>
              </div>
              <div>
                <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.sharpe')}</div>
                <div className="text-xl font-bold text-zinc-900 font-mono">{strategy.backtestMetrics?.sharpeRatio?.toFixed(2) ?? '-'}</div>
              </div>
              <div>
                <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.drawdown')}</div>
                <div className="text-xl font-bold text-red-600 font-mono">{strategy.backtestMetrics?.maxDrawdown ? `-${strategy.backtestMetrics.maxDrawdown.toFixed(2)}%` : '-'}</div>
              </div>
            </div>
          </div>

            {/* Strategy Description */}
            <div className="rounded-xl border border-zinc-200 bg-white overflow-hidden">
              <div className="flex border-b border-zinc-200">
                <button
                  onClick={() => setActiveTab('overview')}
                  className={`flex-1 py-4 text-sm font-bold transition-colors ${
                    activeTab === 'overview' ? 'bg-zinc-50 text-black border-b-2 border-black' : 'text-zinc-500 hover:text-black'
                  }`}
                >
                  {t('strategyDetail.tabs.overview')}
                </button>
                <button
                  onClick={() => setActiveTab('metrics')}
                  className={`flex-1 py-4 text-sm font-bold transition-colors ${
                    activeTab === 'metrics' ? 'bg-zinc-50 text-black border-b-2 border-black' : 'text-zinc-500 hover:text-black'
                  }`}
                >
                  {t('strategyDetail.tabs.metrics')}
                </button>
              </div>

              <div className="p-6">
                {activeTab === 'overview' && (
                  <div className="space-y-6">
                    <div>
                      <h3 className="mb-4 font-bold text-zinc-900">{t('strategyDetail.description')}</h3>
                      <p className="text-zinc-600 leading-relaxed">
                        {strategy.description || 'No description available.'}
                      </p>
                    </div>

                    {strategy.vaultAddress && (
                      <div className="grid grid-cols-2 gap-4 p-4 rounded-lg bg-zinc-50 border border-zinc-100">
                        <div>
                          <div className="text-xs text-zinc-500 mb-1">Vault Address</div>
                          <a
                            href={`https://testnet.purrsec.com/address/${strategy.vaultAddress}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-mono text-sm text-blue-600 hover:underline break-all"
                          >
                            {strategy.vaultAddress.slice(0, 10)}...{strategy.vaultAddress.slice(-8)}
                          </a>
                        </div>
                        <div>
                          <div className="text-xs text-zinc-500 mb-1">EVM Balance (USDC)</div>
                          <div className="font-mono text-sm font-bold text-zinc-900">
                            ${(strategy.evmBalance || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}
                          </div>
                        </div>
                      </div>
                    )}

                    <div className="flex flex-wrap gap-2">
                      <span className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600 capitalize">{strategy.category}</span>
                      {tradedAssets.map(asset => (
                        <span key={asset} className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600">{asset}</span>
                      ))}
                    </div>
                  </div>
                )}

                {activeTab === 'metrics' && (
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-4 rounded-lg bg-zinc-50 border border-zinc-100">
                      <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.winRate')}</div>
                      <div className="text-lg font-bold text-zinc-900">{totalTrades > 0 ? `${winRate.toFixed(1)}%` : '-'}</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-50 border border-zinc-100">
                      <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.trades')}</div>
                      <div className="text-lg font-bold text-zinc-900">{totalTrades > 0 ? totalTrades.toLocaleString() : '-'}</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-50 border border-zinc-100">
                      <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.profitFactor')}</div>
                      <div className="text-lg font-bold text-emerald-600">{totalTrades > 0 ? (profitFactor === Infinity ? '∞' : profitFactor.toFixed(2)) : '-'}</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-50 border border-zinc-100">
                      <div className="text-xs text-zinc-500 mb-1">{t('strategyDetail.avgTrade')}</div>
                      <div className={`text-lg font-bold ${avgTradePnl >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>{totalTrades > 0 ? `$${avgTradePnl.toFixed(2)}` : '-'}</div>
                    </div>
                  </div>
                )}
              </div>
            </div>
        </div>

        {/* Right Column: Actions */}
        <div className="space-y-6">
          <div className="sticky top-6 space-y-6">
            {/* Risk Card */}
            <div className="rounded-xl border border-zinc-200 bg-white p-6">
              <h3 className="mb-4 font-bold text-zinc-900">{t('strategyDetail.riskAnalysis')}</h3>

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-zinc-500">{t('strategyDetail.riskLevel')}</span>
                  <span className="flex items-center gap-1 text-sm font-medium text-amber-600">
                    <AlertTriangle size={14} />
                    {strategy.riskLevel === 'high' ? 'High' : strategy.riskLevel === 'low' ? 'Low' : 'Medium'}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-zinc-500">{t('strategyDetail.runningDays')}</span>
                  <span className="flex items-center gap-1 text-sm font-medium text-zinc-900">
                    <Clock size={14} />
                    {runningDays > 0 ? `${runningDays}d` : '-'}
                  </span>
                </div>

                <div className="pt-4 border-t border-zinc-100">
                  <div className="rounded-md bg-amber-50 p-3 text-xs text-amber-700 leading-relaxed">
                    {t('strategyDetail.riskWarning')}
                  </div>
                </div>
              </div>
            </div>

            {/* Hiring Callout */}
            <div className="rounded-xl border border-dashed border-zinc-300 bg-zinc-50 p-6 text-center">
              <h4 className="font-bold text-zinc-900 mb-2">{t('strategyDetail.hiring.title')}</h4>
              <p className="text-sm text-zinc-500 mb-4">
                {t('strategyDetail.hiring.desc')}
              </p>
              <Link
                to="/submit-agent"
                className="inline-flex items-center gap-2 text-sm font-bold text-black hover:underline"
              >
                {t('strategyDetail.hiring.button')}
                <ArrowRight size={14} />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StrategyDetail;
