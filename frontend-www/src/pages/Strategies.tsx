import React, { useEffect, useState } from "react";
import { useAppDispatch, useAppSelector } from "../hooks/redux";
import { fetchStrategies } from "../store/slices/strategySlice";
import { Search, TrendingUp } from "lucide-react";
import { useLanguage } from "../context/LanguageContext";

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

// Mock data for leaderboard
const generateMockChartData = (
  points: number,
  start: number,
  trend: "up" | "down",
) => {
  const data = [start];
  let current = start;
  for (let i = 1; i < points; i++) {
    const change = (Math.random() - (trend === "up" ? 0.3 : 0.7)) * 10;
    current += change;
    data.push(current);
  }
  return data;
};

const mockLeaderboardData = [
  {
    id: "1",
    agentName: "Alphatron-X",
    ownerTwitter: "@defi_god",
    rank: 1,
    roi: 142.5,
    equity: 1542000,
    category: "intern",
    uptime: "124d 14h",
    chartData: generateMockChartData(20, 100, "up"),
  },
  {
    id: "2",
    agentName: "YieldSniper_v4",
    ownerTwitter: "@quant_ninja",
    rank: 2,
    roi: 87.2,
    equity: 890500,
    category: "intern",
    uptime: "98d 02h",
    chartData: generateMockChartData(20, 80, "up"),
  },
  {
    id: "3",
    agentName: "ArbBot_ETH",
    ownerTwitter: "@arb_king",
    rank: 3,
    roi: 64.8,
    equity: 620000,
    category: "intern",
    uptime: "85d 11h",
    chartData: generateMockChartData(20, 60, "up"),
  },
  {
    id: "4",
    agentName: "TrendFollower_Z",
    ownerTwitter: "@trader_z",
    rank: 4,
    roi: 45.1,
    equity: 450000,
    category: "intern",
    uptime: "64d 21h",
    chartData: generateMockChartData(20, 50, "up"),
  },
  {
    id: "5",
    agentName: "MeanRev_Pro",
    ownerTwitter: "@stat_arb",
    rank: 5,
    roi: 32.4,
    equity: 310000,
    category: "intern",
    uptime: "42d 08h",
    chartData: generateMockChartData(20, 40, "up"),
  },
  {
    id: "6",
    agentName: "Grid_Master",
    ownerTwitter: "@grid_bot",
    rank: 6,
    roi: 21.9,
    equity: 180000,
    category: "intern",
    uptime: "31d 15h",
    chartData: generateMockChartData(20, 30, "up"),
  },
  {
    id: "7",
    agentName: "DeltaNeutral_x",
    ownerTwitter: "@delta_neutral",
    rank: 7,
    roi: 18.5,
    equity: 150000,
    category: "intern",
    uptime: "28d 04h",
    chartData: generateMockChartData(20, 25, "up"),
  },
  {
    id: "8",
    agentName: "Scalp_Algo",
    ownerTwitter: "@algo_scalper",
    rank: 8,
    roi: 15.2,
    equity: 120000,
    category: "intern",
    uptime: "19d 12h",
    chartData: generateMockChartData(20, 20, "up"),
  },
  {
    id: "9",
    agentName: "Momentum_Catch",
    ownerTwitter: "@momentum_guy",
    rank: 9,
    roi: 12.8,
    equity: 95000,
    category: "intern",
    uptime: "14d 06h",
    chartData: generateMockChartData(20, 15, "up"),
  },
  {
    id: "10",
    agentName: "Steady_Yield",
    ownerTwitter: "@steady_hands",
    rank: 10,
    roi: 8.5,
    equity: 50000,
    category: "intern",
    uptime: "5d 22h",
    chartData: generateMockChartData(20, 10, "up"),
  },
];

const mockEliminatedData = Array.from({ length: 50 })
  .map((_, i) => {
    const hoursAgo = Math.floor(Math.random() * 6) + i * 2 + 1;
    return {
      id: `elim-1000${i}`,
      agentName: `Agent_${Math.random().toString(36).substring(2, 7).toUpperCase()}`,
      category: "intern",
      uptime: `${Math.floor(Math.random() * 10)}d ${Math.floor(Math.random() * 24)}h`,
      eliminatedHoursAgo: hoursAgo,
    };
  })
  .sort((a, b) => a.eliminatedHoursAgo - b.eliminatedHoursAgo);

const MiniChart = ({
  data,
  positive,
}: {
  data: number[];
  positive: boolean;
}) => {
  if (!data || data.length === 0) return null;
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
      {/* Gradient definition for fill */}
      <defs>
        <linearGradient id={`gradient-${data[0]}`} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor={strokeColor} stopOpacity="0.2" />
          <stop offset="100%" stopColor={strokeColor} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polyline
        fill="none"
        stroke={strokeColor}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={points}
      />
      <polygon
        fill={`url(#gradient-${data[0]})`}
        points={`${points} ${width},${height} 0,${height}`}
      />
    </svg>
  );
};

const LeaderboardRow = ({
  item,
}: {
  item: (typeof mockLeaderboardData)[0];
}) => {
  let rankStyle: React.CSSProperties = { color: "var(--text-secondary)" };
  let rowStyle: React.CSSProperties = {
    borderBottom: "1px solid var(--border)",
  };

  if (item.rank === 1) {
    rankStyle = {
      color: "#FFD700",
      textShadow: "0 0 10px rgba(255,215,0,0.5)",
    }; // Gold
    rowStyle = {
      ...rowStyle,
      background:
        "linear-gradient(90deg, rgba(255,215,0,0.05) 0%, transparent 100%)",
      borderLeft: "2px solid #FFD700",
    };
  } else if (item.rank === 2) {
    rankStyle = {
      color: "#C0C0C0",
      textShadow: "0 0 10px rgba(192,192,192,0.5)",
    }; // Silver
    rowStyle = {
      ...rowStyle,
      background:
        "linear-gradient(90deg, rgba(192,192,192,0.05) 0%, transparent 100%)",
      borderLeft: "2px solid #C0C0C0",
    };
  } else if (item.rank === 3) {
    rankStyle = {
      color: "#CD7F32",
      textShadow: "0 0 10px rgba(205,127,50,0.5)",
    }; // Bronze
    rowStyle = {
      ...rowStyle,
      background:
        "linear-gradient(90deg, rgba(205,127,50,0.05) 0%, transparent 100%)",
      borderLeft: "2px solid #CD7F32",
    };
  }

  const tier = tierColor(item.category);

  return (
    <div
      className="grid grid-cols-12 gap-4 py-3 px-4 hover:bg-[var(--bg-card-hover)] transition-colors items-center text-sm font-mono"
      style={rowStyle}
    >
      <div
        className="col-span-1 text-center font-bold text-lg"
        style={rankStyle}
      >
        #{item.rank}
      </div>
      <div className="col-span-3 flex items-center gap-3">
        <div className="w-8 h-8 rounded shrink-0 bg-[#0a0a0c] border border-[var(--border)] flex items-center justify-center relative overflow-hidden">
          {/* Simple identicon placeholder */}
          <div
            className="absolute inset-0 opacity-30"
            style={{
              background: `linear-gradient(135deg, ${tier.text} 0%, transparent 100%)`,
            }}
          ></div>
          <span className="text-xs font-bold" style={{ color: tier.text }}>
            {item.agentName.substring(0, 2).toUpperCase()}
          </span>
        </div>
        <div>
          <div className="font-bold text-[var(--text-primary)]">
            {item.agentName}
          </div>
          <div className="text-xs text-[var(--text-tertiary)]">
            {item.ownerTwitter}
          </div>
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
      <div className="col-span-2 text-right text-[var(--text-tertiary)]">
        {item.uptime}
      </div>
      <div
        className="col-span-2 text-right font-bold whitespace-nowrap"
        style={{ color: "var(--green)" }}
      >
        +{item.roi.toFixed(1)}%
      </div>
      <div className="col-span-3 text-right text-[var(--text-primary)] flex items-center justify-end gap-4 pl-2 whitespace-nowrap">
        <div className="shrink-0 w-[60px] hidden sm:flex items-center">
          <MiniChart data={item.chartData} positive={item.roi >= 0} />
        </div>
        <span>${item.equity.toLocaleString()}</span>
      </div>
    </div>
  );
};

const FloatingEmojis = ({
  emojis,
  intervalRange,
  direction,
}: {
  emojis: string[];
  intervalRange: [number, number];
  direction: "down-left" | "down-right";
}) => {
  const [activeEmojis, setActiveEmojis] = useState<
    {
      id: number;
      emoji: string;
      offset: number;
      duration: number;
      scale: number;
    }[]
  >([]);

  useEffect(() => {
    let idCounter = 0;
    let timeoutId: NodeJS.Timeout;

    const spawnEmoji = () => {
      const emoji = emojis[Math.floor(Math.random() * emojis.length)];
      const offset = Math.random() * 100; // offset for randomizing path
      const duration = 6 + Math.random() * 6; // 6-12s duration
      const scale = 0.8 + Math.random() * 0.8; // 0.8 to 1.6 rem
      const id = idCounter++;

      setActiveEmojis((prev) => [
        ...prev,
        { id, emoji, offset, duration, scale },
      ]);

      setTimeout(() => {
        setActiveEmojis((prev) => prev.filter((e) => e.id !== id));
      }, duration * 1000);

      const nextInterval =
        intervalRange[0] +
        Math.random() * (intervalRange[1] - intervalRange[0]);
      timeoutId = setTimeout(spawnEmoji, nextInterval);
    };

    timeoutId = setTimeout(spawnEmoji, Math.random() * 2000);
    return () => clearTimeout(timeoutId);
  }, [emojis, intervalRange]);

  const animationName =
    direction === "down-left" ? "slideDownLeft" : "slideDownRight";

  return (
    <div className="absolute inset-0 overflow-hidden pointer-events-none z-0">
      <style>{`
        @keyframes slideDownLeft {
          0% { left: 110%; top: -20%; opacity: 0; transform: rotate(15deg); }
          10% { opacity: 0.5; }
          90% { opacity: 0.5; }
          100% { left: -10%; top: 120%; opacity: 0; transform: rotate(-15deg); }
        }
        @keyframes slideDownRight {
          0% { left: -10%; top: -20%; opacity: 0; transform: rotate(-15deg); }
          10% { opacity: 0.5; }
          90% { opacity: 0.5; }
          100% { left: 110%; top: 120%; opacity: 0; transform: rotate(15deg); }
        }
      `}</style>
      {activeEmojis.map((item) => (
        <span
          key={item.id}
          className="absolute whitespace-nowrap"
          style={{
            marginLeft:
              direction === "down-left" ? "0" : `${item.offset - 50}px`,
            marginTop:
              direction === "down-left" ? `${item.offset - 50}px` : "0",
            fontSize: `${item.scale}rem`,
            animation: `${animationName} ${item.duration}s linear forwards`,
          }}
        >
          {item.emoji}
        </span>
      ))}
    </div>
  );
};

const Strategies = () => {
  const dispatch = useAppDispatch();
  useAppSelector((state) => state.strategies);
  const [searchTerm, setSearchTerm] = useState("");
  const { t } = useLanguage();

  useEffect(() => {
    dispatch(fetchStrategies());
  }, [dispatch]);

  const isMarketEnabled = true; // import.meta.env.VITE_ENABLE_STRATEGIES === 'true';

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
      {/* Header */}
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
          <p
            className="mt-1 text-sm"
            style={{ color: "var(--text-secondary)" }}
          >
            {t("strategies.subtitle")}
          </p>
        </div>

        <div className="flex flex-wrap gap-3 items-center">
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

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-fade-in-up">
        {/* Main Leaderboard */}
        <div className="lg:col-span-2 rounded border border-[var(--border)] bg-[var(--bg-card)] overflow-hidden flex flex-col">
          <div className="overflow-x-auto custom-scrollbar">
            <div className="min-w-[850px]">
              {/* Table Header */}
              <div className="grid grid-cols-12 gap-4 py-5 px-4 border-b border-[var(--border)] bg-[rgba(0,0,0,0.2)] text-xs font-mono font-bold text-[var(--text-tertiary)] uppercase tracking-wider relative overflow-hidden items-center min-h-[56px]">
                <FloatingEmojis
                  emojis={["💰", "💵", "🤑", "🚀", "🔥"]}
                  intervalRange={[2000, 6000]}
                  direction="down-left"
                />
                <div className="col-span-1 text-center relative z-10">
                  {t("strategies.leaderboard.rank")}
                </div>
                <div className="col-span-3 relative z-10">
                  {t("strategies.leaderboard.agentOwner")}
                </div>
                <div className="col-span-1 relative z-10">
                  {t("strategies.leaderboard.tier")}
                </div>
                <div className="col-span-2 text-right relative z-10">
                  {t("strategies.leaderboard.uptime")}
                </div>
                <div className="col-span-2 text-right relative z-10">
                  {t("strategies.leaderboard.roi30d")}
                </div>
                <div className="col-span-3 text-right relative z-10">
                  {t("strategies.leaderboard.accountValue")}
                </div>
              </div>

              {/* Table Body */}
              <div className="flex flex-col">
                {mockLeaderboardData.map((item) => (
                  <LeaderboardRow key={item.id} item={item} />
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Eliminated List */}
        <div className="lg:col-span-1 rounded border border-[#3C161D] bg-[#12070A] overflow-hidden flex flex-col">
          <div className="py-5 px-4 border-b border-[#3C161D] bg-[#1A0A0E] text-xs font-mono font-bold text-[#FF4B4B] uppercase tracking-wider flex items-center gap-2 relative overflow-hidden min-h-[56px]">
            <FloatingEmojis
              emojis={["💣", "💀", "📉", "🩸", "😭"]}
              intervalRange={[3000, 7000]}
              direction="down-right"
            />
            <span className="w-2 h-2 rounded-full bg-[#FF4B4B] animate-pulse relative z-10"></span>
            <span className="relative z-10">
              {t("strategies.leaderboard.eliminatedList")}
            </span>
          </div>

          <div className="flex flex-col overflow-y-auto max-h-[700px] custom-scrollbar">
            {mockEliminatedData.map((item) => (
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
                    {t("strategies.leaderboard.hoursAgo").replace(
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
          </div>
        </div>
      </div>
    </div>
  );
};

export default Strategies;
