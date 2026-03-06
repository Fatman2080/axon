import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategies } from '../store/slices/strategySlice';
import { Link } from 'react-router-dom';
import { Search, TrendingUp, ArrowRight } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';

const tierColor = (category: string) => {
  switch (category?.toLowerCase()) {
    case 'partner': return { bg: 'rgba(255,184,0,0.08)', text: 'var(--tier-partner)', border: 'rgba(255,184,0,0.2)' };
    case 'manager': return { bg: 'rgba(138,43,226,0.08)', text: 'var(--tier-manager)', border: 'rgba(138,43,226,0.2)' };
    case 'analyst': return { bg: 'rgba(42,127,255,0.1)', text: 'var(--tier-analyst)', border: 'rgba(42,127,255,0.2)' };
    default: return { bg: 'rgba(142,146,155,0.1)', text: 'var(--tier-intern)', border: 'rgba(142,146,155,0.2)' };
  }
};

const Strategies = () => {
  const dispatch = useAppDispatch();
  const { items: strategies, loading } = useAppSelector((state) => state.strategies);
  const [searchTerm, setSearchTerm] = useState('');
  const [filter] = useState('all');
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

  const isMarketEnabled = import.meta.env.VITE_ENABLE_STRATEGIES === 'true';

  if (!isMarketEnabled) {
    return (
      <div className="flex flex-col items-center justify-center py-32 text-center animate-fade-in-up">
        <div className="h-16 w-16 mb-6 rounded-full flex items-center justify-center" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
          <TrendingUp size={24} style={{ color: 'var(--text-tertiary)' }} />
        </div>
        <h2 className="text-xl font-bold tracking-tight mb-3" style={{ color: 'var(--text-primary)' }}>
          {t('strategies.underConstructionTitle')}
        </h2>
        <p className="text-sm font-mono max-w-md" style={{ color: 'var(--text-secondary)' }}>
          {t('strategies.underConstruction')}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-fade-in-up">
      {/* Header */}
      <div
        className="flex flex-col justify-between gap-4 md:flex-row md:items-center pb-6"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <div>
          <h1 className="text-2xl font-bold tracking-tight" style={{ color: 'var(--text-primary)' }}>
            {t('strategies.title')}
          </h1>
          <p className="mt-1 text-sm" style={{ color: 'var(--text-secondary)' }}>
            {t('strategies.subtitle')}
          </p>
        </div>

        <div className="flex flex-wrap gap-3 items-center">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" style={{ color: 'var(--text-tertiary)' }} />
            <input
              type="text"
              placeholder={t('strategies.searchPlaceholder')}
              className="h-9 pl-9 pr-4 text-sm font-mono"
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: '4px',
                color: 'var(--text-primary)',
                width: '200px',
              }}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              onFocus={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--neon-green)'}
              onBlur={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
            />
          </div>
        </div>
      </div>

      {/* Cards */}
      {loading ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map(i => (
            <div
              key={i}
              className="h-60 rounded animate-pulse"
              style={{ background: 'var(--bg-card)' }}
            />
          ))}
        </div>
      ) : filteredStrategies.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <TrendingUp size={40} className="mb-4" style={{ color: 'var(--text-tertiary)' }} />
          <p className="font-mono text-sm" style={{ color: 'var(--text-secondary)' }}>
            // NO_AGENTS_FOUND
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredStrategies.map((strategy) => {
            const tier = tierColor(strategy.category);
            const pnlPositive = strategy.pnlContribution >= 0;

            return (
              <div
                key={strategy.id}
                className="group flex flex-col justify-between rounded p-5 transition-all duration-200"
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                }}
                onMouseEnter={e => {
                  (e.currentTarget as HTMLElement).style.borderColor = 'var(--border-hover)';
                  (e.currentTarget as HTMLElement).style.background = 'var(--bg-card-hover)';
                }}
                onMouseLeave={e => {
                  (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)';
                  (e.currentTarget as HTMLElement).style.background = 'var(--bg-card)';
                }}
              >
                <div>
                  <div className="mb-4 flex items-start justify-between">
                    <div>
                      <span
                        className="inline-block text-[10px] font-mono font-bold uppercase tracking-widest px-2 py-0.5 rounded mb-2"
                        style={{ background: tier.bg, color: tier.text, border: `1px solid ${tier.border}` }}
                      >
                        {strategy.category}
                      </span>
                      <h3 className="text-base font-bold" style={{ color: 'var(--text-primary)' }}>
                        {strategy.name}
                      </h3>
                    </div>
                  </div>

                  <div className="space-y-3 mb-5">
                    {/* PnL */}
                    <div className="flex justify-between text-sm items-center">
                      <span className="font-mono text-xs uppercase tracking-widest" style={{ color: 'var(--text-tertiary)' }}>
                        {t('strategies.card.apr')}
                      </span>
                      <span
                        className="font-mono font-bold"
                        style={{ color: pnlPositive ? 'var(--green)' : 'var(--red)' }}
                      >
                        {pnlPositive ? '+' : ''}{strategy.pnlContribution.toFixed(2)}
                      </span>
                    </div>

                    {/* TVL */}
                    <div>
                      <div className="flex justify-between text-sm items-center mb-1.5">
                        <span className="font-mono text-xs uppercase tracking-widest" style={{ color: 'var(--text-tertiary)' }}>
                          {t('strategies.card.tvl')}
                        </span>
                        <span className="font-mono text-sm" style={{ color: 'var(--text-primary)' }}>
                          ${strategy.currentTvl.toLocaleString()}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <Link
                  to={`/strategies/${strategy.id}`}
                  className="flex items-center justify-between text-sm font-mono font-bold transition-all group-hover:gap-3 pt-4"
                  style={{
                    color: 'var(--text-secondary)',
                    borderTop: '1px solid var(--border)',
                  }}
                  onMouseEnter={e => (e.currentTarget as HTMLElement).style.color = 'var(--neon-green)'}
                  onMouseLeave={e => (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)'}
                >
                  {t('strategies.card.hire')}
                  <ArrowRight size={14} />
                </Link>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default Strategies;
