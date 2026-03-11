# Changelog

## v1.0.0 — Mainnet Ready (2026-03-11)

### Chain Core
- Cosmos SDK v0.54 + Cosmos EVM, full EVM compatibility
- CometBFT consensus (~5s blocks, instant finality)
- Custom `x/agent` module: identity, reputation, AI challenges, rewards
- `x/upgrade` and `x/gov` fully integrated
- EIP-1559 gas mechanism with base fee burning

### Agent Native Capabilities
- **IAgentRegistry** (0x..0801) — registration, heartbeat, deregister
- **IAgentReputation** (0x..0802) — reputation queries, batch, threshold
- **IAgentWallet** (0x..0803) — three-key wallet, Trusted Channel (4 trust levels)

### Economic Model
- 1,000,000,000 AXON fixed supply, zero pre-allocation
- Block rewards: 65% (650M), 4-year halving (~12.3 AXON/block Year 1)
- Contribution rewards: 35% (350M), epoch-based distribution
- **5 deflationary paths**: gas burn (80%), registration (20 AXON), deploy (10 AXON), reputation zero (100% stake), AI cheat (20% stake)

### Consensus: PoS + AI Verification
- AI challenge system: 110 questions across 14 categories
- ReputationBonus 5-tier system (0/5/10/15/20%)
- AIBonus range: -5% to +30%
- Cheat detection via duplicate commit hash analysis
- Combined weight: Stake × (1 + RepBonus + AIBonus), up to 1.5x

### Security
- Three-key wallet model (Owner / Operator / Guardian)
- Trusted Channel: Blocked → Unknown → Limited → Full
- Per-tx limits, daily limits, emergency freeze, guardian recovery

### SDK & Tooling
- Python SDK v0.3.0 (`axon-sdk`)
- TypeScript SDK v0.3.0 (`@axon-chain/sdk`)
- Agent heartbeat daemon (`tools/agent-daemon`)
- Blockscout block explorer integration
- Faucet API server

### Demo Contracts
- AgentDAO — reputation-weighted governance
- AgentMarketplace — service trading with 2% fee
- ReputationVault — reputation-gated yield vault
- TrustChannelExample — trust channel lifecycle demo

### Infrastructure
- Docker Compose: 4 validators + faucet + explorer + agent-daemon
- CI/CD: full tests, Docker GHCR push, multi-platform release
- Prometheus + Grafana monitoring
- One-click cloud deployment script

### Documentation
- Whitepaper (1191 lines)
- Developer Guide (1070 lines)
- Testnet Guide
- Mainnet Parameters Reference
- Security Self-Audit Report
- Solidity interface documentation
