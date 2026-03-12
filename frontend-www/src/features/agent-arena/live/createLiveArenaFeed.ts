import type {
  AgentArenaSnapshotAgent,
  AgentArenaStreamEvent,
  AgentArenaStreamMessage,
} from "../../../types";
import { resolveArenaAvatarVariant } from "../avatarRules";
import type {
  ArenaAgentEntity,
  ArenaEvent,
  ArenaFeed,
  ArenaScenarioPreset,
} from "../types";

function hashString(input: string) {
  let value = 0;
  for (let index = 0; index < input.length; index += 1) {
    value = (value * 31 + input.charCodeAt(index)) >>> 0;
  }
  return value;
}

function toTimestamp(value?: string) {
  if (!value) return Date.now();
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? Date.now() : parsed;
}

function toSlotIndex(agentId: string) {
  return hashString(agentId) % 2048;
}

function mapSnapshotAgent(agent: AgentArenaSnapshotAgent): ArenaAgentEntity {
  const updatedAt = toTimestamp(
    agent.updatedAt ?? agent.lastSyncedAt ?? agent.lastRealizedAt,
  );
  return {
    id: agent.id,
    name: agent.name,
    avatarVariant: resolveArenaAvatarVariant(agent.id),
    status: agent.status,
    slotIndex: toSlotIndex(agent.id),
    totalPnl: agent.totalPnl,
    lastRealizedPnl: agent.lastRealizedPnl,
    isDead: agent.status === "dead",
    selected: false,
    updatedAt,
    lastEvent: agent.status,
  };
}

function mapStreamEvent(event: AgentArenaStreamEvent): ArenaEvent | null {
  const emittedAt = toTimestamp(event.emittedAt);
  switch (event.type) {
    case "agent_joined":
      if (!event.agent) return null;
      return {
        type: "agent_joined",
        agent: mapSnapshotAgent(event.agent),
        emittedAt,
      };
    case "agent_removed":
      if (!event.agentId) return null;
      return {
        type: "agent_removed",
        agentId: event.agentId,
        emittedAt,
      };
    case "agent_state_changed":
      if (!event.agentId || !event.status) return null;
      return {
        type: "agent_state_changed",
        agentId: event.agentId,
        status: event.status,
        emittedAt,
      };
    case "agent_pnl_realized":
      if (!event.agentId || typeof event.pnlDelta !== "number") return null;
      return {
        type: "agent_pnl_realized",
        agentId: event.agentId,
        pnlDelta: event.pnlDelta,
        emittedAt,
      };
    case "agent_died":
      if (!event.agentId) return null;
      return {
        type: "agent_died",
        agentId: event.agentId,
        emittedAt,
      };
    case "agent_revived":
      if (!event.agentId) return null;
      return {
        type: "agent_revived",
        agentId: event.agentId,
        emittedAt,
      };
    default:
      return null;
  }
}

function resolveWebSocketURL() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/api/agent-arena/ws`;
}

export function createLiveArenaFeed(_preset: ArenaScenarioPreset): ArenaFeed {
  let listeners = new Set<(events: ArenaEvent[]) => void>();
  let running = true;
  let disposed = false;
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let latestSnapshot: ArenaEvent | null = null;

  const emit = (events: ArenaEvent[]) => {
    if (events.length === 0) return;
    listeners.forEach((listener) => listener(events));
  };

  const applySnapshot = (
    agents: AgentArenaSnapshotAgent[],
    generatedAt?: string,
  ) => {
    latestSnapshot = {
      type: "agent_snapshot_reset",
      agents: agents.map(mapSnapshotAgent),
      emittedAt: toTimestamp(generatedAt),
    };
    emit([latestSnapshot]);
  };

  const clearReconnect = () => {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  };

  const scheduleReconnect = () => {
    if (!running || disposed || reconnectTimer !== null) {
      return;
    }
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      connectSocket();
    }, 2500);
  };

  const handleMessage = (message: AgentArenaStreamMessage) => {
    if (message.type === "snapshot" && message.snapshot) {
      applySnapshot(message.snapshot.agents, message.snapshot.generatedAt);
      return;
    }
    if (message.type === "events" && message.events) {
      const events = message.events
        .map(mapStreamEvent)
        .filter((event): event is ArenaEvent => event !== null);
      emit(events);
    }
  };

  const connectSocket = () => {
    if (!running || disposed) return;
    if (
      socket &&
      (socket.readyState === WebSocket.OPEN ||
        socket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    const nextSocket = new WebSocket(resolveWebSocketURL());
    socket = nextSocket;

    nextSocket.addEventListener("message", (event) => {
      try {
        handleMessage(JSON.parse(event.data) as AgentArenaStreamMessage);
      } catch (error) {
        console.warn("[agent-arena] failed to parse websocket payload", error);
      }
    });

    nextSocket.addEventListener("close", () => {
      if (socket === nextSocket) {
        socket = null;
      }
      scheduleReconnect();
    });

    nextSocket.addEventListener("error", () => {
      nextSocket.close();
    });
  };

  const stopFeed = () => {
    running = false;
    clearReconnect();
    if (socket) {
      const activeSocket = socket;
      socket = null;
      activeSocket.close();
    }
  };

  return {
    start() {
      if (disposed) return;
      running = true;
      connectSocket();
    },
    stop() {
      stopFeed();
    },
    dispose() {
      disposed = true;
      stopFeed();
      listeners = new Set();
    },
    setPreset() {},
    setRunning(nextRunning) {
      running = nextRunning;
      if (!nextRunning) {
        stopFeed();
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
