import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategies } from '../store/slices/strategySlice';
import {
  User, Wallet, Calendar,
  CheckCircle, Code,
  ChevronRight, Award, Briefcase, DollarSign, LogOut
} from 'lucide-react';
import { Line } from 'react-chartjs-2';
import { useLanguage } from '../context/LanguageContext';
import { authApi } from '../services/api';
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
import { logoutUser } from '../store/slices/userSlice';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler);

const Profile = () => {
  const dispatch = useAppDispatch();
  const { currentUser: user, loading: userLoading } = useAppSelector((state) => state.user);
  const { items: strategies, loading: strategiesLoading } = useAppSelector((state) => state.strategies);
  const [agentHistory, setAgentHistory] = useState<number[]>([]);
  const chartPeriod = '1D';
  const [agentStats, setAgentStats] = useState<{ accountValue: number; totalPnl: number; initialCapital: number } | null>(null);
  const { t } = useLanguage();

  useEffect(() => {
    dispatch(fetchStrategies());
  }, [dispatch]);

  useEffect(() => {
    if (!user) return;
    authApi.getAgentStats().then((stats) => {
      setAgentStats({
        accountValue: stats.accountValue ?? 0,
        totalPnl: stats.totalPnl ?? 0,
        initialCapital: (stats as any).initialCapital ?? 0,
      });
    }).catch(() => {});
  }, [user]);

  const periodToApi = (period: string) => {
    switch (period) {
      case '1D': return '1d';
      case '1W': return '7d';
      case '1M': return '30d';
      case 'ALL': return 'ALL';
      default: return '7d';
    }
  };

  useEffect(() => {
    if (!user) return;
    const loadData = async () => {
      try {
        const historyData = await authApi.getAgentHistory(periodToApi(chartPeriod));
        if (Array.isArray(historyData.history)) {
          let hist = historyData.history;
          let dts = Array.isArray(historyData.dates) ? historyData.dates : [];
          
          if (dts.length !== hist.length) {
            dts = Array.from({ length: hist.length }, (_, i) => {
              const d = new Date();
              const reverseIndex = hist.length - 1 - i;
              if (chartPeriod === '1D') {
                d.setMinutes(d.getMinutes() - Math.round(reverseIndex * ((24 * 60) / Math.max(hist.length - 1, 1))));
              } else if (chartPeriod === '1W') {
                d.setDate(d.getDate() - Math.round(reverseIndex * (6 / Math.max(hist.length - 1, 1))));
              } else if (chartPeriod === '1M') {
                d.setDate(d.getDate() - Math.round(reverseIndex * (29 / Math.max(hist.length - 1, 1))));
              } else {
                d.setDate(d.getDate() - Math.round(reverseIndex * (90 / Math.max(hist.length - 1, 1))));
              }
              return d.toISOString();
            });
          }
          
          if (chartPeriod !== '1D' && dts.length > 0) {
            const fHist: number[] = [];
            const fDts: string[] = [];
            const seen = new Set();
            for (let i = 0; i < dts.length; i++) {
              const dStr = new Date(dts[i]).toDateString();
              if (!seen.has(dStr)) {
                seen.add(dStr);
                fHist.push(hist[i]);
                fDts.push(dts[i]);
              }
            }
            hist = fHist;
            dts = fDts;
          }
          
          const finalHist = hist as any;
          finalHist.dates = dts;
          setAgentHistory(finalHist);
        } else {
          setAgentHistory([]);
        }
      } catch {
        setAgentHistory([]);
      }
    };
    loadData();
  }, [user]);

  const loading = userLoading || strategiesLoading;

  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map(i => (
          <div key={i} className="h-32 rounded animate-pulse" style={{ background: 'var(--bg-card)' }} />
        ))}
      </div>
    );
  }

  if (!user) {
    return (
      <div className="mx-auto max-w-md py-24 text-center animate-fade-in-up">
        <div
          className="h-16 w-16 mx-auto mb-6 rounded flex items-center justify-center"
          style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
        >
          <User size={28} style={{ color: 'var(--text-tertiary)' }} />
        </div>
        <h2 className="text-xl font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
          {t('profile.loginRequired') || 'Login Required'}
        </h2>
        <p className="text-sm mb-8" style={{ color: 'var(--text-secondary)' }}>
          {t('profile.loginDesc') || 'Please login with X to view your profile.'}
        </p>
        <button
          onClick={() => {
            const nextPath = window.location.pathname;
            window.location.href = authApi.getXOAuthStartUrl(undefined, nextPath);
          }}
          className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-bold rounded"
          style={{ background: 'var(--text-primary)', color: '#000' }}
        >
          {t('nav.connectWallet') || 'Login with X'}
        </button>
      </div>
    );
  }

  const mySubmissions = strategies.filter(s => user.agentPublicKey ? s.id === user.agentPublicKey : false);
  const initialCapital = agentStats?.initialCapital ?? 0;
  const accountValue = agentStats?.accountValue ?? (user.totalInvestment ?? 0);
  const totalPnl = agentStats?.totalPnl ?? (user.totalProfit ?? 0);
  const totalInvestment = initialCapital > 0 ? initialCapital : accountValue;
  const totalProfit = totalPnl;
  const roi = initialCapital > 0 ? (totalPnl / initialCapital) * 100 : 0;
  const agentCount = user.agentPublicKey ? 1 : (user.agentCount ?? 0);

  const chartSeries = agentHistory.length > 0 ? agentHistory : [];

  const chartData = {
    labels: (agentHistory as any).dates && (agentHistory as any).dates.length > 0
      ? (agentHistory as any).dates.map((d: string) => {
          const date = new Date(d);
          if (chartPeriod === '1D') return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
          if (chartPeriod === '1W') return [
            date.toLocaleDateString([], { month: 'short', day: 'numeric' }),
            date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          ];
          if (chartPeriod === 'ALL') return date.toLocaleDateString([], { year: 'numeric', month: 'short' });
          return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
        })
      : Array.from({ length: chartSeries.length }, (_, i) => `Point ${i + 1}`),
    datasets: [{
      label: t('profile.portfolio'),
      data: chartSeries,
      borderColor: '#00FF66',
      backgroundColor: 'rgba(0, 255, 102, 0.04)',
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
          maxTicksLimit: 6,
          maxRotation: 0,
          font: { family: 'monospace', size: 10 },
          color: '#555B66',
        }
      },
      y: { display: false, grid: { display: false } },
    },
    interaction: { mode: 'nearest' as const, axis: 'x' as const, intersect: false },
  };

  const summaryCards = [
    { label: initialCapital > 0 ? t('profile.initialCapital') || 'Initial Capital' : t('profile.totalEquity'), value: `$${totalInvestment.toLocaleString()}`, icon: Wallet, color: 'var(--neon-green)' },
    { label: t('profile.unrealizedPnL'), value: `${totalProfit > 0 ? '+' : ''}$${totalProfit.toLocaleString()}`, icon: Award, color: totalProfit >= 0 ? 'var(--green)' : 'var(--red)' },
    { label: 'ROI', value: initialCapital > 0 ? `${roi >= 0 ? '+' : ''}${roi.toFixed(2)}%` : '--', icon: DollarSign, color: roi >= 0 ? 'var(--green)' : 'var(--red)' },
    { label: t('profile.activeAgents'), value: `${agentCount}`, icon: Briefcase, color: 'var(--tier-manager)' },
  ];

  return (
    <div className="mx-auto max-w-5xl pb-20 animate-fade-in-up">
      {/* Profile Header */}
      <div
        className="mb-6 flex flex-col gap-5 md:flex-row md:items-start md:justify-between p-6 rounded"
        style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
      >
        <div className="flex items-center gap-5">
          <div className="relative shrink-0">
            <div
              className="h-16 w-16 overflow-hidden rounded"
              style={{ border: '2px solid var(--border)' }}
            >
              {user.avatar ? (
                <img src={user.avatar} alt={user.name} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full w-full items-center justify-center" style={{ background: 'var(--bg-input)' }}>
                  <User size={28} style={{ color: 'var(--text-tertiary)' }} />
                </div>
              )}
            </div>
            <div
              className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full"
              style={{ background: 'var(--green)' }}
            >
              <CheckCircle size={12} className="text-black" />
            </div>
          </div>

          <div>
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>{user.name}</h1>
            </div>
            <div className="flex items-center gap-4 text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>
              <span className="flex items-center gap-1.5">
                <Calendar size={12} />
                {t('profile.joined')}{' '}
                {(user as any).createdAt
                  ? new Date((user as any).createdAt).toLocaleDateString()
                  : user.joinedAt ? new Date(user.joinedAt).toLocaleDateString() : '-'}
              </span>
            </div>
          </div>
        </div>

        <button
          onClick={() => dispatch(logoutUser())}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded self-start"
          style={{
            background: 'var(--red-dim)',
            color: 'var(--red)',
            border: '1px solid rgba(255,42,42,0.2)',
          }}
          onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = 'rgba(255,42,42,0.15)'}
          onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = 'var(--red-dim)'}
        >
          <LogOut size={14} />
          Logout
        </button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {summaryCards.map(card => (
          <div
            key={card.label}
            className="rounded p-4"
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
          >
            <div className="flex items-center gap-2 mb-2" style={{ color: 'var(--text-tertiary)' }}>
              <card.icon size={14} />
              <span className="text-xs font-mono uppercase tracking-widest">{card.label}</span>
            </div>
            <div className="text-xl font-bold font-mono" style={{ color: card.color }}>
              {card.value}
            </div>
          </div>
        ))}
      </div>

      {/* Portfolio Chart */}
      <div
        className="rounded p-6 mb-6"
        style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
      >
        <div className="mb-5 flex items-end justify-between">
          <div>
            <div className="text-xs font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>
              {t('profile.portfolio')}
            </div>
            <div className="text-3xl font-bold font-mono tracking-tight" style={{ color: 'var(--text-primary)' }}>
              ${accountValue.toLocaleString()}
            </div>
            <div className="mt-1 flex items-center gap-2 text-sm">
              <span className="font-mono" style={{ color: totalPnl >= 0 ? 'var(--green)' : 'var(--red)' }}>
                {totalPnl > 0 ? '+' : ''}${totalPnl.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                {initialCapital > 0 && ` (${roi >= 0 ? '+' : ''}${roi.toFixed(1)}%)`}
              </span>
            </div>
          </div>
        </div>

        {chartSeries.length > 0 ? (
          <div className="h-[240px] w-full">
            <Line data={chartData} options={chartOptions} />
          </div>
        ) : (
          <div className="h-[240px] w-full flex items-center justify-center text-sm font-mono" style={{ color: 'var(--text-tertiary)' }}>
            // NO_HISTORY — agent not yet connected
          </div>
        )}
      </div>

      {/* Strategy Submissions */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-bold" style={{ color: 'var(--text-primary)' }}>{t('profile.myAgents')}</h2>
          <button className="text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>{t('profile.viewAll')}</button>
        </div>

        <div className="space-y-2">
          {mySubmissions.length === 0 ? (
            <div
              className="rounded p-8 text-center"
              style={{
                background: 'var(--bg-card)',
                border: '1px dashed rgba(255,255,255,0.1)',
              }}
            >
              <Briefcase className="mx-auto mb-3" size={28} style={{ color: 'var(--text-tertiary)' }} />
              <p className="text-sm font-semibold mb-1" style={{ color: 'var(--text-secondary)' }}>{t('profile.noAgents')}</p>
              <p className="text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>{t('profile.startBuilding')}</p>
            </div>
          ) : (
            mySubmissions.map((strategy) => (
              <div
                key={strategy.id}
                className="group flex items-center justify-between rounded p-4 transition-all"
                style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border-hover)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
              >
                <div className="flex items-center gap-4">
                  <div
                    className="flex h-9 w-9 items-center justify-center rounded"
                    style={{
                      background: strategy.status === 'active' ? 'var(--green-dim)' :
                        strategy.status === 'pending' ? 'rgba(255,184,0,0.08)' : 'rgba(255,255,255,0.04)',
                      color: strategy.status === 'active' ? 'var(--green)' :
                        strategy.status === 'pending' ? 'var(--tier-partner)' : 'var(--text-tertiary)',
                    }}
                  >
                    <Code size={16} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-bold" style={{ color: 'var(--text-primary)' }}>
                        {strategy.name === user.name ? (strategy.id.slice(0, 8) + '...' + strategy.id.slice(-6)) : (strategy.name || strategy.id)}
                      </h3>
                      <span
                        className="text-[9px] font-mono font-bold uppercase tracking-widest px-1.5 py-0.5 rounded"
                        style={{
                          background: strategy.status === 'active' ? 'var(--green-dim)' :
                            strategy.status === 'pending' ? 'rgba(255,184,0,0.08)' : 'rgba(255,255,255,0.04)',
                          color: strategy.status === 'active' ? 'var(--green)' :
                            strategy.status === 'pending' ? 'var(--tier-partner)' : 'var(--text-tertiary)',
                        }}
                      >
                        {strategy.status}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-1">
                      <div className="text-[10px] font-mono" style={{ color: 'var(--text-tertiary)' }}>
                        {strategy.category.toUpperCase()}
                      </div>
                      <div className="flex items-center gap-1 text-[10px] font-mono" style={{ color: 'var(--tier-intern)' }}>
                        <Wallet size={10} />
                        {strategy.id.slice(0, 6)}...{strategy.id.slice(-4)}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-5">
                  <div className="hidden text-right sm:block">
                    <div className="text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>{t('profile.sharpe')}</div>
                    <div className="font-mono font-bold text-sm" style={{ color: 'var(--text-primary)' }}>
                      {strategy.backtestMetrics?.sharpeRatio.toFixed(2) ?? '-'}
                    </div>
                  </div>
                  <ChevronRight size={14} style={{ color: 'var(--text-tertiary)' }} className="group-hover:text-white transition-colors" />
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

export default Profile;
