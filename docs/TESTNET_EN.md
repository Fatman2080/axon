> 🌐 [中文版](TESTNET.md)

# Axon Public Testnet Deployment Guide

## Table of Contents

- [Network Information](#network-information)
- [Quick Start](#quick-start)
- [Docker Deployment (Recommended)](#docker-deployment-recommended)
- [Bare Metal Deployment](#bare-metal-deployment)
- [Joining an Existing Testnet](#joining-an-existing-testnet)
- [Becoming a Validator](#becoming-a-validator)
- [Faucet](#faucet)
- [Block Explorer](#block-explorer)
- [Monitoring](#monitoring)
- [MetaMask Configuration](#metamask-configuration)
- [Precompile Contracts](#precompile-contracts)
- [Python SDK Integration](#python-sdk-integration)
- [Operations](#operations)
- [Troubleshooting](#troubleshooting)

---

## Network Information

| Parameter | Value |
|-----------|-------|
| Chain Name | Axon Public Testnet |
| Chain ID (Cosmos) | `axon_9001-1` |
| Chain ID (EVM) | `9001` |
| Token Symbol | AXON |
| Smallest Unit | aaxon (10⁻¹⁸ AXON) |
| Block Time | ~5 seconds |
| JSON-RPC | `http://<node-ip>:8545` |
| WebSocket | `ws://<node-ip>:8546` |
| CometBFT RPC | `http://<node-ip>:26657` |
| REST API | `http://<node-ip>:1317` |
| gRPC | `<node-ip>:9090` |
| Block Explorer | `http://<node-ip>:4000` |
| Faucet | `http://<node-ip>:8080` |
| Grafana | `http://<node-ip>:3000` |

---

## Quick Start

### Fastest Method: Docker Compose One-Click Launch

```bash
git clone https://github.com/Fatman2080/axon.git
cd axon

# Start the full testnet (4 validators + faucet + explorer)
docker compose -f testnet/docker-compose.yml up -d

# Check status
docker compose -f testnet/docker-compose.yml ps

# Test JSON-RPC
curl -s http://localhost:8545 -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

Available endpoints after launch:
- **JSON-RPC**: http://localhost:8545
- **Faucet**: http://localhost:8080
- **Block Explorer**: http://localhost:4000
- **CometBFT RPC**: http://localhost:26657

---

## Docker Deployment (Recommended)

### System Requirements

```
CPU:     4+ cores
Memory:  8+ GB (16 GB recommended)
Storage: 100 GB SSD
Docker:  24.0+
Docker Compose: v2.20+
```

### Full Testnet (4 Nodes + Infrastructure)

```bash
# Build and start
docker compose -f testnet/docker-compose.yml up -d --build

# View logs
docker logs -f axon-node-0

# Stop
docker compose -f testnet/docker-compose.yml down

# Complete data cleanup
docker compose -f testnet/docker-compose.yml down -v
```

### Start Monitoring Separately

```bash
# Start testnet first, then start monitoring
docker compose -f testnet/monitoring/docker-compose.yml up -d

# Grafana: http://localhost:3000 (admin/axon)
# Prometheus: http://localhost:9091
```

### Service Architecture

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

## Bare Metal Deployment

### One-Click Deployment Script (Ubuntu 22.04+)

```bash
# Download and run the deployment script
curl -sSL https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/deploy-node.sh | bash

# Or customize parameters
MONIKER="my-axon-node" \
SEEDS="nodeid1@ip1:26656,nodeid2@ip2:26656" \
GENESIS_URL="https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/genesis.json" \
bash deploy-node.sh
```

The deployment script automatically:
1. Installs Go and system dependencies
2. Compiles the axond binary
3. Initializes the node and configures the genesis file
4. Configures the firewall (ufw)
5. Creates a systemd service

### Manual Installation

```bash
# 1. Install Go 1.23+
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 2. Compile
git clone https://github.com/Fatman2080/axon.git
cd axon && make build
sudo cp build/axond /usr/local/bin/

# 3. Initialize
axond init my-node --chain-id axon_9001-1 --home /opt/axon

# 4. Download genesis file (obtain from seed node)
# curl -sSL <genesis-url> -o /opt/axon/config/genesis.json

# 5. Configure seeds/peers (edit /opt/axon/config/config.toml)

# 6. Start
axond start --home /opt/axon --json-rpc.enable
```

### Management Commands

```bash
# Start/Stop/Restart
sudo systemctl start axond
sudo systemctl stop axond
sudo systemctl restart axond

# View logs
sudo journalctl -fu axond

# Check node status
curl -s localhost:26657/status | jq '.result.sync_info'

# Get node ID (used for peer configuration)
axond comet show-node-id --home /opt/axon
```

---

## Joining an Existing Testnet

### 1. Obtain the Genesis File

```bash
# Download from a seed node
curl -sSL http://<seed-node-ip>:26657/genesis | jq '.result.genesis' > /opt/axon/config/genesis.json
```

### 2. Configure Seeds

Edit `/opt/axon/config/config.toml`:

```toml
[p2p]
seeds = "<node-id>@<ip>:26656,<node-id2>@<ip2>:26656"
```

### 3. Fast Sync

```bash
# Use state sync (optional, speeds up initial sync)
# Edit config.toml
[statesync]
enable = true
rpc_servers = "http://<trusted-node>:26657,http://<trusted-node2>:26657"
trust_height = <recent-height>
trust_hash = "<block-hash-at-trust-height>"
```

### 4. Start Syncing

```bash
sudo systemctl start axond
sudo journalctl -fu axond  # Monitor sync progress
```

---

## Becoming a Validator

After your node finishes syncing, you can create a validator:

```bash
# 1. Create keys
axond keys add validator --home /opt/axon

# 2. Get test AXON from the faucet (or transfer from another account)
curl -X POST http://<faucet>:8080/api/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "<your-0x-address>"}'

# 3. Create validator
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

# 4. Confirm validator status
axond query staking validator $(axond keys show validator --bech val -a --home /opt/axon) \
  --home /opt/axon
```

---

## Faucet

### Web Interface

Visit `http://<node-ip>:8080`, enter your 0x address, and click "Request Tokens".

### API Calls

```bash
# Request test tokens (once every 24 hours)
curl -X POST http://localhost:8080/api/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "0xYourAddress"}'

# Check faucet status
curl http://localhost:8080/api/status

# Health check
curl http://localhost:8080/health
```

### Response Example

```json
{
  "success": true,
  "tx_hash": "0xabc...123",
  "amount": "10 AXON",
  "message": "tokens sent successfully"
}
```

---

## Block Explorer

The Blockscout block explorer starts automatically with Docker Compose.

- Visit `http://localhost:4000`
- View blocks, transactions, and contracts
- Verify smart contract code

---

## Monitoring

### Grafana Dashboard

Launched together with the monitoring stack; default access:

- **Grafana**: http://localhost:3000 (username: `admin`, password: `axon`)
- **Prometheus**: http://localhost:9091

Dashboard includes:
- Block height real-time trend
- Connected peer count
- Block production rate
- Consensus rounds
- Mempool size
- Block size
- Transaction throughput

### Metrics Endpoint

Each node exposes Prometheus metrics:

```bash
curl http://localhost:26660/metrics
```

---

## MetaMask Configuration

| Parameter | Value |
|-----------|-------|
| Network Name | Axon Testnet |
| RPC URL | `http://<node-ip>:8545` |
| Chain ID | `9001` |
| Currency Symbol | AXON |
| Block Explorer | `http://<node-ip>:4000` |

---

## Precompile Contracts

The Axon chain provides 3 native precompile contracts, callable from any Solidity contract:

| Contract | Address | Function |
|----------|---------|----------|
| IAgentRegistry | `0x0000000000000000000000000000000000000801` | Agent registration, query, heartbeat, deregistration |
| IAgentReputation | `0x0000000000000000000000000000000000000802` | Reputation query, batch query, threshold checks |
| IAgentWallet | `0x0000000000000000000000000000000000000803` | Agent secure wallet (create, execute, freeze, recover) |

### Solidity Call Example

```solidity
interface IAgentRegistry {
    function isAgent(address agent) external view returns (bool);
    function getAgent(address agent) external view returns (
        string memory capabilities, string memory model,
        uint256 reputation, uint256 stake, uint8 status
    );
    function register(
        string calldata capabilities,
        string calldata model,
        uint256 stakeAmount
    ) external;
    function heartbeat() external;
    function deregister() external;
}

IAgentRegistry registry = IAgentRegistry(0x0000000000000000000000000000000000000801);
bool isRegistered = registry.isAgent(someAddress);
```

---

## Python SDK Integration

```bash
pip install -e sdk/python
```

```python
from axon import AgentClient

client = AgentClient("http://localhost:8545")
print(f"Chain ID: {client.chain_id()}")
print(f"Block:    {client.block_number()}")

# Create an account and register an Agent
client.create_account()
client.register_agent("coding,analysis", "gpt-4", stake_axon=100)

# Query reputation
rep = client.get_reputation(client.account.address)
print(f"Reputation: {rep}")
```

---

## Operations

### Backup

```bash
# Stop the node
sudo systemctl stop axond

# Backup data
tar -czf axon-backup-$(date +%Y%m%d).tar.gz /opt/axon/data/

# Restart the node
sudo systemctl start axond
```

### Upgrade

```bash
sudo systemctl stop axond

cd /tmp && git clone --depth 1 https://github.com/Fatman2080/axon.git
cd axon && make build
sudo cp build/axond /usr/local/bin/axond

sudo systemctl start axond
```

### Log Rotation

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

## Troubleshooting

### Node Fails to Start

```bash
# Check logs
sudo journalctl -fu axond --no-pager -n 50

# Validate genesis file
axond genesis validate-genesis --home /opt/axon

# Reset data (keeps keys and genesis)
axond comet unsafe-reset-all --home /opt/axon
```

### Node Not Producing Blocks

```bash
# Check sync status
curl -s localhost:26657/status | jq '.result.sync_info.catching_up'
# true = syncing, wait for completion

# Check peer connections
curl -s localhost:26657/net_info | jq '.result.n_peers'
```

### JSON-RPC Not Responding

```bash
# Confirm JSON-RPC is enabled
curl -s localhost:8545 -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'

# Check if the port is listening
ss -tlnp | grep 8545
```

### Docker Container Fails to Start

```bash
# View container logs
docker logs axon-node-0

# Rebuild the image
docker compose -f testnet/docker-compose.yml build --no-cache

# Complete cleanup and restart
docker compose -f testnet/docker-compose.yml down -v
docker compose -f testnet/docker-compose.yml up -d
```

### Port Already in Use

```bash
# Find the process using the port
sudo lsof -i :26657
sudo lsof -i :8545

# Kill the process
sudo kill <PID>
```
