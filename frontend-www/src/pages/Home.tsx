import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategies } from '../store/slices/strategySlice';
import { Link } from 'react-router-dom';
import { ArrowRight, Shield, Lock, BarChart3, Database, Eye, Award } from 'lucide-react';
import { VaultOverview, DailySlotsResponse, TreasurySnapshot } from "../types";
import { useLanguage } from '../context/LanguageContext';
import { marketApi } from '../services/api';

const defaultStats: VaultOverview = {
  totalTvl: 0,
  totalEvmBalance: 0,
  totalL1Value: 0,
  agentCount: 0,
  totalPnl: 0,
  totalInitialCapital: 0,
  positions: [],
  recentFills: [],
};
const StatCard = ({ label, value, accent }: { label: string; value: string; accent?: string }) => (
  <div
    className="p-5 rounded cyber-card tracking-tight"
    style={{
      background: 'var(--bg-card)',
    }}
  >
    <div className="text-xs font-mono uppercase tracking-widest mb-2" style={{ color: 'var(--text-tertiary)' }}>
      {label}
    </div>
    <div
      className="text-2xl font-bold font-mono"
      style={{ color: accent || 'var(--text-primary)' }}
    >
      {value}
    </div>
  </div>
);

/* ═══════════════════════════════════════════
   Architecture Diagram — Canvas-based animated flow
   ═══════════════════════════════════════════ */
/* Architecture Diagram hidden per user request 
const ArchitectureDiagram = () => { ... } */

const Home = () => {
  const dispatch = useAppDispatch();
  const { items: strategies } = useAppSelector((state) => state.strategies);
  const [dailySlots, setDailySlots] = useState<DailySlotsResponse | null>(null);
  const [vaultStats, setVaultStats] = useState<VaultOverview | null>(null);
  const [treasury, setTreasury] = useState<TreasurySnapshot | null>(null);
  const [treasuryHistory, setTreasuryHistory] = useState<TreasurySnapshot[]>([]);
  const { t } = useLanguage();

  const fetchSlots = () => {
    marketApi.getDailySlots().then(setDailySlots).catch(() => {});
  };

  useEffect(() => {
    dispatch(fetchStrategies());
    marketApi.getVaultOverview().then(setVaultStats).catch(() => {});
    marketApi.getTreasury().then(setTreasury).catch(() => {});
    marketApi.getTreasuryHistory('30d').then(data => setTreasuryHistory(Array.isArray(data) ? data : [])).catch(() => {});
    fetchSlots();
    const refreshInterval = setInterval(fetchSlots, 60000);
    return () => clearInterval(refreshInterval);
  }, [dispatch]);

  useEffect(() => {
    if (!dailySlots) return;
    const resetTime = new Date(dailySlots.resetsAt).getTime();
    const diff = resetTime - Date.now();
    if (diff <= 0) { fetchSlots(); return; }
    const timer = setTimeout(fetchSlots, diff);
    return () => clearTimeout(timer);
  }, [dailySlots]);

  const stats = vaultStats ?? defaultStats;
  const agentCount = stats?.agentCount || strategies?.length || 0;
  const tvlValue = treasury?.totalFunds ?? 0;
  const totalPnl = treasury?.vaultPnl ?? stats?.totalPnl ?? 0;

  const fmtUsd = (v: number) =>
    v >= 1000000
      ? `$${(v / 1000000).toFixed(1)}M`
      : `$${v.toLocaleString(undefined, { maximumFractionDigits: 0 })}`;

  const features = [
    {
      icon: Eye,
      title: t('home.cards.rideVolatility.title'),
      desc: t('home.cards.rideVolatility.desc'),
      action: t('home.cards.rideVolatility.action'),
      href: '/submit-agent',
      accent: '#00FF66',
      number: '01',
    },
    {
      icon: Award,
      title: t('home.cards.copyWhales.title'),
      desc: t('home.cards.copyWhales.desc'),
      action: t('home.cards.copyWhales.action'),
      href: '/submit-agent',
      accent: '#00FF41',
      number: '02',
    },
    {
      icon: Lock,
      title: t('home.cards.openClawEdge.title'),
      desc: t('home.cards.openClawEdge.desc'),
      action: t('home.cards.openClawEdge.action'),
      href: '/submit-agent',
      accent: '#FFB800',
      number: '03',
    },
  ];

  const riskItems = [
    { icon: Shield, title: t('home.risk.hardConstraints.title'), desc: t('home.risk.hardConstraints.desc'), color: 'var(--green)' },
    { icon: Lock, title: t('home.risk.nonCustodial.title'), desc: t('home.risk.nonCustodial.desc'), color: 'var(--neon-green)' },
    { icon: BarChart3, title: t('home.risk.promotion.title'), desc: t('home.risk.promotion.desc'), color: 'var(--tier-analyst)' },
    { icon: Database, title: t('home.risk.verifiable.title'), desc: t('home.risk.verifiable.desc'), color: 'var(--tier-manager)' },
  ];

  return (
    <div className="space-y-20 pb-24 animate-fade-in-up">

      {/* ── Hero ── */}
      <section className="relative pt-12 pb-8 max-w-5xl mx-auto text-center">
        {/* Live badge */}
        <div className="inline-flex items-center gap-2 mb-8">
          <span
            className="flex items-center gap-2 text-xs font-mono px-3 py-1.5 rounded"
            style={{
              background: 'var(--green-dim)',
              color: 'var(--green)',
              border: '1px solid rgba(0,255,102,0.2)',
            }}
          >
            <span className="status-dot" />
            {t('home.tag')}
          </span>
        </div>

        <h1
          className="text-5xl md:text-7xl font-black tracking-[-0.04em] mb-6 leading-[1.05] font-mono"
          style={{ 
            color: 'var(--text-primary)',
            textShadow: '0 0 40px rgba(0, 255, 65, 0.15)'
          }}
        >
          {t('home.titleStart')}
          <span
            style={{
              color: 'var(--neon-green)',
              textShadow: '0 0 20px rgba(0, 255, 65, 0.4)'
            }}
          >
            {t('home.titleHighlight')}
          </span>
          {t('home.titleEnd')}
        </h1>

        <p className="text-lg md:text-xl max-w-2xl mx-auto mb-4 leading-relaxed font-mono" style={{ color: 'var(--text-secondary)' }}>
          {t('home.subtitle')}
        </p>
        <p className="text-base font-semibold mb-10 font-mono" style={{ color: 'var(--text-primary)' }}>
          {t('home.subtitleBold')}
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link
            to="/submit-agent"
            className="h-11 px-8 rounded text-sm font-bold flex items-center gap-2 transition-all hover:-translate-y-px animate-pulse-glow"
            style={{ background: 'var(--neon-green)', color: '#000' }}
          >
            {t('home.ctaApply')}
            <ArrowRight size={16} />
          </Link>
          <Link
            to="/strategies"
            className="h-11 px-8 rounded text-sm font-medium flex items-center gap-2 transition-all"
            style={{
              background: 'transparent',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
            }}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--border-hover)'; (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)'; }}
          >
            {t('home.ctaSee')}
          </Link>
        </div>

        {/* Daily slots */}
        <div className="mt-10 flex flex-col items-center gap-2">
          <div className="flex items-center gap-2 text-sm font-mono" style={{ color: 'var(--text-secondary)' }}>
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--red)' }} />
              <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: 'var(--red)' }} />
            </span>
            {t('home.dailySlots.label')}{' '}
            <span className="font-bold" style={{ color: 'var(--text-primary)' }}>
              {dailySlots ? `${dailySlots.remaining} / ${dailySlots.total}` : '— / —'}
            </span>
            {' '}{t('home.dailySlots.remaining')}
          </div>
          <div className="h-1 w-56 rounded-full overflow-hidden" style={{ background: 'rgba(255,255,255,0.06)' }}>
            <div
              className="h-full transition-all duration-1000 ease-out"
              style={{
                width: `${dailySlots ? (dailySlots.remaining / dailySlots.total) * 100 : 0}%`,
                background: 'linear-gradient(90deg, var(--green), var(--neon-green))',
              }}
            />
          </div>
          <p className="text-xs font-mono animate-pulse" style={{ color: 'var(--text-tertiary)' }}>
            {t('home.dailySlots.resets')}
          </p>
        </div>

        {/* Stats Grid */}
        <div
          className="grid grid-cols-2 md:grid-cols-3 gap-3 mt-16 pt-12"
          style={{ borderTop: '1px solid var(--border)' }}
        >
          <StatCard label={t('home.stats.tvl')} value={fmtUsd(tvlValue)} accent="var(--neon-green)" />
          <StatCard label={t('home.stats.agents')} value={`${agentCount}+`} accent="var(--green)" />
          <StatCard
            label={t('home.stats.yield')}
            value={`${totalPnl >= 0 ? '+' : ''}${fmtUsd(Math.abs(totalPnl))}`}
            accent={totalPnl >= 0 ? 'var(--green)' : 'var(--red)'}
          />
        </div>

        {/* Total Assets Chart */}
        {treasuryHistory.length > 1 && (
          <div className="mt-8 p-5 rounded cyber-card" style={{ background: 'var(--bg-card)' }}>
            <div className="flex items-center justify-between mb-3">
              <div className="text-xs font-mono uppercase tracking-widest" style={{ color: 'var(--text-tertiary)' }}>
                Total Assets (30d)
              </div>
              <div className="text-sm font-bold" style={{ color: 'var(--neon-green)' }}>
                {fmtUsd(treasuryHistory[treasuryHistory.length - 1]?.totalFunds ?? 0)}
              </div>
            </div>
            <svg viewBox={`0 0 ${Math.max(treasuryHistory.length - 1, 1)} 100`} className="w-full" style={{ height: '120px' }} preserveAspectRatio="none">
              {(() => {
                const rawValues = treasuryHistory.map(s => s.totalFunds || 0);
                // filter out 0 values if there are valid ones
                const validValues = rawValues.filter(v => v > 0);
                const values = rawValues.map(v => (v > 0 ? v : (validValues[0] || 0)));
                
                const min = Math.min(...values);
                const max = Math.max(...values);
                const pMin = min === max ? min * 0.99 : min - (max - min) * 0.1;
                const pMax = min === max ? max * 1.01 : max + (max - min) * 0.1;
                const range = pMax - pMin || 1;
                const points = values.map((v, i) => `${i},${100 - ((v - pMin) / range) * 100}`).join(' ');
                return (
                  <>
                    <polyline fill="none" stroke="var(--neon-green)" strokeWidth="2" vectorEffect="non-scaling-stroke" points={points} />
                    <polyline fill="url(#treasuryGrad)" stroke="none" points={`0,100 ${points} ${Math.max(values.length - 1, 1)},100`} />
                    <defs>
                      <linearGradient id="treasuryGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="var(--neon-green)" stopOpacity="0.15" />
                        <stop offset="100%" stopColor="var(--neon-green)" stopOpacity="0" />
                      </linearGradient>
                    </defs>
                  </>
                );
              })()}
            </svg>
          </div>
        )}
      </section>

      {/* ── Visual Architecture Diagram (Hidden) ── */}
      {/* <ArchitectureDiagram /> */}

      {/* ── Feature Cards — Modern Fintech Bento Style ── */}
      <section className="max-w-[1200px] mx-auto">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-px rounded-xl overflow-hidden" style={{ background: 'var(--border)' }}>
          {features.map((f, i) => (
            <div
              key={i}
              className="group relative flex flex-col justify-between p-8 min-h-[320px] transition-all duration-300 overflow-hidden"
              style={{ background: 'var(--bg-card)' }}
              onMouseEnter={e => {
                (e.currentTarget as HTMLElement).style.background = 'rgba(22,24,26,1)';
              }}
              onMouseLeave={e => {
                (e.currentTarget as HTMLElement).style.background = 'var(--bg-card)';
              }}
            >
              {/* Top number */}
              <div className="flex items-start justify-between mb-8">
                <span className="text-[64px] font-black font-mono leading-none opacity-[0.04] group-hover:opacity-[0.08] transition-opacity select-none" style={{ color: f.accent }}>
                  {f.number}
                </span>
                <div
                  className="h-11 w-11 rounded-lg flex items-center justify-center transition-all duration-300 group-hover:scale-110"
                  style={{ background: `${f.accent}10`, border: `1px solid ${f.accent}30` }}
                >
                  <f.icon size={22} style={{ color: f.accent }} />
                </div>
              </div>

              <div>
                <h3 className="text-lg font-bold mb-3 tracking-tight font-mono" style={{ color: 'var(--text-primary)' }}>
                  {f.title}
                </h3>
                <p className="text-sm leading-relaxed mb-6 font-mono" style={{ color: 'var(--text-secondary)' }}>
                  {f.desc}
                </p>

                <Link
                  to={f.href}
                  className="inline-flex items-center gap-2 text-sm font-bold font-mono transition-all group-hover:gap-3"
                  style={{ color: f.accent }}
                >
                  {f.action}
                  <ArrowRight size={14} className="transition-transform group-hover:translate-x-1" />
                </Link>
              </div>

              {/* Bottom accent line */}
              <div
                className="absolute bottom-0 left-0 w-full h-px transition-all duration-500 group-hover:h-[2px]"
                style={{ background: `linear-gradient(90deg, transparent, ${f.accent}, transparent)`, opacity: 0.3 }}
              />
            </div>
          ))}
        </div>
      </section>

      {/* ── Risk Management ── */}
      <section className="max-w-5xl mx-auto">
        <div className="text-center mb-12">
          <div
            className="inline-block text-xs font-mono uppercase tracking-widest px-3 py-1 rounded mb-4"
            style={{
              background: 'rgba(138,43,226,0.08)',
              color: 'var(--tier-manager)',
              border: '1px solid rgba(138,43,226,0.2)',
            }}
          >
            RISK_FRAMEWORK
          </div>
          <h2 className="text-3xl font-extrabold tracking-tight mb-3 font-mono" style={{ color: 'var(--text-primary)' }}>
            {t('home.risk.title')}
          </h2>
          <p className="max-w-2xl mx-auto font-mono" style={{ color: 'var(--text-secondary)' }}>
            {t('home.risk.subtitle')}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {riskItems.map((item, i) => (
            <div
              key={i}
              className="flex gap-5 p-6 rounded cyber-card"
              style={{ background: 'var(--bg-card)' }}
            >
              <div
                className="shrink-0 h-10 w-10 rounded flex items-center justify-center"
                style={{ background: `${item.color}14`, border: `1px solid ${item.color}30` }}
              >
                <item.icon size={20} style={{ color: item.color }} />
              </div>
              <div>
                <h3 className="font-bold mb-1.5" style={{ color: 'var(--text-primary)' }}>
                  {item.title}
                </h3>
                <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
                  {item.desc}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

    </div>
  );
};

export default Home;
