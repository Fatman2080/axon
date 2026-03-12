export type ArenaAvatarVariant = 'adam' | 'alex' | 'amelia' | 'bob';

export type ArenaAgentStatus = 'idle' | 'running' | 'phone' | 'dead';

export interface ArenaAgentEntity {
  id: string;
  name: string;
  avatarVariant: ArenaAvatarVariant;
  status: ArenaAgentStatus;
  slotIndex: number;
  totalPnl: number;
  lastRealizedPnl: number;
  isDead: boolean;
  selected: boolean;
  updatedAt: number;
  lastEvent: string;
}

export type ArenaEvent =
  | {
      type: 'agent_snapshot_reset';
      agents: ArenaAgentEntity[];
      emittedAt: number;
    }
  | {
      type: 'agent_joined';
      agent: ArenaAgentEntity;
      emittedAt: number;
    }
  | {
      type: 'agent_removed';
      agentId: string;
      emittedAt: number;
    }
  | {
      type: 'agent_state_changed';
      agentId: string;
      status: Exclude<ArenaAgentStatus, 'dead'>;
      emittedAt: number;
    }
  | {
      type: 'agent_pnl_realized';
      agentId: string;
      pnlDelta: number;
      emittedAt: number;
    }
  | {
      type: 'agent_died';
      agentId: string;
      emittedAt: number;
    }
  | {
      type: 'agent_revived';
      agentId: string;
      emittedAt: number;
    };

export type ArenaScenarioScript = 'steady' | 'pnl-storm' | 'death-wave';

export interface ArenaScenarioPreset {
  agentCount: 50 | 300 | 1000;
  eventRate: 10 | 50 | 200;
  script: ArenaScenarioScript;
}

export interface ArenaPerfStats {
  fps: number;
  liveAgents: number;
  deadAgents: number;
  renderedAgents: number;
  pooledAgents: number;
  pooledFloaters: number;
  activeFloaters: number;
  queuedEvents: number;
  eventsPerSecond: number;
}

export interface ArenaFeed {
  start(): void;
  stop(): void;
  dispose(): void;
  setPreset(preset: ArenaScenarioPreset): void;
  setRunning(running: boolean): void;
  subscribe(listener: (events: ArenaEvent[]) => void): () => void;
}
