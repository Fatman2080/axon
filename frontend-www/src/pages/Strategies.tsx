import React, { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useAppDispatch, useAppSelector } from "../hooks/redux";
import { fetchStrategies } from "../store/slices/strategySlice";
import { Search, TrendingUp, ArrowRight, Trophy, Grid } from "lucide-react";
import { useLanguage } from "../context/LanguageContext";
import type { Strategy } from "../types";

const tierColor = (category: string) => {
  switch (category?.toLowerCase()) {
    case "partner":
      return {
        bg: "rgba(255,184,0,0.08)",
        text: "var(--tier-partner)",
        border: "rgba(255,184,0,0.2)",
      };
    case "manager":
      return {
        bg: "rgba(138,43,226,0.08)",
        text: "var(--tier-manager)",
        border: "rgba(138,43,226,0.2)",
      };
    case "analyst":
      return {
        bg: "rgba(42,127,255,0.1)",
        text: "var(--tier-analyst)",
        border: "rgba(42,127,255,0.2)",
      };
    default:
      return {
        bg: "rgba(142,146,155,0.1)",
        text: "var(--tier-intern)",
        border: "rgba(142,146,155,0.2)",
      };
  }
};

const formatMoney = (value: number) => {
  if (!Number.isFinite(value)) return "$0";
  if (Math.abs(value) >= 1_000_000) {
    return `$${(value / 1_000_000).toFixed(2)}M`;
  }
  return `$${value.toLocaleString(undefined, { maximumFractionDigits: 2 })}`;
};

const hoursSince = (ts?: string): number | null => {
  if (!ts) return null;
  const t = new Date(ts).getTime();
  if (!Number.isFinite(t) || t <= 0) return null;
  const diffMs = Date.now() - t;
  if (diffMs < 0) return 0;
  return Math.floor(diffMs / 3600000);
};

const formatUptime = (days?: number, startedAt?: string): string => {
  if (days && days > 0) {
    const d = Math.floor(days);
    return `${d}d`;
  }
  const h = hoursSince(startedAt);
  if (h === null) return "--";
  const d = Math.floor(h / 24);
  const hr = h % 24;
  if (d > 0) return `${d}d ${hr}h`;
  return `${hr}h`;
};

const formatOwner = (creator: string | undefined, notPublicLabel: string): string => {
  const v = (creator || "").trim();
  if (!v) return notPublicLabel;
  if (v.startsWith("@")) return v;
  if (v.includes(" ")) return v;
  return `@${v}`;
};

const calcROI = (s: Strategy): number => {
  const pnl = Number(s.pnlContribution || 0);
  const baseFromInitial = Number(s.initialCapital || 0);
  if (baseFromInitial > 0) {
    return (pnl / baseFromInitial) * 100;
  }
  const tvl = Number(s.currentTvl || 0);
  if (tvl <= 0) return 0;
  return (pnl / tvl) * 100;
};

const buildChartData = (s: Strategy): number[] => {
  const daily = Array.isArray(s.performance?.daily)
    ? s.performance.daily.filter((x) => Number.isFinite(x))
    : [];
  if (daily.length >= 2) {
    return daily.slice(-20);
  }
  return [];
};

type LeaderboardItem = {
  id: string;
  agentName: string;
  owner: string;
  ownerAvatar?: string;
  ownerUserId?: string;
  rank: number;
  roi: number;
  equity: number;
  category: string;
  uptime: string;
  chartData: number[];
};

type EliminatedItem = {
  id: string;
  agentName: string;
  category: string;
  uptime: string;
  eliminatedHoursAgo: number | null;
};

const MiniChart = ({
  data,
  positive,
}: {
  data: number[];
  positive: boolean;
}) => {
  if (!data || data.length < 2) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const width = 60;
  const height = 24;

  const points = data
    .map((val, i) => {
      const x = (i / (data.length - 1)) * width;
      const y =
        max === min
          ? height / 2
          : height - ((val - min) / (max - min)) * height;
      return `${x},${y}`;
    })
    .join(" ");

  const strokeColor = positive ? "var(--green)" : "var(--red)";

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className="overflow-visible"
    >
      <polyline
        fill="none"
        stroke={strokeColor}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={points}
      />
    </svg>
  );
};

const LeaderboardRow = ({ item }: { item: LeaderboardItem }) => {
  const tier = tierColor(item.category);
  const roiColor = item.roi >= 0 ? "var(--green)" : "var(--red)";
  const roiPrefix = item.roi >= 0 ? "+" : "";
  const [avatarFailed, setAvatarFailed] = useState(false);

  useEffect(() => {
    setAvatarFailed(false);
  }, [item.ownerAvatar]);

  const rowClassName = [
    "grid grid-cols-12 gap-4 py-3 px-4 items-center text-sm font-mono",
    item.ownerUserId ? "hover:bg-[var(--bg-card-hover)] transition-colors cursor-pointer" : "",
  ]
    .join(" ")
    .trim();
  const rowChildren = (
    <>
      <div className="col-span-1 text-center font-bold text-lg">#{item.rank}</div>
      <div className="col-span-3 flex items-center gap-3">
        <div className="w-8 h-8 rounded shrink-0 bg-[#0a0a0c] border border-[var(--border)] flex items-center justify-center relative overflow-hidden">
          {item.ownerAvatar && !avatarFailed ? (
            <img
              src={item.ownerAvatar}
              alt={item.owner}
              className="h-full w-full object-cover"
              onError={() => setAvatarFailed(true)}
            />
          ) : (
            <>
              <div
                className="absolute inset-0 opacity-30"
                style={{
                  background: `linear-gradient(135deg, ${tier.text} 0%, transparent 100%)`,
                }}
              ></div>
              <span className="text-xs font-bold" style={{ color: tier.text }}>
                {item.agentName.substring(0, 2).toUpperCase()}
              </span>
            </>
          )}
        </div>
        <div>
          <div className="font-bold text-[var(--text-primary)]">{item.agentName}</div>
          <div className="text-xs text-[var(--text-tertiary)]">{item.owner}</div>
        </div>
      </div>
      <div className="col-span-1 flex items-center">
        <span
          className="inline-block text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded"
          style={{
            background: tier.bg,
            color: tier.text,
            border: `1px solid ${tier.border}`,
          }}
        >
          {item.category}
        </span>
      </div>
      <div className="col-span-2 text-right text-[var(--text-tertiary)]">{item.uptime}</div>
      <div className="col-span-2 text-right font-bold whitespace-nowrap" style={{ color: roiColor }}>
        {roiPrefix}
        {item.roi.toFixed(1)}%
      </div>
      <div className="col-span-3 text-right text-[var(--text-primary)] flex items-center justify-end gap-4 pl-2 whitespace-nowrap">
        <div className="shrink-0 w-[60px] hidden sm:flex items-center justify-end">
          <MiniChart data={item.chartData} positive={item.roi >= 0} />
        </div>
        <span>{formatMoney(item.equity)}</span>
      </div>
    </>
  );

  if (!item.ownerUserId) {
    return (
      <div className={rowClassName} style={{ borderBottom: "1px solid var(--border)" }}>
        {rowChildren}
      </div>
    );
  }

  return (
    <Link to={`/profile/${item.ownerUserId}`} className="block">
      <div className={rowClassName} style={{ borderBottom: "1px solid var(--border)" }}>
        {rowChildren}
      </div>
    </Link>
  );
};

const Strategies = () => {
  const dispatch = useAppDispatch();
  const { items, loading } = useAppSelector((state) => state.strategies);
  const [searchTerm, setSearchTerm] = useState("");
  const [filter] = useState("all");
  const [viewMode, setViewMode] = useState<"grid" | "leaderboard">(
    "leaderboard",
  );
  const { t } = useLanguage();

  useEffect(() => {
    dispatch(fetchStrategies());
  }, [dispatch]);

  const { leaderboard, eliminated, filteredStrategies } = useMemo(() => {
    const q = searchTerm.trim().toLowerCase();
    const filtered = items.filter((s) => {
      if (!q) return true;
      return (
        s.name.toLowerCase().includes(q) ||
        s.id.toLowerCase().includes(q) ||
        (s.creator || "").toLowerCase().includes(q)
      );
    });

    const active = filtered
      .filter((s) => s.agentStatus === "active")
      .sort((a, b) => calcROI(b) - calcROI(a));

    const leaderboardRows: LeaderboardItem[] = active.map((s, idx) => ({
      id: s.id,
      agentName: s.name || `${s.id.slice(0, 8)}...`,
      owner: formatOwner(s.creator, t("strategies.leaderboard.notPublic")),
      ownerAvatar: s.creatorAvatar,
      ownerUserId: s.ownerUserId,
      rank: idx + 1,
      roi: calcROI(s),
      equity: Number(s.currentTvl || 0),
      category: s.category || "intern",
      uptime: formatUptime(s.runningDays, s.startedAt),
      chartData: buildChartData(s),
    }));

    const eliminatedRows: EliminatedItem[] = filtered
      .filter((s) => s.agentStatus !== "active")
      .sort((a, b) => {
        const ah = hoursSince(a.lastSyncedAt);
        const bh = hoursSince(b.lastSyncedAt);
        if (ah === null && bh === null) return 0;
        if (ah === null) return 1;
        if (bh === null) return -1;
        return ah - bh;
      })
      .map((s) => ({
        id: s.id,
        agentName: s.name || `${s.id.slice(0, 8)}...`,
        category: s.category || "intern",
        uptime: formatUptime(s.runningDays, s.startedAt),
        eliminatedHoursAgo: hoursSince(s.lastSyncedAt),
      }));

    const cardFiltered = items.filter((strategy) => {
      const isActive = strategy.agentStatus === "active";
      const matchesFilter = filter === "all" || strategy.category === filter;
      const matchesSearch = strategy.name.toLowerCase().includes(q);
      return isActive && matchesFilter && matchesSearch;
    });

    return {
      leaderboard: leaderboardRows,
      eliminated: eliminatedRows,
      filteredStrategies: cardFiltered,
    };
  }, [items, searchTerm, filter, t]);

  const isMarketEnabled = true;

  if (!isMarketEnabled) {
    return (
      <div className="flex flex-col items-center justify-center py-32 text-center animate-fade-in-up">
        <div
          className="h-16 w-16 mb-6 rounded-full flex items-center justify-center"
          style={{
            background: "var(--bg-card)",
            border: "1px solid var(--border)",
          }}
        >
          <TrendingUp size={24} style={{ color: "var(--text-tertiary)" }} />
        </div>
        <h2
          className="text-xl font-bold tracking-tight mb-3"
          style={{ color: "var(--text-primary)" }}
        >
          {t("strategies.underConstructionTitle")}
        </h2>
        <p
          className="text-sm font-mono max-w-md"
          style={{ color: "var(--text-secondary)" }}
        >
          {t("strategies.underConstruction")}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-fade-in-up">
      <div
        className="flex flex-col justify-between gap-4 md:flex-row md:items-center pb-6"
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        <div>
          <h1
            className="text-2xl font-bold tracking-tight"
            style={{ color: "var(--text-primary)" }}
          >
            {t("strategies.title")}
          </h1>
          <p className="mt-1 text-sm" style={{ color: "var(--text-secondary)" }}>
            {t("strategies.subtitle")}
          </p>
        </div>

        <div className="flex flex-wrap gap-3 items-center">
          {/* View Toggle */}
          <div className="flex bg-[var(--bg-input)] rounded border border-[var(--border)] p-0.5">
            <button
              onClick={() => setViewMode("grid")}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold rounded transition-all ${viewMode === "grid" ? "bg-[rgba(0,255,65,0.12)] text-[var(--neon-green)]" : "text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]"}`}
              style={{ fontFamily: "var(--font-mono)" }}
            >
              <Grid size={14} />
              {t("strategies.viewMode.grid")}
            </button>
            <button
              onClick={() => setViewMode("leaderboard")}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold rounded transition-all ${viewMode === "leaderboard" ? "bg-[rgba(0,255,65,0.12)] text-[var(--neon-green)]" : "text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]"}`}
              style={{ fontFamily: "var(--font-mono)" }}
            >
              <Trophy size={14} />
              {t("strategies.viewMode.leaderboard")}
            </button>
          </div>
          {/* Search */}
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
              style={{ color: "var(--text-tertiary)" }}
            />
            <input
              type="text"
              placeholder={t("strategies.searchPlaceholder")}
              className="h-9 pl-9 pr-4 text-sm font-mono"
              style={{
                background: "var(--bg-card)",
                border: "1px solid var(--border)",
                borderRadius: "4px",
                color: "var(--text-primary)",
                width: "200px",
              }}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              onFocus={(e) =>
                ((e.currentTarget as HTMLElement).style.borderColor =
                  "var(--neon-green)")
              }
              onBlur={(e) =>
                ((e.currentTarget as HTMLElement).style.borderColor =
                  "var(--border)")
              }
            />
          </div>
        </div>
      </div>

      {viewMode === "leaderboard" ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-fade-in-up">
          <div className="lg:col-span-2 rounded border border-[var(--border)] bg-[var(--bg-card)] overflow-hidden flex flex-col">
            <div className="overflow-x-auto custom-scrollbar">
              <div className="min-w-[850px]">
                <div className="grid grid-cols-12 gap-4 py-5 px-4 border-b border-[var(--border)] bg-[rgba(0,0,0,0.2)] text-xs font-mono font-bold text-[var(--text-tertiary)] uppercase tracking-wider items-center min-h-[56px]">
                  <div className="col-span-1 text-center">{t("strategies.leaderboard.rank")}</div>
                  <div className="col-span-3">{t("strategies.leaderboard.agentOwner")}</div>
                  <div className="col-span-1">{t("strategies.leaderboard.tier")}</div>
                  <div className="col-span-2 text-right">{t("strategies.leaderboard.uptime")}</div>
                  <div className="col-span-2 text-right">{t("strategies.leaderboard.roi30d")}</div>
                  <div className="col-span-3 text-right">{t("strategies.leaderboard.accountValue")}</div>
                </div>

                <div className="flex flex-col">
                  {leaderboard.map((item) => (
                    <LeaderboardRow key={item.id} item={item} />
                  ))}
                  {leaderboard.length === 0 && (
                    <div className="py-10 text-center text-sm font-mono" style={{ color: "var(--text-tertiary)" }}>
                      {loading ? t("strategies.leaderboard.loading") : t("strategies.leaderboard.noActiveAgents")}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-1 rounded border border-[#3C161D] bg-[#12070A] overflow-hidden flex flex-col">
            <div className="py-5 px-4 border-b border-[#3C161D] bg-[#1A0A0E] text-xs font-mono font-bold text-[#FF4B4B] uppercase tracking-wider flex items-center gap-2 min-h-[56px]">
              <span className="w-2 h-2 rounded-full bg-[#FF4B4B] animate-pulse"></span>
              <span>{t("strategies.leaderboard.eliminatedList")}</span>
            </div>

            <div className="flex flex-col overflow-y-auto max-h-[700px] custom-scrollbar">
              {eliminated.map((item) => (
                <div
                  key={item.id}
                  className="py-2.5 px-4 border-b border-[#2A0F14] hover:bg-[#1A0A0E] transition-colors flex justify-between items-center text-sm font-mono text-[#D4C3C6]"
                >
                  <div className="flex flex-col gap-1">
                    <span
                      className="font-bold text-[#FF7A7A] truncate max-w-[120px]"
                      title={item.agentName}
                    >
                      {item.agentName}
                    </span>
                    <span className="text-[10px] text-[#FF4B4B] opacity-80 uppercase">
                      {item.eliminatedHoursAgo === null
                        ? "--"
                        : t("strategies.leaderboard.hoursAgo").replace(
                            "{h}",
                            item.eliminatedHoursAgo.toString(),
                          )}
                    </span>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className="text-[10px] text-[var(--text-tertiary)] border border-[var(--text-tertiary)] rounded px-1 uppercase">
                      {item.category}
                    </span>
                    <span className="text-[10px] opacity-70 border-b border-dotted border-[#6A4040]">
                      {t("strategies.leaderboard.survived")} {item.uptime}
                    </span>
                  </div>
                </div>
              ))}
              {eliminated.length === 0 && (
                <div className="py-10 text-center text-xs font-mono text-[#8f6f73]">
                  {t("strategies.leaderboard.noEliminatedAgents")}
                </div>
              )}
            </div>
          </div>
        </div>
      ) : /* Cards */
      loading ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div
              key={i}
              className="h-60 rounded animate-pulse"
              style={{ background: "var(--bg-card)" }}
            />
          ))}
        </div>
      ) : filteredStrategies.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <TrendingUp
            size={40}
            className="mb-4"
            style={{ color: "var(--text-tertiary)" }}
          />
          <p
            className="font-mono text-sm"
            style={{ color: "var(--text-secondary)" }}
          >
            // NO_AGENTS_FOUND
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredStrategies.map((strategy) => {
            const tier = tierColor(strategy.category);
            const pnlPositive = strategy.pnlContribution >= 0;

            return (
              <div
                key={strategy.id}
                className="group flex flex-col justify-between rounded p-5 transition-all duration-200"
                style={{
                  background: "var(--bg-card)",
                  border: "1px solid var(--border)",
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.borderColor =
                    "var(--border-hover)";
                  (e.currentTarget as HTMLElement).style.background =
                    "var(--bg-card-hover)";
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.borderColor =
                    "var(--border)";
                  (e.currentTarget as HTMLElement).style.background =
                    "var(--bg-card)";
                }}
              >
                <div>
                  <div className="mb-4 flex items-start justify-between">
                    <div>
                      <span
                        className="inline-block text-[10px] font-mono font-bold uppercase tracking-widest px-2 py-0.5 rounded mb-2"
                        style={{
                          background: tier.bg,
                          color: tier.text,
                          border: `1px solid ${tier.border}`,
                        }}
                      >
                        {strategy.category}
                      </span>
                      <h3
                        className="text-base font-bold"
                        style={{ color: "var(--text-primary)" }}
                      >
                        {strategy.name}
                      </h3>
                    </div>
                  </div>

                  <div className="space-y-3 mb-5">
                    {/* PnL */}
                    <div className="flex justify-between text-sm items-center">
                      <span
                        className="font-mono text-xs uppercase tracking-widest"
                        style={{ color: "var(--text-tertiary)" }}
                      >
                        {t("strategies.card.apr")}
                      </span>
                      <span
                        className="font-mono font-bold"
                        style={{
                          color: pnlPositive ? "var(--green)" : "var(--red)",
                        }}
                      >
                        {pnlPositive ? "+" : ""}
                        {strategy.pnlContribution.toFixed(2)}
                      </span>
                    </div>

                    {/* TVL */}
                    <div>
                      <div className="flex justify-between text-sm items-center mb-1.5">
                        <span
                          className="font-mono text-xs uppercase tracking-widest"
                          style={{ color: "var(--text-tertiary)" }}
                        >
                          {t("strategies.card.tvl")}
                        </span>
                        <span
                          className="font-mono text-sm"
                          style={{ color: "var(--text-primary)" }}
                        >
                          ${strategy.currentTvl.toLocaleString()}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <Link
                  to={`/strategies/${strategy.id}`}
                  className="flex items-center justify-between text-sm font-mono font-bold transition-all group-hover:gap-3 pt-4"
                  style={{
                    color: "var(--text-secondary)",
                    borderTop: "1px solid var(--border)",
                  }}
                  onMouseEnter={(e) =>
                    ((e.currentTarget as HTMLElement).style.color =
                      "var(--neon-green)")
                  }
                  onMouseLeave={(e) =>
                    ((e.currentTarget as HTMLElement).style.color =
                      "var(--text-secondary)")
                  }
                >
                  {t("strategies.card.hire")}
                  <ArrowRight size={14} />
                </Link>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default Strategies;
