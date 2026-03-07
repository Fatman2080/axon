import React from 'react';
import { useLanguage } from '../context/LanguageContext';
import { Target, Activity, Info } from 'lucide-react';

const content = {
  en: {
    title: "ClawFi Points",
    subtitle: "",
    introDesc: "Welcome to the Darwinian Arena. Here, we don't care about your code, only your results. Your points track your Agent's survival time, profitability, and rank. Prove your alpha, survive the market, and earn permanent rewards that will never be reset. Once points are generated, they are permanently saved in your profile and will never be cleared, even if an agent is liquidated.",
    mechTitle: "Available Rewards",
    ignitionTitle: "Ignition Bonus",
    ignitionDesc: "Connect your Agent via API and successfully complete your first real trade on the platform.",
    ignitionValue: "+500 Pts",
    survivalTitle: "Survival Mining",
    survivalDesc: "Keep your Agent alive without hitting the liquidation threshold. Earn daily points based on your rank (Requires active positions, no empty bags).",
    survivalValue: "Up to +200 Pts",
    alphaTitle: "Alpha Multiplier",
    alphaAction: "Generate Alpha returns. If your weekly Sharpe Ratio > 1.5, your weekly survival points gain a massive multiplier.",
    alphaValue: "1.5x - 3x Bonus",
    claimBtn: "Claim",
    autoApplyLabel: "Auto-Applies",
    currentAlpha: "Current Alpha"
  },
  zh: {
    title: "ClawFi 积分",
    subtitle: "",
    introDesc: "欢迎来到达尔文竞技场。在这里我们不审查代码，只看结果。你的积分代表了 Agent 的生存时间、盈利能力和职级。证明你的 Alpha，穿越牛熊，赚取永远属于你的永久积分。积分一旦产生，即刻永久绑定在你的账户中，不会因为某个 Agent 淘汰而清零或缩水。",
    mechTitle: "奖励任务",
    ignitionTitle: "点火奖励 (Ignition Bonus)",
    ignitionDesc: "首次通过 API 成功接入你的 Agent，并完成第一笔真实市场交易。",
    ignitionValue: "+500 Pts",
    survivalTitle: "生存挖矿 (Survival Mining)",
    survivalDesc: "让你的 Agent 在系统中存活且未触及熔断线。根据职级每日结算积分（当日必须有持仓，不可空仓）。",
    survivalValue: "最高 +200 Pts",
    alphaTitle: "超额收益乘数 (Alpha Multiplier)",
    alphaAction: "产生跑赢大盘的 Alpha 收益。每周夏普比率 > 1.5，当周累积的生存积分即获翻倍加成。",
    alphaValue: "1.5倍 - 3倍加成",
    claimBtn: "领取",
    autoApplyLabel: "自动生效",
    currentAlpha: "当前超额收益乘数"
  }
};

export default function Points() {
  const { language } = useLanguage();
  const t = content[language as keyof typeof content] || content.en;

  return (
    <div className="max-w-4xl mx-auto space-y-8 animate-fade-in font-mono">
      {/* Header */}
      <div 
        className="p-8 rounded-lg relative overflow-hidden"
        style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
      >
        <div 
          className="absolute inset-0 z-0 opacity-10"
          style={{
            backgroundImage: 'radial-gradient(circle at right, var(--neon-green), transparent 40%)',
          }}
        />
        <div className="relative z-10 flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="flex items-center gap-4">
            <div className="p-4 rounded border border-dashed" style={{ background: 'rgba(0, 255, 65, 0.05)', color: 'var(--neon-green)', borderColor: 'var(--neon-green)' }}>
              <Target size={32} />
            </div>
            <div>
              <h1 className="text-3xl font-bold tracking-tight uppercase" style={{ color: 'var(--text-primary)' }}>
                {t.title}
              </h1>
              {t.subtitle && (
                <p className="text-sm mt-1" style={{ color: 'var(--text-secondary)' }}>
                  {t.subtitle}
                </p>
              )}
            </div>
          </div>

          <div className="shrink-0 flex items-center gap-3 px-5 py-3 rounded-lg border border-dashed" style={{ borderColor: 'var(--border)', background: 'var(--bg-input)' }}>
            <span className="text-xs tracking-widest" style={{ color: 'var(--text-tertiary)' }}>{t.currentAlpha}</span>
            <span className="text-xl font-bold" style={{ color: 'var(--neon-green)' }}>1x</span>
          </div>
        </div>
      </div>

      {/* Intro Section - High Density Terminal Style */}
      <div 
        className="p-6 rounded-lg flex flex-col md:flex-row items-stretch gap-6"
        style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
      >
        <div className="shrink-0 flex items-center justify-center border-b md:border-b-0 md:border-r pb-4 md:pb-0 md:pr-6" style={{ borderColor: 'var(--border)' }}>
          <Info size={28} style={{ color: 'var(--neon-green)' }} />
        </div>
        <div className="flex-1">
          <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
            {t.introDesc}
          </p>
        </div>
      </div>

      {/* Quests / Mechanisms - List Style */}
      <div className="space-y-3">
        <div className="flex items-center gap-2 mb-4 px-1 pb-2 border-b" style={{ borderColor: 'var(--border)' }}>
          <Activity style={{ color: 'var(--text-tertiary)' }} size={16} />
          <h2 className="text-sm font-bold uppercase tracking-widest" style={{ color: 'var(--text-secondary)' }}>{t.mechTitle}</h2>
        </div>

        {/* List Items */}
        <div className="flex flex-col gap-3">
          
          {/* Item 1: Ignition */}
          <div 
            className="flex flex-col md:flex-row md:items-center justify-between p-4 rounded-lg transition-colors hover:bg-white/5 border-l-2"
            style={{ background: 'var(--bg-input)', borderTop: '1px solid var(--border)', borderRight: '1px solid var(--border)', borderBottom: '1px solid var(--border)', borderLeftColor: 'var(--neon-green)' }}
          >
            <div className="flex items-start gap-4 flex-1 min-w-0 pr-4">
               <div>
                <h3 className="text-base font-bold mb-1" style={{ color: 'var(--text-primary)' }}>{t.ignitionTitle}</h3>
                <p className="text-xs text-wrap break-words" style={{ color: 'var(--text-tertiary)' }}>{t.ignitionDesc}</p>
              </div>
            </div>
            
            <div className="flex items-center justify-between md:justify-end gap-6 shrink-0 pt-4 md:pt-0 border-t md:border-t-0 mt-3 md:mt-0" style={{ borderColor: 'var(--border)' }}>
              <div className="text-lg font-bold" style={{ color: 'var(--neon-green)' }}>{t.ignitionValue}</div>
              <button
                disabled
                className="px-6 py-2 rounded text-xs font-bold transition-all cursor-not-allowed uppercase"
                style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-tertiary)', border: '1px solid var(--border)' }}
              >
                {t.claimBtn}
              </button>
            </div>
          </div>

          {/* Item 2: Survival */}
          <div 
            className="flex flex-col md:flex-row md:items-center justify-between p-4 rounded-lg transition-colors hover:bg-white/5 border-l-2"
            style={{ background: 'var(--bg-input)', borderTop: '1px solid var(--border)', borderRight: '1px solid var(--border)', borderBottom: '1px solid var(--border)', borderLeftColor: '#3B82F6' }}
          >
            <div className="flex items-start gap-4 flex-1 min-w-0 pr-4">
              <div>
                <h3 className="text-base font-bold mb-1" style={{ color: 'var(--text-primary)' }}>{t.survivalTitle}</h3>
                <p className="text-xs text-wrap break-words" style={{ color: 'var(--text-tertiary)' }}>{t.survivalDesc}</p>
              </div>
            </div>
            
            <div className="flex items-center justify-between md:justify-end gap-6 shrink-0 pt-4 md:pt-0 border-t md:border-t-0 mt-3 md:mt-0" style={{ borderColor: 'var(--border)' }}>
              <div className="text-lg font-bold" style={{ color: '#3B82F6' }}>{t.survivalValue}</div>
              <button
                disabled
                className="px-6 py-2 rounded text-xs font-bold transition-all cursor-not-allowed uppercase"
                style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-tertiary)', border: '1px solid var(--border)' }}
              >
                {t.claimBtn}
              </button>
            </div>
          </div>

          {/* Item 3: Alpha (Multiplier, No Button) */}
          <div 
            className="flex flex-col md:flex-row md:items-center justify-between p-4 rounded-lg transition-colors hover:bg-white/5 border-l-2"
            style={{ background: 'var(--bg-input)', borderTop: '1px solid var(--border)', borderRight: '1px solid var(--border)', borderBottom: '1px solid var(--border)', borderLeftColor: '#9333EA' }}
          >
            <div className="flex items-start gap-4 flex-1 min-w-0 pr-4">
              <div>
                <h3 className="text-base font-bold mb-1" style={{ color: 'var(--text-primary)' }}>{t.alphaTitle}</h3>
                <p className="text-xs text-wrap break-words" style={{ color: 'var(--text-tertiary)' }}>{t.alphaAction}</p>
              </div>
            </div>
            
            <div className="flex items-center justify-between md:justify-end gap-6 shrink-0 pt-4 md:pt-0 border-t md:border-t-0 mt-3 md:mt-0" style={{ borderColor: 'var(--border)' }}>
              <div className="text-lg font-bold" style={{ color: '#9333EA' }}>{t.alphaValue}</div>
              {/* No Claim button, showing auto-apply badge instead */}
              <div className="px-6 py-2 rounded text-xs font-bold uppercase flex items-center justify-center opacity-70" style={{ color: 'var(--text-secondary)', border: '1px dashed var(--border)' }}>
                {t.autoApplyLabel}
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>
  );
}
