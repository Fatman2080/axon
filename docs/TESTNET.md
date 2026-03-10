# Axon 公开测试网部署指南

## 目录

- [网络信息](#网络信息)
- [快速开始](#快速开始)
- [Docker 部署（推荐）](#docker-部署推荐)
- [裸机部署](#裸机部署)
- [加入现有测试网](#加入现有测试网)
- [成为验证者](#成为验证者)
- [水龙头](#水龙头)
- [区块浏览器](#区块浏览器)
- [监控](#监控)
- [MetaMask 配置](#metamask-配置)
- [预编译合约](#预编译合约)
- [Python SDK 接入](#python-sdk-接入)
- [运维](#运维)
- [故障排除](#故障排除)

---

## 网络信息

| 参数 | 值 |
|------|-----|
| 链名称 | Axon Public Testnet |
| Chain ID (Cosmos) | `axon_9001-1` |
| Chain ID (EVM) | `9001` |
| 代币符号 | AXON |
| 最小单位 | aaxon (10⁻¹⁸ AXON) |
| 区块时间 | ~5 秒 |
| JSON-RPC | `http://<node-ip>:8545` |
| WebSocket | `ws://<node-ip>:8546` |
| CometBFT RPC | `http://<node-ip>:26657` |
| REST API | `http://<node-ip>:1317` |
| gRPC | `<node-ip>:9090` |
| 区块浏览器 | `http://<node-ip>:4000` |
| 水龙头 | `http://<node-ip>:8080` |
| Grafana | `http://<node-ip>:3000` |

---

## 快速开始

### 最快方式：Docker Compose 一键启动

```bash
git clone https://github.com/Fatman2080/axon.git
cd axon

# 启动完整测试网（4 验证者 + 水龙头 + 浏览器）
docker compose -f testnet/docker-compose.yml up -d

# 查看状态
docker compose -f testnet/docker-compose.yml ps

# 测试 JSON-RPC
curl -s http://localhost:8545 -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

启动后可用端点：
- **JSON-RPC**: http://localhost:8545
- **水龙头**: http://localhost:8080
- **区块浏览器**: http://localhost:4000
- **CometBFT RPC**: http://localhost:26657

---

## Docker 部署（推荐）

### 系统要求

```
CPU:    4+ 核
内存:   8+ GB（推荐 16 GB）
存储:   100 GB SSD
Docker: 24.0+
Docker Compose: v2.20+
```

### 完整测试网（4 节点 + 基础设施）

```bash
# 构建并启动
docker compose -f testnet/docker-compose.yml up -d --build

# 查看日志
docker logs -f axon-node-0

# 停止
docker compose -f testnet/docker-compose.yml down

# 完全清除数据
docker compose -f testnet/docker-compose.yml down -v
```

### 单独启动监控

```bash
# 先启动测试网，然后启动监控
docker compose -f testnet/monitoring/docker-compose.yml up -d

# Grafana: http://localhost:3000 (admin/axon)
# Prometheus: http://localhost:9091
```

### 服务架构

```
┌─────────────────────────────────────────────────────┐
│                   Docker Network                     │
│                                                      │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐      │
│  │ axon-node-0│ │ axon-node-1│ │ axon-node-2│ ...  │
│  │  :26656 P2P│ │            │ │            │      │
│  │  :26657 RPC│ │            │ │            │      │
│  │  :8545 EVM │ │            │ │            │      │
│  └─────┬──────┘ └────────────┘ └────────────┘      │
│        │                                             │
│  ┌─────┴──────┐  ┌───────────┐  ┌──────────────┐   │
│  │  Blockscout │  │  Faucet   │  │  Prometheus  │   │
│  │  :4000      │  │  :8080    │  │  + Grafana   │   │
│  └─────────────┘  └───────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────┘
```

---

## 裸机部署

### 一键部署脚本（Ubuntu 22.04+）

```bash
# 下载并运行部署脚本
curl -sSL https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/deploy-node.sh | bash

# 或自定义参数
MONIKER="my-axon-node" \
SEEDS="nodeid1@ip1:26656,nodeid2@ip2:26656" \
GENESIS_URL="https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/genesis.json" \
bash deploy-node.sh
```

部署脚本会自动：
1. 安装 Go 和系统依赖
2. 编译 axond 二进制文件
3. 初始化节点并配置创世文件
4. 配置防火墙（ufw）
5. 创建 systemd 服务

### 手动安装

```bash
# 1. 安装 Go 1.23+
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 2. 编译
git clone https://github.com/Fatman2080/axon.git
cd axon && make build
sudo cp build/axond /usr/local/bin/

# 3. 初始化
axond init my-node --chain-id axon_9001-1 --home /opt/axon

# 4. 下载创世文件（从种子节点获取）
# curl -sSL <genesis-url> -o /opt/axon/config/genesis.json

# 5. 配置 seeds/peers（编辑 /opt/axon/config/config.toml）

# 6. 启动
axond start --home /opt/axon --json-rpc.enable
```

### 管理命令

```bash
# 启动/停止/重启
sudo systemctl start axond
sudo systemctl stop axond
sudo systemctl restart axond

# 查看日志
sudo journalctl -fu axond

# 查看节点状态
curl -s localhost:26657/status | jq '.result.sync_info'

# 查看节点 ID（用于 peer 配置）
axond comet show-node-id --home /opt/axon
```

---

## 加入现有测试网

### 1. 获取创世文件

```bash
# 从种子节点下载
curl -sSL http://<seed-node-ip>:26657/genesis | jq '.result.genesis' > /opt/axon/config/genesis.json
```

### 2. 配置 Seeds

编辑 `/opt/axon/config/config.toml`：

```toml
[p2p]
seeds = "<node-id>@<ip>:26656,<node-id2>@<ip2>:26656"
```

### 3. 快速同步

```bash
# 使用状态同步（可选，加速初始同步）
# 编辑 config.toml
[statesync]
enable = true
rpc_servers = "http://<trusted-node>:26657,http://<trusted-node2>:26657"
trust_height = <recent-height>
trust_hash = "<block-hash-at-trust-height>"
```

### 4. 启动同步

```bash
sudo systemctl start axond
sudo journalctl -fu axond  # 观察同步进度
```

---

## 成为验证者

节点完成同步后，可以创建验证者：

```bash
# 1. 创建密钥
axond keys add validator --home /opt/axon

# 2. 从水龙头获取测试 AXON（或从其他账户转账）
curl -X POST http://<faucet>:8080/api/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "<your-0x-address>"}'

# 3. 创建验证者
axond tx staking create-validator \
  --amount=10000000000000000000000000aaxon \
  --pubkey=$(axond comet show-validator --home /opt/axon) \
  --moniker="my-validator" \
  --chain-id=axon_9001-1 \
  --commission-rate=0.10 \
  --commission-max-rate=0.20 \
  --commission-max-change-rate=0.01 \
  --min-self-delegation=1 \
  --from=validator \
  --home=/opt/axon

# 4. 确认验证者状态
axond query staking validator $(axond keys show validator --bech val -a --home /opt/axon) \
  --home /opt/axon
```

---

## 水龙头

### Web 界面

访问 `http://<node-ip>:8080`，输入你的 0x 地址，点击 "Request Tokens"。

### API 调用

```bash
# 请求测试代币（每 24 小时一次）
curl -X POST http://localhost:8080/api/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "0xYourAddress"}'

# 查看水龙头状态
curl http://localhost:8080/api/status

# 健康检查
curl http://localhost:8080/health
```

### 响应示例

```json
{
  "success": true,
  "tx_hash": "0xabc...123",
  "amount": "10 AXON",
  "message": "tokens sent successfully"
}
```

---

## 区块浏览器

Blockscout 区块浏览器随 Docker Compose 自动启动。

- 访问 `http://localhost:4000`
- 查看区块、交易、合约
- 验证智能合约代码

---

## 监控

### Grafana 仪表盘

随监控栈一起启动，默认访问：

- **Grafana**: http://localhost:3000（用户名: `admin`，密码: `axon`）
- **Prometheus**: http://localhost:9091

仪表盘包含：
- 区块高度实时趋势
- 连接的 Peer 数量
- 出块速率
- 共识轮次
- 内存池大小
- 区块大小
- 交易吞吐量

### 监控指标端点

每个节点暴露 Prometheus 指标：

```bash
curl http://localhost:26660/metrics
```

---

## MetaMask 配置

| 参数 | 值 |
|------|-----|
| 网络名称 | Axon Testnet |
| RPC URL | `http://<node-ip>:8545` |
| Chain ID | `9001` |
| 货币符号 | AXON |
| 区块浏览器 | `http://<node-ip>:4000` |

---

## 预编译合约

Axon 链提供 3 个原生预编译合约，所有 Solidity 合约均可直接调用：

| 合约 | 地址 | 功能 |
|------|------|------|
| IAgentRegistry | `0x0000000000000000000000000000000000000801` | Agent 注册、查询、心跳、注销 |
| IAgentReputation | `0x0000000000000000000000000000000000000802` | 信誉查询、批量查询、阈值判断 |
| IAgentWallet | `0x0000000000000000000000000000000000000803` | Agent 安全钱包（创建、执行、冻结、恢复） |

### Solidity 调用示例

```solidity
interface IAgentRegistry {
    function isAgent(address agent) external view returns (bool);
    function getAgent(address agent) external view returns (
        string memory capabilities, string memory model,
        uint256 reputation, uint256 stake, uint8 status
    );
    function register(string calldata capabilities, string calldata model) external payable;
    function heartbeat() external;
    function deregister() external;
}

IAgentRegistry registry = IAgentRegistry(0x0000000000000000000000000000000000000801);
bool isRegistered = registry.isAgent(someAddress);
```

---

## Python SDK 接入

```bash
pip install -e sdk/python
```

```python
from axon import AgentClient

client = AgentClient("http://localhost:8545")
print(f"Chain ID: {client.chain_id()}")
print(f"Block:    {client.block_number()}")

# 创建账户并注册 Agent
client.create_account()
client.register_agent("coding,analysis", "gpt-4", stake_axon=100)

# 查询信誉
rep = client.get_reputation(client.account.address)
print(f"Reputation: {rep}")
```

---

## 运维

### 备份

```bash
# 停止节点
sudo systemctl stop axond

# 备份数据
tar -czf axon-backup-$(date +%Y%m%d).tar.gz /opt/axon/data/

# 重启节点
sudo systemctl start axond
```

### 升级

```bash
sudo systemctl stop axond

cd /tmp && git clone --depth 1 https://github.com/Fatman2080/axon.git
cd axon && make build
sudo cp build/axond /usr/local/bin/axond

sudo systemctl start axond
```

### 日志轮转

```bash
# /etc/logrotate.d/axond
/var/log/axond.log {
    daily
    rotate 14
    compress
    missingok
    notifempty
}
```

---

## 故障排除

### 节点无法启动

```bash
# 检查日志
sudo journalctl -fu axond --no-pager -n 50

# 验证创世文件
axond genesis validate-genesis --home /opt/axon

# 重置数据（保留密钥和创世）
axond comet unsafe-reset-all --home /opt/axon
```

### 节点不出块

```bash
# 检查同步状态
curl -s localhost:26657/status | jq '.result.sync_info.catching_up'
# true = 正在同步，等待完成

# 检查 peer 连接
curl -s localhost:26657/net_info | jq '.result.n_peers'
```

### JSON-RPC 无响应

```bash
# 确认 JSON-RPC 已启用
curl -s localhost:8545 -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'

# 检查端口是否监听
ss -tlnp | grep 8545
```

### Docker 容器启动失败

```bash
# 查看容器日志
docker logs axon-node-0

# 重新构建镜像
docker compose -f testnet/docker-compose.yml build --no-cache

# 完全清除重启
docker compose -f testnet/docker-compose.yml down -v
docker compose -f testnet/docker-compose.yml up -d
```

### 端口被占用

```bash
# 查找占用进程
sudo lsof -i :26657
sudo lsof -i :8545

# 终止占用进程
sudo kill <PID>
```
