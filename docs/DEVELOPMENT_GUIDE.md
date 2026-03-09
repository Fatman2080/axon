# Axon 开发说明书

## 总览

本文档将 Axon 公链的全部开发工作拆分为 **6 个阶段、28 个独立任务**。每个任务自包含，有明确的输入、输出和验收标准。按顺序逐一完成即可。

```
Phase 0  环境搭建              3 个任务     ← 你在这里
Phase 1  链骨架可运行           5 个任务
Phase 2  Agent 模块完整实现     6 个任务
Phase 3  EVM 集成 + 预编译      5 个任务
Phase 4  经济模型实现           4 个任务
Phase 5  测试网 + SDK           5 个任务
```

---

## 当前状态（已完成）

```
✅ 白皮书 v1.0（docs/whitepaper.md）
✅ 项目目录结构
✅ go.mod 依赖定义
✅ Protobuf 消息定义（proto/axon/agent/v1/）
✅ x/agent 模块骨架代码（types, keeper, module）
✅ EVM 预编译合约骨架（precompiles/）
✅ Solidity 接口文件（contracts/interfaces/）
✅ App 入口 + CMD 骨架
✅ Makefile, README, .gitignore, CONTRIBUTING
```

---

## Phase 0 — 环境搭建

### 任务 0.1：开发环境配置

```
目标：本机可以编译 Go + Cosmos SDK 项目
依赖：无
预计：1 小时

步骤：
  1. 安装 Go 1.23+（brew install go）
  2. 安装 buf（protobuf 工具，brew install bufbuild/buf/buf）
  3. 安装 golangci-lint（brew install golangci-lint）
  4. 安装 Docker（后续测试网需要）
  5. 确认：go version / buf --version 输出正确

验收：
  □ go version 显示 1.23+
  □ buf --version 正常
  □ make 命令可用
```

### 任务 0.2：初始化 Git 仓库

```
目标：代码进入版本控制
依赖：0.1
预计：15 分钟

步骤：
  1. cd axon/
  2. git init
  3. git add .
  4. git commit -m "initial: project structure and module skeleton"
  5. （可选）创建 GitHub 仓库并 push

验收：
  □ git log 显示首次提交
  □ .gitignore 正确排除 build/ 和 vendor/
```

### 任务 0.3：Protobuf 代码生成

```
目标：从 .proto 文件生成 Go 代码，模块类型可编译
依赖：0.1
预计：2 小时

步骤：
  1. 创建 proto/buf.yaml 配置文件
  2. 创建 proto/buf.gen.yaml 代码生成配置
  3. 运行 buf generate proto
  4. 生成的 .pb.go 文件出现在 x/agent/types/ 下
  5. 确认 go build ./x/agent/types/ 编译通过

输出文件：
  x/agent/types/agent.pb.go
  x/agent/types/params.pb.go
  x/agent/types/genesis.pb.go
  x/agent/types/tx.pb.go
  x/agent/types/query.pb.go
  x/agent/types/tx_grpc.pb.go
  x/agent/types/query_grpc.pb.go

验收：
  □ buf generate 无报错
  □ go build ./x/agent/types/ 通过
  □ 生成的 .pb.go 文件数量 ≥ 7
```

---

## Phase 1 — 链骨架可运行

### 任务 1.1：集成 Cosmos EVM 到 app.go

```
目标：App 同时包含 Cosmos SDK 核心模块 + EVM 模块
依赖：0.3
预计：4-6 小时

步骤：
  1. 参考 github.com/cosmos/evm/example_chain 的 app.go
  2. 在 app/app.go 中注册以下模块：
     - Cosmos SDK: auth, bank, staking, distribution, gov,
                   slashing, params, consensus, genutil, upgrade
     - Cosmos EVM: evm, feemarket
     - 自定义: x/agent
  3. 配置 store keys、keepers、module manager
  4. 设置 BeginBlocker / EndBlocker 顺序
  5. 配置 JSON-RPC 服务器

关键参考：
  https://pkg.go.dev/github.com/cosmos/evm/example_chain
  https://evm.cosmos.network/docs/documentation/integration/

验收：
  □ go build ./app/ 编译通过
  □ 所有 Keeper 正确初始化
  □ ModuleManager 包含全部模块
```

### 任务 1.2：完善 CMD 入口

```
目标：axond 二进制文件支持标准 Cosmos 命令
依赖：1.1
预计：2-3 小时

步骤：
  1. 在 cmd/axond/main.go 中接入 Cosmos server commands
  2. 注册 init / start / tx / query 等子命令
  3. 注册 x/agent 的 CLI 命令（tx agent register 等）
  4. 注册 EVM 的 JSON-RPC 命令
  5. 配置 Bech32 前缀（axon / axonvaloper）

验收：
  □ make build 生成 build/axond
  □ axond version 输出正确
  □ axond init test --chain-id axon-local-1 正常执行
```

### 任务 1.3：创世状态配置

```
目标：可以生成有效的创世文件
依赖：1.2
预计：2 小时

步骤：
  1. 定义创世参数：
     - 区块时间 5s
     - 验证者上限 100
     - 最低质押 10,000 AXON
     - Agent 注册质押 100 AXON
     - Epoch 长度 720 块
  2. 在 app.go 中设置 DefaultGenesis
  3. 创建创世账户脚本
  4. 配置 EIP-1559 Gas 参数

验收：
  □ axond init 生成 genesis.json
  □ genesis.json 包含 x/agent 模块的默认参数
  □ genesis.json 包含 EVM 模块配置
```

### 任务 1.4：单节点启动

```
目标：单个节点可以启动并出块
依赖：1.3
预计：2-3 小时

步骤：
  1. 初始化节点：axond init test-node --chain-id axon-local-1
  2. 创建验证者账户
  3. 添加创世账户（带初始余额）
  4. 生成创世交易（gentx）
  5. 收集创世交易
  6. 启动节点：axond start
  7. 确认区块正常产出

验收：
  □ axond start 后节点运行
  □ 日志显示区块递增
  □ curl localhost:26657/status 返回节点信息
  □ 区块时间 ~5 秒
```

### 任务 1.5：EVM JSON-RPC 验证

```
目标：以太坊工具可以连接
依赖：1.4
预计：1-2 小时

步骤：
  1. 确认 JSON-RPC 端口 8545 开放
  2. 测试 eth_chainId 返回正确 chain ID
  3. 测试 eth_blockNumber 返回递增区块号
  4. MetaMask 添加自定义网络并连接
  5. 测试简单 ETH 转账（AXON 转账）

验收：
  □ curl -X POST localhost:8545 -d '{"method":"eth_chainId"}' 有响应
  □ MetaMask 可以连接
  □ 可以发送原生代币转账
```

---

## Phase 2 — Agent 模块完整实现

### 任务 2.1：Agent 注册与注销

```
目标：Agent 可以链上注册/注销
依赖：1.4
预计：3-4 小时

步骤：
  1. 确认 msg_server.go 中 Register 逻辑正确
  2. 实现质押锁定 + 20 AXON 销毁
  3. 实现 Deregister 退还质押（扣除已销毁部分）
  4. 添加 7 天冷却期逻辑
  5. 编写单元测试

测试命令：
  axond tx agent register --capabilities "coding" --model "gpt-4" --stake 100axon
  axond query agent agent <address>
  axond tx agent deregister

验收：
  □ 注册成功，链上可查询到 Agent
  □ 注册时 20 AXON 被销毁（总供应量减少）
  □ 注销后质押退还（减去销毁部分）
  □ 单元测试通过
```

### 任务 2.2：心跳机制

```
目标：Agent 通过心跳维持在线状态
依赖：2.1
预计：2-3 小时

步骤：
  1. 实现 Heartbeat 消息处理（已有骨架）
  2. 在 BeginBlocker 中检测心跳超时
  3. 超时 Agent 标记为 Offline，信誉 -1
  4. 编写测试：模拟超时场景

验收：
  □ Agent 发送心跳后 status = Online
  □ 超过 720 块不心跳，自动变为 Offline
  □ Offline 时信誉自动扣减
```

### 任务 2.3：信誉系统

```
目标：信誉按规则自动变化
依赖：2.2
预计：3-4 小时

步骤：
  1. 实现信誉增加规则：
     - 验证者正常出块：每 Epoch +1
     - 持续在线心跳：每 1000 块 +1
     - 链上活跃（≥10 笔/Epoch）：+0.5
  2. 实现信誉减少规则：
     - 验证者离线/未出块：-5
     - Slashing（双签）：-50
     - 长时间无心跳：每 Epoch -1
  3. 信誉上限 100，下限 0
  4. 信誉归零时质押 100% 销毁

验收：
  □ 信誉在 0-100 范围内正确变化
  □ 验证者出块获得信誉加分
  □ 信誉归零触发质押销毁
  □ 信誉不可转让/购买
```

### 任务 2.4：AI 挑战 — 出题与提交

```
目标：链能自动出题，验证者能提交答案
依赖：2.3
预计：4-6 小时

步骤：
  1. 创建 AI 挑战题库（初始 100 道题）
     - 文本分类、逻辑推理、代码分析、知识问答
     - 每题有标准答案或评分规则
  2. 每 Epoch 开始时，链从题库随机抽题
  3. 实现 Commit 阶段：验证者在 50 块内提交答案哈希
  4. 实现 Reveal 阶段：截止后揭示明文答案
  5. 存储每个 Epoch 的挑战和响应

验收：
  □ 每个 Epoch 自动生成一个挑战
  □ 验证者可以提交 commit hash
  □ 验证者可以 reveal 明文
  □ reveal 与 commit 不匹配时拒绝
```

### 任务 2.5：AI 挑战 — 评估与计分

```
目标：答案被评估，AIBonus 影响出块权重
依赖：2.4
预计：4-6 小时

步骤：
  1. Epoch 结束时触发评估：
     - 确定性题目：比对标准答案
     - 开放性题目：验证者交叉评分取中位数
  2. 计算 AIBonus：
     - 优秀 → 15-30%
     - 一般 → 5-10%
     - 未参与 → 0%
     - 明显错误 → -5%
  3. AIBonus 写入验证者状态
  4. 修改出块权重：Stake × (1 + ReputationBonus + AIBonus)

验收：
  □ Epoch 结束时自动评估
  □ AIBonus 正确写入
  □ 出块权重受 AIBonus 影响
  □ 高 AIBonus 验证者出块频率更高
```

### 任务 2.6：Agent CLI 命令

```
目标：完整的 CLI 交互
依赖：2.1 - 2.5
预计：2-3 小时

步骤：
  1. 实现 x/agent/client/cli/tx.go：
     - axond tx agent register
     - axond tx agent update
     - axond tx agent heartbeat
     - axond tx agent deregister
     - axond tx agent submit-challenge
     - axond tx agent reveal-challenge
  2. 实现 x/agent/client/cli/query.go：
     - axond query agent params
     - axond query agent agent <addr>
     - axond query agent agents
     - axond query agent reputation <addr>
     - axond query agent challenge

验收：
  □ 所有 tx 子命令可执行
  □ 所有 query 子命令返回正确数据
  □ --help 显示用法说明
```

---

## Phase 3 — EVM 集成 + 预编译合约

### 任务 3.1：预编译合约 — IAgentRegistry 完整实现

```
目标：Solidity 合约可以调用 isAgent / getAgent
依赖：2.1
预计：4-6 小时

步骤：
  1. 在 precompiles/registry/registry.go 中：
     - 从 EVM 调用参数中提取 Cosmos SDK Context
     - isAgent：调用 keeper.IsAgent()，打包返回 bool
     - getAgent：调用 keeper.GetAgent()，打包返回全部字段
     - register：解析 msg.value 作为质押，调用 keeper
     - heartbeat：调用 keeper
     - deregister：调用 keeper
  2. 在 app.go 中注册预编译到 EVM 模块
  3. 编写 Solidity 测试合约调用预编译

验收：
  □ Solidity 合约调用 IAgentRegistry(0x..0801).isAgent() 返回正确
  □ 通过预编译注册 Agent 等同于 CLI 注册
  □ Gas 消耗显著低于普通合约调用
```

### 任务 3.2：预编译合约 — IAgentReputation 完整实现

```
目标：Solidity 合约可以查询信誉
依赖：2.3, 3.1
预计：2-3 小时

步骤：
  1. getReputation：查询单个 Agent 信誉
  2. getReputations：批量查询
  3. meetsReputation：判断是否达标
  4. 注册到 EVM

验收：
  □ Solidity 合约可查询任意 Agent 信誉
  □ 批量查询正常工作
  □ meetsReputation 阈值判断正确
```

### 任务 3.3：预编译合约 — IAgentWallet 完整实现

```
目标：Agent 可以创建并使用安全钱包
依赖：3.1
预计：6-8 小时（最复杂的预编译）

步骤：
  1. 实现钱包状态存储（KV Store 或 EVM 状态）
  2. createWallet：创建钱包，设定规则
  3. execute：检查限额/冷却，执行转账
  4. freeze：Guardian 冻结
  5. recover：Guardian 更换操作密钥
  6. getWalletInfo：查询状态
  7. 日限额重置逻辑（每 17,280 块 ≈ 24 小时）

验收：
  □ Agent 可创建钱包并设定限额
  □ 超限额交易被拒绝
  □ Guardian 可以冻结和恢复
  □ 日限额每天自动重置
```

### 任务 3.4：EVM 兼容性完整测试

```
目标：标准以太坊工具全部可用
依赖：3.1 - 3.3
预计：3-4 小时

步骤：
  1. 使用 Hardhat 部署 ERC-20 合约
  2. 使用 Foundry 部署并测试合约
  3. 使用 MetaMask 发送交易
  4. 使用 ethers.js / web3.py 交互
  5. 测试预编译合约在 Solidity 中的调用
  6. 验证 EIP-1559 Gas 机制

验收：
  □ Hardhat deploy 成功
  □ Foundry forge test 通过
  □ MetaMask 转账成功
  □ ethers.js 脚本执行正常
  □ ERC-20 合约运行正常
```

### 任务 3.5：合约部署销毁机制

```
目标：部署合约时额外销毁 10 AXON
依赖：3.4
预计：2-3 小时

步骤：
  1. Hook 进 EVM 合约创建流程
  2. 检测 CREATE / CREATE2 操作码
  3. 从部署者余额扣除 10 AXON 并销毁
  4. 余额不足时拒绝部署

验收：
  □ 部署合约后总供应量减少 10 AXON
  □ 余额不足 10 AXON 时部署失败
  □ 普通转账不受影响
```

---

## Phase 4 — 经济模型实现

### 任务 4.1：区块奖励分配

```
目标：验证者按规则获得区块奖励
依赖：2.5
预计：4-6 小时

步骤：
  1. 创建 x/rewards 模块或在 x/agent 中实现
  2. 每区块铸造奖励：
     - Year 1-4: ~12.3 AXON/block
     - 4 年减半逻辑
  3. 分配规则：
     - Proposer 25%
     - 其他活跃验证者 50%（按权重）
     - AI 挑战表现 25%
  4. 权重计算：Stake × (1 + ReputationBonus + AIBonus)
  5. 实现 Mint 模块参数

验收：
  □ 每个区块正确铸造 AXON
  □ Proposer 获得 25%
  □ 权重高的验证者获得更多
  □ 减半逻辑在正确区块高度生效
```

### 任务 4.2：Agent 贡献奖励引擎

```
目标：活跃 Agent 自动获得贡献奖励
依赖：2.3, 4.1
预计：6-8 小时（最复杂的经济逻辑）

步骤：
  1. 定义贡献评分指标：
     - 部署合约数 × 权重
     - 合约被调用次数 × 权重
     - 交易活跃度 × 权重
     - 信誉分 × 权重
  2. 防刷机制：
     - 自调用不计分
     - 单 Agent 上限 = Epoch 池的 2%
     - 信誉 < 20 不参与
     - 注册 < 7 天不参与
  3. 每 Epoch 结束时计算并分配
  4. Year 1-4: ~35M/year 释放速度

验收：
  □ 活跃 Agent 每 Epoch 获得奖励
  □ 不活跃 Agent 获得 0
  □ 防刷机制有效（自调用不得分）
  □ 单 Agent 上限生效
```

### 任务 4.3：Gas 销毁（EIP-1559）

```
目标：Base Fee 100% 销毁
依赖：3.4
预计：2-3 小时

步骤：
  1. 确认 Cosmos EVM 的 FeeMarket 模块配置
  2. 设置 Base Fee → 100% 销毁（不进验证者）
  3. 设置 Priority Fee → 100% 给 Proposer
  4. 验证总供应量随交易减少

验收：
  □ 每笔交易的 Base Fee 被销毁
  □ 总供应量查询值递减
  □ Priority Fee 正确给到出块者
```

### 任务 4.4：通缩机制集成测试

```
目标：所有销毁路径正确工作
依赖：4.1 - 4.3, 3.5
预计：2-3 小时

步骤：
  逐一验证：
  1. Gas 销毁 ✓
  2. Agent 注册销毁 20 AXON ✓
  3. 合约部署销毁 10 AXON ✓
  4. 信誉归零质押全额销毁 ✓
  5. AI 挑战作弊惩罚销毁 ✓
  查询 total supply 确认每次都减少

验收：
  □ 5 种销毁路径全部验证
  □ total supply 正确反映所有销毁
  □ 编写自动化测试覆盖所有路径
```

---

## Phase 5 — 测试网 + SDK

### 任务 5.1：本地多节点测试网

```
目标：4 个节点组成本地网络
依赖：Phase 1-4 全部完成
预计：4-6 小时

步骤：
  1. 编写 scripts/localnet.sh
  2. 生成 4 个验证者密钥
  3. 配置 persistent_peers
  4. 启动 4 节点网络
  5. 验证出块、共识、交易

验收：
  □ 4 个节点正常出块
  □ 交易在所有节点同步
  □ 停掉 1 个节点（< 1/3），网络继续运行
  □ Agent 注册/查询跨节点一致
```

### 任务 5.2：区块浏览器

```
目标：可视化查看链上数据
依赖：5.1
预计：2-4 小时

步骤：
  1. 部署 Blockscout（开源 EVM 区块浏览器）
  2. 配置连接本地 JSON-RPC
  3. 验证区块、交易、合约显示正确
  4.（可选）自定义 Agent 信息展示页面

验收：
  □ 浏览器显示区块列表
  □ 可以查看交易详情
  □ 可以查看合约代码
```

### 任务 5.3：Python Agent SDK

```
目标：Agent 可以用 Python 接入链
依赖：5.1
预计：4-6 小时

步骤：
  1. 创建 sdk/python/axon/ 包
  2. 实现 AgentClient 类：
     - connect(rpc_url)
     - register_agent(capabilities, model, stake)
     - heartbeat()
     - query_agent(address)
     - query_reputation(address)
     - deploy_contract(bytecode)
     - call_contract(address, method, args)
  3. 底层使用 web3.py 与 JSON-RPC 交互
  4. 编写示例脚本

验收：
  □ pip install -e sdk/python 安装成功
  □ Python 脚本可以注册 Agent
  □ Python 脚本可以查询信誉
  □ Python 脚本可以部署合约
```

### 任务 5.4：自动化测试套件

```
目标：关键路径有测试覆盖
依赖：5.1
预计：4-6 小时

步骤：
  1. x/agent 模块单元测试（Go）：
     - 注册/注销
     - 心跳/超时
     - 信誉变化
     - AI 挑战 commit/reveal
  2. 预编译合约测试（Solidity + Go）
  3. 集成测试（多节点场景）
  4. 目标覆盖率 > 70%

验收：
  □ make test 全部通过
  □ make test-cover 显示 > 70%
  □ CI（GitHub Actions）自动运行
```

### 任务 5.5：公开测试网部署

```
目标：外部可以连接的测试网
依赖：5.1 - 5.4
预计：4-8 小时

步骤：
  1. 准备 3-5 台云服务器（或用 Akash）
  2. 部署验证者节点
  3. 配置种子节点和持久对等节点
  4. 开放 RPC / JSON-RPC / P2P 端口
  5. 部署 Blockscout 浏览器
  6. 编写接入文档
  7. 水龙头合约（领取测试 AXON）

验收：
  □ 外部节点可以同步
  □ MetaMask 可以连接测试网
  □ 水龙头可以领测试币
  □ Agent 注册全流程可走通
```

---

## 任务依赖关系图

```
Phase 0
  0.1 环境搭建
   ├→ 0.2 Git 初始化
   └→ 0.3 Protobuf 生成
        │
Phase 1  │
        ↓
  1.1 集成 Cosmos EVM
   → 1.2 CMD 入口
    → 1.3 创世配置
     → 1.4 单节点启动 ←── 里程碑：链能跑 ──
      → 1.5 EVM 验证
        │
Phase 2  │
        ↓
  2.1 Agent 注册/注销
   → 2.2 心跳机制
    → 2.3 信誉系统
     → 2.4 AI 挑战-提交
      → 2.5 AI 挑战-评估
       → 2.6 CLI 命令
        │
Phase 3  │（与 Phase 2 部分并行）
        ↓
  3.1 预编译-Registry
   → 3.2 预编译-Reputation
    → 3.3 预编译-Wallet
     → 3.4 EVM 兼容测试
      → 3.5 合约部署销毁
        │
Phase 4  │
        ↓
  4.1 区块奖励
   → 4.2 贡献奖励引擎
    → 4.3 Gas 销毁
     → 4.4 通缩集成测试 ←── 里程碑：经济模型完整 ──
        │
Phase 5  │
        ↓
  5.1 多节点测试网
   ├→ 5.2 区块浏览器
   ├→ 5.3 Python SDK
   ├→ 5.4 自动化测试
   └→ 5.5 公开测试网 ←── 里程碑：对外开放 ──
```

---

## 工作量估算

```
Phase 0    环境搭建         ~4 小时
Phase 1    链骨架可运行     ~12-16 小时
Phase 2    Agent 模块       ~18-26 小时
Phase 3    EVM + 预编译     ~17-24 小时
Phase 4    经济模型         ~14-20 小时
Phase 5    测试网 + SDK     ~18-30 小时
─────────────────────────────────────
总计                        ~83-120 小时

按每天 6 小时有效开发：14-20 个工作日
按每天 4 小时兼职开发：21-30 个工作日
```

---

## 使用方式

每次开发时：

1. 打开本文档，找到当前任务
2. 告诉 AI："执行任务 X.X"
3. AI 按照任务描述完成开发
4. 验收通过后，进入下一个任务

这样每次对话的上下文很小，不会混乱。
