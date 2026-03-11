> 🌐 [English Version](DEVELOPER_GUIDE_EN.md)

# Axon 开发者指南

> 面向外部开发者的完整接入手册——从零启动节点到部署合约、注册 Agent、使用预编译 API。

---

## 目录

1. [快速开始（5 分钟运行节点）](#1-快速开始5-分钟运行节点)
2. [网络信息](#2-网络信息)
3. [Agent 注册指南](#3-agent-注册指南)
4. [智能合约部署教程](#4-智能合约部署教程)
5. [预编译合约 API 文档](#5-预编译合约-api-文档)
6. [信任通道使用指南](#6-信任通道使用指南)
7. [代币经济说明](#7-代币经济说明)
8. [MetaMask 配置](#8-metamask-配置)
9. [FAQ](#9-faq)

---

## 1. 快速开始（5 分钟运行节点）

### 1.1 前置条件

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.22+ | 编译节点二进制文件 |
| Git | 任意 | 拉取源码 |
| Docker（可选）| 20+ | 一键启动测试网 |
| Make | 任意 | 构建工具 |

### 1.2 方式一：从源码构建

```bash
# 1. 克隆仓库
git clone https://github.com/Fatman2080/axon.git

# 2. 编译
cd axon && go build -o axond ./cmd/axond

# 3. 初始化本地节点
./axond init mynode --chain-id axon_8210-1

# 4. 创建验证者账户并添加创世余额
./axond keys add mykey
./axond genesis add-genesis-account mykey 1000000000000000000000aaxon

# 5. 生成创世交易
./axond genesis gentx mykey 10000000000000000000000aaxon --chain-id axon_8210-1

# 6. 收集创世交易
./axond genesis collect-gentxs

# 7. 启动节点
./axond start --json-rpc.enable
```

启动后你将看到区块高度递增的日志，约每 5 秒出一个块。

### 1.3 方式二：使用 Docker（推荐）

```bash
# 启动完整测试网（4 验证者 + 水龙头 + 区块浏览器）
docker compose -f testnet/docker-compose.yml up -d
```

启动后可用的服务：

| 服务 | 地址 |
|------|------|
| JSON-RPC (EVM) | http://localhost:8545 |
| 水龙头 | http://localhost:8080 |
| 区块浏览器 | http://localhost:4000 |

### 1.4 方式三：本地脚本快速启动

```bash
make build
bash scripts/local_node.sh
./build/axond start --home ~/.axond --chain-id axon_8210-1 --json-rpc.enable
```

### 1.5 验证节点运行

```bash
# 检查 Tendermint 状态
curl http://localhost:26657/status

# 检查 EVM 链 ID
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
# 返回: "0x201a" (8210)

# 检查最新区块高度
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

---

## 2. 网络信息

### 2.1 链参数

| 参数 | 值 |
|------|------|
| Chain ID (EVM) | `8210` |
| Chain ID (Cosmos) | `axon_8210-1` |
| 区块时间 | ~5 秒 |
| 终局性 | 即时（单区块确认，无分叉） |
| 原生代币 | AXON |
| 最小单位 | aaxon（`1 AXON = 10^18 aaxon`，与 ETH/wei 对齐） |
| Bech32 前缀 | `axon`（账户）/ `axonvaloper`（验证者） |
| 验证者上限 | 100 |
| Epoch 长度 | 720 区块（≈ 1 小时） |

### 2.2 RPC 端点

| 协议 | 地址 | 用途 |
|------|------|------|
| Tendermint RPC | `http://localhost:26657` | Cosmos 原生查询、WebSocket 订阅 |
| JSON-RPC (EVM) | `http://localhost:8545` | MetaMask、Hardhat、ethers.js 等以太坊工具 |
| WebSocket (EVM) | `ws://localhost:8546` | 实时事件订阅 |
| REST API | `http://localhost:1317` | Cosmos REST 查询（x/bank、x/staking 等） |
| gRPC | `localhost:9090` | Cosmos gRPC，适合后端服务 |

### 2.3 Gas 机制

Axon 使用 EIP-1559 动态 Gas 机制：

- **Base Fee**：根据区块利用率动态调整，100% 销毁（通缩）
- **Priority Fee**：用户/Agent 自定义小费，100% 给出块者
- 最大区块 Gas：**40,000,000**
- Gas 价格远低于以太坊，适合 Agent 高频交互

---

## 3. Agent 注册指南

Agent 是 Axon 的一等公民。注册后获得链级身份和信誉，所有链上合约均可查询。

### 3.1 使用 CLI 注册

```bash
# 注册 Agent（质押 100 AXON，其中 20 AXON 永久销毁）
axond tx agent register \
  --capabilities "nlp,reasoning,code-generation" \
  --model "gpt-4" \
  --stake 100axon \
  --from my-agent-key

# 查询 Agent 信息
axond query agent agent $(axond keys show my-agent-key -a)

# 发送心跳保持在线
axond tx agent heartbeat --from my-agent-key

# 查询信誉分
axond query agent reputation $(axond keys show my-agent-key -a)
```

### 3.2 使用 Python SDK

```python
from axon import AgentClient

client = AgentClient(
    rpc_url="http://localhost:8545",
    private_key="0xYOUR_PRIVATE_KEY",
)

# 注册 Agent（质押 100 AXON）
tx = client.register_agent(
    capabilities=["nlp", "reasoning", "code-generation"],
    model="gpt-4",
    stake_amount=100,
)
client.wait_for_tx(tx)

# 发送心跳保持在线
client.heartbeat()

# 查询 Agent 信息
agent_info = client.query_agent("0xAGENT_ADDRESS")
print(f"信誉: {agent_info.reputation}, 在线: {agent_info.is_online}")

# 查询信誉分
rep = client.get_reputation("0xAGENT_ADDRESS")
print(f"信誉分: {rep}")
```

### 3.3 使用 TypeScript SDK

```typescript
import { AgentClient } from '@axon-chain/sdk';

const client = new AgentClient(
  "http://localhost:8545",
  "0xYOUR_PRIVATE_KEY"
);

// 注册 Agent
await client.registerAgent({
  capabilities: ["nlp", "reasoning", "code-generation"],
  model: "gpt-4",
  stakeAxon: "100",
});

// 发送心跳
await client.heartbeat();

// 查询信誉
const rep = await client.getReputation("0xAGENT_ADDRESS");
console.log(`Reputation: ${rep}`);
```

### 3.4 使用 Solidity（通过预编译合约）

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./interfaces/IAgentRegistry.sol";

contract MyAgentRegistrar {
    IAgentRegistry constant REGISTRY =
        IAgentRegistry(0x0000000000000000000000000000000000000801);

    function registerSelf() external payable {
        // msg.value 需 >= 100 AXON (100 * 10^18 aaxon)
        REGISTRY.register{value: msg.value}("nlp,reasoning", "gpt-4");
    }

    function checkAgent(address account) external view returns (bool) {
        return REGISTRY.isAgent(account);
    }
}
```

### 3.5 Agent 生命周期

```
┌─────────────┐     质押 100 AXON      ┌─────────────┐
│  未注册      │ ─────────────────────→ │  已注册      │
│  (Unregistered)                      │  (Registered) │
└─────────────┘     20 AXON 销毁       └──────┬──────┘
                                              │
                    ┌─────────────────────────┤
                    │                         │
                    ▼                         ▼
            ┌─────────────┐          ┌─────────────┐
            │  在线        │ ←─心跳── │  离线        │
            │  (Online)   │          │  (Offline)   │
            └──────┬──────┘          └─────────────┘
                   │                  720 块无心跳 → 自动离线
                   │
          ┌────────┴────────┐
          │                 │
          ▼                 ▼
  信誉增长(活跃)      AI 挑战(每 Epoch)
  +0.5 ~ +1/Epoch    答对 → AIBonus
                     未答 → 无惩罚
                   │
                   ▼
            ┌─────────────┐
            │  注销        │    80 AXON 返还（7 天冷却期后）
            │  (Deregister)│    20 AXON 已在注册时销毁
            └─────────────┘
```

**关键参数：**

| 操作 | 费用/要求 | 备注 |
|------|----------|------|
| 注册 | 质押 100 AXON（20 永久销毁） | 单地址每 24 小时最多注册 3 个 |
| 心跳 | 仅 Gas 费 | 每 720 块至少发送 1 次 |
| AI 挑战 | 仅限验证者 | 每 Epoch 自动出题，答对加成 15-30% |
| 注销 | 7 天冷却期后返还 80 AXON | 信誉归零则质押全额销毁 |
| 初始信誉 | 10 分 | 上限 100，不可转移、不可购买 |

---

## 4. 智能合约部署教程

Axon 完全 EVM 兼容，支持 Solidity ^0.8.x。所有以太坊开发工具均可直接使用。

### 4.1 使用 Hardhat

**安装与初始化：**

```bash
mkdir my-axon-dapp && cd my-axon-dapp
npm init -y
npm install --save-dev hardhat @nomicfoundation/hardhat-toolbox
npx hardhat init
```

**配置 `hardhat.config.js`：**

```javascript
require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.20",
  networks: {
    axon_local: {
      url: "http://localhost:8545",
      chainId: 8210,
      accounts: ["0xYOUR_PRIVATE_KEY"],
    },
    axon_testnet: {
      url: "https://rpc-testnet.axon.network",
      chainId: 9001,
      accounts: ["0xYOUR_PRIVATE_KEY"],
    },
  },
};
```

**编写合约 `contracts/HelloAxon.sol`：**

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract HelloAxon {
    string public greeting = "Hello from Axon!";

    function setGreeting(string memory _greeting) external {
        greeting = _greeting;
    }
}
```

**部署脚本 `scripts/deploy.js`：**

```javascript
const { ethers } = require("hardhat");

async function main() {
  const HelloAxon = await ethers.getContractFactory("HelloAxon");
  const contract = await HelloAxon.deploy();
  await contract.waitForDeployment();
  console.log("HelloAxon deployed to:", await contract.getAddress());
}

main().catch(console.error);
```

**执行部署：**

```bash
npx hardhat run scripts/deploy.js --network axon_local
```

### 4.2 使用 Foundry

**安装 Foundry：**

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

**配置 `foundry.toml`：**

```toml
[profile.default]
src = "src"
out = "out"
libs = ["lib"]
solc_version = "0.8.20"

[rpc_endpoints]
axon_local = "http://localhost:8545"
axon_testnet = "https://rpc-testnet.axon.network"
```

**部署合约：**

```bash
# 编译
forge build

# 部署
forge create src/HelloAxon.sol:HelloAxon \
  --rpc-url http://localhost:8545 \
  --chain-id 8210 \
  --private-key 0xYOUR_PRIVATE_KEY

# 运行测试
forge test
```

**调用预编译合约（Foundry 示例）：**

```bash
# 查询某地址是否为注册 Agent
cast call 0x0000000000000000000000000000000000000801 \
  "isAgent(address)(bool)" \
  0xAGENT_ADDRESS \
  --rpc-url http://localhost:8545
```

### 4.3 注意事项

| 项目 | 说明 |
|------|------|
| 部署费用 | 除 Gas 外，额外销毁 **10 AXON**（防止垃圾合约） |
| 最大区块 Gas | 40,000,000 |
| EVM 版本 | 完全兼容（Shanghai） |
| Solidity 版本 | 推荐 ^0.8.20 |
| 余额不足 | 账户余额 < 10 AXON + Gas 时，部署将失败 |

**部署前检查清单：**

```bash
# 确认账户余额足够（至少 10 AXON + Gas 费）
cast balance 0xYOUR_ADDRESS --rpc-url http://localhost:8545

# 测试网领取测试币
curl -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "0xYOUR_ADDRESS"}'
```

---

## 5. 预编译合约 API 文档

Axon 的 Agent 原生能力通过固定地址的预编译合约暴露。这些合约由 Go 原生代码执行，性能比普通 Solidity 合约高 10-100 倍。

### 5.1 IAgentRegistry（`0x0000000000000000000000000000000000000801`）

Agent 身份注册与管理。

#### `isAgent(address account) → bool` [view]

检查地址是否为已注册的 Agent。

```solidity
bool registered = IAgentRegistry(0x0000000000000000000000000000000000000801)
    .isAgent(0x1234...);
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `account` | `address` | 待查询地址 |
| **返回** | `bool` | `true` = 已注册 |

---

#### `getAgent(address account) → (agentId, capabilities, model, reputation, isOnline)` [view]

获取 Agent 完整信息。

```solidity
(
    string memory agentId,
    string[] memory capabilities,
    string memory model,
    uint64 reputation,
    bool isOnline
) = IAgentRegistry(0x0000000000000000000000000000000000000801)
    .getAgent(0x1234...);
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `account` | `address` | Agent 地址 |
| **返回** | | |
| `agentId` | `string` | Agent 标识符 |
| `capabilities` | `string[]` | 能力标签列表 |
| `model` | `string` | AI 模型标识 |
| `reputation` | `uint64` | 信誉分（0-100） |
| `isOnline` | `bool` | 当前是否在线 |

---

#### `register(string capabilities, string model)` [payable]

注册为 Agent。`msg.value` 需 ≥ 100 AXON（`100 * 10^18 aaxon`）。其中 20 AXON 永久销毁，80 AXON 锁定为质押。

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801)
    .register{value: 100 ether}("nlp,reasoning", "gpt-4");
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `capabilities` | `string` | 逗号分隔的能力标签 |
| `model` | `string` | AI 模型标识 |
| `msg.value` | `uint256` | 质押金额（≥ 100 AXON） |

---

#### `updateAgent(string capabilities, string model)`

更新 Agent 的能力标签和模型信息。仅已注册 Agent 可调用。

| 参数 | 类型 | 说明 |
|------|------|------|
| `capabilities` | `string` | 新的能力标签 |
| `model` | `string` | 新的模型标识 |

---

#### `heartbeat()`

发送心跳以维持在线状态。Agent 需每 720 个区块（约 1 小时）至少发送一次，否则自动标记为离线，信誉开始衰减。

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801).heartbeat();
```

---

#### `deregister()`

注销 Agent 身份。进入 7 天冷却期后，剩余质押（80 AXON）解锁返还。

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801).deregister();
```

---

### 5.2 IAgentReputation（`0x0000000000000000000000000000000000000802`）

链级信誉查询，只读。信誉由全网验证者共识维护，不可转移、不可购买。

#### `getReputation(address agent) → uint64` [view]

查询单个 Agent 的信誉分（0-100）。

```solidity
uint64 rep = IAgentReputation(0x0000000000000000000000000000000000000802)
    .getReputation(0x1234...);
```

---

#### `getReputations(address[] agents) → uint64[]` [view]

批量查询多个 Agent 的信誉分。

```solidity
address[] memory agents = new address[](2);
agents[0] = 0x1234...;
agents[1] = 0x5678...;

uint64[] memory reps = IAgentReputation(0x0000000000000000000000000000000000000802)
    .getReputations(agents);
// reps[0] = agent1 信誉, reps[1] = agent2 信誉
```

---

#### `meetsReputation(address agent, uint64 minReputation) → bool` [view]

判断 Agent 信誉是否达到指定阈值。在合约中做准入控制时推荐使用此方法，比先查询再比较更节约 Gas。

```solidity
bool qualified = IAgentReputation(0x0000000000000000000000000000000000000802)
    .meetsReputation(agentAddr, 50);

require(qualified, "reputation too low");
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `agent` | `address` | Agent 地址 |
| `minReputation` | `uint64` | 最低信誉要求 |
| **返回** | `bool` | `true` = 达标 |

---

### 5.3 IAgentWallet（`0x0000000000000000000000000000000000000803`）

Agent 安全钱包——三密钥架构（Owner / Operator / Guardian），内置限额、冷却、冻结、信任通道。

#### `createWallet(operator, guardian, txLimit, dailyLimit, cooldownBlocks) → address` 

创建 Agent 专用安全钱包。调用者为 Owner。

```solidity
address wallet = IAgentWallet(0x0000000000000000000000000000000000000803)
    .createWallet(
        0xOperatorAddr,           // Operator: Agent 日常操作密钥
        0xGuardianAddr,           // Guardian: 紧急恢复人
        10 ether,                 // 单笔限额: 10 AXON
        50 ether,                 // 日限额: 50 AXON
        100                       // 大额冷却: 100 个区块 (~8 分钟)
    );
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `operator` | `address` | Agent 日常操作地址 |
| `guardian` | `address` | 紧急恢复/冻结地址 |
| `txLimit` | `uint256` | 单笔交易最大金额（aaxon） |
| `dailyLimit` | `uint256` | 每日累计最大支出（aaxon） |
| `cooldownBlocks` | `uint256` | 超限额交易的冷却区块数 |
| **返回** | `address` | 钱包合约地址 |

---

#### `execute(wallet, target, value, data)`

通过钱包执行交易。安全检查根据 `target` 的信任等级决定：

- **Full (3)**：无限额限制
- **Limited (2)**：使用信任通道配置的独立限额
- **Unknown (1)**：使用钱包默认限额
- **Blocked (0)**：拒绝执行

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .execute(
        walletAddr,
        targetContract,
        1 ether,                  // 转 1 AXON
        abi.encodeWithSignature("swap(uint256)", 1000)
    );
```

---

#### `freeze(address wallet)`

冻结钱包，阻止所有支出交易。**Guardian 或 Owner** 可调用。

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .freeze(walletAddr);
```

---

#### `recover(address wallet, address newOperator)`

恢复被冻结的钱包，同时更换操作密钥。**仅 Guardian** 可调用。

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .recover(walletAddr, 0xNewOperator);
```

---

#### `setTrust(wallet, target, level, txLimit, dailyLimit, expiresAt)`

为指定合约设置信任等级和限额。**仅 Owner** 可调用。

```solidity
// 授权 Uniswap 路由为 Full Trust
IAgentWallet(0x0000000000000000000000000000000000000803)
    .setTrust(
        walletAddr,
        0xUniswapRouter,
        3,                        // level: Full
        0,                        // txLimit: Full 下忽略
        0,                        // dailyLimit: Full 下忽略
        0                         // expiresAt: 0 = 永不过期
    );
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `wallet` | `address` | 钱包地址 |
| `target` | `address` | 目标合约地址 |
| `level` | `uint8` | 0=Blocked, 1=Unknown, 2=Limited, 3=Full |
| `txLimit` | `uint256` | Limited 下的单笔限额 |
| `dailyLimit` | `uint256` | Limited 下的日限额 |
| `expiresAt` | `uint256` | 授权过期区块高度（0 = 永不过期） |

---

#### `removeTrust(address wallet, address target)`

移除对某合约的信任授权，回退为 Unknown。**仅 Owner** 可调用。

---

#### `getTrust(wallet, target) → (level, txLimit, dailyLimit, authorizedAt, expiresAt)` [view]

查询某合约的信任配置。

| 返回字段 | 类型 | 说明 |
|---------|------|------|
| `level` | `uint8` | 信任等级 |
| `txLimit` | `uint256` | 单笔限额 |
| `dailyLimit` | `uint256` | 日限额 |
| `authorizedAt` | `uint256` | 授权时的区块高度 |
| `expiresAt` | `uint256` | 过期区块高度 |

---

#### `getWalletInfo(address wallet) → (txLimit, dailyLimit, dailySpent, isFrozen, owner, operator, guardian)` [view]

查询钱包状态与配置。

```solidity
(
    uint256 txLimit,
    uint256 dailyLimit,
    uint256 dailySpent,
    bool isFrozen,
    address owner,
    address operator,
    address guardian
) = IAgentWallet(0x0000000000000000000000000000000000000803)
    .getWalletInfo(walletAddr);
```

---

### 5.4 完整示例：高信誉 Agent 协作合约

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IAgentRegistry {
    function isAgent(address account) external view returns (bool);
}

interface IAgentReputation {
    function meetsReputation(address agent, uint64 minReputation) external view returns (bool);
}

contract AgentDAO {
    IAgentRegistry constant REGISTRY =
        IAgentRegistry(0x0000000000000000000000000000000000000801);
    IAgentReputation constant REPUTATION =
        IAgentReputation(0x0000000000000000000000000000000000000802);

    mapping(address => bool) public members;
    uint64 public minReputation;

    event MemberJoined(address indexed agent);
    event TaskExecuted(address indexed agent, address target);

    constructor(uint64 _minReputation) {
        minReputation = _minReputation;
    }

    modifier onlyQualifiedAgent() {
        require(REGISTRY.isAgent(msg.sender), "not a registered agent");
        require(
            REPUTATION.meetsReputation(msg.sender, minReputation),
            "reputation too low"
        );
        _;
    }

    function join() external onlyQualifiedAgent {
        members[msg.sender] = true;
        emit MemberJoined(msg.sender);
    }

    function executeTask(address target, bytes calldata data) external {
        require(members[msg.sender], "not a member");
        (bool success, ) = target.call(data);
        require(success, "task execution failed");
        emit TaskExecuted(msg.sender, target);
    }
}
```

---

## 6. 信任通道使用指南

信任通道是 Agent 安全钱包的核心功能——让 Agent 对不同合约设定不同的授权等级和限额。

### 6.1 三密钥模型

```
┌─────────────────────────────────────────────────────┐
│                  Agent 安全钱包                       │
├─────────────┬─────────────────┬─────────────────────┤
│  Owner      │  Operator       │  Guardian            │
│  (所有者)    │  (操作者)        │  (守护者)            │
├─────────────┼─────────────────┼─────────────────────┤
│  设定规则    │  日常签署交易     │  冻结钱包            │
│  授权信任    │  受限额约束      │  恢复/换钥           │
│  更换 Guardian│  可被更换       │  离线保管            │
└─────────────┴─────────────────┴─────────────────────┘
```

- **Owner**：通常是人类或高安全 Agent，离线保管，设定安全规则和信任通道
- **Operator**：Agent 持有的热密钥，日常签署交易，权限受规则约束
- **Guardian**：紧急恢复人，可以冻结钱包、更换 Operator、恢复资金

### 6.2 信任等级

| 等级 | 值 | 说明 | 限额 |
|------|---|------|------|
| **Blocked** | 0 | 禁止交互 | 一切交易被拒绝 |
| **Unknown** | 1 | 默认等级 | 使用钱包全局限额 |
| **Limited** | 2 | 有限信任 | 使用通道独立限额 |
| **Full** | 3 | 完全信任 | 无限额限制 |

### 6.3 场景示例一：授权已验证的 DeFi 合约

该 DEX 已通过审计且广泛使用，设为 Full Trust：

```solidity
IAgentWallet wallet = IAgentWallet(0x0000000000000000000000000000000000000803);

// Owner 调用：授权 DEX 路由合约为完全信任
wallet.setTrust(
    myWallet,
    0xTrustedDEXRouter,
    3,                // Full trust
    0,                // 忽略
    0,                // 忽略
    0                 // 永不过期
);

// Operator 现在可以不受限额限制地调用该 DEX
wallet.execute(myWallet, 0xTrustedDEXRouter, 0, swapCalldata);
```

### 6.4 场景示例二：授权未知合约（有限信任）

新发现的合约，尚未完全验证，设为 Limited Trust 并设定独立限额：

```solidity
// Owner 调用：授权新合约为有限信任，单笔最多 5 AXON，日最多 20 AXON
wallet.setTrust(
    myWallet,
    0xNewProtocol,
    2,                        // Limited trust
    5 ether,                  // 单笔限额: 5 AXON
    20 ether,                 // 日限额: 20 AXON
    block.number + 720 * 24   // 24 小时后自动过期
);

// Operator 可以调用，但受独立限额约束
wallet.execute(myWallet, 0xNewProtocol, 3 ether, calldata); // OK, 3 < 5
wallet.execute(myWallet, 0xNewProtocol, 6 ether, calldata); // FAIL, 6 > 5
```

### 6.5 场景示例三：拉黑恶意合约

```solidity
// Owner 调用：拉黑已知恶意合约
wallet.setTrust(
    myWallet,
    0xMaliciousContract,
    0,                // Blocked
    0, 0, 0
);

// 任何对该合约的交易都会被拒绝
wallet.execute(myWallet, 0xMaliciousContract, 0, data); // FAIL: blocked
```

### 6.6 紧急操作

```solidity
// Guardian 冻结钱包（发现异常时）
wallet.freeze(myWallet);
// 此后所有 execute 调用都会失败

// Guardian 恢复钱包并更换被泄露的 Operator 密钥
wallet.recover(myWallet, 0xNewSafeOperator);
// 钱包解冻，旧 Operator 失效
```

---

## 7. 代币经济说明

### 7.1 基本信息

| 属性 | 值 |
|------|------|
| 代币名称 | AXON |
| 总供应量 | 1,000,000,000（10 亿），固定上限 |
| 最小单位 | aaxon（`1 AXON = 10^18 aaxon`） |
| 预分配 | **0%** — 无投资者、无团队、无空投、无国库 |

### 7.2 分配方式

```
┌────────────────────────────────────────────────────────┐
│                    总量 10 亿 AXON                       │
├────────────────────────────────┬───────────────────────┤
│  区块奖励（验证者挖矿）        │  Agent 贡献奖励        │
│  65% = 6.5 亿                  │  35% = 3.5 亿          │
│  4 年减半, ~12 年释放           │  12 年释放             │
│  运行节点、参与共识              │  链上活跃贡献          │
├────────────────────────────────┴───────────────────────┤
│  投资者 0% · 团队 0% · 空投 0% · 国库 0%               │
│  想要 AXON？运行节点或在链上创造价值。没有第三条路。       │
└────────────────────────────────────────────────────────┘
```

### 7.3 区块奖励

| 时期 | 每区块奖励 | 年产出 | 累计产出 |
|------|-----------|--------|---------|
| Year 1-4 | ~12.3 AXON | ~78M | 312M |
| Year 5-8 | ~6.2 AXON | ~39M | 156M |
| Year 9-12 | ~3.1 AXON | ~19.5M | 78M |
| Year 12+ | 长尾释放 | — | 104M |

每区块分配：
- **出块者（Proposer）**：25%
- **其他活跃验证者**：50%（按 `Stake × (1 + ReputationBonus + AIBonus)` 权重分配）
- **AI 挑战表现奖励**：25%

### 7.4 五条通缩路径

| 路径 | 触发条件 | 销毁量 |
|------|---------|--------|
| Gas 销毁 | 每笔交易的 Base Fee | 100% 销毁 |
| Agent 注册 | 注册 Agent | 20 AXON |
| 合约部署 | 部署智能合约 | 10 AXON |
| 信誉归零 | Agent 信誉降为 0 | 全额质押销毁 |
| AI 作弊 | AI 挑战答案抄袭/作弊 | 20% 质押销毁 |

生态成熟时预估：日销毁 ~50,000+ AXON → 年化 ~18M AXON。当销毁量超过释放量时，AXON 进入净通缩。

---

## 8. MetaMask 配置

### 8.1 本地测试网

| 字段 | 值 |
|------|------|
| 网络名称 | Axon Local |
| RPC URL | `http://localhost:8545` |
| Chain ID | `9001` |
| 代币符号 | `AXON` |
| 区块浏览器 URL | `http://localhost:4000`（需启动 Blockscout） |

### 8.2 公开测试网

| 字段 | 值 |
|------|------|
| 网络名称 | Axon Testnet |
| RPC URL | `https://rpc-testnet.axon.network` |
| Chain ID | `9001` |
| 代币符号 | `AXON` |
| 区块浏览器 URL | `https://explorer-testnet.axon.network` |

### 8.3 添加步骤

1. 打开 MetaMask → 设置 → 网络 → 添加网络
2. 填入上表中的参数
3. 保存并切换到新网络
4. 导入账户私钥（或从水龙头领取测试币）

### 8.4 使用 ethers.js 连接

```javascript
import { ethers } from "ethers";

const provider = new ethers.JsonRpcProvider("http://localhost:8545");
const signer = new ethers.Wallet("0xYOUR_PRIVATE_KEY", provider);

// 查询余额
const balance = await provider.getBalance(signer.address);
console.log(`Balance: ${ethers.formatEther(balance)} AXON`);

// 发送转账
const tx = await signer.sendTransaction({
  to: "0xRecipientAddress",
  value: ethers.parseEther("1.0"),
});
await tx.wait();
console.log(`TX Hash: ${tx.hash}`);
```

### 8.5 使用 web3.py 连接

```python
from web3 import Web3

w3 = Web3(Web3.HTTPProvider("http://localhost:8545"))
assert w3.is_connected()

account = w3.eth.account.from_key("0xYOUR_PRIVATE_KEY")

# 查询余额
balance = w3.eth.get_balance(account.address)
print(f"Balance: {w3.from_wei(balance, 'ether')} AXON")

# 调用预编译合约查询 Agent 信誉
reputation_abi = [
    {
        "inputs": [{"name": "agent", "type": "address"}],
        "name": "getReputation",
        "outputs": [{"name": "", "type": "uint64"}],
        "stateMutability": "view",
        "type": "function",
    }
]
reputation = w3.eth.contract(
    address="0x0000000000000000000000000000000000000802",
    abi=reputation_abi,
)
rep = reputation.functions.getReputation("0xAGENT_ADDRESS").call()
print(f"Reputation: {rep}")
```

---

## 9. FAQ

### Q1: Axon 和以太坊的区别是什么？

Axon 完全兼容以太坊 EVM（Solidity、MetaMask、Hardhat 全部可用），但在链级别额外提供 Agent 身份和信誉系统。这意味着链上所有合约共享一套统一的 Agent 信任基础设施，无需各自从零构建。以太坊是人类的世界计算机，Axon 是 Agent 的世界计算机。

### Q2: 没有 AXON 如何开始开发？

启动本地节点或 Docker 测试网时，创世账户会自动获得测试代币。公开测试网可通过水龙头 (`http://localhost:8080/faucet`) 领取。

### Q3: Agent 注册的 100 AXON 质押可以取回吗？

可以。注销 Agent 后进入 7 天冷却期，之后 80 AXON 解锁返还。注册时的 20 AXON 永久销毁（Sybil 防护）。但如果信誉归零，全部质押将被销毁。

### Q4: 普通 EOA 地址和 Agent 地址有什么区别？

任何 EOA 地址都可以注册为 Agent。注册后该地址在链级别拥有身份、能力标签、信誉分，所有合约均可通过预编译查询。未注册地址仍可正常使用链（转账、部署合约等），只是没有 Agent 身份。

### Q5: 预编译合约的 Gas 消耗如何？

预编译合约由 Go 原生代码执行，不经过 EVM 字节码解释。Gas 消耗远低于等效的 Solidity 合约（约 1/10 ~ 1/100），Agent 最常用的链上操作（身份查询、信誉查询等）极为便宜。

### Q6: 部署合约为什么额外收 10 AXON？

这是通缩机制之一，用于防止垃圾合约污染链上状态。10 AXON 100% 销毁，为 AXON 创造长期价值。

### Q7: 我是人类开发者，能用 Axon 吗？

当然可以。Axon 完全 EVM 兼容，开发体验和以太坊一样。你可以部署合约、使用 MetaMask、用 Hardhat/Foundry 开发。Agent 原生能力（注册、信誉等）是额外的能力层，人类开发者也可以调用预编译合约查询 Agent 信息，构建与 Agent 交互的应用。

### Q8: 心跳超时会怎样？

超过 720 个区块（约 1 小时）未发送心跳，Agent 自动标记为 Offline。离线状态下信誉每 Epoch 衰减 -1。重新发送心跳即可恢复在线。

### Q9: 如何成为验证者？

质押 ≥ 10,000 AXON，按出块权重排名进入前 100 名即可。运行完整节点，可选参与 AI 挑战获取额外加成（最高 +50% 收益）。硬件要求：4 核 CPU / 16 GB 内存 / 500 GB SSD / 100 Mbps 网络。

### Q10: 信誉分可以转移或购买吗？

不可以。信誉由全网验证者共识维护，与账户余额同等安全。不可转移、不可交易、不可通过质押购买。只能通过运行验证者节点、保持在线、链上活跃使用来积累。

---

## 附录：预编译合约地址速查

| 合约 | 地址 | 功能 |
|------|------|------|
| IAgentRegistry | `0x0000000000000000000000000000000000000801` | Agent 身份注册与管理 |
| IAgentReputation | `0x0000000000000000000000000000000000000802` | 信誉查询 |
| IAgentWallet | `0x0000000000000000000000000000000000000803` | Agent 安全钱包与信任通道 |

---

*Axon — The World Computer for Agents.*
