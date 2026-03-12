import { startTransition, useEffect, useRef } from "react";
import { ArenaEngine } from "../engine/ArenaEngine";
import type {
  ArenaAgentEntity,
  ArenaFeed,
  ArenaPerfStats,
  ArenaScenarioPreset,
} from "../types";

interface AgentArenaCanvasProps {
  preset: ArenaScenarioPreset;
  running: boolean;
  createFeed: (preset: ArenaScenarioPreset) => ArenaFeed;
  onSelectAgent: (agent: ArenaAgentEntity | null) => void;
  onStats: (stats: ArenaPerfStats) => void;
}

export function AgentArenaCanvas({
  preset,
  running,
  createFeed,
  onSelectAgent,
  onStats,
}: AgentArenaCanvasProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const engineRef = useRef<ArenaEngine | null>(null);
  const feedRef = useRef<ArenaFeed | null>(null);

  useEffect(() => {
    let mounted = true;

    const bootstrap = async () => {
      if (!hostRef.current) return;

      const engine = new ArenaEngine({
        onSelectAgent: (agent) => {
          startTransition(() => onSelectAgent(agent));
        },
        onStats: (stats) => {
          startTransition(() => onStats(stats));
        },
      });
      await engine.init(hostRef.current);

      if (!mounted) {
        engine.dispose();
        return;
      }

      const feed = createFeed(preset);
      const unsubscribe = feed.subscribe((events) => engine.enqueue(events));
      feed.setPreset(preset);
      feed.setRunning(running);
      if (running) {
        feed.start();
      } else {
        feed.stop();
      }

      engineRef.current = engine;
      feedRef.current = feed;

      return () => {
        unsubscribe();
      };
    };

    let cleanupSubscription: (() => void) | undefined;
    bootstrap().then((cleanup) => {
      cleanupSubscription = cleanup;
    });

    return () => {
      mounted = false;
      cleanupSubscription?.();
      feedRef.current?.dispose();
      engineRef.current?.dispose();
      feedRef.current = null;
      engineRef.current = null;
    };
  }, [createFeed, onSelectAgent, onStats]);

  useEffect(() => {
    feedRef.current?.setPreset(preset);
  }, [preset]);

  useEffect(() => {
    feedRef.current?.setRunning(running);
    if (running) {
      feedRef.current?.start();
    } else {
      feedRef.current?.stop();
    }
  }, [running]);

  return (
    <div
      ref={hostRef}
      className="relative min-h-[560px] w-full overflow-hidden rounded-2xl border lg:min-h-[640px]"
      style={{
        borderColor: "rgba(255,255,255,0.08)",
        background:
          "radial-gradient(circle at top, rgba(0,255,65,0.08), transparent 28%), linear-gradient(180deg, rgba(14,18,26,0.98) 0%, rgba(7,9,14,1) 100%)",
        boxShadow:
          "0 30px 80px rgba(0,0,0,0.35), inset 0 1px 0 rgba(255,255,255,0.05)",
      }}
    >
      <div className="pointer-events-none absolute right-4 top-4 z-10 flex items-center gap-2">
        <div
          className="pointer-events-auto flex items-center gap-1 rounded-full border p-1"
          style={{
            borderColor: "rgba(255,255,255,0.08)",
            background: "rgba(8,11,18,0.8)",
          }}
        >
          <button
            onClick={() => engineRef.current?.zoomOut()}
            className="rounded-full w-8 h-8 text-sm font-bold"
            style={{
              background: "rgba(255,255,255,0.04)",
              color: "var(--text-primary)",
            }}
          >
            -
          </button>
          <button
            onClick={() => engineRef.current?.resetCamera()}
            className="rounded-full px-3 h-8 text-[11px] font-bold uppercase tracking-[0.14em]"
            style={{
              background: "rgba(255,255,255,0.04)",
              color: "var(--text-primary)",
            }}
          >
            Reset
          </button>
          <button
            onClick={() => engineRef.current?.zoomIn()}
            className="rounded-full w-8 h-8 text-sm font-bold"
            style={{
              background: "rgba(255,255,255,0.04)",
              color: "var(--text-primary)",
            }}
          >
            +
          </button>
        </div>
      </div>
    </div>
  );
}
