# Axon 下一步开发计划

**版本：v1.0 — 2026 年 3 月**
**当前状态：本地测试网可运行，核心模块骨架完成**

---

## 当前完成度概览

```
✅ 已完成                              ⚠️ 部分完成                    ❌ 未实现
─────────────────────────────────────────────────────────────────────────
✅ 链骨架 (Cosmos SDK + EVM)           ⚠️ Python SDK（骨架）          ❌ 公网多节点部署
✅ x/agent 模块（注册/心跳/信誉）       ⚠️ 单元测试（覆盖率低）        ❌ 链升级机制
✅ AI 挑战（commit/reveal/评估）        ⚠️ AI 题库（30 道，目标 100+） ❌ 治理模块集成
✅ 预编译 IAgentRegistry               ⚠️ 接入文档（无）              ❌ IBC 跨链
✅ 预编译 IAgentReputation                                           ❌ 以太坊桥
✅ 预编译 IAgentWallet + 信任通道                                     ❌ 安全审计
✅ 区块奖励（650M 硬顶 + 4 年减半）
✅ 贡献奖励（350M 硬顶 + 防刷）
✅ 零预分配代币经济
✅ Blockscout 区块浏览器
✅ 水龙头
✅ CI (GitHub Actions)
✅ 信任通道安全体系
✅ Gas Base Fee 智能销毁（80%/50%）
✅ 合约部署销毁 10 AXON
✅ 信誉归零→质押全额销毁
✅ AI 作弊检测与惩罚销毁
✅ 出块权重动态调整（ReputationBonus 分级）
✅ IAgentWallet Solidity 接口同步
```

---

## 开发路线

```
Sprint 1    通缩模型完成           7 个任务    ← 当前优先
Sprint 2    SDK + 文档 + 测试      6 个任务
Sprint 3    公开测试网             5 个任务
Sprint 4    主网准备               5 个任务
Sprint 5    生态扩展               4 个任务（远期）
```

---

## Sprint 1 — 通缩模型完成（白皮书 §8.6 五大销毁路径）

> 白皮书最核心的卖点之一是"多层通缩机制"，当前 5 条路径只实现了注册销毁 1 条。
> 优先补全，确保经济模型与白皮书一致。

### 任务 1.1：Gas Base Fee 100% 销毁

```
目标：EIP-1559 的 Base Fee 全部销毁，Priority Fee 给出块者
依赖：无
位置：app/agent_module.go → BeginBlock / FeeMarket 配置
预计：3-4 小时

步骤：
  1. 确认 Cosmos EVM FeeMarket 模块当前 Base Fee 去向
  2. 在 BeginBlocker 中添加逻辑：
     - 收集上一区块的 Base Fee 总额
     - 调用 bankKeeper.BurnCoins 销毁
  3. Priority Fee 确保发给 Proposer
  4. 添加事件 "gas_fee_burned" 记录每块销毁量
  5. 验证 total supply 随交易递减

验收：
  □ 发送交易后 total supply 减少
  □ Priority Fee 正确到账出块者
  □ 事件日志记录销毁数量
```

### 任务 1.2：合约部署销毁 10 AXON

```
目标：部署合约时额外销毁 10 AXON（白皮书 §8.6 路径 3）
依赖：无
位置：app/evm_hooks.go（新建）或 app/agent_module.go
预计：4-5 小时

步骤：
  1. 实现 EVM PostTxProcessing Hook
  2. 检测交易是否为合约创建（to == nil 或 CREATE/CREATE2）
  3. 从部署者余额扣除 10 AXON 并调用 BurnCoins
  4. 余额不足 10 AXON 时回滚整笔交易
  5. 记录贡献计分：IncrementDeployCount

验收：
  □ 部署合约后 total supply 减少 10 AXON
  □ 余额不足时部署失败
  □ 普通转账不受影响
  □ 贡献计分正确递增
```

### 任务 1.3：信誉归零 → 质押全额销毁

```
目标：Agent 信誉降为 0 时，质押 100% 销毁（白皮书 §8.6 路径 4）
依赖：无
位置：x/agent/keeper/reputation.go
预计：2-3 小时

步骤：
  1. 在 UpdateReputation 中检测信誉降为 0
  2. 触发：从模块账户销毁该 Agent 的全部质押
  3. 自动注销 Agent（设为 SUSPENDED）
  4. 发出事件 "agent_slashed_zero_reputation"

验收：
  □ 信誉降为 0 时质押自动销毁
  □ Agent 状态变为 SUSPENDED
  □ total supply 减少对应质押金额
```

### 任务 1.4：AI 挑战作弊检测与惩罚

```
目标：检测明显作弊行为并罚没质押（白皮书 §8.6 路径 5）
依赖：无
位置：x/agent/keeper/challenge.go → EvaluateEpochChallenges
预计：3-4 小时

步骤：
  1. 在 EvaluateEpochChallenges 中增加作弊检测：
     - 重复答案检测：多个验证者提交完全相同的 commit hash
     - 空答案 + 高 gas 抢先提交检测
  2. 检测到作弊：
     - 罚没 20% 质押并销毁
     - 信誉 -20
     - AIBonus 设为 -5
  3. 发出事件 "ai_challenge_cheat_detected"

验收：
  □ 重复答案被标记为作弊
  □ 质押部分销毁
  □ 信誉扣减
```

### 任务 1.5：出块权重动态调整

```
目标：验证者出块权重 = Stake × (1 + ReputationBonus + AIBonus)（白皮书 §7.3）
依赖：1.4
位置：x/agent/keeper/abci.go → EndBlocker
预计：4-5 小时

步骤：
  1. 实现 ReputationBonus 计算：
     - 信誉 < 30 → 0%
     - 信誉 30-50 → 5%
     - 信誉 50-70 → 10%
     - 信誉 70-90 → 15%
     - 信誉 > 90 → 20%
  2. 综合 AIBonus（已有）
  3. 在 EndBlocker 中调用 stakingKeeper 更新验证者 Power
  4. Power = DelegatedTokens × (100 + RepBonus + AIBonus) / 100

验收：
  □ 高信誉 + 高 AI 分的验证者出块频率更高
  □ 信誉低的验证者 Power 降低
  □ 权重变化通过事件日志可查
```

### 任务 1.6：IAgentWallet Solidity 接口同步

```
目标：更新 Solidity 接口文件，与信任通道实现一致
依赖：无
位置：contracts/interfaces/IAgentWallet.sol
预计：1 小时

步骤：
  1. 更新 createWallet 签名（新增 operator 参数，caller = owner）
  2. 新增 setTrust / removeTrust / getTrust 方法
  3. 更新 getWalletInfo 输出（新增 owner 字段）
  4. 添加事件定义

验收：
  □ Solidity 接口与 Go 预编译 ABI 完全一致
```

### 任务 1.7：通缩集成测试

```
目标：验证 5 条通缩路径全部正常工作
依赖：1.1 - 1.4
预计：2-3 小时

步骤：
  逐一验证：
  1. Gas 销毁 — 发送转账交易，查 total supply
  2. Agent 注册销毁 — 注册 Agent，查 20 AXON 被销毁
  3. 合约部署销毁 — 部署合约，查 10 AXON 被销毁
  4. 信誉归零销毁 — 模拟信誉归零，查质押被销毁
  5. AI 作弊销毁 — 模拟作弊场景，查质押部分销毁

验收：
  □ 5 条路径全部验证通过
  □ total supply 查询值正确反映所有销毁
  □ 编写自动化测试脚本
```

---

## Sprint 2 — SDK + 文档 + 测试

> 没有 SDK 和文档，外部开发者无法接入。这是公开测试网的前提。

### 任务 2.1：Python SDK 完善

```
目标：Agent 可以用 Python 完成全流程
依赖：Sprint 1 完成
位置：sdk/python/axon/
预计：6-8 小时

步骤：
  1. 完善 AgentClient 类：
     - register_agent / heartbeat / deregister
     - query_agent / query_reputation / query_agents
     - deploy_contract / call_contract / send_tx
     - create_wallet / execute_wallet / set_trust
  2. 底层通过 web3.py 对接 JSON-RPC
  3. Cosmos SDK 原生交易通过 gRPC / REST 对接
  4. 编写完整示例脚本
  5. 发布到 PyPI（axon-sdk）

验收：
  □ pip install axon-sdk 安装成功
  □ 全流程脚本（注册→心跳→部署合约→查询）运行通过
```

### 任务 2.2：TypeScript SDK

```
目标：前端和 Node.js 生态支持
依赖：2.1
位置：sdk/typescript/
预计：6-8 小时

步骤：
  1. 基于 ethers.js / viem 封装
  2. 实现与 Python SDK 对等的功能
  3. 支持 Browser + Node.js
  4. 发布到 npm（@axon-chain/sdk）

验收：
  □ npm install @axon-chain/sdk 安装成功
  □ 可在浏览器和 Node.js 中使用
```

### 任务 2.3：开发者文档

```
目标：外部开发者可以自助接入
依赖：2.1, 2.2
位置：docs/
预计：4-6 小时

内容：
  1. 快速开始（5 分钟运行节点）
  2. Agent 注册指南（CLI + SDK）
  3. 智能合约部署教程（Hardhat + Foundry）
  4. 预编译合约 API 文档（Registry / Reputation / Wallet）
  5. 信任通道使用指南
  6. 代币经济说明
  7. FAQ

格式：Markdown，可选部署 VitePress 或 Docusaurus
```

### 任务 2.4：AI 题库扩充到 100+

```
目标：AI 挑战题目多样化，防止记忆攻击
依赖：无
位置：x/agent/keeper/challenge.go → challengePool
预计：3-4 小时

步骤：
  1. 扩充题目到 100+ 道
  2. 覆盖领域：算法、区块链、密码学、网络、数据库、
     设计模式、操作系统、机器学习、数学、Axon 专有
  3. 确保每题有唯一标准答案
  4. 添加难度分级（easy / medium / hard）

验收：
  □ 题库 ≥ 100 道
  □ 覆盖 ≥ 10 个领域
  □ 所有答案可自动评分
```

### 任务 2.5：单元测试补全

```
目标：核心路径测试覆盖率 > 70%
依赖：Sprint 1 完成
位置：x/agent/keeper/*_test.go, precompiles/wallet/*_test.go
预计：6-8 小时

重点覆盖：
  1. 区块奖励分配（分配比例、减半、硬顶）
  2. 贡献奖励分配（计分、防刷、2% 上限）
  3. 信誉变化（加减、归零触发销毁）
  4. AI 挑战全流程（出题→提交→揭示→评分→作弊检测）
  5. 钱包信任通道（全部 trust level 场景）
  6. 通缩路径（5 条）

验收：
  □ make test 全部通过
  □ 覆盖率 > 70%
```

### 任务 2.6：EVM 兼容性完整测试

```
目标：标准以太坊工具全部验证通过
依赖：Sprint 1 完成
位置：contracts/test/
预计：3-4 小时

步骤：
  1. Hardhat 部署 ERC-20 合约
  2. Foundry forge test 通过
  3. MetaMask 发送交易
  4. ethers.js 脚本调用预编译合约
  5. 验证 EIP-1559 Gas 机制
  6. 验证合约部署销毁机制

验收：
  □ 全部以太坊标准工具可用
  □ ERC-20 / ERC-721 合约正常运行
```

---

## Sprint 3 — 公开测试网（白皮书 Q3 目标）

### 任务 3.1：多节点公网部署

```
目标：外部节点可同步的公开测试网
依赖：Sprint 2 完成
预计：6-8 小时

步骤：
  1. 准备 3-5 台云服务器（或 Akash 去中心化云）
  2. 使用 testnet/init-testnet.sh 初始化多验证者
  3. 部署种子节点，开放 P2P / RPC / JSON-RPC
  4. 部署 Blockscout 区块浏览器（公网可访问）
  5. 部署水龙头（公网可访问）
  6. 配置监控（Prometheus + Grafana）

验收：
  □ 外部节点可以 sync
  □ MetaMask 可连接测试网
  □ 水龙头可以领测试币
```

### 任务 3.2：Agent 自动化心跳守护进程

```
目标：Agent 节点自动发送心跳，保持在线
依赖：3.1
预计：3-4 小时

步骤：
  1. 编写守护脚本 / sidecar 程序
  2. 每 100 块自动发送心跳交易
  3. 自动参与 AI 挑战（调用本地 AI 模型回答）
  4. 集成到 Docker 部署方案

验收：
  □ Agent 节点启动后自动保持 ONLINE
  □ 自动参与 AI 挑战
```

### 任务 3.3：首批示范合约

```
目标：展示 Agent 生态可能性
依赖：3.1
预计：4-6 小时

合约：
  1. AgentDAO — 高信誉 Agent 协作 DAO
  2. AgentMarketplace — Agent 服务交易市场
  3. ReputationGatedVault — 信誉门控金库
  4. TrustChannelExample — 信任通道使用示例

验收：
  □ 4 个合约部署到测试网
  □ 有使用文档和示例代码
```

### 任务 3.4：持续集成 / 持续部署

```
目标：代码合并后自动测试 + 自动部署测试网
依赖：3.1
预计：3-4 小时

步骤：
  1. GitHub Actions 添加 e2e 测试
  2. main 分支合并后自动构建 Docker 镜像
  3. 自动推送到 GitHub Container Registry
  4.（可选）自动滚动更新测试网节点

验收：
  □ PR 合并后自动跑测试
  □ Docker 镜像自动发布
```

### 任务 3.5：公开测试网发布

```
目标：对外宣布测试网上线
依赖：3.1 - 3.4
预计：2-3 小时

输出：
  1. 测试网接入指南（含 RPC URL、Chain ID、水龙头地址）
  2. GitHub Release v0.4.0-testnet
  3. 社交媒体公告
  4. 开发者 Discord / Telegram 频道

目标指标：
  □ 首月 50+ 外部验证者
  □ 首月 100+ 链上合约
```

---

## Sprint 4 — 主网准备（白皮书 Q4 目标）

### 任务 4.1：链升级机制

```
位置：集成 x/upgrade 模块
目标：支持不停链软件升级
预计：4-6 小时
```

### 任务 4.2：治理模块集成

```
位置：集成 x/gov 模块
目标：链参数可通过提案投票调整
预计：4-6 小时
```

### 任务 4.3：安全审计

```
外部：委托专业审计公司
重点：奖励分配、预编译合约、钱包安全
预计：4-8 周（外部周期）
```

### 任务 4.4：正式创世配置

```
目标：正式 genesis.json + 初始验证者集合
预计：2-3 小时
```

### 任务 4.5：主网上线

```
目标：正式创世区块
输出：主网 RPC、Chain ID、区块浏览器、水龙头关闭
```

---

## Sprint 5 — 生态扩展（2027 远期）

| 任务 | 说明 | 复杂度 |
|------|------|--------|
| IBC 跨链 | 接入 Cosmos 生态（Osmosis、Stride 等） | 高 |
| 以太坊桥 | ERC-20 AXON 双向桥接 | 高 |
| Block-STM 并行执行 | TPS 提升到 10,000+ | 很高 |
| Agent 治理权重 | 高信誉 Agent 投票权加成 | 中 |
| 多语言 SDK | Go SDK 补全 | 中 |
| 区块时间优化 | 5s → 2s | 中 |

---

## 工作量估算

```
Sprint 1    通缩模型完成         ~20-28 小时     ← 最优先
Sprint 2    SDK + 文档 + 测试    ~28-38 小时
Sprint 3    公开测试网           ~18-25 小时
Sprint 4    主网准备             ~14-20 小时 + 审计周期
Sprint 5    生态扩展             远期规划
──────────────────────────────────────────────
Sprint 1-3 合计                  ~66-91 小时

按每天 6 小时：11-15 个工作日可完成到公开测试网
```

---

## 依赖关系

```
Sprint 1（通缩完成）
  1.1 Gas 销毁
  1.2 合约部署销毁
  1.3 信誉归零销毁        → 1.7 通缩集成测试
  1.4 AI 作弊惩罚
  1.5 出块权重调整
  1.6 Solidity 接口同步
            │
Sprint 2（SDK + 测试）
  2.1 Python SDK
  2.2 TypeScript SDK
  2.3 开发者文档           → 2.6 EVM 兼容测试
  2.4 AI 题库扩充
  2.5 单元测试
            │
Sprint 3（公开测试网）
  3.1 公网部署
  3.2 心跳守护进程         → 3.5 公开测试网发布
  3.3 示范合约
  3.4 CI/CD
            │
Sprint 4（主网）
  4.1 升级机制
  4.2 治理模块             → 4.5 主网上线
  4.3 安全审计
  4.4 创世配置
```

---

## 建议执行顺序

每次开发时打开本文档，告诉 AI 执行对应任务编号即可。

### Sprint 1 进度

| 任务 | 状态 | 说明 |
|------|------|------|
| 1.1 Gas Base Fee 销毁 | ✅ 完成 | BeginBlocker 顺序修复 + 智能比例(80%/50%) |
| 1.2 合约部署销毁 10 AXON | ✅ 已有 | `app/evm_hooks.go` |
| 1.3 信誉归零→质押销毁 | ✅ 已有 | `x/agent/keeper/reputation.go` |
| 1.4 AI 作弊检测与惩罚 | ✅ 完成 | 重复 commit hash 检测 + 20%质押销毁 + 信誉-20 |
| 1.5 出块权重动态调整 | ✅ 完成 | ReputationBonus 5级分层(0/5/10/15/20%) |
| 1.6 IAgentWallet 接口同步 | ✅ 完成 | Solidity 接口新增 setTrust/removeTrust/getTrust |
| 1.7 通缩集成测试 | ⏳ 待做 | |

**下一步：执行任务 1.7（通缩集成测试），然后进入 Sprint 2。**
