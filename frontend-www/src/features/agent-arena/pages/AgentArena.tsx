import { useState } from "react";
import { AgentArenaCanvas } from "../components/AgentArenaCanvas";
import { createLiveArenaFeed } from "../live/createLiveArenaFeed";
import type {
  ArenaAgentEntity,
  ArenaPerfStats,
  ArenaScenarioPreset,
} from "../types";

const LIVE_PRESET: ArenaScenarioPreset = {
  agentCount: 300,
  eventRate: 10,
  script: "steady",
};

const EMPTY_STATS: ArenaPerfStats = {
  fps: 0,
  liveAgents: 0,
  deadAgents: 0,
  renderedAgents: 0,
  pooledAgents: 0,
  pooledFloaters: 0,
  activeFloaters: 0,
  queuedEvents: 0,
  eventsPerSecond: 0,
};

function formatUsd(value: number) {
  const sign = value >= 0 ? "+" : "-";
  return `${sign}$${Math.abs(value).toLocaleString(undefined, { maximumFractionDigits: 2 })}`;
}

export default function AgentArena() {
  const [running, setRunning] = useState(true);
  const [selectedAgent, setSelectedAgent] = useState<ArenaAgentEntity | null>(
    null,
  );
  const [stats, setStats] = useState<ArenaPerfStats>(EMPTY_STATS);
  const [reconnectKey, setReconnectKey] = useState(0);

  return (
    <div className="pb-16 animate-fade-in-up">
      <div className="mb-6 flex flex-col gap-4">
        <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
          <div>
            <h1
              className="text-3xl font-bold tracking-tight"
              style={{ color: "var(--text-primary)" }}
            >
              Agent Arena
            </h1>
            <p
              className="mt-2 max-w-3xl text-sm leading-6"
              style={{ color: "var(--text-secondary)" }}
            >
              Live pixel office view for all assigned agents. The arena reflects
              persisted agent snapshots and realized fills from the backend
              stream, while the lab route remains dedicated to mock stress
              testing.
            </p>
          </div>
          <div className="flex items-center gap-2 self-start md:self-auto">
            <button
              onClick={() => setRunning((value) => !value)}
              className="rounded-lg px-4 py-2 text-sm font-semibold transition"
              style={{
                color: "#08110c",
                background: running ? "var(--neon-green)" : "#facc15",
                boxShadow: running
                  ? "0 0 20px rgba(0,255,65,0.18)"
                  : "0 0 20px rgba(250,204,21,0.18)",
              }}
            >
              {running ? "Pause Stream" : "Resume Stream"}
            </button>
            <button
              onClick={() => {
                setReconnectKey((value) => value + 1);
                setSelectedAgent(null);
                setStats(EMPTY_STATS);
              }}
              className="rounded-lg border px-4 py-2 text-sm font-semibold transition"
              style={{
                borderColor: "rgba(255,255,255,0.12)",
                color: "var(--text-primary)",
                background: "rgba(255,255,255,0.04)",
              }}
            >
              Reconnect
            </button>
          </div>
        </div>

        <div
          className="grid gap-3 rounded-2xl border p-4 md:grid-cols-[1fr_auto]"
          style={{
            borderColor: "var(--border)",
            background: "rgba(17,19,21,0.88)",
          }}
        >
          <div className="grid gap-3 sm:grid-cols-3">
            <div
              className="rounded-xl border px-3 py-3"
              style={{
                borderColor: "rgba(255,255,255,0.06)",
                background: "rgba(255,255,255,0.03)",
              }}
            >
              <div
                className="text-[10px] font-bold uppercase tracking-[0.18em]"
                style={{ color: "var(--text-tertiary)" }}
              >
                Live Agents
              </div>
              <div className="mt-1 text-lg font-bold text-white">
                {stats.liveAgents}
              </div>
            </div>
            <div
              className="rounded-xl border px-3 py-3"
              style={{
                borderColor: "rgba(255,255,255,0.06)",
                background: "rgba(255,255,255,0.03)",
              }}
            >
              <div
                className="text-[10px] font-bold uppercase tracking-[0.18em]"
                style={{ color: "var(--text-tertiary)" }}
              >
                Dead Agents
              </div>
              <div className="mt-1 text-lg font-bold text-[#ff5f56]">
                {stats.deadAgents}
              </div>
            </div>
            <div
              className="rounded-xl border px-3 py-3"
              style={{
                borderColor: "rgba(255,255,255,0.06)",
                background: "rgba(255,255,255,0.03)",
              }}
            >
              <div
                className="text-[10px] font-bold uppercase tracking-[0.18em]"
                style={{ color: "var(--text-tertiary)" }}
              >
                Events/s
              </div>
              <div className="mt-1 text-lg font-bold text-white">
                {stats.eventsPerSecond}
              </div>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3 md:min-w-[220px]">
            <div
              className="rounded-xl border px-3 py-2"
              style={{
                borderColor: "rgba(255,255,255,0.06)",
                background: "rgba(255,255,255,0.03)",
              }}
            >
              <div
                className="text-[10px] font-bold uppercase tracking-[0.18em]"
                style={{ color: "var(--text-tertiary)" }}
              >
                FPS
              </div>
              <div className="mt-1 text-lg font-bold text-white">
                {stats.fps}
              </div>
            </div>
            <div
              className="rounded-xl border px-3 py-2"
              style={{
                borderColor: "rgba(255,255,255,0.06)",
                background: "rgba(255,255,255,0.03)",
              }}
            >
              <div
                className="text-[10px] font-bold uppercase tracking-[0.18em]"
                style={{ color: "var(--text-tertiary)" }}
              >
                Queued
              </div>
              <div className="mt-1 text-lg font-bold text-white">
                {stats.queuedEvents}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="min-w-0">
          <AgentArenaCanvas
            key={reconnectKey}
            createFeed={createLiveArenaFeed}
            preset={LIVE_PRESET}
            running={running}
            onSelectAgent={setSelectedAgent}
            onStats={setStats}
          />
        </div>

        <div className="space-y-4">
          <section
            className="rounded-2xl border p-4"
            style={{
              borderColor: "var(--border)",
              background: "rgba(17,19,21,0.94)",
            }}
          >
            <div
              className="mb-3 text-xs font-bold uppercase tracking-[0.22em]"
              style={{ color: "var(--text-tertiary)" }}
            >
              Runtime Stats
            </div>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div
                className="rounded-xl border p-3"
                style={{
                  borderColor: "rgba(255,255,255,0.06)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <div style={{ color: "var(--text-tertiary)" }}>Rendered</div>
                <div className="mt-1 text-xl font-bold">
                  {stats.renderedAgents}
                </div>
              </div>
              <div
                className="rounded-xl border p-3"
                style={{
                  borderColor: "rgba(255,255,255,0.06)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <div style={{ color: "var(--text-tertiary)" }}>Floaters</div>
                <div className="mt-1 text-xl font-bold">
                  {stats.activeFloaters}
                </div>
              </div>
              <div
                className="rounded-xl border p-3"
                style={{
                  borderColor: "rgba(255,255,255,0.06)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <div style={{ color: "var(--text-tertiary)" }}>
                  Pooled Views
                </div>
                <div className="mt-1 text-xl font-bold">
                  {stats.pooledAgents}
                </div>
              </div>
              <div
                className="rounded-xl border p-3"
                style={{
                  borderColor: "rgba(255,255,255,0.06)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <div style={{ color: "var(--text-tertiary)" }}>
                  Pooled Floaters
                </div>
                <div className="mt-1 text-xl font-bold">
                  {stats.pooledFloaters}
                </div>
              </div>
            </div>
          </section>

          {selectedAgent && (
            <section
              className="rounded-2xl border p-4"
              style={{
                borderColor: "var(--border)",
                background: "rgba(17,19,21,0.94)",
              }}
            >
              <div className="mb-3 flex items-center justify-between">
                <div
                  className="text-xs font-bold uppercase tracking-[0.22em]"
                  style={{ color: "var(--text-tertiary)" }}
                >
                  Selected Agent
                </div>
                <span
                  className="rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-[0.18em]"
                  style={{
                    background: selectedAgent.isDead
                      ? "rgba(255,95,86,0.14)"
                      : "rgba(0,255,65,0.12)",
                    color: selectedAgent.isDead
                      ? "#ff5f56"
                      : "var(--neon-green)",
                  }}
                >
                  {selectedAgent.status}
                </span>
              </div>
              <div className="space-y-3 text-sm">
                <div>
                  <div
                    className="text-xs uppercase tracking-[0.18em]"
                    style={{ color: "var(--text-tertiary)" }}
                  >
                    Name
                  </div>
                  <div className="mt-1 text-lg font-bold text-white">
                    {selectedAgent.name}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div
                    className="rounded-xl border p-3"
                    style={{
                      borderColor: "rgba(255,255,255,0.06)",
                      background: "rgba(255,255,255,0.03)",
                    }}
                  >
                    <div style={{ color: "var(--text-tertiary)" }}>
                      Total PnL
                    </div>
                    <div
                      className="mt-1 text-base font-bold"
                      style={{
                        color:
                          selectedAgent.totalPnl >= 0 ? "#00ff66" : "#ff5f56",
                      }}
                    >
                      {formatUsd(selectedAgent.totalPnl)}
                    </div>
                  </div>
                  <div
                    className="rounded-xl border p-3"
                    style={{
                      borderColor: "rgba(255,255,255,0.06)",
                      background: "rgba(255,255,255,0.03)",
                    }}
                  >
                    <div style={{ color: "var(--text-tertiary)" }}>
                      Last Realized
                    </div>
                    <div
                      className="mt-1 text-base font-bold"
                      style={{
                        color:
                          selectedAgent.lastRealizedPnl >= 0
                            ? "#00ff66"
                            : "#ff5f56",
                      }}
                    >
                      {formatUsd(selectedAgent.lastRealizedPnl)}
                    </div>
                  </div>
                </div>
                <div
                  className="rounded-xl border p-3"
                  style={{
                    borderColor: "rgba(255,255,255,0.06)",
                    background: "rgba(255,255,255,0.03)",
                  }}
                >
                  <div style={{ color: "var(--text-tertiary)" }}>Slot</div>
                  <div className="mt-1 text-base font-bold text-white">
                    #{selectedAgent.slotIndex}
                  </div>
                </div>
                <div
                  className="rounded-xl border p-3"
                  style={{
                    borderColor: "rgba(255,255,255,0.06)",
                    background: "rgba(255,255,255,0.03)",
                  }}
                >
                  <div style={{ color: "var(--text-tertiary)" }}>
                    Last Event
                  </div>
                  <div className="mt-1 text-base font-bold text-white">
                    {selectedAgent.lastEvent}
                  </div>
                </div>
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}
