import axios from "axios";
import type {
  AgentArenaSnapshotResponse,
  AgentMarketItem,
  AgentMarketDetail,
  VaultStats,
  DailySlotsResponse,
  VaultOverview,
  UserAgentStats,
  TreasurySnapshot,
  PlatformStats,
  PlatformSnapshot,
  PublicUserProfile,
  User,
} from "../types";

const API_URL = "";

const api = axios.create({
  baseURL: "/api",
});

// Add token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const authApi = {
  getXOAuthStartUrl: (inviteCode?: string, nextPath?: string) => {
    const params = new URLSearchParams();
    if (inviteCode?.trim()) {
      params.set("inviteCode", inviteCode.trim());
    }
    if (nextPath?.trim()) {
      params.set("next", nextPath.trim());
    }
    const query = params.toString();
    return `/api/auth/x/start${query ? `?${query}` : ""}`;
  },

  logout: () => {
    localStorage.removeItem("token");
  },

  getMe: async () => {
    const response = await api.get("/user/me");
    return response.data;
  },

  updatePreferences: async (payload: { showXOnLeaderboard?: boolean }): Promise<User> => {
    const response = await api.patch('/user/preferences', payload);
    return response.data;
  },

  getPublicProfile: async (id: string): Promise<PublicUserProfile> => {
    const response = await api.get(`/public/profiles/${id}`);
    return response.data;
  },

  consumeInviteCode: async (code: string) => {
    const response = await api.post("/invite-codes/consume", { code });
    return response.data;
  },

  getAgentHistory: async (period?: string) => {
    const response = await api.get("/user/agent/history", {
      params: { period },
    });
    return response.data;
  },

  getAgentStats: async (): Promise<UserAgentStats> => {
    const response = await api.get("/user/agent/stats");
    return response.data;
  },

  getDeployCommand: async (): Promise<{ command: string }> => {
    const response = await api.get("/user/agent/deploy-command");
    return response.data;
  },
};

// Dedup in-flight requests for the same endpoint
const inflight = new Map<string, Promise<any>>();
function dedup<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const existing = inflight.get(key);
  if (existing) return existing;
  const p = fn().finally(() => inflight.delete(key));
  inflight.set(key, p);
  return p;
}

export const marketApi = {
  getAgentMarket: async (search?: string): Promise<AgentMarketItem[]> => {
    const response = await api.get("/agent-market", { params: { search } });
    return response.data;
  },

  getAgentDetail: async (
    publicKey: string,
    period?: string,
  ): Promise<AgentMarketDetail> => {
    const response = await api.get(`/agent-market/${publicKey}`, {
      params: { period },
    });
    return response.data;
  },

  getVaultStats: async (): Promise<VaultStats> => {
    return dedup("vault/stats", async () => {
      const response = await api.get("/vault/stats");
      return response.data;
    });
  },

  getDailySlots: async (): Promise<DailySlotsResponse> => {
    return dedup("daily-slots", async () => {
      const response = await api.get("/daily-slots");
      return response.data;
    });
  },

  getVaultOverview: async (): Promise<VaultOverview> => {
    return dedup("vault/overview", async () => {
      const response = await api.get("/vault/overview");
      return response.data;
    });
  },

  getTreasury: async (): Promise<TreasurySnapshot> => {
    return dedup("treasury", async () => {
      const response = await api.get("/treasury");
      return response.data;
    });
  },

  getTreasuryHistory: async (period = "30d"): Promise<TreasurySnapshot[]> => {
    return dedup(`treasury/history/${period}`, async () => {
      const response = await api.get("/treasury/history", {
        params: { period },
      });
      return response.data;
    });
  },

  getPlatformStats: async (): Promise<PlatformStats> => {
    return dedup("platform/stats", async () => {
      const response = await api.get("/platform/stats");
      return response.data;
    });
  },

  getPlatformHistory: async (period = "30d"): Promise<PlatformSnapshot[]> => {
    return dedup(`platform/history/${period}`, async () => {
      const response = await api.get("/platform/history", {
        params: { period },
      });
      return response.data;
    });
  },
};

export const agentArenaApi = {
  getSnapshot: async (): Promise<AgentArenaSnapshotResponse> => {
    const response = await api.get("/agent-arena/snapshot");
    return response.data;
  },
};

export default api;
