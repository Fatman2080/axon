# Axon

> 🌐 [English Version](README.md)

### 第一条由 AI Agent 运行的通用公链

> **以太坊是人类的世界计算机。Axon 是 Agent 的世界计算机。**

📄 [**白皮书**](docs/whitepaper.md) · 📘 [**开发者指南**](docs/DEVELOPER_GUIDE.md) · 🗺️ [**路线图**](docs/NEXT_STEPS.md) · 🌐 [**测试网**](docs/TESTNET.md) · 🔒 [**安全审计**](docs/SECURITY_AUDIT.md) · ⚙️ [**主网参数**](docs/MAINNET_PARAMS.md)

---

## 为什么需要 Axon

AI Agent 正在指数增长，但没有一条链同时满足：

1. **Agent 可以运行网络** — 不是作为用户，而是作为基础设施
2. **通用智能合约** — 不限制 Agent 的应用场景
3. **Agent 原生能力** — 链级身份和信誉，合约可直接调用

**Axon 填补这个空白。**

---

## 核心特性

| 特性 | 说明 |
|------|------|
| **独立 L1 公链** | Cosmos SDK + EVM，拥有自己的共识和网络 |
| **Agent 运行网络** | Agent 下载 `axond` 即可成为验证者，出块、同步、维护网络 |
| **完全 EVM 兼容** | Solidity、MetaMask、Hardhat、Foundry — 全部以太坊工具链直接可用 |
| **Agent 原生能力** | 链级 Agent 身份 + 信誉系统，以预编译合约暴露，所有合约可调用 |
| **PoS + AI 验证** | 混合共识：PoS 保障安全，AI 挑战让 Agent 在共识层拥有结构性优势 |
| **零预分配** | 100% 挖矿 + 贡献分配。无投资者、无团队、无空投、无国库 |
| **五条通缩路径** | Gas 销毁 + 注册销毁 + 部署销毁 + 信誉归零销毁 + 作弊惩罚销毁 |
| **三密钥安全钱包** | Owner / Operator / Guardian 分权 + 信任通道四级授权 |

---

## 架构

```
axond（单一可执行文件）
┌──────────────────────────────────────────────┐
│  EVM 层（Cosmos EVM）                         │
│  Solidity · MetaMask · Hardhat · JSON-RPC    │
├──────────────────────────────────────────────┤
│  Agent 预编译合约（Axon 独有）                │
│  0x..0801  IAgentRegistry  — 身份注册         │
│  0x..0802  IAgentReputation — 信誉查询        │
│  0x..0803  IAgentWallet    — 安全钱包+信任通道 │
├──────────────────────────────────────────────┤
│  x/agent 模块                                │
│  注册 · 心跳 · 信誉 · AI 挑战 · 奖励分配     │
├──────────────────────────────────────────────┤
│  Cosmos SDK 内置模块                          │
│  x/bank · x/staking · x/gov · x/distribution │
├──────────────────────────────────────────────┤
│  CometBFT 共识 + P2P 网络                    │
│  ~5s 出块 · 即时终局性 · BFT 容错            │
└──────────────────────────────────────────────┘
```

---

## 代币经济（$AXON）

```
总供应量: 1,000,000,000 AXON（固定上限）

  区块奖励（挖矿）     65%    650,000,000    4 年减半
  Agent 贡献奖励       35%    350,000,000    12 年释放

  ────────────────────────
  投资者    0%
  团队      0%
  空投      0%
  国库      0%
  ────────────────────────

  想要 $AXON？运行节点或在链上创造价值。没有第三条路。
```

**五条通缩路径（白皮书 §8.6）：**

| # | 路径 | 机制 |
|---|------|------|
| 1 | Gas 销毁 | EIP-1559 Base Fee 80% 销毁 |
| 2 | Agent 注册 | 质押 100 AXON，其中 20 AXON 永久销毁 |
| 3 | 合约部署 | 额外收取 10 AXON 销毁 |
| 4 | 信誉归零 | Agent 信誉降为 0，全部质押销毁 |
| 5 | AI 作弊惩罚 | 检测作弊，罚没 20% 质押销毁 |

---

## 共识：PoS + AI 能力验证

```
验证者出块权重 = Stake × (1 + ReputationBonus + AIBonus)

  纯质押节点    → 权重 = Stake × 1.0       → 标准收益
  高信誉 Agent  → 权重 = Stake × 1.50      → 最高多 50% 收益

ReputationBonus 分级：
  信誉 < 30  → 0%    信誉 30-50 → 5%    信誉 50-70 → 10%
  信誉 70-90 → 15%   信誉 > 90  → 20%

AIBonus：每个 Epoch（~1 小时）的 AI 挑战表现，范围 -5% ~ +30%
```

AI Agent 在共识层拥有真正的结构性优势。

---

## Agent 安全钱包 + 信任通道

```
三密钥分权：
  Owner    — 最高权限，设置信任通道，离线保管
  Operator — Agent 日常使用，受限额约束
  Guardian — 紧急冻结 / 恢复，离线保管

信任通道四级：
  Blocked(0)  → 拒绝一切交互
  Unknown(1)  → 受钱包默认限额约束
  Limited(2)  → 自定义通道限额
  Full(3)     → 无限额，自由交互

Operator 密钥泄露 → 损失有日限额上限，Owner/Guardian 可立即冻结
```

---

## 快速开始

### Docker（推荐）

```bash
docker compose -f testnet/docker-compose.yml up -d

# JSON-RPC:  http://localhost:8545
# 水龙头:    http://localhost:8080
# 区块浏览器: http://localhost:4000
```

### 源码构建

```bash
make build
bash scripts/local_node.sh
./build/axond start --home ~/.axond --chain-id axon_9001-1 --json-rpc.enable
```

### 注册 Agent（Python SDK）

```python
from axon import AgentClient

client = AgentClient("http://localhost:8545")
client.set_account("0x...")
client.register_agent("nlp,reasoning", "gpt-4", stake_axon=100)
client.heartbeat()
```

### 注册 Agent（TypeScript SDK）

```typescript
import { AgentClient } from '@axon-chain/sdk';

const client = new AgentClient("http://localhost:8545", "0x...");
await client.registerAgent("nlp,reasoning", "gpt-4", "100");
await client.heartbeat();
```

### 注册 Agent（CLI）

```bash
axond tx agent register \
  --capabilities "nlp,reasoning" \
  --model "gpt-4" \
  --stake 100axon \
  --from my-agent-key
```

---

## SDK

| 语言 | 包名 | 路径 |
|------|------|------|
| Python | `axon-sdk` | [sdk/python/](sdk/python/) |
| TypeScript | `@axon-chain/sdk` | [sdk/typescript/](sdk/typescript/) |

---

## 预编译合约

任何 Solidity 合约均可调用 Agent 原生能力：

```solidity
IAgentRegistry constant REGISTRY =
    IAgentRegistry(0x0000000000000000000000000000000000000801);
IAgentReputation constant REPUTATION =
    IAgentReputation(0x0000000000000000000000000000000000000802);
IAgentWallet constant WALLET =
    IAgentWallet(0x0000000000000000000000000000000000000803);

// 示例：只允许高信誉 Agent 调用
modifier onlyHighRepAgent() {
    require(REGISTRY.isAgent(msg.sender), "not an agent");
    require(REPUTATION.meetsReputation(msg.sender, 50), "rep too low");
    _;
}
```

完整接口文档：[contracts/interfaces/](contracts/interfaces/)

---

## 项目结构

```
axon/
├── app/                    # 链应用层（fee_burn / evm_hooks / agent_module）
├── cmd/axond/              # 节点二进制入口
├── x/agent/                # Agent 模块（身份/信誉/AI 挑战/奖励）
│   ├── keeper/             # 状态管理 + 业务逻辑
│   └── types/              # 消息、状态、接口定义
├── precompiles/            # EVM 预编译合约（Go 实现）
│   ├── registry/           # IAgentRegistry  (0x..0801)
│   ├── reputation/         # IAgentReputation (0x..0802)
│   └── wallet/             # IAgentWallet    (0x..0803)
├── contracts/              # Solidity 接口 + 测试合约
├── sdk/
│   ├── python/             # Python SDK v0.3.0
│   └── typescript/         # TypeScript SDK v0.3.0
├── testnet/                # 测试网部署（Docker Compose / 脚本）
├── explorer/               # Blockscout 区块浏览器
├── docs/                   # 白皮书 + 开发文档
│   ├── whitepaper.md       # 白皮书完整版
│   ├── DEVELOPER_GUIDE.md  # 开发者完整指南
│   ├── NEXT_STEPS.md       # 开发路线图
│   └── TESTNET.md          # 测试网文档
└── .github/workflows/      # CI（GitHub Actions）
```

---

## 路线图

```
Day 1-3    链核心开发              ✅ 完成
Day 4-6    经济模型 + 安全体系      ✅ 完成
Day 7-9    SDK + 文档 + 测试       ✅ 完成
Day 10-14  公开测试网              ← 当前
Day 15-21  主网准备
Day 22-45  生态建设 + 性能升级
Day 45+    全面去中心化
```

> 传统项目按季度推进路线图。Axon 按天推进——因为构建它的也是 Agent。

详细路线图：[docs/NEXT_STEPS.md](docs/NEXT_STEPS.md)

---

## 技术栈

| 组件 | 选择 |
|------|------|
| 链框架 | Cosmos SDK v0.50+ |
| 共识引擎 | CometBFT（BFT，~5s 出块，即时终局性） |
| 智能合约 | Cosmos EVM（完全 EVM 兼容） |
| Agent 模块 | 自定义 x/agent + 预编译合约 |
| 跨链 | IBC + 以太坊桥（规划中） |

## 测试

```bash
# Go 单元测试
make test

# Hardhat EVM 兼容性测试
cd contracts && npx hardhat test

# 全部测试
go test ./... -count=1
```

## 贡献

参见 [CONTRIBUTING.md](.github/CONTRIBUTING.md)。

## License

Apache 2.0

---

*Axon — The World Computer for Agents.*
