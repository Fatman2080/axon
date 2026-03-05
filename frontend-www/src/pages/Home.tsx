import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { fetchStrategies } from '../store/slices/strategySlice';
import { Link } from 'react-router-dom';
import { ArrowRight, TrendingUp, Coins, Shield, Database, Zap, BarChart3, Lock } from 'lucide-react';
import { VaultStats, VaultOverview, DailySlotsResponse } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { marketApi } from '../services/api';

const defaultStats: VaultOverview = {
  totalTvl: 0,
  totalEvmBalance: 0,
  totalL1Value: 0,
  agentCount: 0,
  totalPnl: 0,
  positions: [],
  recentFills: [],
};

const Home = () => {
  const dispatch = useAppDispatch();
  const { items: strategies } = useAppSelector((state) => state.strategies);
  const [dailySlots, setDailySlots] = useState<DailySlotsResponse | null>(null);
  const [countdown, setCountdown] = useState('');
  const [vaultStats, setVaultStats] = useState<VaultOverview | null>(null);
  const { t } = useLanguage();

  const fetchSlots = () => {
    marketApi.getDailySlots().then(setDailySlots).catch(() => {});
  };

  useEffect(() => {
    dispatch(fetchStrategies());
    marketApi.getVaultOverview().then(setVaultStats).catch(() => {});
    fetchSlots();

    const refreshInterval = setInterval(fetchSlots, 30000);
    return () => clearInterval(refreshInterval);
  }, [dispatch]);

  useEffect(() => {
    if (!dailySlots) return;
    const tick = () => {
      const now = Date.now();
      const resetTime = new Date(dailySlots.resetsAt).getTime();
      const diff = resetTime - now;
      if (diff <= 0) {
        setCountdown('00:00:00');
        fetchSlots();
        return;
      }
      const h = String(Math.floor(diff / 3600000)).padStart(2, '0');
      const m = String(Math.floor((diff % 3600000) / 60000)).padStart(2, '0');
      const s = String(Math.floor((diff % 60000) / 1000)).padStart(2, '0');
      setCountdown(`${h}:${m}:${s}`);
    };
    tick();
    const timer = setInterval(tick, 1000);
    return () => clearInterval(timer);
  }, [dailySlots]);

  const stats = vaultStats ?? defaultStats;
  const agentCount = stats.agentCount || strategies.length;
  const tvlValue = stats.totalTvl;
  const totalPnl = stats.totalPnl ?? 0;
  const apy = tvlValue > 0 && totalPnl !== 0 ? (totalPnl / (tvlValue - totalPnl)) * 100 : 0;

  return (
    <div className="space-y-24 pb-20">
      {/* Hero Section */}
      <section className="relative pt-20 pb-16 text-center max-w-5xl mx-auto px-4">
        <div className="inline-flex items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50/50 px-3 py-1 text-xs font-medium text-zinc-600 mb-8 backdrop-blur-sm">
          <span className="flex h-2 w-2 relative">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          {t('home.tag')}
        </div>
        
        <h1 className="text-5xl md:text-7xl font-bold tracking-tighter text-zinc-900 mb-8 leading-[1.1]">
          {t('home.title').split(' ').slice(0, 3).join(' ')} <br />
          <span className="bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent">
             {t('home.title').split(' ').slice(3).join(' ')}
          </span>
        </h1>
        
        <p className="text-xl text-zinc-500 max-w-2xl mx-auto mb-12 leading-relaxed">
          {t('home.subtitle')}
          <span className="block mt-2 font-medium text-zinc-900">
            {t('home.subtitleBold')}
          </span>
        </p>
        
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link
            to="/submit-agent"
            className="h-12 px-8 rounded-full bg-zinc-900 text-white font-medium flex items-center justify-center hover:bg-zinc-800 transition-all hover:scale-105 active:scale-95"
          >
            {t('home.ctaApply')}
          </Link>
          <Link
            to="/strategies"
            className="h-12 px-8 rounded-full border border-zinc-200 text-zinc-900 font-medium flex items-center justify-center hover:bg-zinc-50 transition-all"
          >
            {t('home.ctaSee')}
          </Link>
        </div>

        {/* Daily Slots Counter */}
        <div className="mt-8 flex flex-col items-center gap-2 animate-fade-in-up">
          <div className="flex items-center gap-2 text-sm font-medium text-zinc-600">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500"></span>
            </span>
            {t('home.dailySlots.label')} <span className="font-mono font-bold text-zinc-900">{dailySlots ? `${dailySlots.remaining} / ${dailySlots.total}` : '— / —'}</span> {t('home.dailySlots.remaining')}
          </div>
          <div className="h-1.5 w-64 rounded-full bg-zinc-100 overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-emerald-500 to-emerald-400 transition-all duration-1000 ease-out"
              style={{ width: `${dailySlots ? (dailySlots.remaining / dailySlots.total) * 100 : 0}%` }}
            ></div>
          </div>
          <p className="text-xs text-zinc-400 font-mono">{t('home.dailySlots.resets')} {countdown || '--:--:--'}</p>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mt-24 border-t border-zinc-100 pt-12">
          <div>
            <div className="text-3xl font-bold tracking-tight text-zinc-900 mb-1">
              ${tvlValue >= 1000000 ? (tvlValue / 1000000).toFixed(1) + 'M+' : tvlValue.toLocaleString(undefined, { maximumFractionDigits: 0 })}
            </div>
            <div className="text-sm text-zinc-500 font-medium">{t('home.stats.tvl')}</div>
          </div>
          <div>
            <div className="text-3xl font-bold tracking-tight text-zinc-900 mb-1">
              {agentCount > 0 ? agentCount : '0'}+
            </div>
            <div className="text-sm text-zinc-500 font-medium">{t('home.stats.agents')}</div>
          </div>
          <div>
            <div className="text-3xl font-bold tracking-tight text-zinc-900 mb-1">
              {apy !== 0 ? `${apy.toFixed(1)}%` : '0.0%'}
            </div>
            <div className="text-sm text-zinc-500 font-medium">{t('home.stats.apy')}</div>
          </div>
          <div>
            <div className="text-3xl font-bold tracking-tight text-zinc-900 mb-1">
              {totalPnl >= 0 ? '+' : ''}${Math.abs(totalPnl) >= 1000000 ? (totalPnl / 1000000).toFixed(1) + 'M' : totalPnl.toLocaleString(undefined, { maximumFractionDigits: 0 })}
            </div>
            <div className="text-sm text-zinc-500 font-medium">{t('home.stats.yield')}</div>
          </div>
        </div>
      </section>

      {/* Feature Grid - "Open Claw" Style */}
      <section className="max-w-[1600px] mx-auto px-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Card 1: Vault */}
          <div className="group relative rounded-2xl border border-zinc-200 bg-white p-8 hover:border-zinc-300 transition-colors h-[400px] flex flex-col justify-between overflow-hidden">
            <div className="relative z-10">
              <div className="h-12 w-12 rounded-xl bg-zinc-100 flex items-center justify-center mb-6 text-zinc-900">
                <TrendingUp size={24} />
              </div>
              <h3 className="text-2xl font-bold text-zinc-900 mb-3">{t('home.cards.rideVolatility.title')}</h3>
              <p className="text-zinc-500 leading-relaxed">
                {t('home.cards.rideVolatility.desc')}
              </p>
            </div>
            <div className="absolute right-[-20px] bottom-[-20px] opacity-5 group-hover:opacity-10 transition-opacity">
              <TrendingUp size={200} />
            </div>
            <Link to="/submit-agent" className="relative z-10 flex items-center gap-2 text-sm font-bold text-zinc-900 hover:gap-3 transition-all">
              {t('home.cards.rideVolatility.action')} <ArrowRight size={16} />
            </Link>
          </div>

          {/* Card 2: Marketplace */}
          <div className="group relative rounded-2xl border border-zinc-200 bg-white p-8 hover:border-zinc-300 transition-colors h-[400px] flex flex-col justify-between overflow-hidden">
            <div className="relative z-10">
              <div className="h-12 w-12 rounded-xl bg-zinc-100 flex items-center justify-center mb-6 text-zinc-900">
                <Coins size={24} />
              </div>
              <h3 className="text-2xl font-bold text-zinc-900 mb-3">{t('home.cards.copyWhales.title')}</h3>
              <p className="text-zinc-500 leading-relaxed">
                {t('home.cards.copyWhales.desc')}
              </p>
            </div>
            <div className="absolute right-[-20px] bottom-[-20px] opacity-5 group-hover:opacity-10 transition-opacity">
              <Coins size={200} />
            </div>
            <Link to="/strategies" className="relative z-10 flex items-center gap-2 text-sm font-bold text-zinc-900 hover:gap-3 transition-all">
              {t('home.cards.copyWhales.action')} <ArrowRight size={16} />
            </Link>
          </div>

          {/* Card 3: Developer */}
          <div className="group relative rounded-2xl border border-zinc-200 bg-zinc-900 p-8 hover:border-zinc-700 transition-colors h-[400px] flex flex-col justify-between overflow-hidden">
            <div className="relative z-10">
              <div className="h-12 w-12 rounded-xl bg-zinc-800 flex items-center justify-center mb-6 text-white">
                <Zap size={24} />
              </div>
              <h3 className="text-2xl font-bold text-white mb-3">{t('home.cards.openClawEdge.title')}</h3>
              <p className="text-zinc-400 leading-relaxed">
                {t('home.cards.openClawEdge.desc')}
              </p>
            </div>
            <div className="absolute right-[-20px] bottom-[-20px] opacity-5 group-hover:opacity-10 transition-opacity">
              <Zap size={200} className="text-white" />
            </div>
            <Link to="/submit-agent" className="relative z-10 flex items-center gap-2 text-sm font-bold text-white hover:gap-3 transition-all">
              {t('home.cards.openClawEdge.action')} <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      </section>



      {/* How it works */}
      <section className="max-w-5xl mx-auto px-4 py-16">
        <div className="mb-16 text-center">
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 mb-4">{t('home.risk.title')}</h2>
          <p className="text-zinc-500 max-w-2xl mx-auto">
            {t('home.risk.subtitle')}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
          <div className="flex gap-6">
            <div className="shrink-0">
              <div className="h-12 w-12 rounded-full bg-emerald-50 flex items-center justify-center text-emerald-600">
                <Shield size={24} />
              </div>
            </div>
            <div>
              <h3 className="text-xl font-bold text-zinc-900 mb-2">{t('home.risk.hardConstraints.title')}</h3>
              <p className="text-zinc-500 leading-relaxed">
                {t('home.risk.hardConstraints.desc')}
              </p>
            </div>
          </div>
          
          <div className="flex gap-6">
            <div className="shrink-0">
              <div className="h-12 w-12 rounded-full bg-blue-50 flex items-center justify-center text-blue-600">
                <Lock size={24} />
              </div>
            </div>
            <div>
              <h3 className="text-xl font-bold text-zinc-900 mb-2">{t('home.risk.nonCustodial.title')}</h3>
              <p className="text-zinc-500 leading-relaxed">
                {t('home.risk.nonCustodial.desc')}
              </p>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="shrink-0">
              <div className="h-12 w-12 rounded-full bg-amber-50 flex items-center justify-center text-amber-600">
                <BarChart3 size={24} />
              </div>
            </div>
            <div>
              <h3 className="text-xl font-bold text-zinc-900 mb-2">{t('home.risk.promotion.title')}</h3>
              <p className="text-zinc-500 leading-relaxed mb-3">
                {t('home.risk.promotion.desc')}
              </p>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="shrink-0">
              <div className="h-12 w-12 rounded-full bg-purple-50 flex items-center justify-center text-purple-600">
                <Database size={24} />
              </div>
            </div>
            <div>
              <h3 className="text-xl font-bold text-zinc-900 mb-2">{t('home.risk.verifiable.title')}</h3>
              <p className="text-zinc-500 leading-relaxed">
                {t('home.risk.verifiable.desc')}
              </p>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
};

export default Home;
