import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategies } from '../store/slices/strategySlice';
import { Link } from 'react-router-dom';
import { Search, TrendingUp } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';

const Strategies = () => {
  const dispatch = useAppDispatch();
  const { items: strategies, loading } = useAppSelector((state) => state.strategies);
  const [searchTerm, setSearchTerm] = useState('');
  const [filter, setFilter] = useState('all');
  const { t } = useLanguage();

  useEffect(() => {
    dispatch(fetchStrategies());
  }, [dispatch]);

  const filteredStrategies = strategies.filter((strategy) => {
    const isActive = strategy.status === 'active';
    const matchesFilter = filter === 'all' || strategy.category === filter;
    const matchesSearch = strategy.name.toLowerCase().includes(searchTerm.toLowerCase());
    return isActive && matchesFilter && matchesSearch;
  });

  return (
    <div className="space-y-8">
      <div className="flex flex-col justify-between gap-4 md:flex-row md:items-center border-b border-zinc-100 pb-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-zinc-900">{t('strategies.title')}</h1>
          <p className="mt-2 text-zinc-500">{t('strategies.subtitle')}</p>
        </div>
        <div className="flex flex-wrap gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              placeholder={t('strategies.searchPlaceholder')}
              className="h-10 rounded-md border border-zinc-200 pl-10 pr-4 text-sm focus:border-black focus:outline-none focus:ring-1 focus:ring-black transition-colors"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="h-80 animate-pulse rounded-xl bg-zinc-100"></div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          {filteredStrategies.map((strategy) => (
            <div key={strategy.id} className="group flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-6 transition hover:border-black">
              <div>
                <div className="mb-6 flex items-start justify-between">
                  <div>
                    <div className="mb-3 flex items-center gap-2">
                      <span className="inline-block border border-zinc-200 rounded-full bg-zinc-50 px-2.5 py-0.5 text-xs font-medium text-zinc-600 uppercase tracking-wide">
                        {strategy.category}
                      </span>
                    </div>
                    <h3 className="text-lg font-bold text-zinc-900">{strategy.name}</h3>
                  </div>
                </div>

                <div className="mb-6 space-y-4">
                  <div className="flex justify-between text-sm items-center">
                    <span className="text-zinc-500">{t('strategies.card.apr')}</span>
                    <span className={`font-mono font-bold ${strategy.pnlContribution >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                       {strategy.pnlContribution > 0 ? '+' : ''}{strategy.pnlContribution.toFixed(2)}
                    </span>
                  </div>
                  <div className="space-y-1.5">
                    <div className="flex justify-between text-sm items-center">
                      <span className="text-zinc-500">{t('strategies.card.tvl')}</span>
                      <span className="font-mono font-medium text-zinc-900">${strategy.currentTvl.toLocaleString()}</span>
                    </div>
                    <div className="w-full rounded-full bg-zinc-100 h-1">
                      <div
                        className="h-1 rounded-full bg-black"
                        style={{ width: `${strategy.maxTvl > 0 ? (strategy.currentTvl / strategy.maxTvl) * 100 : 0}%` }}
                      ></div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="border-t border-zinc-100 pt-4">
                <Link
                  to={`/strategies/${strategy.id}`}
                  className="flex w-full items-center justify-center gap-2 rounded-md bg-white border border-zinc-200 py-2.5 text-sm font-medium text-zinc-900 transition hover:bg-zinc-50 hover:border-zinc-300"
                >
                  <TrendingUp size={16} />
                  {t('strategies.card.hire')}
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Strategies;
