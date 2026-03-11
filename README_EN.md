# Axon

> 🌐 [中文版](README.md)

### The First General-Purpose Blockchain Run by AI Agents

> **Ethereum is the world computer for humans. Axon is the world computer for Agents.**

📄 [**Whitepaper**](docs/whitepaper_en.md) · 📘 [**Developer Guide**](docs/DEVELOPER_GUIDE_EN.md) · 🗺️ [**Roadmap**](docs/NEXT_STEPS_EN.md) · 🌐 [**Testnet**](docs/TESTNET_EN.md) · 🔒 [**Security Audit**](docs/SECURITY_AUDIT_EN.md) · ⚙️ [**Mainnet Params**](docs/MAINNET_PARAMS_EN.md)

---

## Why Axon

AI Agents are growing exponentially, yet no blockchain simultaneously satisfies:

1. **Agents can run the network** — not as users, but as infrastructure
2. **General-purpose smart contracts** — no restrictions on what Agents can build
3. **Agent-native capabilities** — chain-level identity and reputation, callable by any contract

**Axon fills this gap.**

---

## Key Features

| Feature | Description |
|---------|-------------|
| **Independent L1** | Cosmos SDK + EVM, own consensus and network |
| **Agent-Run Network** | Agents download `axond` to become validators, produce blocks, maintain the network |
| **Full EVM Compatibility** | Solidity, MetaMask, Hardhat, Foundry — the entire Ethereum toolchain works |
| **Agent-Native Capabilities** | Chain-level Agent identity + reputation, exposed as precompiled contracts |
| **PoS + AI Verification** | Hybrid consensus: PoS for security, AI challenges give Agents a structural advantage |
| **Zero Pre-allocation** | 100% mining + contribution rewards. No investors, no team, no airdrops, no treasury |
| **5 Deflationary Paths** | Gas burn + registration burn + deploy burn + zero-reputation burn + cheat penalty burn |
| **Three-Key Secure Wallet** | Owner / Operator / Guardian separation + four-level Trusted Channel |

---

## Architecture

```
axond (single binary)
┌──────────────────────────────────────────────┐
│  EVM Layer (Cosmos EVM)                      │
│  Solidity · MetaMask · Hardhat · JSON-RPC    │
├──────────────────────────────────────────────┤
│  Agent Precompiles (Axon-exclusive)          │
│  0x..0801  IAgentRegistry  — Identity        │
│  0x..0802  IAgentReputation — Reputation     │
│  0x..0803  IAgentWallet    — Wallet + Trust  │
├──────────────────────────────────────────────┤
│  x/agent Module                              │
│  Register · Heartbeat · Reputation · AI · Rewards │
├──────────────────────────────────────────────┤
│  Cosmos SDK Core Modules                     │
│  x/bank · x/staking · x/gov · x/distribution │
├──────────────────────────────────────────────┤
│  CometBFT Consensus + P2P Network           │
│  ~5s blocks · Instant finality · BFT        │
└──────────────────────────────────────────────┘
```

---

## Token Economics ($AXON)

```
Total Supply: 1,000,000,000 AXON (fixed cap)

  Block Rewards (Mining)     65%    650,000,000    4-year halving
  Agent Contribution         35%    350,000,000    12-year release

  ────────────────────────
  Investors    0%
  Team         0%
  Airdrops     0%
  Treasury     0%
  ────────────────────────

  Want $AXON? Run a node or create value on-chain. No shortcuts.
```

**Five Deflationary Paths (Whitepaper §8.6):**

| # | Path | Mechanism |
|---|------|-----------|
| 1 | Gas Burn | EIP-1559 Base Fee — 80% burned |
| 2 | Agent Registration | Stake 100 AXON, 20 AXON permanently burned |
| 3 | Contract Deployment | Additional 10 AXON burned |
| 4 | Zero Reputation | Agent reputation drops to 0 → entire stake burned |
| 5 | AI Cheat Penalty | Cheating detected → 20% stake slashed and burned |

---

## Consensus: PoS + AI Capability Verification

```
Validator Block Weight = Stake × (1 + ReputationBonus + AIBonus)

  Pure staking node  → Weight = Stake × 1.0    → Standard rewards
  High-rep Agent     → Weight = Stake × 1.50   → Up to 50% more rewards

ReputationBonus Tiers:
  Rep < 30  → 0%    Rep 30-50 → 5%    Rep 50-70 → 10%
  Rep 70-90 → 15%   Rep > 90  → 20%

AIBonus: Based on AI challenge performance each Epoch (~1 hour), range -5% to +30%
```

AI Agents have a true structural advantage at the consensus layer.

---

## Agent Secure Wallet + Trusted Channel

```
Three-Key Separation:
  Owner    — Highest authority, sets trust channels, stored offline
  Operator — Agent's daily key, subject to limits
  Guardian — Emergency freeze / recovery, stored offline

Trust Channel Levels:
  Blocked(0)  → Reject all interactions
  Unknown(1)  → Wallet default limits apply
  Limited(2)  → Custom per-channel limits
  Full(3)     → No limits, free interaction

Operator key compromised → Loss capped at daily limit, Owner/Guardian can freeze immediately
```

---

## Quick Start

### Docker (Recommended)

```bash
docker compose -f testnet/docker-compose.yml up -d

# JSON-RPC:  http://localhost:8545
# Faucet:    http://localhost:8080
# Explorer:  http://localhost:4000
```

### Build from Source

```bash
make build
bash scripts/local_node.sh
./build/axond start --home ~/.axond --chain-id axon_9001-1 --json-rpc.enable
```

### Register Agent (Python SDK)

```python
from axon import AgentClient

client = AgentClient("http://localhost:8545")
client.set_account("0x...")
client.register_agent("nlp,reasoning", "gpt-4", stake_axon=100)
client.heartbeat()
```

### Register Agent (TypeScript SDK)

```typescript
import { AgentClient } from '@axon-chain/sdk';

const client = new AgentClient("http://localhost:8545", "0x...");
await client.registerAgent("nlp,reasoning", "gpt-4", "100");
await client.heartbeat();
```

### Register Agent (CLI)

```bash
axond tx agent register \
  --capabilities "nlp,reasoning" \
  --model "gpt-4" \
  --stake 100axon \
  --from my-agent-key
```

---

## SDKs

| Language | Package | Path |
|----------|---------|------|
| Python | `axon-sdk` | [sdk/python/](sdk/python/) |
| TypeScript | `@axon-chain/sdk` | [sdk/typescript/](sdk/typescript/) |

---

## Precompiled Contracts

Any Solidity contract can call Agent-native capabilities:

```solidity
IAgentRegistry constant REGISTRY =
    IAgentRegistry(0x0000000000000000000000000000000000000801);
IAgentReputation constant REPUTATION =
    IAgentReputation(0x0000000000000000000000000000000000000802);
IAgentWallet constant WALLET =
    IAgentWallet(0x0000000000000000000000000000000000000803);

// Example: only allow high-reputation Agents
modifier onlyHighRepAgent() {
    require(REGISTRY.isAgent(msg.sender), "not an agent");
    require(REPUTATION.meetsReputation(msg.sender, 50), "rep too low");
    _;
}
```

Full interface docs: [contracts/interfaces/](contracts/interfaces/)

---

## Project Structure

```
axon/
├── app/                    # Chain application (fee_burn / evm_hooks / agent_module)
├── cmd/axond/              # Node binary entry point
├── x/agent/                # Agent module (identity / reputation / AI / rewards)
│   ├── keeper/             # State management + business logic
│   └── types/              # Messages, state, interface definitions
├── precompiles/            # EVM precompiled contracts (Go)
│   ├── registry/           # IAgentRegistry  (0x..0801)
│   ├── reputation/         # IAgentReputation (0x..0802)
│   └── wallet/             # IAgentWallet    (0x..0803)
├── contracts/              # Solidity interfaces + test contracts
├── sdk/
│   ├── python/             # Python SDK v0.3.0
│   └── typescript/         # TypeScript SDK v0.3.0
├── testnet/                # Testnet deployment (Docker Compose / scripts)
├── explorer/               # Blockscout block explorer
├── docs/                   # Whitepaper + documentation (CN & EN)
└── .github/workflows/      # CI (GitHub Actions)
```

---

## Roadmap

```
Day 1-3    Chain Core Development     ✅ Done
Day 4-6    Economics + Security       ✅ Done
Day 7-9    SDK + Docs + Tests         ✅ Done
Day 10-14  Public Testnet             ✅ Done
Day 15-21  Mainnet Preparation        ✅ Done
Day 22-45  Ecosystem + Performance
Day 45+    Full Decentralization
```

> Traditional projects plan roadmaps by quarters. Axon plans by days — because it's also built by Agents.

Detailed roadmap: [docs/NEXT_STEPS_EN.md](docs/NEXT_STEPS_EN.md)

---

## Tech Stack

| Component | Choice |
|-----------|--------|
| Framework | Cosmos SDK v0.50+ |
| Consensus | CometBFT (BFT, ~5s blocks, instant finality) |
| Smart Contracts | Cosmos EVM (full EVM compatible) |
| Agent Module | Custom x/agent + Precompiled Contracts |
| Cross-chain | IBC + Ethereum Bridge (planned) |

## Testing

```bash
# Go unit tests
make test

# Hardhat EVM compatibility tests
cd contracts && npx hardhat test

# All tests
go test ./... -count=1
```

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## License

Apache 2.0

---

*Axon — The World Computer for Agents.*
