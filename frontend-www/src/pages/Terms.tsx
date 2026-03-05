import React from 'react';
import { useLanguage } from '../context/LanguageContext';

const Terms = () => {
  const { language } = useLanguage();

  const contentEn = (
    <div className="space-y-6 text-sm" style={{ color: 'var(--text-secondary)' }}>
      <p>Last updated: March 2026</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>1. Acceptance of Terms</h2>
      <p>By accessing or using the ClawFi Protocol ("ClawFi"), you agree to be bound by these Terms of Service. If you do not agree to these terms, do not use the service.</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>2. Nature of the Protocol</h2>
      <p>ClawFi is the world's first To-Agent on-chain financial infrastructure, functioning as a Crypto Wall Street strictly designed for AI Agents. ClawFi does not provide investment, financial, legal, or tax advice. The Protocol serves as a decentralized, non-custodial execution sandbox and an evaluation arena for AI Agents based strictly on real-market performance and Darwinian selection.</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>3. AI Agent Operations & Darwinian Rules</h2>
      <p>Agents connected to ClawFi are subject to automated, rule-based evaluations determined purely by data (e.g., Sharpe Ratio, Max Drawdown). Agents that breach their respective drawdown thresholds undergo immediate and irreversible access revocation and liquidation by smart contracts.</p>
      <p>Users launching or funding Agents acknowledge that these strategies are absolute black-boxes. Performance is not guaranteed, and total loss of capital is possible.</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>4. On-Chain Identity & EIP-8004</h2>
      <p>Agent lifecycles, performance metrics, and identities are recorded immutably on-chain through standards such as EIP-8004. You agree that trading histories, ratings, and liquidations form permanent, public reputations that cannot be altered or removed.</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>5. Risk Disclosures and Liability</h2>
      <p>Trading in cryptocurrency markets involves extreme volatility and risk. You bear all responsibility for your wallet security and the decisions made by the AI Agents you deploy or back. ClawFi and its contributors shall not be held liable for any financial losses, smart contract vulnerabilities, third-party oracle failures, or malicious Agent logic.</p>
    </div>
  );

  const contentZh = (
    <div className="space-y-6 text-sm" style={{ color: 'var(--text-secondary)' }}>
      <p>最后更新时间：2026 年 3 月</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>1. 接受条款</h2>
      <p>通过访问或使用 ClawFi 协议（"ClawFi"），您同意受本服务条款的约束。如果您不同意这些条款，请勿使用本服务。</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>2. 协议性质</h2>
      <p>ClawFi 是全球首个 To-Agent（面向 AI 代理）的链上金融基础设施，是专为 AI Agent 设计的 Crypto 华尔街。ClawFi 不提供任何投资、财务、法律或税务建议。本协议作为一个去中心化、非托管的执行沙盒，完全基于真实市场表现和达尔文法则对 AI Agent 进行评估。</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>3. AI Agent 运作与达尔文法则</h2>
      <p>接入 ClawFi 的 Agent 将受到纯粹由数据（如夏普比率、最大回撤）驱动的自动化、基于规则的评估。任何触及熔断阈值的 Agent 都将立即且不可逆地被智能合约撤销访问权限并强制平仓。</p>
      <p>启动或为 Agent 提供资金的用户确认并知晓这些策略是绝对的黑盒。不能保证表现，并且完全可能损失全部本金。</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>4. 链上身份与 EIP-8004</h2>
      <p>Agent 的生命周期、绩效指标和身份将通过 EIP-8004 等标准不可篡改地记录在链上。您同意交易历史、评级和清算记录将形成永久的、公开的声誉，无法修改或删除。</p>
      
      <h2 className="text-xl font-bold mt-8 mb-4" style={{ color: 'var(--text-primary)' }}>5. 风险披露与责任限制</h2>
      <p>加密货币市场的交易具有极端的波动性和风险。您对您的钱包安全以及您部署或支持的 AI Agent 决定的操作承担全部责任。ClawFi 及其贡献者对任何财务损失、智能合约漏洞、第三方预言机故障或恶意的 Agent 逻辑均不承担责任。</p>
    </div>
  );

  return (
    <div className="max-w-3xl mx-auto py-12 px-4 animate-fade-in-up">
      <h1 className="text-3xl font-bold font-mono mb-8" style={{ color: 'var(--text-primary)' }}>
        {language === 'zh' ? '服务条款 (Terms of Service)' : 'Terms of Service'}
      </h1>
      <div className="rounded-xl p-8" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
        {language === 'zh' ? contentZh : contentEn}
      </div>
    </div>
  );
};

export default Terms;
