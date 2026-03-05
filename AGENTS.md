# OpenFi Project Summary

## 1. 项目定位
- 这是一个前后端一体化项目：
  - `frontend-www`：用户端（Vite + React）
  - `frontend-admin`：管理端（Vite + Vue3）
  - `server`：后端（Go + Echo + SQLite）
- 服务端统一托管 `www/admin` 静态资源并提供 API，避免跨域。

## 2. 目录结构
- `frontend-www/`：用户端源码
- `frontend-admin/`：管理端源码
- `server/src/`：后端源码（主代码目录）
- `server/config/`：运行配置（JSON）
- `server/dist/`：编译产物
- `server/local_run/`：`dev_run.sh` 一键运行目录

## 3. 当前技术栈
- 后端：Go + Echo + SQLite
- 管理端：Vite + Vue3 + TypeScript
- 用户端：Vite + React + TypeScript
- 鉴权：自定义 token（HMAC-SHA256）+ X OAuth（服务端发起）

## 4. 关键业务流程

### 4.1 邀请码与用户绑定
1. 管理端创建邀请码（单个或批量）。
2. 新用户通过 X 登录后必须填写邀请码。
3. 后端校验邀请码并消耗一次使用次数。

### 4.2 Agent 账号池导入与分配
1. 管理端粘贴加密 JSON 到导入界面。
2. 后端使用配置中的 `agentPool.fixedKey` 做 AES-256-GCM 解密。
3. 解密得到私钥数组，派生公钥并入库到 agent 账号池。
4. 用户邀请码验证成功后，从账号池分配一个 `unused` 账号给该用户。
5. 用户端市场与个人中心按分配到的 agent 公钥查询更新。
### 4.3 数据
1. Agent 市场和首页的数据来自 agent 账户公钥的统计数据。Hyperliquid L1 接口提供 accountValue/unrealizedPnl，EVM 链上读取 vault 地址和 USDC 余额。历史数据由服务端定时同步并存储到 agent_snapshots 表。
2. 每个 agent 的 TVL = L1 accountValue + EVM USDC 余额。总 TVL 为所有 agent 的 TVL 之和。
3. Vault 页面（`/vault`）展示聚合 TVL 统计和 agent 列表，首页 Stats Grid 使用真实 TVL 数据。
4. Agent 支持 name/description/category 字段，通过管理后台编辑。
5. Agent 分级（agentLevel）和用户评分评论功能已暂时移除。

### 4.3.1 前端 UI 已有但后端数据暂未接入的功能点
- 回测指标 (sharpeRatio, maxDrawdown)：详情页显示为 "-"
- 性能指标详情 (profit factor, avg trade, win rate)：Metrics tab 显示为 "-"
- Agent 参数设置 (stopLoss, takeProfit, maxPosition)：使用默认值
- LP 相关数据 (lpShares, lpValue)：用户端使用默认值
- APY / Yield 数据：首页仍使用静态值（无历史数据计算）
## 5. 智能合约集成

- **Allocator 合约**: `0x0CAE2ceD373970211b5f3c7cAbc42b38e5040711` (Hyperliquid Testnet EVM)
- Agent 公钥对应链上 AgentVault（BeaconProxy），由 Allocator 管理
- 服务端通过 EVM JSON-RPC 读取 vault 地址、USDC 余额和 Agent 状态
- 配置: `contracts.rpcURL` + `contracts.allocatorAddress`
- 数据流: agent 公钥 → `Allocator.getAgentsInfo(address[])` → 批量返回状态和余额 → `Allocator.agentVaults()` → vault 地址
- Agent 状态枚举: `Inactive`(0) / `Active`(1) / `Revoked`(2)，从合约 `AgentVault.Status` 同步
- **统计数据（TVL 等）只计算 `active` 状态的 agent**
- L1 数据 (accountValue) + EVM 余额 = 单个 agent TVL，总 TVL 为所有活跃 agent TVL 之和
- API 端点: `GET /api/vault/stats` 返回 `{ totalTvl, totalEvmBalance, totalL1Value, agentCount }`（仅活跃 agent）

## 6. Agent 导入协议
- Admin API: `POST /admin/api/agent-accounts/import`
- 请求体：
```json
{
  "encryptedPayload": "{\"status\":\"ok\",\"format\":\"AES-GCM-256\",\"encrypted_data\":\"...\",\"count\":12}"
}
```
- `encrypted_data` 支持 hex/base64（内容为 `nonce(12B)+ciphertext+tag`）。

## 7. 路由约定（统一入口）
- `/admin/api/*`：管理端后端接口
- `/admin/*`：管理端静态资源
- `/api/*`：用户端后端接口
- `/*`：用户端静态资源

## 8. 静态资源模式
- `frontend.mode = "release"`：读取 `frontend-www/dist` 与 `frontend-admin/dist`
- `frontend.mode = "dev"`：反向代理到 Vite dev server
  - www -> `frontend.dev.wwwDevServer`
  - admin -> `frontend.dev.adminDevServer`

## 9. 端口与基础地址（当前默认）
- Server: `9333`
- WWW dev: `9334`
- Admin dev: `9335`
- `appBaseUrl`: `http://localhost:9333`

## 10. 配置规范
- 不使用环境变量，统一使用 `server/config/*.json`。
- 关键配置：
  - `appBaseUrl`
  - `server.port`, `server.tokenSecret`
  - `storage.dbPath`
  - `agentPool.fixedKey`（长度必须 32）
  - `hyperliquid.baseURL`
  - `contracts.rpcURL`, `contracts.allocatorAddress`
  - `xOAuth.clientId/clientSecret/scopes/...`
  - `frontend.mode/release/dev`

## 11. X OAuth 回调策略
- 回调 URL 由 `appBaseUrl` 自动生成：
  - 授权回调：`<appBaseUrl>/api/auth/x/callback`
  - 前端成功/失败回跳：`<appBaseUrl>/auth/x/callback`
- X 平台配置回调时应填写：`<appBaseUrl>/api/auth/x/callback`

## 12. 常用命令
- 后端编译：`cd server && ./build.sh`
- 后端一键开发运行：`cd server && ./dev_run.sh`
- 用户端开发：`cd frontend-www && npm run dev`
- 管理端开发：`cd frontend-admin && npm run dev`
- 后端测试：`cd server && go test ./src`
