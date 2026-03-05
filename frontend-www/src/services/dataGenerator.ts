import { Strategy, Agent, User, Trade, Review, Vault } from '../types';

const STRATEGY_CATEGORIES = ['arbitrage', 'trend', 'grid', 'martingale'] as const;
const RISK_LEVELS = ['low', 'medium', 'high'] as const;

export const generateVault = (): Vault => {
  const chainBalance = 50000 + Math.random() * 20000;
  const clusterBalance = 150000 + Math.random() * 50000;
  const hypeBalance = 300000 + Math.random() * 100000;
  const agentBalance = 500000 + Math.random() * 200000;
  const totalTvl = chainBalance + clusterBalance + hypeBalance + agentBalance;

  return {
    id: 'vault_hyperliquid_01',
    name: 'Hyperliquid AI Alpha Vault',
    totalTvl, 
    sharePrice: 1.0 + Math.random() * 0.2,
    apy: 45.2, // 45.2% APY
    totalProfit: 2340000,
    agentCount: 156,
    depositorCount: 1234 + Math.floor(Math.random() * 500),
    performanceHistory: Array.from({ length: 30 }, () => Math.random() * 1.5),
    chainBalance,
    clusterBalance,
    hypeBalance,
    agentBalance,
  };
};

export const generateStrategies = (count: number): Strategy[] => {
  return Array.from({ length: count }).map((_, i) => {
    // 权重分配：大部分是小额，少量大额
    const levelWeight = Math.random();
    let maxTvl = 10000;
    if (levelWeight > 0.95) maxTvl = 5000000;
    else if (levelWeight > 0.8) maxTvl = 500000;
    else if (levelWeight > 0.5) maxTvl = 50000;

    const currentTvl = Math.floor(maxTvl * (0.3 + Math.random() * 0.6)); // 30% - 90% full
    const pnlContribution = currentTvl * (Math.random() * 0.4 - 0.1); // -10% to +30%

    return {
      id: `strat_${i + 1}`,
      name: `Agent ${String.fromCharCode(65 + i)}`,
      description: 'An advanced AI-driven trading strategy optimizing for market volatility.',
      category: STRATEGY_CATEGORIES[Math.floor(Math.random() * STRATEGY_CATEGORIES.length)],
      minInvestment: Math.floor(Math.random() * 10) * 100 + 100,
      riskLevel: RISK_LEVELS[Math.floor(Math.random() * RISK_LEVELS.length)],
      expectedReturn: Math.floor(Math.random() * 30) + 5,
      runningDays: Math.floor(Math.random() * 300) + 30,
      userCount: Math.floor(Math.random() * 1000) + 50,
      creator: `Creator_${i + 1}`,
      currentTvl,
      maxTvl,
      pnlContribution,
      performanceFee: 0.2, // 20%
      status: 'active', // 默认生成的是已激活的策略
      codeType: 'python',
      backtestMetrics: {
        sharpeRatio: 1.5 + Math.random(),
        maxDrawdown: Math.random() * 10
      },
      performance: {
        daily: Array.from({ length: 30 }, () => Math.random() * 2 - 0.5),
        monthly: Array.from({ length: 12 }, () => Math.random() * 10 - 2),
      },
      parameters: {
        stopLoss: 5,
        takeProfit: 10,
        maxPosition: 1000,
      },
      social: {
        twitter: 'https://twitter.com/clawfi',
        website: 'https://clawfi.io',
        discord: 'https://discord.gg/clawfi'
      },
      rating: (Math.floor(Math.random() * 20) + 30) / 10,
      reviews: generateReviews(Math.floor(Math.random() * 5)),
    };
  });
};

const generateReviews = (count: number): Review[] => {
  return Array.from({ length: count }).map((_, i) => ({
    id: `review_${i}`,
    userId: `user_${Math.floor(Math.random() * 100)}`,
    rating: Math.floor(Math.random() * 2) + 4,
    comment: 'Great strategy! Consistent returns.',
    createdAt: new Date(Date.now() - Math.random() * 10000000000).toISOString(),
  }));
};

export const generateAgents = (strategies: Strategy[], count: number): Agent[] => {
  return Array.from({ length: count }).map((_, i) => {
    const strategy = strategies[Math.floor(Math.random() * strategies.length)];
    const investment = Math.floor(Math.random() * 5000) + 500;
    const profit = investment * (Math.random() * 0.2);
    
    return {
      id: `agent_${i + 1}`,
      name: `${strategy.name} Bot`,
      strategyId: strategy.id,
      userId: 'current_user',
      status: Math.random() > 0.2 ? 'running' : 'stopped',
      investment,
      currentValue: investment + profit,
      totalProfit: profit,
      todayProfit: profit * 0.1,
      createdAt: new Date(Date.now() - Math.random() * 10000000000).toISOString(),
      lastTrade: new Date().toISOString(),
      trades: generateTrades(5),
      config: {
        tradingPairs: ['BTC/USDT', 'ETH/USDT'],
        riskLevel: 5,
        autoReinvest: true,
      },
    };
  });
};

const generateTrades = (count: number): Trade[] => {
  return Array.from({ length: count }).map((_, i) => ({
    id: `trade_${i}`,
    pair: 'BTC/USDT',
    type: Math.random() > 0.5 ? 'buy' : 'sell',
    amount: Math.random() * 0.1,
    price: 45000 + Math.random() * 2000,
    profit: Math.random() * 50 - 10,
    timestamp: new Date(Date.now() - i * 3600000).toISOString(),
  }));
};

export const generateUser = (): User => ({
  id: 'current_user',
  email: 'user@example.com',
  walletAddress: '0x123...abc',
  name: 'Crypto Trader',
  level: 'premium',
  totalInvestment: 50000,
  totalProfit: 12500,
  agentCount: 2,
  lpShares: 45000,
  lpValue: 48000,
  joinedAt: new Date(Date.now() - 10000000000).toISOString(),
  preferences: {
    notifications: true,
    autoUpdate: false,
    currency: 'USD',
  },
});
