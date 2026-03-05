import { Strategy, Agent, User } from '../types';
import { generateStrategies, generateAgents, generateUser } from './dataGenerator';

// In-memory storage
let strategies: Strategy[] = [];
let agents: Agent[] = [];
let user: User | null = null;

const DELAY = 800; // Simulate network delay

const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

// Initialize data
const initData = () => {
  if (strategies.length === 0) {
    strategies = generateStrategies(12);
    user = generateUser();
    agents = generateAgents(strategies, 4);
    
    // Update user stats based on agents
    if (user) {
      user.agentCount = agents.length;
      // Removed legacy logic that overwrote LP investment with Agent TVL
      // user.totalInvestment = agents.reduce((sum, agent) => sum + agent.investment, 0);
      // user.totalProfit = agents.reduce((sum, agent) => sum + agent.totalProfit, 0);
    }
  }
};

initData();

export const mockApi = {
  getStrategies: async (): Promise<Strategy[]> => {
    await delay(DELAY);
    return [...strategies];
  },

  getStrategy: async (id: string): Promise<Strategy | undefined> => {
    await delay(DELAY);
    return strategies.find(s => s.id === id);
  },

  getAgents: async (): Promise<Agent[]> => {
    await delay(DELAY);
    return [...agents];
  },

  createAgent: async (agentData: Partial<Agent>): Promise<Agent> => {
    await delay(DELAY);
    const newAgent: Agent = {
      ...agentData as any,
      id: `agent_${Date.now()}`,
      createdAt: new Date().toISOString(),
      lastTrade: new Date().toISOString(),
      trades: [],
      currentValue: agentData.investment || 0,
      totalProfit: 0,
      todayProfit: 0,
      status: 'running'
    };
    agents.push(newAgent);
    
    // Update user stats
    if (user) {
      user.agentCount = agents.length;
      // user.totalInvestment += newAgent.investment; // Legacy logic
    }
    
    return newAgent;
  },

  updateAgentStatus: async (id: string, status: 'running' | 'stopped'): Promise<boolean> => {
    await delay(DELAY);
    const agent = agents.find(a => a.id === id);
    if (agent) {
      agent.status = status;
      return true;
    }
    return false;
  },

  getUser: async (): Promise<User> => {
    await delay(DELAY);
    return { ...user! };
  },

  login: async (): Promise<User> => {
    await delay(DELAY + 500); // Slightly longer for "auth"
    if (!user) {
      initData(); // Ensure user exists
    }
    // Add fake Twitter data if not present
    const twitterUser = {
      ...user!,
      name: 'Kai',
      avatar: 'https://pbs.twimg.com/profile_images/1780044485541699584/p78MCSE3_400x400.jpg', // Mock Twitter avatar
      walletAddress: '0x123...abc', // Mock address associated with account
    };
    user = twitterUser;
    return twitterUser;
  },

  logout: async (): Promise<boolean> => {
    await delay(DELAY);
    return true;
  },

  submitStrategy: async (strategyData: Partial<Strategy>): Promise<Strategy> => {
    await delay(DELAY);
    const newStrategy: Strategy = {
      ...strategyData as any,
      id: `strat_${Date.now()}`,
      status: 'pending',
      runningDays: 0,
      userCount: 0,
      agentLevel: 'intern',
      currentTvl: 0,
      maxTvl: 10000,
      pnlContribution: 0,
      performanceFee: 0.5, // Default 50% for new agents
      performance: { daily: [], monthly: [] },
      rating: 0,
      reviews: [],
      backtestMetrics: strategyData.backtestMetrics || { sharpeRatio: 0, maxDrawdown: 0 }
    };
    strategies.push(newStrategy);
    return newStrategy;
  }
};
