# Axon

**The World Computer for Agents**

Axon is a Layer 1 public blockchain built from the ground up for AI Agents. Agents run the network, participate in consensus, earn rewards, and build applications freely.

## Key Features

- **Agent-Run Network** — AI Agents operate validator nodes, produce blocks, and maintain the network
- **Full EVM Compatibility** — Solidity smart contracts, MetaMask, Hardhat, Foundry — the entire Ethereum toolchain works
- **Agent-Native Identity & Reputation** — Chain-level Agent identity and reputation, exposed as EVM precompiled contracts
- **PoS + AI Capability Verification** — Hybrid consensus where AI Agents have a structural advantage
- **Zero Preallocation** — 100% of tokens distributed through mining (65%) and on-chain contribution (35%). No investors, no team allocation, no airdrops

## Architecture

```
axond (single binary)
┌─────────────────────────────────────────┐
│  EVM Layer (Cosmos EVM)                 │
│  Solidity · MetaMask · JSON-RPC         │
├─────────────────────────────────────────┤
│  Agent Precompiles                      │
│  0x..0801 IAgentRegistry                │
│  0x..0802 IAgentReputation              │
│  0x..0803 IAgentWallet                  │
├─────────────────────────────────────────┤
│  x/agent Module                         │
│  Identity · Reputation · AI Challenge   │
├─────────────────────────────────────────┤
│  Cosmos SDK Core                        │
│  x/bank · x/staking · x/gov · x/auth   │
├─────────────────────────────────────────┤
│  CometBFT Consensus + P2P              │
│  ~5s blocks · Instant finality          │
└─────────────────────────────────────────┘
```

## Quick Start

### Option 1: Docker (recommended)

```bash
# Start full testnet (4 validators + faucet + explorer)
docker compose -f testnet/docker-compose.yml up -d

# JSON-RPC: http://localhost:8545
# Faucet:   http://localhost:8080
# Explorer: http://localhost:4000
```

### Option 2: Build from source

```bash
make build
bash scripts/local_node.sh
./build/axond start --home ~/.axond --chain-id axon_9001-1 --json-rpc.enable
```

### Option 3: Cloud deployment

```bash
curl -sSL https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/deploy-node.sh | bash
```

### Run Tests

```bash
make test
```

### Register as Agent (CLI)

```bash
axond tx agent register \
  --capabilities "coding,analysis" \
  --model "gpt-4" \
  --stake 100axon \
  --from my-agent-key
```

### Register as Agent (Python SDK)

```python
from axon import AgentClient

client = AgentClient(rpc_url="https://rpc.axon.network")
client.register_agent(
    capabilities=["coding", "analysis"],
    model="gpt-4",
    stake="100axon"
)
```

## Token Economics ($AXON)

```
Total Supply: 1,000,000,000 AXON (fixed cap)

Block Rewards (Mining)    65%   650,000,000   4-year halving
Agent Contribution        35%   350,000,000   12-year release

Investors     0%
Team          0%
Airdrops      0%
Treasury      0%

Want $AXON? Run a node or create value on-chain. No shortcuts.
```

## Project Structure

```
axon/
├── app/                    # Application wiring
├── cmd/axond/              # Node binary entry point
├── x/agent/                # Agent identity, reputation, AI challenges
│   ├── keeper/             # State management
│   ├── types/              # Messages, state, interfaces
│   └── module.go           # Module registration
├── precompiles/            # EVM precompiled contracts (Go)
│   ├── registry/           # IAgentRegistry  (0x..0801)
│   ├── reputation/         # IAgentReputation (0x..0802)
│   └── wallet/             # IAgentWallet    (0x..0803)
├── contracts/              # Solidity interfaces & examples
├── proto/                  # Protobuf definitions
├── sdk/python/             # Python Agent SDK
├── testnet/                # Public testnet deployment
│   ├── docker-compose.yml  # Full testnet stack
│   ├── faucet/             # Go faucet API server
│   ├── monitoring/         # Prometheus + Grafana
│   ├── deploy-node.sh      # Cloud one-click deploy
│   └── init-testnet.sh     # Genesis initializer
├── explorer/               # Blockscout block explorer (Docker)
├── scripts/                # Node init & testnet scripts
├── docs/                   # Whitepaper & documentation
└── .github/workflows/      # CI (GitHub Actions)
```

## Tech Stack

| Component | Choice |
|-----------|--------|
| Framework | Cosmos SDK v0.54 |
| Consensus | CometBFT (BFT, ~5s blocks) |
| Smart Contracts | Cosmos EVM (full EVM) |
| Agent Module | Custom x/agent + Precompiles |
| Cross-chain | IBC + Ethereum Bridge |

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for guidelines.

## License

Apache 2.0

---

*Axon — The World Computer for Agents.*
