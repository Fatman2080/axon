import React, { useEffect, useState } from 'react';
import { User, UserAgentStats } from '../../types';
import { authApi } from '../../services/api';
import { useLanguage } from '../../context/LanguageContext';
import { Link } from 'react-router-dom';
import { Plus, TrendingUp, TrendingDown, Activity, Clock, Shield, ArrowRight } from 'lucide-react';
import { Line } from 'react-chartjs-2';

interface UserDashboardProps {
  user: User;
}

const UserDashboard: React.FC<UserDashboardProps> = ({ user }) => {
  const { t } = useLanguage();
  const activeAgents = user.agents?.filter(a => a.status === 'running') || [];
  const [historyData, setHistoryData] = useState<number[]>([]);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [agentStats, setAgentStats] = useState<UserAgentStats | null>(null);
  const accountValue = agentStats?.accountValue ?? user.totalInvestment ?? 0;
  const totalPnl = agentStats?.totalPnl ?? user.totalProfit ?? 0;
  const agentCount = user.agentPublicKey ? 1 : (user.agentCount ?? 0);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [histRes, statsRes] = await Promise.all([
          authApi.getAgentHistory().catch(() => ({ history: [] })),
          authApi.getAgentStats().catch(() => null),
        ]);
        if (histRes.history && histRes.history.length > 0) {
          setHistoryData(histRes.history);
        }
        if (statsRes) setAgentStats(statsRes);
      } catch {
        // ignore
      } finally {
        setHistoryLoading(false);
      }
    };
    loadData();
  }, []);

  const chartValues = historyData;

  const chartData = {
    labels: chartValues.map((_, i) => `Point ${i + 1}`),
    datasets: [
      {
        label: t('profile.portfolio'),
        data: chartValues,
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
      x: { display: false },
      y: { display: false },
    },
    interaction: {
      mode: 'nearest' as const,
      axis: 'x' as const,
      intersect: false,
    },
  };

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Welcome & Quick Actions */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-zinc-900">
            {`${t('dashboard.welcome')} ${user.name || 'Trader'}`}
          </h1>
          <p className="text-zinc-500 mt-1">
            {t('dashboard.overview')}
          </p>
        </div>
        <div className="flex gap-3">
          <Link
            to="/strategies"
            className="flex items-center gap-2 rounded-lg bg-black px-4 py-2 text-sm font-bold text-white hover:bg-zinc-800 transition-colors"
          >
            <Plus size={16} />
            {t('dashboard.newAgent')}
          </Link>
        </div>
      </div>

      {/* Portfolio Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="col-span-1 md:col-span-2 rounded-xl border border-zinc-200 bg-white p-6 shadow-sm">
          <div className="flex justify-between items-start mb-6">
            <div>
              <div className="text-sm font-medium text-zinc-500 mb-1">{t('dashboard.totalEquity')}</div>
              <div className="text-3xl font-bold text-zinc-900 font-mono">
                ${historyData.length > 0
                  ? historyData[historyData.length - 1].toLocaleString(undefined, { maximumFractionDigits: 2 })
                  : accountValue.toLocaleString(undefined, { maximumFractionDigits: 2 })}
              </div>
              <div className={`flex items-center gap-1 text-sm font-medium mt-1 ${totalPnl >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                {totalPnl >= 0 ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                {totalPnl >= 0 ? '+' : ''}${totalPnl.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                <span className="text-zinc-400 font-normal ml-1">(All Time)</span>
              </div>
            </div>
            <div className="hidden sm:block">
              <div className="flex gap-2">
                {['1D', '1W', '1M', 'ALL'].map(period => (
                  <button
                    key={period}
                    className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                      period === '1W' ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-100'
                    }`}
                  >
                    {period}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <div className="h-[200px] w-full">
            {historyLoading ? (
              <div className="h-full flex items-center justify-center text-zinc-400 text-sm">Loading chart...</div>
            ) : chartValues.length === 0 ? (
              <div className="h-full flex items-center justify-center text-zinc-400 text-sm">No history data</div>
            ) : (
              <Line data={chartData} options={chartOptions} />
            )}
          </div>
        </div>

        <div className="space-y-6">
          <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm h-full flex flex-col justify-center">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-emerald-50 text-emerald-600">
                <Activity size={20} />
              </div>
              <span className="text-sm font-medium text-zinc-500">{t('dashboard.activeAgents')}</span>
            </div>
            <div className="text-3xl font-bold text-zinc-900 font-mono mb-1">
              {activeAgents.length}
            </div>
            <div className="text-xs text-zinc-400">
              {agentCount} total deployed
            </div>
          </div>

          <div className="rounded-xl border border-zinc-200 bg-zinc-900 p-6 shadow-sm h-full flex flex-col justify-center text-white">
            <div className="flex items-center gap-3 mb-2">
              <div className="p-2 rounded-lg bg-zinc-800 text-emerald-400">
                <Shield size={20} />
              </div>
              <span className="text-sm font-medium text-zinc-400">{t('dashboard.riskScore')}</span>
            </div>
            <div className="text-3xl font-bold font-mono mb-1">
              Low
            </div>
            <div className="text-xs text-zinc-500">
              Conservative portfolio
            </div>
          </div>
        </div>
      </div>

      {/* Active Agents List */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-zinc-900">{t('dashboard.myAgents')}</h2>
          <Link to="/vault" className="text-sm font-medium text-zinc-500 hover:text-black flex items-center gap-1">
            {t('dashboard.viewAll')} <ArrowRight size={14} />
          </Link>
        </div>

        {activeAgents.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {activeAgents.map(agent => (
              <div key={agent.id} className="group rounded-xl border border-zinc-200 bg-white p-5 hover:border-black transition-colors">
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <h3 className="font-bold text-zinc-900">{agent.name}</h3>
                    <div className="text-xs text-zinc-500 font-mono mt-1">
                      Started {new Date(agent.createdAt).toLocaleDateString()}
                    </div>
                  </div>
                  <span className="flex h-2 w-2 relative">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                  </span>
                </div>

                <div className="space-y-3 mb-4">
                  <div className="flex justify-between text-sm">
                    <span className="text-zinc-500">TVL</span>
                    <span className="font-mono font-medium">${agent.currentValue.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-zinc-500">PnL (24h)</span>
                    <span className={`font-mono font-medium ${agent.todayProfit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                      {agent.todayProfit >= 0 ? '+' : ''}{agent.todayProfit}%
                    </span>
                  </div>
                </div>

                <div className="pt-4 border-t border-zinc-100 flex gap-2">
                  <button className="flex-1 rounded-md bg-zinc-50 py-2 text-xs font-bold text-zinc-600 hover:bg-zinc-100 transition-colors">
                    Settings
                  </button>
                  <button className="flex-1 rounded-md bg-zinc-50 py-2 text-xs font-bold text-red-600 hover:bg-red-50 transition-colors">
                    Stop
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-zinc-300 bg-zinc-50 p-12 text-center">
            <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-zinc-200 flex items-center justify-center text-zinc-400">
              <Clock size={24} />
            </div>
            <h3 className="text-lg font-bold text-zinc-900 mb-2">{t('dashboard.noAgents')}</h3>
            <p className="text-zinc-500 mb-6 max-w-md mx-auto">
              {t('dashboard.noAgentsDesc')}
            </p>
            <Link
              to="/strategies"
              className="inline-flex items-center gap-2 rounded-lg bg-black px-6 py-3 text-sm font-bold text-white hover:bg-zinc-800 transition-colors"
            >
              <Plus size={16} />
              {t('dashboard.deployFirst')}
            </Link>
          </div>
        )}
      </div>
    </div>
  );
};

export default UserDashboard;
