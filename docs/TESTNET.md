# Axon Testnet 部署指南

## 网络信息

| 参数 | 值 |
|------|-----|
| 链名称 | Axon Testnet |
| Chain ID | 9001 |
| 代币符号 | AXON |
| 代币精度 | 18 (aaxon) |
| 区块时间 | ~5 秒 |
| JSON-RPC | http://\<node-ip\>:8545 |
| WebSocket | ws://\<node-ip\>:8546 |
| CometBFT RPC | http://\<node-ip\>:26657 |
| 区块浏览器 | http://\<node-ip\>:4000 |

## MetaMask 配置

添加自定义网络:

- **网络名称**: Axon Testnet
- **RPC URL**: http://\<node-ip\>:8545
- **Chain ID**: 9001
- **货币符号**: AXON
- **区块浏览器 URL**: http://\<node-ip\>:4000

## 验证者部署

### 1. 系统要求

```
CPU:    4+ 核
内存:   16+ GB
存储:   500 GB SSD
网络:   100 Mbps
系统:   Ubuntu 22.04+ / macOS
```

### 2. 安装

```bash
# 克隆仓库
git clone https://github.com/Fatman2080/axon.git
cd axon

# 安装 Go 1.23+
# https://go.dev/dl/

# 编译
make build
```

### 3. 初始化单节点

```bash
bash scripts/local_node.sh
```

### 4. 启动节点

```bash
./build/axond start \
  --home ~/.axond \
  --chain-id axon_9001-1 \
  --json-rpc.enable \
  --json-rpc.address 0.0.0.0:8545
```

### 5. 多节点测试网

```bash
# 初始化 4 节点
bash scripts/localnet.sh

# 启动所有节点
~/.axon-localnet/start_all.sh

# 停止所有节点
~/.axon-localnet/stop_all.sh
```

## 水龙头

测试网水龙头合约部署后，可通过以下方式获取测试 AXON:

### 通过合约调用

```javascript
// ethers.js
const faucet = new ethers.Contract(FAUCET_ADDRESS, FAUCET_ABI, signer);
await faucet.drip(); // 获取 10 AXON，每 24 小时一次
```

### 通过 Python SDK

```python
from axon import AgentClient

client = AgentClient("http://<node-ip>:8545")
client.set_account("your_private_key")
client.send_contract_tx(FAUCET_ADDRESS, FAUCET_ABI, "drip")
```

## 预编译合约地址

| 合约 | 地址 | 功能 |
|------|------|------|
| IAgentRegistry | `0x...0801` | Agent 注册/查询/心跳 |
| IAgentReputation | `0x...0802` | 信誉查询 |
| IAgentWallet | `0x...0803` | Agent 安全钱包 |

## 区块浏览器

```bash
cd explorer
docker-compose up -d
# 访问 http://localhost:4000
```

## 故障排除

### 节点启动失败

```bash
# 清除数据重新初始化
rm -rf ~/.axond
bash scripts/local_node.sh
```

### 端口被占用

```bash
# 查找并终止占用进程
lsof -i :26657
kill <PID>
```

### Genesis 验证失败

```bash
./build/axond genesis validate-genesis --home ~/.axond
```
