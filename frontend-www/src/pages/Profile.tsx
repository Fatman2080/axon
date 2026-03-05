import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchUser } from '../store/slices/userSlice';
import { fetchStrategies } from '../store/slices/strategySlice';
import {
  User, Mail, Wallet, Calendar, Bell, Shield, Code, Clock,
  CheckCircle, XCircle, Copy, ExternalLink, Settings, LogOut,
  ChevronRight, Award, Briefcase, DollarSign
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

const Profile = () => {
  const dispatch = useAppDispatch();
  const { currentUser: user, loading: userLoading } = useAppSelector((state) => state.user);
  const { items: strategies, loading: strategiesLoading } = useAppSelector((state) => state.strategies);
  const [activeTab, setActiveTab] = useState('portfolio');
  const [agentHistory, setAgentHistory] = useState<number[]>([]);
  const [chartPeriod, setChartPeriod] = useState('1W');
  const { t } = useLanguage();

  useEffect(() => {
    dispatch(fetchUser());
    dispatch(fetchStrategies());
  }, [dispatch]);

  useEffect(() => {
    if (!user) return;
    const loadData = async () => {
      try {
        const historyData = await authApi.getAgentHistory(chartPeriod);
        setAgentHistory(Array.isArray(historyData.history) ? historyData.history : []);
      } catch {
        setAgentHistory([]);
      }
    };
    loadData();
  }, [user, chartPeriod]);

  const loading = userLoading || strategiesLoading;

  if (loading) {
    return <div className="h-96 w-full animate-pulse rounded-xl bg-zinc-100"></div>;
  }

  if (!user) {
    return (
      <div className="mx-auto max-w-md py-20 text-center">
        <User size={48} className="mx-auto mb-4 text-zinc-300" />
        <h2 className="text-xl font-bold text-zinc-900 mb-2">{t('profile.loginRequired') || 'Login Required'}</h2>
        <p className="text-sm text-zinc-500 mb-6">{t('profile.loginDesc') || 'Please login with X to view your profile.'}</p>
        <button
          onClick={() => {
            const nextPath = window.location.pathname;
            window.location.href = authApi.getXOAuthStartUrl(undefined, nextPath);
          }}
          className="inline-flex items-center gap-2 rounded-full bg-black px-6 py-2.5 text-sm font-bold text-white transition hover:bg-zinc-800"
        >
          {t('nav.connectWallet') || 'Login with X'}
        </button>
      </div>
    );
  }

  const mySubmissions = strategies.filter(s => user.agentPublicKey ? s.id === user.agentPublicKey : false);

  const accountValue = user.totalInvestment ?? 0;
  const totalPnl = user.totalProfit ?? 0;
  const totalInvestment = accountValue;
  const totalProfit = totalPnl;
  const agentCount = user.agentPublicKey ? 1 : (user.agentCount ?? 0);
  const lpShares = user.lpShares ?? 0;

  const chartSeries = agentHistory.length > 0
    ? agentHistory
    : [];

  const chartData = {
    labels: Array.from({ length: chartSeries.length }, (_, i) => `Day ${i + 1}`),
    datasets: [
      {
        label: t('profile.portfolio'),
        data: chartSeries,
        borderColor: '#10b981', // Emerald 500
        backgroundColor: 'rgba(16, 185, 129, 0.05)',
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
      x: { display: false },
      y: { 
        display: false,
        grid: { display: false } 
      },
    },
    interaction: {
      mode: 'nearest' as const,
      axis: 'x' as const,
      intersect: false,
    },
  };

  return (
    <div className="mx-auto max-w-6xl pb-20">
      {/* Header Profile Section */}
      <div className="mb-8 flex flex-col gap-6 md:flex-row md:items-start md:justify-between">
        <div className="flex items-center gap-6">
          <div className="relative h-24 w-24 shrink-0">
            <div className="h-full w-full overflow-hidden rounded-full border-2 border-zinc-100 bg-zinc-50">
              {user.avatar ? (
                <img src={user.avatar} alt={user.name} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full w-full items-center justify-center text-zinc-300">
                  <User size={40} />
                </div>
              )}
            </div>
            <div className="absolute bottom-0 right-0 flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500 ring-2 ring-white">
              <CheckCircle size={14} className="text-white" />
            </div>
          </div>

          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-zinc-900">{user.name}</h1>
              <span className={`rounded-md px-2 py-0.5 text-xs font-bold uppercase tracking-wider ${
                user.level === 'vip' ? 'bg-amber-100 text-amber-700' : 'bg-zinc-100 text-zinc-600'
              }`}>
                {user.level}
              </span>
            </div>
            <div className="mt-2 flex items-center gap-4 text-sm text-zinc-500">
              <div className="flex items-center gap-1.5 font-mono">
                <Wallet size={14} />
                {user.agentPublicKey
                  ? `${user.agentPublicKey.slice(0, 6)}...${user.agentPublicKey.slice(-4)}`
                  : (user.walletAddress || '-')}
                <Copy size={12} className="cursor-pointer hover:text-black" />
              </div>
              <div className="flex items-center gap-1.5">
                <Calendar size={14} />
                {t('profile.joined')} {(user as any).createdAt ? new Date((user as any).createdAt).toLocaleDateString() : user.joinedAt ? new Date(user.joinedAt).toLocaleDateString() : '-'}
              </div>
            </div>
          </div>
        </div>

        <button className="flex items-center gap-2 rounded-md border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors">
          <LogOut size={16} />
        </button>
      </div>

      {/* Account Summary Stats Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div className="rounded-xl border border-zinc-200 bg-white p-5">
          <div className="flex items-center gap-2 text-zinc-500 mb-2">
            <Wallet size={16} />
            <span className="text-xs font-medium">{t('profile.totalEquity')}</span>
          </div>
          <div className="text-xl font-bold text-zinc-900 font-mono">
            ${totalInvestment.toLocaleString()}
          </div>
        </div>
        <div className="rounded-xl border border-zinc-200 bg-white p-5">
          <div className="flex items-center gap-2 text-zinc-500 mb-2">
            <Award size={16} />
            <span className="text-xs font-medium">{t('profile.unrealizedPnL')}</span>
          </div>
          <div className={`text-xl font-bold font-mono ${totalProfit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
            {totalProfit > 0 ? '+' : ''}${totalProfit.toLocaleString()}
          </div>
        </div>
        <div className="rounded-xl border border-zinc-200 bg-white p-5">
          <div className="flex items-center gap-2 text-zinc-500 mb-2">
            <Briefcase size={16} />
            <span className="text-xs font-medium">{t('profile.activeAgents')}</span>
          </div>
          <div className="text-xl font-bold text-zinc-900 font-mono">{agentCount}</div>
        </div>
        <div className="rounded-xl border border-zinc-200 bg-white p-5">
          <div className="flex items-center gap-2 text-zinc-500 mb-2">
            <DollarSign size={16} />
            <span className="text-xs font-medium">{t('profile.vaultShares')}</span>
          </div>
          <div className="text-xl font-bold text-zinc-900 font-mono">{lpShares.toLocaleString()}</div>
        </div>
      </div>

      {/* Portfolio Chart - Full Width */}
      <div className="rounded-xl border border-zinc-200 bg-white p-6 mb-8">
        <div className="mb-6 flex items-end justify-between">
          <div>
            <div className="text-sm font-medium text-zinc-500 mb-1">{t('profile.portfolio')}</div>
            <div className="text-3xl font-bold text-zinc-900 font-mono tracking-tight">
              ${(totalInvestment + totalProfit).toLocaleString()}
            </div>
            <div className="mt-1 flex items-center gap-2 text-sm">
              <span className={`font-medium ${totalPnl >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                {totalPnl > 0 ? '+' : ''}${totalPnl.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                {totalInvestment > 0 && totalPnl !== 0 && ` (${((totalPnl / (totalInvestment - totalPnl)) * 100).toFixed(1)}%)`}
              </span>
            </div>
          </div>
          {chartSeries.length > 0 && (
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
          )}
        </div>
        {chartSeries.length > 0 ? (
          <div className="h-[280px] w-full">
            <Line data={chartData} options={chartOptions} />
          </div>
        ) : (
          <div className="h-[280px] w-full flex items-center justify-center text-zinc-400 text-sm">
            {t('profile.noChartData') || 'No performance data available yet'}
          </div>
        )}
      </div>

      {/* Submissions List - Full Width */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-zinc-900">{t('profile.myStrategies')}</h2>
          <button className="text-sm font-medium text-zinc-500 hover:text-black">{t('profile.viewAll')}</button>
        </div>

        <div className="space-y-3">
          {mySubmissions.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-300 bg-zinc-50 p-8 text-center">
              <Briefcase className="mx-auto mb-3 text-zinc-400" size={32} />
              <p className="text-sm font-medium text-zinc-900">{t('profile.noStrategies')}</p>
              <p className="text-xs text-zinc-500 mt-1">{t('profile.startBuilding')}</p>
            </div>
          ) : (
            mySubmissions.map((strategy) => (
              <div key={strategy.id} className="group flex items-center justify-between rounded-xl border border-zinc-200 bg-white p-4 transition hover:border-black">
                <div className="flex items-center gap-4">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                    strategy.status === 'active' ? 'bg-emerald-50 text-emerald-600' :
                    strategy.status === 'pending' ? 'bg-amber-50 text-amber-600' :
                    'bg-zinc-100 text-zinc-500'
                  }`}>
                    <Code size={20} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-bold text-zinc-900">{strategy.name}</h3>
                      <span className={`rounded-sm px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide ${
                        strategy.status === 'active' ? 'bg-emerald-100 text-emerald-700' :
                        strategy.status === 'pending' ? 'bg-amber-100 text-amber-700' :
                        'bg-zinc-100 text-zinc-600'
                      }`}>
                        {strategy.status}
                      </span>
                    </div>
                    <div className="text-xs text-zinc-500 font-mono mt-0.5">
                      {strategy.category.toUpperCase()}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-6">
                  <div className="hidden text-right sm:block">
                    <div className="text-xs text-zinc-500">{t('profile.sharpe')}</div>
                    <div className="font-mono font-medium text-zinc-900">
                      {strategy.backtestMetrics?.sharpeRatio.toFixed(2) ?? '-'}
                    </div>
                  </div>
                  <ChevronRight size={16} className="text-zinc-300 group-hover:text-black transition-colors" />
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
