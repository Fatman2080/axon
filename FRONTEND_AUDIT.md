# 用户端前端数值与功能审查清单

> 审查时间：2026-03-06  
> 审查范围：`frontend-www` 所有用户端页面（不含管理端）

---

## 🔴 严重问题（建议上线前修复）

---

### BUG-01 | Profile 页财务数值全部为 0

| 项目                  | 详情                                                                                                                                                                                                                                                                                               |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/profile`                                                                                                                                                                                                                                                                                         |
| **对应前端字段/功能** | Summary Cards 区域：`Total Equity`、`Unrealized PnL`、`Vault Shares` 四个卡片；Portfolio 区域大数字 `$totalInvestment + totalProfit` 及百分比涨跌                                                                                                                                                  |
| **对应前端文件**      | `frontend-www/src/pages/Profile.tsx` L96–L101, L142–L146, L246–L251                                                                                                                                                                                                                                |
| **问题描述**          | 前端读取 `user.totalInvestment`、`user.totalProfit`、`user.lpShares`、`user.agentCount` 来渲染这四个卡片。但后端 `/api/user/me` 返回的 `User` struct（`src/models.go`）**完全没有这些字段**，返回的只有 `id, name, avatar, agentPublicKey` 等身份信息，导致四个值永远为 `0` 或 `undefined`。       |
| **修正方案**          | 在 `src/handlers_public.go` 的 `handleGetMe` 中，获取用户基本信息后，额外调用 `store.listAgentSnapshots(user.AgentPublicKey, 120, "ALL")` 取最新/最早账户价值，拼入响应体字段 `accountValue` 和 `totalPnl`；或者前端在 Profile 加载时额外调用已有的 `GET /api/user/agent/stats` 并合并到 UI 状态。 |
| **对应后端文件**      | `src/handlers_public.go` → `handleGetMe`（L257–L267）；`src/handlers_public.go` → `handleUserAgentStats`（L506–L553）可复用                                                                                                                                                                        |

---

### [暂时未展示] BUG-02 | StrategyDetail Sharpe Ratio / Max Drawdown 永远显示「-」

| 项目                  | 详情                                                                                                                                                                                                                                                                           |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                                              |
| **对应前端字段/功能** | Overview 卡片底部三列指标区：`Sharpe`、`Drawdown`；Metrics Tab 下不显示（因为用的是 fills 计算）                                                                                                                                                                               |
| **对应前端文件**      | `frontend-www/src/pages/StrategyDetail.tsx` L199–L200；`frontend-www/src/store/slices/strategySlice.ts` → `mapAgentToStrategy`（L27–L61）                                                                                                                                      |
| **问题描述**          | `StrategyDetail` 从 Redux 取 `strategy.backtestMetrics?.sharpeRatio` 和 `strategy.backtestMetrics?.maxDrawdown`。但 `mapAgentToStrategy()` 函数在将后端 `AgentMarketItem` 映射为前端 `Strategy` 时，**未填充 `backtestMetrics` 字段**，导致它始终为 `undefined`，UI 显示 `-`。 |
| **修正方案**          | 方案A（推荐）：在 `mapAgentToStrategy` 中，根据已有的 `detail.history` 数据在前端实时计算 Sharpe（参考 `Agents.tsx` L71–L84 的算法），填入 `backtestMetrics`。方案B：后端 `handleAgentMarketDetail` 响应体中增加计算好的统计数据。                                             |
| **对应开发文件**      | `frontend-www/src/store/slices/strategySlice.ts` → `mapAgentToStrategy` 及 `fetchStrategyById`（L71–L93）                                                                                                                                                                      |

---

### [暂时未展示] BUG-03 | Vault Address 链接指向测试网浏览器

| 项目                  | 详情                                                                                                                                       |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| **页面**              | `/strategies/:id`                                                                                                                          |
| **对应前端字段/功能** | Overview Tab → Vault Address 区块（仅当 `strategy.vaultAddress` 有值时显示）                                                               |
| **对应前端文件**      | `frontend-www/src/pages/StrategyDetail.tsx` L254–L263                                                                                      |
| **问题描述**          | 链接硬编码为 `https://testnet.purrsec.com/address/...`，主网上线后需切换为正式区块浏览器地址，否则用户点击后看到的是测试网数据（或 404）。 |
| **修正方案**          | 将硬编码域名改为环境变量或配置常量，例如 `VITE_EXPLORER_URL`，在 `vite.config.ts` / `.env` 文件中区分测试/生产环境。                       |
| **对应开发文件**      | `frontend-www/src/pages/StrategyDetail.tsx` L255；`frontend-www/vite.config.ts`；需新增 `.env.production`                                  |

---

## ⚠️ 中等问题（影响数值准确性或用户理解）

---

### [暂时未展示] BUG-04 | Strategies 卡片字段标签「APR」与实际值不符

| 项目                  | 详情                                                                                                                                                                                                                                              |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies`（策略列表）                                                                                                                                                                                                                         |
| **对应前端字段/功能** | 每张策略卡片第一行数值：标签显示 `APR`，值显示如 `+12.50`                                                                                                                                                                                         |
| **对应前端文件**      | `frontend-www/src/pages/Strategies.tsx` L134–L143；翻译键 `strategies.card.apr`，位于 `frontend-www/src/i18n/index.ts`                                                                                                                            |
| **问题描述**          | 标签写的是 `APR`（年化利率/百分比），但实际渲染的是 `strategy.pnlContribution`，这个字段是**绝对金额 PnL**（从第一个快照到最新快照的价值差，单位美元），例如显示 `+12.50` 但没有 `$` 或 `%`，含义完全模糊。APR 意味着用户会以为这是百分比收益率。 |
| **修正方案**          | 将标签改为 `PnL` 或 `Total Return`，同时在数值前加 `$` 符号；或者将后端提供的 PnL 换算为相对上一期 TVL 的百分比后再显示，并恢复 `APR` 标签。                                                                                                      |
| **对应开发文件**      | `frontend-www/src/pages/Strategies.tsx` L135, L141；`frontend-www/src/i18n/index.ts`（`strategies.card.apr` 键）                                                                                                                                  |

---

### BUG-05 | Agents 页（Vault）APY 是累计收益率而非年化

| 项目                  | 详情                                                                                                                                                                                                             |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/agents`（Vault 总览页）                                                                                                                                                                                        |
| **对应前端字段/功能** | 右上角 Header 区域 `APY` 数值                                                                                                                                                                                    |
| **对应前端文件**      | `frontend-www/src/pages/Agents.tsx` L86、L212–L214                                                                                                                                                               |
| **问题描述**          | 当前公式：`apy = (totalPnl / (totalTvl - totalPnl)) * 100`。这是**累计收益率**（ROI），非年化（APY/APR）。Season One 初期运行天数很短时，若有利润则这个数字会很大（日化放大到100%+），反而误导用户认为年化很高。 |
| **修正方案**          | 如果确实是累计收益率，改标签为 `Total ROI` 或 `Return`；若需要真正年化，需从后端获取运行天数：`apy = (totalPnl / principal) / runningDays * 365 * 100`。                                                         |
| **对应开发文件**      | `frontend-www/src/pages/Agents.tsx` L86                                                                                                                                                                          |

---

### BUG-06 | Agents 页 Share Price 无数据时显示虚构值

| 项目                  | 详情                                                                                                                                                                    |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/agents`                                                                                                                                                               |
| **对应前端字段/功能** | 图表区域左上角 `SHARE PRICE` 大数字（如 `$1.0020`）                                                                                                                     |
| **对应前端文件**      | `frontend-www/src/pages/Agents.tsx` L67–L69                                                                                                                             |
| **问题描述**          | 当用户没有绑定 agent 或 agent 没有历史快照时，Share Price 回退值为 `1 + totalTvl / 1_000_000 * 0.2`，这是一个**完全虚构的估算值**，没有任何实际依据，会对用户产生误导。 |
| **修正方案**          | 无数据时显示 `-` 或 `N/A`，与图表区「NO_DATA — awaiting agent telemetry」保持一致。                                                                                     |
| **对应开发文件**      | `frontend-www/src/pages/Agents.tsx` L67–L69                                                                                                                             |

---

### [已修复/暂时未展示] BUG-07 | Strategies TVL 进度条 maxTvl 硬编码 10,000,000

| 项目                  | 详情                                                                                                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies`                                                                                                                                                             |
| **对应前端字段/功能** | 每张策略卡片 TVL 下方的进度条                                                                                                                                             |
| **对应前端文件**      | `frontend-www/src/pages/Strategies.tsx` L159；`frontend-www/src/store/slices/strategySlice.ts` L42                                                                        |
| **问题描述**          | `maxTvl` 在 `mapAgentToStrategy` 中硬编码为 `10_000_000`（1000万美元），初期每个 agent 资金仅约 100 美元（Season One 分配资金），进度条永远接近 0，毫无意义且显得像 bug。 |
| **修正方案**          | 方案A：从后端返回实际最大管理资金上限（可加到 `agent_accounts` 表字段或配置里）；方案B：暂时隐藏该进度条，或将 `maxTvl` 设为当前 TVL 的若干倍动态计算。                   |
| **对应开发文件**      | `frontend-www/src/store/slices/strategySlice.ts` L42；`src/models.go` 可新增字段                                                                                          |

---

## ℹ️ 轻微问题（体验优化建议）

---

### [部分暂时未展示] BUG-08 | 图表 X 轴标签为「Day 1, Day 2」无时间戳

| 项目                  | 详情                                                                                                                                               |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/agents`、`/strategies/:id`、`/profile`                                                                                                           |
| **对应前端字段/功能** | 所有折线图的 X 轴 Tooltip 时间标签                                                                                                                 |
| **对应前端文件**      | `Agents.tsx` L90；`StrategyDetail.tsx` L39；`Profile.tsx` L106                                                                                     |
| **问题描述**          | 图表数据标签用 `Day ${i+1}` 或 `Point ${i+1}`，tooltip 悬浮后看不到任何时间信息，用户无法判断数据对应的具体日期。                                  |
| **修正方案**          | 后端 `listAgentSnapshots` 已返回 `createdAt` 字段，可将其一并传给前端；前端将 `history` 改为对象数组 `{value, time}`，X 轴用格式化后的日期字符串。 |
| **对应开发文件**      | `src/handlers_public.go` → `handleAgentMarketDetail`（L342–L394）；三个前端图表文件                                                                |

---

### BUG-09 | Home 页 APY 显示「--」（占位符）

| 项目                  | 详情                                                                            |
| --------------------- | ------------------------------------------------------------------------------- |
| **页面**              | `/`（首页）                                                                     |
| **对应前端字段/功能** | Stats Grid 第三个卡片，标签为「Total Trading Volume」或「AVG APY」              |
| **对应前端文件**      | `frontend-www/src/pages/Home.tsx` L227                                          |
| **问题描述**          | 硬编码 `value="--"`，功能未实现，但展示出来会让用户认为是数据加载失败。         |
| **修正方案**          | 如果该指标暂不计划实现，建议隐藏该卡片，或将文字改为「Coming Soon」以明示意图。 |
| **对应开发文件**      | `frontend-www/src/pages/Home.tsx` L227                                          |

---

### BUG-10 | SubmitAgent CLI 命令未检测包是否发布

| 项目                  | 详情                                                                                                                  |
| --------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/submit-agent`                                                                                                       |
| **对应前端字段/功能** | 页面下半部分的 Terminal 命令框：`npx clawfi-cli deploy --key ...`                                                     |
| **对应前端文件**      | `frontend-www/src/pages/SubmitAgent.tsx` L42–L44                                                                      |
| **问题描述**          | `clawfi-cli` npm 包如果尚未发布，用户执行该命令会直接报错 `npm ERR! code E404`，体验极差。                            |
| **修正方案**          | 上线前确认 `clawfi-cli` 已发布到 npm；或在命令下方加上注释「CLI coming soon」，并将命令框设为不可复制状态直到包发布。 |
| **对应开发文件**      | `frontend-www/src/pages/SubmitAgent.tsx` L42–L44, L208                                                                |

---

## 优先级总结

| 优先级 | ID     | 问题                                    | 工作量估计                           |
| ------ | ------ | --------------------------------------- | ------------------------------------ |
| 🔴 P0  | BUG-01 | Profile 财务数值全为 0                  | 中（后端加字段 or 前端多调一个接口） |
| 🔴 P0  | BUG-02 | StrategyDetail Sharpe/Drawdown 永远 `-` | 小（前端填充逻辑）                   |
| 🔴 P0  | BUG-03 | 测试网链接                              | 极小（替换域名+环境变量）            |
| ⚠️ P1  | BUG-04 | APR 标签名与值不符                      | 极小（改标签文字）                   |
| ⚠️ P1  | BUG-05 | APY 是累计非年化                        | 小（改标签或改公式）                 |
| ⚠️ P1  | BUG-06 | Share Price 虚构回退值                  | 极小（改为显示 `-`）                 |
| ⚠️ P1  | BUG-07 | TVL 进度条 maxTvl 无意义                | 小（隐藏或动态值）                   |
| ℹ️ P2  | BUG-08 | 图表无时间轴                            | 中（需前后端配合）                   |
| ℹ️ P2  | BUG-09 | APY 占位符 `--`                         | 极小（隐藏卡片）                     |
| ℹ️ P2  | BUG-10 | CLI 包未发布                            | 依赖外部任务                         |

---

## 第二部分：空壳功能 / 幽灵 UI 清单

> 这些功能有翻译文案、有界面占位、有翻译键，但**完全没有对应的后端 API 或前端逻辑支撑**。
> 策略：上线前必须从页面中移除，或明确标注「Coming Soon」，否则影响产品可信度。

---

### [已清理/暂时未展示] GHOST-01 | StrategyDetail「为 Agent 投票」功能

| 项目                  | 详情                                                                                                                                                                                                                                                            |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                               |
| **对应前端字段/功能** | 右侧边栏（当前是 Risk Card + Hiring Callout），翻译键 `strategyDetail.vote.*` 已存在                                                                                                                                                                            |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L229–L237（英文）、L541–L548（中文）；**StrategyDetail.tsx 中 JSX 完全无该卡片**                                                                                                                                        |
| **问题描述**          | 翻译文件完整定义了「Vote for Manager」卡片的所有文案（title、desc、button、perfFee、devShare、pnlContribution），说明原本设计了这个功能，但 `StrategyDetail.tsx` 中**完全没有渲染该卡片**，相关 i18n key 形成了孤立的翻译幽灵。后端也没有任何 vote 相关的 API。 |
| **处置方案**          | 短期：清理 `translations.ts` 中未使用的 `vote.*` 翻译键（或保留为备用）；中期：如果投票功能计划实现，需要后端新增投票接口 + 前端新增 Vote Card 组件。                                                                                                           |
| **对应开发文件**      | `frontend-www/src/i18n/translations.ts` L229–L237；`frontend-www/src/pages/StrategyDetail.tsx`（需新增 Vote 卡片）                                                                                                                                              |

---

### [暂时未展示] GHOST-02 | StrategyDetail「Followers / 粉丝数」数据

| 项目                  | 详情                                                                                                                                                                                                                                                                  |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                                     |
| **对应前端字段/功能** | 页面顶部 Header Card 中 strategy 名称旁的统计数字（`userCount`/`followers`）                                                                                                                                                                                          |
| **对应前端文件**      | `frontend-www/src/store/slices/strategySlice.ts` L40（`userCount: 0` 硬编码）                                                                                                                                                                                         |
| **问题描述**          | `Strategy` 类型有 `userCount`（支持者数量）字段，但在 `mapAgentToStrategy` 中永远赋值为 `0`，因为后端 `AgentMarketItem` 根本不返回该字段，后端也无 followers 表或统计逻辑。当前页面 JSX 中没有展示该字段，但如果 UI 后续加上「X followers」之类的显示，它会永远为 0。 |
| **处置方案**          | 确认 Followers 功能是否上线计划。如无：保持现状；如有：后端需要在 `agent_accounts` 表或独立 `agent_followers` 表中统计并在 API 中返回。                                                                                                                               |
| **对应开发文件**      | `frontend-www/src/store/slices/strategySlice.ts` L40；`src/models.go`（AgentMarketItem 需新增 followerCount 字段）                                                                                                                                                    |

---

### GHOST-03 | Profile「邀请赚佣」功能

| 项目                  | 详情                                                                                                                                                                                                                                                    |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/profile`                                                                                                                                                                                                                                              |
| **对应前端字段/功能** | 个人中心：翻译键 `profile.inviteEarn`（邀请赚佣）、`profile.inviteDesc`（推荐开发者或 LP 加入 ClawFi，永久赚取其平台费用的 10%）、`profile.copy`（复制按钮）                                                                                            |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L287–L289（英文）、L599–L601（中文）；`frontend-www/src/pages/Profile.tsx`（**JSX 中完全找不到这些翻译键的渲染**）                                                                                              |
| **问题描述**          | 翻译文件定义了完整的「邀请赚佣」模块（标题、描述、复制按钮），但 `Profile.tsx` 页面 JSX 代码中**完全没有渲染这个区块**。同时后端也没有：邀请码分佣记录、邀请链接生成、佣金计算任何相关逻辑。功能从策划到 i18n 定义了，但既没有前端 UI，也没有后端支持。 |
| **处置方案**          | 短期：清理翻译文件中未使用的键（或标注 TODO）；中期：如果分佣功能上线，需要 (1) 后端新增 referral 表和分佣统计 API，(2) 前端在 Profile 页新增邀请码展示 + 分佣记录卡片。                                                                                |
| **对应开发文件**      | `frontend-www/src/i18n/translations.ts` L287–L289；`frontend-www/src/pages/Profile.tsx`（需新增 Invite & Earn 区块）；`src/store.go`（需新增 referral 相关表和逻辑）                                                                                    |

---

### GHOST-04 | Profile「View All」按钮无跳转

| 项目                  | 详情                                                                                                                       |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/profile`                                                                                                                 |
| **对应前端字段/功能** | 「My Strategies」列表右上方的「View All」按钮                                                                              |
| **对应前端文件**      | `frontend-www/src/pages/Profile.tsx` L290                                                                                  |
| **问题描述**          | 按钮有文案 `t('profile.viewAll')`，但是一个 `<button>` 元素，**没有 `onClick` 事件，没有路由跳转**。点击完全没有任何效果。 |
| **处置方案**          | 改为 `<Link to="/strategies">` 跳转到 Agent Market 页，或者删除该按钮（1行修复）。                                         |
| **对应开发文件**      | `frontend-www/src/pages/Profile.tsx` L290                                                                                  |

---

### GHOST-05 | Profile「My Strategies」列表匹配逻辑导致已绑定 agent 看不到自己

| 项目                  | 详情                                                                                                                                                                                                                                                                                                                                     |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/profile`                                                                                                                                                                                                                                                                                                                               |
| **对应前端字段/功能** | 「My Strategies」列表：展示登录用户自己的策略                                                                                                                                                                                                                                                                                            |
| **对应前端文件**      | `frontend-www/src/pages/Profile.tsx` L95                                                                                                                                                                                                                                                                                                 |
| **问题描述**          | 过滤逻辑为：`strategies.filter(s => user.agentPublicKey ? s.id === user.agentPublicKey : false)`。这要求 strategy 的 `id` 与用户 `agentPublicKey` 完全一致。但 `fetchStrategies` 只返回 `agentStatus === 'active'` 的 agent，如果用户绑定了公钥但 agent 尚未被管理员激活，列表永远为空，用户会看到「No strategies deployed」的错误印象。 |
| **处置方案**          | 单独为已登录用户调用 `/api/user/agent/stats`（或新建 `/api/user/agent`）来获取其 agent 信息，无论是否上架都展示用户自己的 agent 基本信息。                                                                                                                                                                                               |
| **对应开发文件**      | `frontend-www/src/pages/Profile.tsx` L95；`src/handlers_public.go`（`handleUserAgentStats` 可扩展为返回完整 agent 信息）                                                                                                                                                                                                                 |

---

### GHOST-06 | Vault 页残留无用翻译键

| 项目                  | 详情                                                                                                                                                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/vault`                                                                                                                                                                                                                  |
| **对应前端字段/功能** | i18n 中定义了 `vault.depositors`、`vault.managerFee`、`vault.tabs.depositors`、`vault.tabs.trades`、`vault.myPosition.*`、`vault.actions.*`、`vault.riskControls.*`、`vault.assets`、`vault.table.*`                      |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L125–L182（英文）、L437–L494（中文）；`frontend-www/src/pages/Agents.tsx`（当前 JSX 中已**不存在**这些 tab 和功能）                                                               |
| **问题描述**          | 根据上一次会话记录，Vault 页的 Deposit/Withdraw 表单、Depositors Tab、Manager Fee 已从页面 JSX 中移除。但翻译文件中这些文案字符串全部残留，形成大量无用翻译键，增加维护负担。                                             |
| **处置方案**          | 清理 `translations.ts` 中所有 `vault.tabs.*`（tabs.trades、tabs.depositors）、`vault.myPosition.*`、`vault.actions.*`、`vault.riskControls.*`、`vault.depositors`、`vault.managerFee`、`vault.table.*` 等无使用的翻译键。 |
| **对应开发文件**      | `frontend-www/src/i18n/translations.ts`（英文 L125–L182，中文 L437–L494）；`frontend-www/src/pages/Agents.tsx`                                                                                                            |

---

### GHOST-07 | Home 页「Career Progress 职级晋升路线图」有 i18n 无 UI

| 项目                  | 详情                                                                                                                                                                                   |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/`（首页）                                                                                                                                                                            |
| **对应前端字段/功能** | 翻译键 `home.career.*`（The Path to Partner / 合伙人晋升之路，包含 intern/analyst/manager/partner 四个职级的详细描述）                                                                 |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L62–L94（英文）、L374–L406（中文）；`frontend-www/src/pages/Home.tsx`（**JSX 中找不到 `home.career` 的任何引用**）                             |
| **问题描述**          | 翻译文件精心定义了四个职级（实习生 $100 → 分析师 $50k → 基金经理 $500k → 合伙人 $5M）的完整文案，但首页 `Home.tsx` 中完全没有渲染这段内容，所有 `home.career.*` 翻译键都是孤立幽灵键。 |
| **处置方案**          | 方案A：在首页补充渲染 Career Path 图表/卡片（产品层面决定）；方案B：如暂不展示，清理对应翻译键避免积累冗余。                                                                           |
| **对应开发文件**      | `frontend-www/src/i18n/translations.ts` L62–L94；`frontend-www/src/pages/Home.tsx`（可新增 Career Section）                                                                            |

---

### [暂时未展示] GHOST-08 | StrategyDetail「Trade History」Tab 未实现

| 项目                  | 详情                                                                                                                                                                                                                                                             |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                                |
| **对应前端字段/功能** | Tabs 栏的第三个 Tab：`strategyDetail.tabs.history`（交易记录 / Trade History）                                                                                                                                                                                   |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L246（英文）、L558（中文）；`frontend-www/src/pages/StrategyDetail.tsx` L217                                                                                                                                             |
| **问题描述**          | 翻译文件定义了 `tabs.history` 键，但 StrategyDetail.tsx 中的 Tab 渲染只有 `['overview', 'metrics']` 两个，**'history' tab 既没有 button 也没有对应内容区**。后端虽然返回了 `recentFills`（最近50条成交记录），但前端完全没有渲染这张表。数据有，但没有 UI 展示。 |
| **处置方案**          | 在 tabs 数组中增加 `'history'` tab，并参照 `VaultFill` 类型（coin、side、size、price、time、fee、closedPnl）渲染一张成交流水表格；或移除该翻译键。                                                                                                               |
| **对应开发文件**      | `frontend-www/src/pages/StrategyDetail.tsx` L217 的 tabs 数组；`frontend-www/src/types/index.ts` 中 `VaultFill`（L188–L199）                                                                                                                                     |

---

### [暂时未展示] GHOST-09 | StrategyDetail「Career Level / 职级」字段后端有数据前端无展示

| 项目                  | 详情                                                                                                                                                                                                                                                                                        |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                                                           |
| **对应前端字段/功能** | 翻译键 `strategyDetail.careerProgress`、`strategyDetail.currentLevel`、`strategyDetail.aum`、`strategyDetail.manageDesc`                                                                                                                                                                    |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L223–L226；`frontend-www/src/pages/StrategyDetail.tsx`（**JSX 中找不到这些键的渲染**）                                                                                                                                                              |
| **问题描述**          | 翻译文件定义了「职业进度」「当前等级」「资产管理规模」「管理描述」等文案，但 `StrategyDetail.tsx` 页面 JSX 代码中完全不存在对应渲染。后端在 `agent_accounts` 表有 `agent_level` 字段（intern/analyst/manager/partner），但这个字段没有通过 API 返回给前端，前端也没有展示。连接两端都缺失。 |
| **处置方案**          | 如果要展示晋升进度：后端需在 `AgentMarketItem` 响应中增加 `agentLevel` 字段；前端在 StrategyDetail 右侧边栏增加 Career Progress 卡片。暂不展示则清理翻译键。                                                                                                                                |
| **对应开发文件**      | `src/models.go`（AgentMarketItem 缺少 agentLevel 字段）；`src/store.go`（listAgentStats 查询需加 agent_level 列）；`frontend-www/src/pages/StrategyDetail.tsx`                                                                                                                              |

---

### [暂时未展示] GHOST-10 | Rating / 评分系统表已建但无 API 无 UI

| 项目                  | 详情                                                                                                                                                                                                                                                                            |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/strategies/:id`                                                                                                                                                                                                                                                               |
| **对应前端字段/功能** | 翻译键 `strategyDetail.rating`（评分）；`Strategy` 类型中的 `rating: number` 和 `reviews: Review[]`                                                                                                                                                                             |
| **对应前端文件**      | `frontend-www/src/i18n/translations.ts` L222；`frontend-www/src/store/slices/strategySlice.ts` L54–L55；`frontend-www/src/types/index.ts` L43–L44                                                                                                                               |
| **问题描述**          | `Strategy` 类型定义了 `rating`（评分）和 `reviews`（评价列表），`mapAgentToStrategy` 中两者均赋值为 `0` 和 `[]`。后端虽然建了 `agent_reviews` 表（`src/store.go` L157–L166），但没有任何公开 API 供前端读取评分或提交评价，页面 JSX 中也没有显示 rating 和 reviews 的 UI 区块。 |
| **处置方案**          | 评分功能完整性低，建议暂时移除类型定义和翻译键；或后端补充 `GET /api/agent-market/:publicKey/reviews`，前端在 StrategyDetail overview tab 中展示评分。                                                                                                                          |
| **对应开发文件**      | `src/store.go`（`agent_reviews` 表已存在但无 handler）；`frontend-www/src/types/index.ts`（Review 类型）；`frontend-www/src/pages/StrategyDetail.tsx`                                                                                                                           |

---

### GHOST-11 | XAuthCallback 错误页样式与全站不一致

| 项目                  | 详情                                                                                                                                                                                                                  |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **页面**              | `/auth/x/callback`（登录回调失败时）                                                                                                                                                                                  |
| **对应前端字段/功能** | 登录失败/profile加载失败的错误页面卡片                                                                                                                                                                                |
| **对应前端文件**      | `frontend-www/src/pages/XAuthCallback.tsx` L84–L141                                                                                                                                                                   |
| **问题描述**          | 错误页面使用 `bg-amber-50 text-zinc-900 border-red-200` 等亮色 TailwindCSS 类，与全站的暗色系设计（`var(--bg-card)`、`var(--text-primary)` 等）完全不一致，在全局暗色背景下会出现一个孤立的亮白色方块，视觉割裂严重。 |
| **处置方案**          | 将 XAuthCallback 的错误卡片样式改为使用全站 CSS 变量（`var(--bg-card)`、`var(--border)`、`var(--red)` 等），保持视觉一致性（约30分钟修复）。                                                                          |
| **对应开发文件**      | `frontend-www/src/pages/XAuthCallback.tsx` L84–L110、L114–L141                                                                                                                                                        |

---

## 空壳功能优先级总结

| 优先级 | ID       | 功能                                                       | 处置建议                        |
| ------ | -------- | ---------------------------------------------------------- | ------------------------------- |
| 🔴 P0  | GHOST-03 | Profile 邀请赚佣（有 i18n 无 UI 无后端）                   | 清理翻译键或完整实现            |
| 🔴 P0  | GHOST-04 | Profile「View All」按钮无跳转                              | 改为 Link 或删除按钮（1行修复） |
| 🔴 P0  | GHOST-05 | Profile 我的策略匹配逻辑 — 未激活 agent 不显示             | 补充用户专属 agent 查询接口     |
| ⚠️ P1  | GHOST-01 | StrategyDetail Vote 卡片（i18n 有，UI/后端无）             | 清理 i18n 或实现功能            |
| ⚠️ P1  | GHOST-08 | StrategyDetail Trade History Tab（后端有数据，前端无 UI）  | 补充 UI 或移除 i18n 键          |
| ⚠️ P1  | GHOST-09 | StrategyDetail Career Level 字段（后端有数据，前端无展示） | 补充 agentLevel 到 API 和 UI    |
| ⚠️ P1  | GHOST-11 | 登录错误页样式不一致                                       | 统一为暗色主题（30分钟修复）    |
| ℹ️ P2  | GHOST-02 | Followers 数量（永远为0）                                  | 确认是否要实现                  |
| ℹ️ P2  | GHOST-06 | Vault 残留无用翻译键                                       | 清理 i18n 文件                  |
| ℹ️ P2  | GHOST-07 | Home 职级晋升路线图（i18n 有，UI 无）                      | 实现 UI 或清理 i18n 键          |
| ℹ️ P2  | GHOST-10 | Rating/Reviews（表已建，无 API 无 UI）                     | 清理或实现                      |
