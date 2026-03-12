import { useMemo, useState } from "react";
import { AgentArenaCanvas } from "../components/AgentArenaCanvas";
import { createMockArenaFeed } from "../mock/createMockArenaFeed";
import type {
  ArenaAgentEntity,
  ArenaPerfStats,
  ArenaScenarioPreset,
  ArenaScenarioScript,
} from "../types";

const AGENT_COUNTS: ArenaScenarioPreset["agentCount"][] = [50, 300, 1000];
const EVENT_RATES: ArenaScenarioPreset["eventRate"][] = [10, 50, 200];
const SCRIPTS: ArenaScenarioScript[] = ["steady", "pnl-storm", "death-wave"];

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

export default function AgentArenaLab() {
  const [agentCount, setAgentCount] =
    useState<ArenaScenarioPreset["agentCount"]>(300);
  const [eventRate, setEventRate] =
    useState<ArenaScenarioPreset["eventRate"]>(50);
  const [script, setScript] = useState<ArenaScenarioScript>("steady");
  const [running, setRunning] = useState(true);
  const [selectedAgent, setSelectedAgent] = useState<ArenaAgentEntity | null>(
    null,
  );
  const [stats, setStats] = useState<ArenaPerfStats>(EMPTY_STATS);
  const [replayKey, setReplayKey] = useState(0);

  const preset = useMemo(
    () =>
      ({
        agentCount,
        eventRate,
        script,
      }) satisfies ArenaScenarioPreset,
    [agentCount, eventRate, script],
  );

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
              Mock-driven pixel office arena for large agent populations. This
              page is isolated from the current vault UI and is designed to
              validate event throughput, object pooling, selection, and
              rendering stability up to 1000 agents.
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
              {running ? "Pause Feed" : "Resume Feed"}
            </button>
            <button
              onClick={() => {
                setReplayKey((value) => value + 1);
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
              Reset Replay
            </button>
          </div>
        </div>

        <div
          className="grid gap-3 rounded-2xl border p-4 md:grid-cols-[1fr_1fr_1fr_auto]"
          style={{
            borderColor: "var(--border)",
            background: "rgba(17,19,21,0.88)",
          }}
        >
          <div>
            <div
              className="mb-2 text-[11px] font-bold uppercase tracking-[0.18em]"
              style={{ color: "var(--text-tertiary)" }}
            >
              Agent Count
            </div>
            <div className="flex gap-2">
              {AGENT_COUNTS.map((value) => (
                <button
                  key={value}
                  onClick={() => setAgentCount(value)}
                  className="rounded-lg px-3 py-2 text-sm font-semibold"
                  style={{
                    background:
                      agentCount === value
                        ? "rgba(0,255,65,0.12)"
                        : "rgba(255,255,255,0.03)",
                    color:
                      agentCount === value
                        ? "var(--neon-green)"
                        : "var(--text-secondary)",
                    border:
                      agentCount === value
                        ? "1px solid rgba(0,255,65,0.25)"
                        : "1px solid rgba(255,255,255,0.06)",
                  }}
                >
                  {value}
                </button>
              ))}
            </div>
          </div>

          <div>
            <div
              className="mb-2 text-[11px] font-bold uppercase tracking-[0.18em]"
              style={{ color: "var(--text-tertiary)" }}
            >
              Event Rate
            </div>
            <div className="flex gap-2">
              {EVENT_RATES.map((value) => (
                <button
                  key={value}
                  onClick={() => setEventRate(value)}
                  className="rounded-lg px-3 py-2 text-sm font-semibold"
                  style={{
                    background:
                      eventRate === value
                        ? "rgba(96,165,250,0.12)"
                        : "rgba(255,255,255,0.03)",
                    color:
                      eventRate === value ? "#93c5fd" : "var(--text-secondary)",
                    border:
                      eventRate === value
                        ? "1px solid rgba(96,165,250,0.22)"
                        : "1px solid rgba(255,255,255,0.06)",
                  }}
                >
                  {value}/s
                </button>
              ))}
            </div>
          </div>

          <div>
            <div
              className="mb-2 text-[11px] font-bold uppercase tracking-[0.18em]"
              style={{ color: "var(--text-tertiary)" }}
            >
              Scenario
            </div>
            <div className="flex gap-2">
              {SCRIPTS.map((value) => (
                <button
                  key={value}
                  onClick={() => setScript(value)}
                  className="rounded-lg px-3 py-2 text-sm font-semibold capitalize"
                  style={{
                    background:
                      script === value
                        ? "rgba(250,204,21,0.12)"
                        : "rgba(255,255,255,0.03)",
                    color:
                      script === value ? "#facc15" : "var(--text-secondary)",
                    border:
                      script === value
                        ? "1px solid rgba(250,204,21,0.25)"
                        : "1px solid rgba(255,255,255,0.06)",
                  }}
                >
                  {value}
                </button>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3 md:min-w-[260px]">
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
                Events/s
              </div>
              <div className="mt-1 text-lg font-bold text-white">
                {stats.eventsPerSecond}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="min-w-0">
          <AgentArenaCanvas
            key={replayKey}
            createFeed={createMockArenaFeed}
            preset={preset}
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
                <div style={{ color: "var(--text-tertiary)" }}>Live</div>
                <div className="mt-1 text-xl font-bold">{stats.liveAgents}</div>
              </div>
              <div
                className="rounded-xl border p-3"
                style={{
                  borderColor: "rgba(255,255,255,0.06)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <div style={{ color: "var(--text-tertiary)" }}>Dead</div>
                <div className="mt-1 text-xl font-bold text-[#ff5f56]">
                  {stats.deadAgents}
                </div>
              </div>
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
                <div style={{ color: "var(--text-tertiary)" }}>Queued</div>
                <div className="mt-1 text-xl font-bold">
                  {stats.queuedEvents}
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
                <div style={{ color: "var(--text-tertiary)" }}>Floaters</div>
                <div className="mt-1 text-xl font-bold">
                  {stats.activeFloaters}
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
                  <div style={{ color: "var(--text-tertiary)" }}>
                    Last Event
                  </div>
                  <div className="mt-1 text-sm font-semibold text-white">
                    {selectedAgent.lastEvent}
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
                    Slot Index
                  </div>
                  <div className="mt-1 text-sm font-semibold text-white">
                    {selectedAgent.slotIndex}
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
