# OpenFi Server

AI Agent 交易平台后端，基于 Go + Echo + SQLite，集成 Hyperliquid L1 交易和 EVM 智能合约。

## 目录结构

```
openfi-server/
├── src/                    # Go 后端源码
│   ├── main.go             # 入口：配置加载、服务启动、自动同步
│   ├── config.go           # 运行时配置结构
│   ├── models.go           # 数据模型定义
│   ├── store.go            # SQLite schema + 核心存储（用户/邀请码/管理员/设置）
│   ├── store_agents.go     # Agent 账户池 + 快照 + 成交 + Vault 存储
│   ├── store_stats.go      # 仪表盘统计 + 国库快照 + 平台快照 + Agent 绩效
│   ├── agent_service.go    # 数据同步轮次（EVM → Hyperliquid → 快照采集）
│   ├── handlers_admin.go   # 管理后台 API
│   ├── handlers_public.go  # 用户端 API
│   ├── evm_client.go       # Allocator 合约交互（JSON-RPC）
│   ├── hyperliquid.go      # Hyperliquid L1 API 客户端
│   ├── auth.go             # JWT 鉴权中间件
│   ├── x_oauth.go          # X/Twitter OAuth 流程
│   ├── agent_crypto.go     # Agent 私钥加解密（AES-GCM-256）
│   └── static_host.go      # 前端静态资源托管 / 开发代理
├── frontend-www/           # 用户端前端（Vite + React + TypeScript）
├── frontend-admin/         # 管理后台前端（Vite + Vue3 + TypeScript）
├── contracts/              # Solidity 智能合约
│   ├── Allocator.sol       # Allocator 合约（BeaconProxy 模式创建 AgentVault）
│   └── interfaces/         # Hyperliquid EVM 接口
├── config/                 # 配置文件（config.json）
├── dist/                   # 编译输出
├── local_run/              # dev_run.sh 生成的本地运行目录
├── build.sh                # 生产构建脚本
├── dev_run.sh              # 开发一键启动脚本
└── release.sh              # 发布脚本
```

---

## 核心概念与实体关系

### 架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│                         OpenFi 平台                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   用户 (User)                                                    │
│   └── 通过 X/Twitter OAuth 注册登录                               │
│       └── 输入邀请码 (Invite Code)                                │
│           └── 系统分配 Agent 账户 (Agent Account)                  │
│               └── 对应链上 AgentVault 合约 (EVM)                   │
│                   └── 在 Hyperliquid L1 上交易                    │
│                                                                  │
│   后台定时同步 (Sync Round, 默认 60s)                              │
│   ├── Phase 1: 查询 Allocator 合约发现所有 Vault                   │
│   ├── Phase 2: 查询 Hyperliquid L1 获取交易数据                    │
│   ├── Phase 2.5: 采集国库快照 (Treasury Snapshot)                  │
│   ├── Phase 2.6: 采集平台快照 (Platform Snapshot)                  │
│   └── Phase 3: 一致性校验                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 实体关系

#### 1. 用户 (User)

通过 X/Twitter OAuth 自动创建，每个用户有一个 X 平台身份。

| 字段 | 说明 |
|------|------|
| `xId` / `xUsername` | X 平台身份标识 |
| `inviteCodeUsed` | 使用的邀请码 |
| `agentPublicKey` | 分配到的 Agent 账户公钥（Hyperliquid 地址） |
| `agentAssignedAt` | Agent 分配时间 |

#### 2. 邀请码 (Invite Code)

一次性使用的准入凭证，由管理员批量生成。

- 一个邀请码只能被一个用户使用
- 使用邀请码后系统自动分配一个可用的 Agent 账户
- 后端记录每个码被哪个用户在什么时间使用

#### 3. Agent 账户 (Agent Account)

Hyperliquid L1 交易账户的密钥对，由管理员导入。

| 状态 | 说明 |
|------|------|
| `unused` | 池中待分配 |
| `assigned` | 已分配给某用户，一对一绑定 |

- 管理员通过加密 payload 批量导入私钥
- 私钥使用 AES-GCM-256 加密存储，固定密钥配置在 `agentPool.fixedKey`
- 公钥（Hyperliquid 地址）自动从私钥派生
- 管理员可撤销分配、重新分配给其他用户

#### 4. AgentVault 合约 (链上)

Allocator 合约为每个 Agent 创建的 EVM 链上资金容器（BeaconProxy 模式）。

```
Allocator 合约
├── 创建 AgentVault (每个 Agent 一个)
├── 管理资金分配 (initialCapital)
└── 跟踪 vault 有效性 (valid)

AgentVault
├── vaultAddress: EVM 合约地址
├── userAddress: Agent 公钥（Hyperliquid 地址）
├── evmBalance: 链上 USDC 余额
├── initialCapital: 初始资金
└── valid: 是否有效
```

#### 5. Hyperliquid L1 账户

每个 AgentVault 的 `vaultAddress` 在 Hyperliquid L1 上有对应交易账户：

- **Perps 账户价值** (`accountValue`): 永续合约账户总值
- **Spot 余额** (`spotBalance`): 现货 USDC 余额
- **持仓** (`positions`): 当前开仓头寸
- **成交** (`fills`): 历史交易记录

### 完整关系链

```
邀请码 ──(1:1)──▶ 用户 ──(1:1)──▶ Agent 账户 ──(1:1)──▶ AgentVault(EVM) ──(1:1)──▶ Hyperliquid L1
   │                │                  │                     │                          │
   │                │                  │                     │                          │
 管理员生成       X OAuth 注册       密钥池分配           合约自动发现             同步获取交易数据
```

---

## 数据同步 (Sync Round)

后台定时执行（默认 60 秒一轮），同步链上和 L1 数据：

### Phase 1: EVM 合约发现

1. 调用 `Allocator.vaultCount()` 获取 Vault 总数
2. 批量获取 Vault 地址：`getVaultsByRange(start, count)`
3. 批量获取 Vault 信息：`getVaultsInfo(addresses[])`
4. 写入 `agent_vaults` 表

### Phase 2: Hyperliquid L1 数据同步

对每个有活跃用户的 Vault（并发，默认 5 worker）：

1. `FetchAccountData(vaultAddress)` → Perps 账户价值 + 未实现盈亏
2. `FetchSpotBalance(vaultAddress)` → Spot USDC 余额
3. 更新 `agent_vaults` 表
4. 保存 `agent_snapshots`（`accountValue` = L1 Perps 账户价值）
5. 同步失败时记录 `sync_status=error` + 错误原因

### Phase 2.5: 国库快照

汇总三个资金来源 × 三种资金类型：

| | EVM | Perps | Spot |
|---|---|---|---|
| **AgentVaults** | 所有 vault 的链上 USDC | L1 Perps 账户价值 | L1 Spot 余额 |
| **Allocator** | Allocator 合约 USDC | Allocator L1 Perps | Allocator L1 Spot |
| **Owner** | Owner 地址 USDC | Owner L1 Perps | Owner L1 Spot |

**系统总资金** = 以上 9 个值的总和

### Phase 2.6: 平台快照

记录平台级 KPI：总资产(TVL)、总盈亏、用户数、活跃 Agent 数、交易总数

### Phase 3: 一致性校验

检测已分配的 Agent 是否仍在链上 Vault 中有效，发现异常时输出警告日志。

---

## 统计数据说明

### 仪表盘 (Dashboard)

| 指标 | 数据来源 | 含义 |
|------|---------|------|
| **总资产** | `treasury_snapshots.totalFunds` | 三来源 × 三类型 = 9 个余额之和 |
| **EVM / Perps / Spot 总额** | 国库快照各类型汇总 | 按资金类型跨三来源合计 |
| **总盈亏** | `treasury_snapshots.vaultPnl` | 所有 Vault 的未实现盈亏总和 |
| **总用户数** | `COUNT(users)` | 注册用户总数 |
| **Agent 总量 / 已分配 / 未使用** | `agent_accounts` 按 status 计数 | 账户池分配状态 |
| **今日/本周新增** | `users WHERE created_at >= ?` | 时间窗口内新注册用户 |
| **Agent 转化率** | 已分配 / 总用户 | 用户到活跃交易者的转化 |
| **邀请码已使用率** | 已使用码数 / 总码数 | 码的消耗效率 |
| **同步轮次 / 最后同步 / 数据新鲜度** | settings + platform_snapshots | 系统运行健康状态 |

### 国库明细

按来源（AgentVaults / Allocator / Owner）分别展示 EVM / Perps / Spot 三种余额和小计。

### Agent 绩效 (Performance)

| 指标 | 计算方式 |
|------|---------|
| **ROI** | (当前账户价值 - 初始资金) / 初始资金 |
| **胜率** | agent_fills 中 closedPnl > 0 的比例 |
| **最大回撤** | agent_snapshots 历史峰值到谷值的最大跌幅 |
| **Sharpe Ratio** | 日收益率均值 / 标准差 × √365 |
| **交易频率** | 总成交数 / 运行天数 |
| **总已实现盈亏** | agent_fills 中 closedPnl 的总和 |

### Agent 快照 (Snapshots)

每轮同步为每个活跃 Agent 记录一条快照：

- `accountValue` = L1 Perps 账户价值（用于 TVL 口径）
- `unrealizedPnl` = Perps 未实现盈亏
- 用于用户端图表（`/api/user/agent/history`）和绩效计算

### 平台快照 (Platform Snapshots)

每轮同步记录平台级 KPI，用于趋势分析（`/admin/api/dashboard/trends`，`/api/platform/history`）。

---

## API 路由

### 用户端 `/api/*`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/auth/x/start` | X OAuth 发起 |
| GET | `/api/auth/x/callback` | X OAuth 回调 |
| GET | `/api/invite-codes/verify` | 验证邀请码有效性 |
| POST | `/api/invite-codes/consume` | 使用邀请码 + 分配 Agent |
| GET | `/api/user/me` | 当前用户信息 |
| GET | `/api/user/agent/history` | Agent 历史快照（图表数据） |
| GET | `/api/user/agent/stats` | Agent 当前统计 |
| GET | `/api/agent-market` | Agent 市场列表 |
| GET | `/api/agent-market/:publicKey` | Agent 详情 + 绩效 |
| GET | `/api/vault/stats` | Vault 统计汇总 |
| GET | `/api/vault/overview` | Vault 持仓概览 |
| GET | `/api/treasury` | 当前国库快照 |
| GET | `/api/treasury/history` | 国库历史 |
| GET | `/api/platform/stats` | 平台概览（总资产/用户数/Agent数/增长率） |
| GET | `/api/platform/history` | 平台快照历史（趋势图数据） |
| GET | `/api/daily-slots` | 每日配额状态 |

### 管理后台 `/admin/api/*`

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/admin/api/login` | 管理员登录 |
| GET | `/admin/api/dashboard` | 增强仪表盘（金融+统计+增长+健康） |
| GET | `/admin/api/dashboard/trends` | 平台趋势 + 增长率 |
| GET | `/admin/api/agents/:key/performance` | 单 Agent 完整绩效 |
| GET | `/admin/api/agents/leaderboard` | Agent 排行榜 |
| GET | `/admin/api/users` | 用户列表 |
| GET | `/admin/api/invite-codes` | 邀请码列表（含使用者信息） |
| GET | `/admin/api/agent-accounts` | Agent 账户池 |
| POST | `/admin/api/agent-accounts/import` | 导入加密密钥 |
| GET | `/admin/api/agent-vaults` | 链上 Vault 列表（含同步状态） |
| GET | `/admin/api/treasury` | 国库快照 |
| GET | `/admin/api/treasury/history` | 国库历史 |
| `*/settings/*` | 同步/OAuth/合约/配额/调度 | 各项系统设置 |

---

## 构建与运行

### 构建

```bash
./build.sh
```

输出：`dist/openfi-server`

### 开发运行

```bash
./dev_run.sh
```

自动编译 → 复制到 `local_run/` → 使用开发配置启动，前端请求代理到 Vite 开发服务器。

### 直接运行

```bash
go run ./src -config ./config/config.json
```

### 配置文件

所有参数从 `config/*.json` 读取，不需要环境变量。

| 配置项 | 说明 |
|--------|------|
| `appBaseUrl` | 应用根 URL（用于 OAuth 回调等） |
| `server.port` | 监听端口（默认 9333） |
| `server.tokenSecret` | JWT 签名密钥 |
| `storage.dbPath` | SQLite 数据库路径 |
| `agentPool.fixedKey` | Agent 私钥加密密钥（32 字节 UTF-8 或 64 字符 hex） |
| `hyperliquid.baseURL` | Hyperliquid API 地址 |
| `frontend.mode` | `release`（静态文件）或 `dev`（代理到 Vite） |

首次启动时，X OAuth、合约地址、同步间隔等配置从配置文件迁移到数据库 settings 表，之后通过管理后台修改。

---

## 路由策略

```
/admin/api/*    → 管理后台 API（需要管理员 JWT）
/admin/*        → 管理后台前端
/api/*          → 用户端 API
/*              → 用户端前端
```

前端托管模式：
- **release**: 从编译后的 dist 目录提供静态文件
- **dev**: 反向代理到 Vite 开发服务器（支持 HMR）

---

## Agent 密钥导入

`POST /admin/api/agent-accounts/import`

```json
{
  "encryptedPayload": "{\"status\":\"ok\",\"format\":\"AES-GCM-256\",\"encrypted_data\":\"<hex>\",\"count\":12}"
}
```

不接受明文私钥。导入后私钥以 AES-GCM-256 加密存储在数据库中。

---

## 管理员初始化

首次启动时，如果 `admins` 表为空，自动创建默认管理员并在日志中输出临时密码：

```
bootstrap admin created: email=admin@openfi.local temporary_password=xxx (please change immediately)
```
