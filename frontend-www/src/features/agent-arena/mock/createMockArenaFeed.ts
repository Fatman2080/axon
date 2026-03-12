import type {
  ArenaAgentEntity,
  ArenaAgentStatus,
  ArenaEvent,
  ArenaFeed,
  ArenaScenarioPreset,
  ArenaScenarioScript,
} from '../types';
import { resolveArenaAvatarVariant } from '../avatarRules';
const RUNNING_NAMES = [
  'Delta Claw',
  'Gamma Burst',
  'Silent Basis',
  'Iron Arb',
  'Neon Delta',
  'Maker Fang',
  'Vault Echo',
  'Liquid Coil',
];

type MockAgentRecord = ArenaAgentEntity & { removed?: boolean };

const TICK_MS = 100;

function randomBetween(min: number, max: number) {
  return min + Math.random() * (max - min);
}

function randomItem<T>(items: T[]) {
  return items[Math.floor(Math.random() * items.length)];
}

function buildName(index: number) {
  const root = RUNNING_NAMES[index % RUNNING_NAMES.length];
  return `${root} #${String(index + 1).padStart(4, '0')}`;
}

function createAgent(index: number): MockAgentRecord {
  const statusPool: Exclude<ArenaAgentStatus, 'dead'>[] = ['running', 'running', 'running', 'idle', 'phone'];
  const status = randomItem(statusPool);
  const id = `arena-agent-${String(index + 1).padStart(4, '0')}`;
  return {
    id,
    name: buildName(index),
    avatarVariant: resolveArenaAvatarVariant(id),
    status,
    slotIndex: index,
    totalPnl: randomBetween(-1800, 2600),
    lastRealizedPnl: 0,
    isDead: false,
    selected: false,
    updatedAt: Date.now(),
    lastEvent: 'snapshot',
  };
}

function cloneAgent(agent: MockAgentRecord): ArenaAgentEntity {
  return { ...agent };
}

export function createMockArenaFeed(initialPreset: ArenaScenarioPreset): ArenaFeed {
  let preset = initialPreset;
  let running = true;
  let timer: number | null = null;
  let listeners = new Set<(events: ArenaEvent[]) => void>();
  let agents = new Map<string, MockAgentRecord>();
  let generation = 0;
  let deathWaveCursor = 0;
  let latestSnapshot: ArenaEvent | null = null;

  const emit = (events: ArenaEvent[]) => {
    if (events.length === 0) return;
    listeners.forEach((listener) => listener(events));
  };

  const snapshot = () => {
    agents = new Map();
    generation += 1;
    for (let index = 0; index < preset.agentCount; index += 1) {
      const agent = createAgent(index);
      agents.set(agent.id, agent);
    }
    latestSnapshot = {
      type: 'agent_snapshot_reset',
      agents: Array.from(agents.values()).map(cloneAgent),
      emittedAt: Date.now(),
    };
    emit([latestSnapshot]);
  };

  const chooseAliveAgent = () => {
    const alive = Array.from(agents.values()).filter((agent) => !agent.removed && !agent.isDead);
    return alive.length ? randomItem(alive) : null;
  };

  const chooseDeadAgent = () => {
    const dead = Array.from(agents.values()).filter((agent) => !agent.removed && agent.isDead);
    return dead.length ? randomItem(dead) : null;
  };

  const chooseAnyAgent = () => {
    const active = Array.from(agents.values()).filter((agent) => !agent.removed);
    return active.length ? randomItem(active) : null;
  };

  const nextChurnAgent = () => {
    const removed = Array.from(agents.values()).find((agent) => agent.removed);
    if (removed) {
      removed.removed = false;
      removed.isDead = false;
      removed.status = 'idle';
      removed.totalPnl = randomBetween(-500, 1200);
      removed.lastRealizedPnl = 0;
      removed.updatedAt = Date.now();
      removed.lastEvent = 'rejoined';
      return removed;
    }
    const created = createAgent(agents.size % Math.max(preset.agentCount, 50));
    created.id = `arena-agent-${generation}-${String(Math.floor(Math.random() * 9999)).padStart(4, '0')}`;
    created.avatarVariant = resolveArenaAvatarVariant(created.id, created.avatarVariant);
    created.slotIndex = agents.size % Math.max(preset.agentCount, 50);
    agents.set(created.id, created);
    return created;
  };

  const maybeSetStatus = (agent: MockAgentRecord, script: ArenaScenarioScript) => {
    const pool: Exclude<ArenaAgentStatus, 'dead'>[] =
      script === 'steady' ? ['running', 'running', 'idle', 'phone'] : ['running', 'idle', 'phone'];
    const nextStatus = randomItem(pool);
    if (nextStatus === agent.status) return null;
    agent.status = nextStatus;
    agent.updatedAt = Date.now();
    agent.lastEvent = `state:${nextStatus}`;
    return {
      type: 'agent_state_changed' as const,
      agentId: agent.id,
      status: nextStatus,
      emittedAt: agent.updatedAt,
    };
  };

  const applySteady = () => {
    const events: ArenaEvent[] = [];
    const ops = Math.max(1, Math.round((preset.eventRate / 1000) * TICK_MS));
    for (let index = 0; index < ops; index += 1) {
      const roll = Math.random();
      if (roll < 0.62) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        const pnlDelta = randomBetween(-280, 420);
        agent.totalPnl += pnlDelta;
        agent.lastRealizedPnl = pnlDelta;
        agent.updatedAt = Date.now();
        agent.lastEvent = `pnl:${pnlDelta >= 0 ? '+' : ''}${pnlDelta.toFixed(2)}`;
        events.push({
          type: 'agent_pnl_realized',
          agentId: agent.id,
          pnlDelta,
          emittedAt: agent.updatedAt,
        });
      } else if (roll < 0.84) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        const event = maybeSetStatus(agent, 'steady');
        if (event) events.push(event);
      } else if (roll < 0.92) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        agent.isDead = true;
        agent.status = 'dead';
        agent.updatedAt = Date.now();
        agent.lastEvent = 'melted';
        events.push({ type: 'agent_died', agentId: agent.id, emittedAt: agent.updatedAt });
      } else if (roll < 0.97) {
        const agent = chooseDeadAgent();
        if (!agent) continue;
        agent.isDead = false;
        agent.status = 'idle';
        agent.updatedAt = Date.now();
        agent.lastEvent = 'revived';
        events.push({ type: 'agent_revived', agentId: agent.id, emittedAt: agent.updatedAt });
      } else if (roll < 0.985) {
        const leaving = chooseAliveAgent();
        if (!leaving) continue;
        leaving.removed = true;
        leaving.updatedAt = Date.now();
        leaving.lastEvent = 'left';
        events.push({ type: 'agent_removed', agentId: leaving.id, emittedAt: leaving.updatedAt });
      } else {
        const joined = nextChurnAgent();
        events.push({ type: 'agent_joined', agent: cloneAgent(joined), emittedAt: Date.now() });
      }
    }
    return events;
  };

  const applyPnlStorm = () => {
    const events: ArenaEvent[] = [];
    const ops = Math.max(2, Math.round((preset.eventRate / 1000) * TICK_MS));
    for (let index = 0; index < ops; index += 1) {
      const roll = Math.random();
      if (roll < 0.84) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        const pnlDelta = randomBetween(-860, 1100);
        agent.totalPnl += pnlDelta;
        agent.lastRealizedPnl = pnlDelta;
        agent.status = pnlDelta >= 0 ? 'running' : 'phone';
        agent.updatedAt = Date.now();
        agent.lastEvent = `pnl-storm:${pnlDelta >= 0 ? '+' : ''}${pnlDelta.toFixed(2)}`;
        events.push({
          type: 'agent_pnl_realized',
          agentId: agent.id,
          pnlDelta,
          emittedAt: agent.updatedAt,
        });
      } else if (roll < 0.92) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        const event = maybeSetStatus(agent, 'pnl-storm');
        if (event) events.push(event);
      } else if (roll < 0.97) {
        const agent = chooseAliveAgent();
        if (!agent) continue;
        agent.isDead = true;
        agent.status = 'dead';
        agent.updatedAt = Date.now();
        agent.lastEvent = 'storm-eliminated';
        events.push({ type: 'agent_died', agentId: agent.id, emittedAt: agent.updatedAt });
      } else {
        const agent = chooseDeadAgent();
        if (!agent) continue;
        agent.isDead = false;
        agent.status = 'running';
        agent.updatedAt = Date.now();
        agent.lastEvent = 'storm-recovered';
        events.push({ type: 'agent_revived', agentId: agent.id, emittedAt: agent.updatedAt });
      }
    }
    return events;
  };

  const applyDeathWave = () => {
    const events: ArenaEvent[] = [];
    const aliveAgents = Array.from(agents.values()).filter((agent) => !agent.removed && !agent.isDead);
    const deadAgents = Array.from(agents.values()).filter((agent) => !agent.removed && agent.isDead);

    if (aliveAgents.length > 0) {
      const waveSize = Math.max(1, Math.floor(preset.agentCount * 0.015));
      for (let index = 0; index < waveSize; index += 1) {
        const agent = aliveAgents[(deathWaveCursor + index) % aliveAgents.length];
        if (!agent || agent.isDead) continue;
        agent.isDead = true;
        agent.status = 'dead';
        agent.updatedAt = Date.now();
        agent.lastEvent = 'death-wave';
        events.push({ type: 'agent_died', agentId: agent.id, emittedAt: agent.updatedAt });
      }
      deathWaveCursor += waveSize;
    }

    const reviveOps = Math.max(1, Math.floor(deadAgents.length * 0.12));
    for (let index = 0; index < reviveOps; index += 1) {
      const agent = deadAgents[index];
      if (!agent) continue;
      agent.isDead = false;
      agent.status = Math.random() > 0.5 ? 'running' : 'idle';
      agent.updatedAt = Date.now();
      agent.lastEvent = 're-armed';
      events.push({ type: 'agent_revived', agentId: agent.id, emittedAt: agent.updatedAt });
    }

    const pnlOps = Math.max(1, Math.round(events.length * 0.6));
    for (let index = 0; index < pnlOps; index += 1) {
      const agent = chooseAnyAgent();
      if (!agent || agent.isDead || agent.removed) continue;
      const pnlDelta = randomBetween(-420, 580);
      agent.totalPnl += pnlDelta;
      agent.lastRealizedPnl = pnlDelta;
      agent.updatedAt = Date.now();
      agent.lastEvent = `wave-pnl:${pnlDelta >= 0 ? '+' : ''}${pnlDelta.toFixed(2)}`;
      events.push({
        type: 'agent_pnl_realized',
        agentId: agent.id,
        pnlDelta,
        emittedAt: agent.updatedAt,
      });
    }

    return events;
  };

  const tick = () => {
    if (!running) return;
    let events: ArenaEvent[] = [];
    if (preset.script === 'pnl-storm') {
      events = applyPnlStorm();
    } else if (preset.script === 'death-wave') {
      events = applyDeathWave();
    } else {
      events = applySteady();
    }
    emit(events);
  };

  const ensureTimer = () => {
    if (timer !== null) return;
    timer = window.setInterval(tick, TICK_MS);
  };

  const clearTimer = () => {
    if (timer !== null) {
      window.clearInterval(timer);
      timer = null;
    }
  };

  snapshot();
  ensureTimer();

  return {
    start() {
      running = true;
      ensureTimer();
    },
    stop() {
      running = false;
    },
    dispose() {
      running = false;
      clearTimer();
      listeners = new Set();
      agents.clear();
    },
    setPreset(nextPreset) {
      preset = nextPreset;
      deathWaveCursor = 0;
      snapshot();
    },
    setRunning(nextRunning) {
      running = nextRunning;
      if (nextRunning) {
        ensureTimer();
      }
    },
    subscribe(listener) {
      listeners.add(listener);
      if (latestSnapshot) {
        listener([latestSnapshot]);
      }
      return () => {
        listeners.delete(listener);
      };
    },
  };
}
