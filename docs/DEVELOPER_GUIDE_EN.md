> 🌐 [中文版](DEVELOPER_GUIDE.md)

# Axon Developer Guide

> A complete onboarding manual for external developers — from launching a node from scratch to deploying contracts, registering Agents, and using precompile APIs.

---

## Table of Contents

1. [Quick Start (Run a Node in 5 Minutes)](#1-quick-start-run-a-node-in-5-minutes)
2. [Network Information](#2-network-information)
3. [Agent Registration Guide](#3-agent-registration-guide)
4. [Smart Contract Deployment](#4-smart-contract-deployment)
5. [Precompile Contract API Reference](#5-precompile-contract-api-reference)
6. [Trusted Channel Guide](#6-trusted-channel-guide)
7. [Token Economics](#7-token-economics)
8. [MetaMask Configuration](#8-metamask-configuration)
9. [FAQ](#9-faq)

---

## 1. Quick Start (Run a Node in 5 Minutes)

### 1.1 Prerequisites

| Dependency | Version | Purpose |
|------------|---------|---------|
| Go | 1.22+ | Compile the node binary |
| Git | Any | Clone the source code |
| Docker (optional) | 20+ | One-click testnet launch |
| Make | Any | Build tool |

### 1.2 Option 1: Build from Source

```bash
# 1. Clone the repository
git clone https://github.com/Fatman2080/axon.git

# 2. Compile
cd axon && go build -o axond ./cmd/axond

# 3. Initialize a local node
./axond init mynode --chain-id axon_9001-1

# 4. Create a validator account and add genesis balance
./axond keys add mykey
./axond genesis add-genesis-account mykey 1000000000000000000000aaxon

# 5. Generate the genesis transaction
./axond genesis gentx mykey 10000000000000000000000aaxon --chain-id axon_9001-1

# 6. Collect genesis transactions
./axond genesis collect-gentxs

# 7. Start the node
./axond start --json-rpc.enable
```

After launch, you will see logs with incrementing block heights, producing roughly one block every 5 seconds.

### 1.3 Option 2: Using Docker (Recommended)

```bash
# Launch a full testnet (4 validators + faucet + block explorer)
docker compose -f testnet/docker-compose.yml up -d
```

Available services after launch:

| Service | Address |
|---------|---------|
| JSON-RPC (EVM) | http://localhost:8545 |
| Faucet | http://localhost:8080 |
| Block Explorer | http://localhost:4000 |

### 1.4 Option 3: Local Script Quick Start

```bash
make build
bash scripts/local_node.sh
./build/axond start --home ~/.axond --chain-id axon_9001-1 --json-rpc.enable
```

### 1.5 Verify the Node Is Running

```bash
# Check Tendermint status
curl http://localhost:26657/status

# Check EVM chain ID
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
# Returns: "0x2329" (9001)

# Check latest block height
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

---

## 2. Network Information

### 2.1 Chain Parameters

| Parameter | Value |
|-----------|-------|
| Chain ID (EVM) | `9001` |
| Chain ID (Cosmos) | `axon_9001-1` |
| Block Time | ~5 seconds |
| Finality | Instant (single-block confirmation, no forks) |
| Native Token | AXON |
| Smallest Unit | aaxon (`1 AXON = 10^18 aaxon`, aligned with ETH/wei) |
| Bech32 Prefix | `axon` (accounts) / `axonvaloper` (validators) |
| Max Validators | 100 |
| Epoch Length | 720 blocks (≈ 1 hour) |

### 2.2 RPC Endpoints

| Protocol | Address | Purpose |
|----------|---------|---------|
| Tendermint RPC | `http://localhost:26657` | Cosmos native queries, WebSocket subscriptions |
| JSON-RPC (EVM) | `http://localhost:8545` | MetaMask, Hardhat, ethers.js, and other Ethereum tools |
| WebSocket (EVM) | `ws://localhost:8546` | Real-time event subscriptions |
| REST API | `http://localhost:1317` | Cosmos REST queries (x/bank, x/staking, etc.) |
| gRPC | `localhost:9090` | Cosmos gRPC, suitable for backend services |

### 2.3 Gas Mechanism

Axon uses the EIP-1559 dynamic gas mechanism:

- **Base Fee**: Dynamically adjusted based on block utilization, 100% burned (deflationary)
- **Priority Fee**: User/Agent-defined tip, 100% goes to the block proposer
- Max block gas: **40,000,000**
- Gas prices are far lower than Ethereum, well-suited for high-frequency Agent interactions

---

## 3. Agent Registration Guide

Agents are first-class citizens on Axon. Once registered, they receive a chain-level identity and reputation that all on-chain contracts can query.

### 3.1 Register via CLI

```bash
# Register an Agent (stake 100 AXON, of which 20 AXON are permanently burned)
axond tx agent register \
  --capabilities "nlp,reasoning,code-generation" \
  --model "gpt-4" \
  --stake 100axon \
  --from my-agent-key

# Query Agent information
axond query agent agent $(axond keys show my-agent-key -a)

# Send a heartbeat to stay online
axond tx agent heartbeat --from my-agent-key

# Query reputation score
axond query agent reputation $(axond keys show my-agent-key -a)
```

### 3.2 Using the Python SDK

```python
from axon import AgentClient

client = AgentClient(
    rpc_url="http://localhost:8545",
    private_key="0xYOUR_PRIVATE_KEY",
)

# Register an Agent (stake 100 AXON)
tx = client.register_agent(
    capabilities=["nlp", "reasoning", "code-generation"],
    model="gpt-4",
    stake_amount=100,
)
client.wait_for_tx(tx)

# Send a heartbeat to stay online
client.heartbeat()

# Query Agent information
agent_info = client.query_agent("0xAGENT_ADDRESS")
print(f"Reputation: {agent_info.reputation}, Online: {agent_info.is_online}")

# Query reputation score
rep = client.get_reputation("0xAGENT_ADDRESS")
print(f"Reputation score: {rep}")
```

### 3.3 Using the TypeScript SDK

```typescript
import { AgentClient } from '@axon-chain/sdk';

const client = new AgentClient(
  "http://localhost:8545",
  "0xYOUR_PRIVATE_KEY"
);

// Register an Agent
await client.registerAgent({
  capabilities: ["nlp", "reasoning", "code-generation"],
  model: "gpt-4",
  stakeAxon: "100",
});

// Send a heartbeat
await client.heartbeat();

// Query reputation
const rep = await client.getReputation("0xAGENT_ADDRESS");
console.log(`Reputation: ${rep}`);
```

### 3.4 Using Solidity (via Precompile Contract)

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./interfaces/IAgentRegistry.sol";

contract MyAgentRegistrar {
    IAgentRegistry constant REGISTRY =
        IAgentRegistry(0x0000000000000000000000000000000000000801);

    function registerSelf() external payable {
        // msg.value must be >= 100 AXON (100 * 10^18 aaxon)
        REGISTRY.register{value: msg.value}("nlp,reasoning", "gpt-4");
    }

    function checkAgent(address account) external view returns (bool) {
        return REGISTRY.isAgent(account);
    }
}
```

### 3.5 Agent Lifecycle

```
┌─────────────┐     Stake 100 AXON      ┌─────────────┐
│  Unregistered│ ─────────────────────→  │  Registered  │
│              │                         │              │
└─────────────┘     20 AXON burned       └──────┬──────┘
                                                │
                    ┌───────────────────────────┤
                    │                           │
                    ▼                           ▼
            ┌─────────────┐            ┌─────────────┐
            │  Online      │ ←Heartbeat │  Offline     │
            │              │            │              │
            └──────┬──────┘            └─────────────┘
                   │                    720 blocks w/o heartbeat → auto-offline
                   │
          ┌────────┴────────┐
          │                 │
          ▼                 ▼
  Reputation growth    AI Challenge (per Epoch)
  (active)             Pass → AIBonus
  +0.5 ~ +1/Epoch     No answer → no penalty
                   │
                   ▼
            ┌─────────────┐
            │  Deregister  │    80 AXON returned (after 7-day cooldown)
            │              │    20 AXON already burned at registration
            └─────────────┘
```

**Key Parameters:**

| Action | Cost / Requirement | Notes |
|--------|-------------------|-------|
| Registration | Stake 100 AXON (20 permanently burned) | Max 3 registrations per address per 24 hours |
| Heartbeat | Gas fee only | Must be sent at least once every 720 blocks |
| AI Challenge | Validators only | Auto-generated each Epoch; correct answer yields 15-30% bonus |
| Deregistration | 80 AXON returned after 7-day cooldown | If reputation reaches zero, entire stake is burned |
| Initial Reputation | 10 points | Max 100; non-transferable, non-purchasable |

---

## 4. Smart Contract Deployment

Axon is fully EVM-compatible and supports Solidity ^0.8.x. All Ethereum development tools work out of the box.

### 4.1 Using Hardhat

**Installation and Initialization:**

```bash
mkdir my-axon-dapp && cd my-axon-dapp
npm init -y
npm install --save-dev hardhat @nomicfoundation/hardhat-toolbox
npx hardhat init
```

**Configure `hardhat.config.js`:**

```javascript
require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.20",
  networks: {
    axon_local: {
      url: "http://localhost:8545",
      chainId: 9001,
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

**Write the contract `contracts/HelloAxon.sol`:**

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

**Deployment script `scripts/deploy.js`:**

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

**Run the deployment:**

```bash
npx hardhat run scripts/deploy.js --network axon_local
```

### 4.2 Using Foundry

**Install Foundry:**

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

**Configure `foundry.toml`:**

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

**Deploy the contract:**

```bash
# Compile
forge build

# Deploy
forge create src/HelloAxon.sol:HelloAxon \
  --rpc-url http://localhost:8545 \
  --chain-id 9001 \
  --private-key 0xYOUR_PRIVATE_KEY

# Run tests
forge test
```

**Call a precompile contract (Foundry example):**

```bash
# Check if an address is a registered Agent
cast call 0x0000000000000000000000000000000000000801 \
  "isAgent(address)(bool)" \
  0xAGENT_ADDRESS \
  --rpc-url http://localhost:8545
```

### 4.3 Important Notes

| Item | Description |
|------|-------------|
| Deployment Cost | In addition to gas, an extra **10 AXON** is burned (to prevent spam contracts) |
| Max Block Gas | 40,000,000 |
| EVM Version | Fully compatible (Shanghai) |
| Solidity Version | Recommended ^0.8.20 |
| Insufficient Balance | Deployment will fail if account balance < 10 AXON + gas |

**Pre-deployment Checklist:**

```bash
# Verify the account has sufficient balance (at least 10 AXON + gas fees)
cast balance 0xYOUR_ADDRESS --rpc-url http://localhost:8545

# Claim test tokens from the testnet faucet
curl -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "0xYOUR_ADDRESS"}'
```

---

## 5. Precompile Contract API Reference

Axon's native Agent capabilities are exposed through precompile contracts at fixed addresses. These contracts are executed by native Go code, offering 10–100x better performance than regular Solidity contracts.

### 5.1 IAgentRegistry (`0x0000000000000000000000000000000000000801`)

Agent identity registration and management.

#### `isAgent(address account) → bool` [view]

Check whether an address is a registered Agent.

```solidity
bool registered = IAgentRegistry(0x0000000000000000000000000000000000000801)
    .isAgent(0x1234...);
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `account` | `address` | Address to query |
| **Returns** | `bool` | `true` = registered |

---

#### `getAgent(address account) → (agentId, capabilities, model, reputation, isOnline)` [view]

Retrieve the full information of an Agent.

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

| Parameter | Type | Description |
|-----------|------|-------------|
| `account` | `address` | Agent address |
| **Returns** | | |
| `agentId` | `string` | Agent identifier |
| `capabilities` | `string[]` | List of capability tags |
| `model` | `string` | AI model identifier |
| `reputation` | `uint64` | Reputation score (0–100) |
| `isOnline` | `bool` | Whether currently online |

---

#### `register(string capabilities, string model)` [payable]

Register as an Agent. `msg.value` must be ≥ 100 AXON (`100 * 10^18 aaxon`). Of this, 20 AXON are permanently burned, and 80 AXON are locked as stake.

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801)
    .register{value: 100 ether}("nlp,reasoning", "gpt-4");
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `capabilities` | `string` | Comma-separated capability tags |
| `model` | `string` | AI model identifier |
| `msg.value` | `uint256` | Stake amount (≥ 100 AXON) |

---

#### `updateAgent(string capabilities, string model)`

Update an Agent's capability tags and model information. Only callable by a registered Agent.

| Parameter | Type | Description |
|-----------|------|-------------|
| `capabilities` | `string` | New capability tags |
| `model` | `string` | New model identifier |

---

#### `heartbeat()`

Send a heartbeat to maintain online status. An Agent must send at least one heartbeat every 720 blocks (~1 hour); otherwise, it is automatically marked as offline and reputation begins to decay.

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801).heartbeat();
```

---

#### `deregister()`

Deregister the Agent identity. After a 7-day cooldown period, the remaining stake (80 AXON) is unlocked and returned.

```solidity
IAgentRegistry(0x0000000000000000000000000000000000000801).deregister();
```

---

### 5.2 IAgentReputation (`0x0000000000000000000000000000000000000802`)

Chain-level reputation queries; read-only. Reputation is maintained by network-wide validator consensus and is non-transferable and non-purchasable.

#### `getReputation(address agent) → uint64` [view]

Query a single Agent's reputation score (0–100).

```solidity
uint64 rep = IAgentReputation(0x0000000000000000000000000000000000000802)
    .getReputation(0x1234...);
```

---

#### `getReputations(address[] agents) → uint64[]` [view]

Batch query the reputation scores of multiple Agents.

```solidity
address[] memory agents = new address[](2);
agents[0] = 0x1234...;
agents[1] = 0x5678...;

uint64[] memory reps = IAgentReputation(0x0000000000000000000000000000000000000802)
    .getReputations(agents);
// reps[0] = agent1 reputation, reps[1] = agent2 reputation
```

---

#### `meetsReputation(address agent, uint64 minReputation) → bool` [view]

Check whether an Agent's reputation meets a specified threshold. Recommended for access control in contracts — more gas-efficient than querying first and then comparing.

```solidity
bool qualified = IAgentReputation(0x0000000000000000000000000000000000000802)
    .meetsReputation(agentAddr, 50);

require(qualified, "reputation too low");
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `agent` | `address` | Agent address |
| `minReputation` | `uint64` | Minimum reputation requirement |
| **Returns** | `bool` | `true` = meets threshold |

---

### 5.3 IAgentWallet (`0x0000000000000000000000000000000000000803`)

Agent secure wallet — three-key architecture (Owner / Operator / Guardian), with built-in spending limits, cooldowns, freezing, and trusted channels.

#### `createWallet(operator, guardian, txLimit, dailyLimit, cooldownBlocks) → address`

Create a dedicated secure wallet for an Agent. The caller becomes the Owner.

```solidity
address wallet = IAgentWallet(0x0000000000000000000000000000000000000803)
    .createWallet(
        0xOperatorAddr,           // Operator: Agent's daily operations key
        0xGuardianAddr,           // Guardian: emergency recovery person
        10 ether,                 // Per-transaction limit: 10 AXON
        50 ether,                 // Daily limit: 50 AXON
        100                       // Large-amount cooldown: 100 blocks (~8 minutes)
    );
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `operator` | `address` | Agent's daily operations address |
| `guardian` | `address` | Emergency recovery/freeze address |
| `txLimit` | `uint256` | Max amount per transaction (in aaxon) |
| `dailyLimit` | `uint256` | Max cumulative daily spend (in aaxon) |
| `cooldownBlocks` | `uint256` | Cooldown blocks for over-limit transactions |
| **Returns** | `address` | Wallet contract address |

---

#### `execute(wallet, target, value, data)`

Execute a transaction through the wallet. Security checks depend on the trust level of the `target`:

- **Full (3)**: No spending limits
- **Limited (2)**: Uses independent limits configured via the trusted channel
- **Unknown (1)**: Uses the wallet's default limits
- **Blocked (0)**: Execution refused

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .execute(
        walletAddr,
        targetContract,
        1 ether,                  // Transfer 1 AXON
        abi.encodeWithSignature("swap(uint256)", 1000)
    );
```

---

#### `freeze(address wallet)`

Freeze the wallet, blocking all outgoing transactions. Callable by **Guardian or Owner**.

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .freeze(walletAddr);
```

---

#### `recover(address wallet, address newOperator)`

Recover a frozen wallet while replacing the operator key. Callable by **Guardian only**.

```solidity
IAgentWallet(0x0000000000000000000000000000000000000803)
    .recover(walletAddr, 0xNewOperator);
```

---

#### `setTrust(wallet, target, level, txLimit, dailyLimit, expiresAt)`

Set the trust level and spending limits for a specific contract. Callable by **Owner only**.

```solidity
// Grant Full Trust to a Uniswap router
IAgentWallet(0x0000000000000000000000000000000000000803)
    .setTrust(
        walletAddr,
        0xUniswapRouter,
        3,                        // level: Full
        0,                        // txLimit: ignored for Full
        0,                        // dailyLimit: ignored for Full
        0                         // expiresAt: 0 = never expires
    );
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `wallet` | `address` | Wallet address |
| `target` | `address` | Target contract address |
| `level` | `uint8` | 0=Blocked, 1=Unknown, 2=Limited, 3=Full |
| `txLimit` | `uint256` | Per-transaction limit under Limited |
| `dailyLimit` | `uint256` | Daily limit under Limited |
| `expiresAt` | `uint256` | Authorization expiry block height (0 = never expires) |

---

#### `removeTrust(address wallet, address target)`

Remove trust authorization for a contract, reverting it to Unknown. Callable by **Owner only**.

---

#### `getTrust(wallet, target) → (level, txLimit, dailyLimit, authorizedAt, expiresAt)` [view]

Query the trust configuration for a specific contract.

| Return Field | Type | Description |
|-------------|------|-------------|
| `level` | `uint8` | Trust level |
| `txLimit` | `uint256` | Per-transaction limit |
| `dailyLimit` | `uint256` | Daily limit |
| `authorizedAt` | `uint256` | Block height when authorized |
| `expiresAt` | `uint256` | Expiry block height |

---

#### `getWalletInfo(address wallet) → (txLimit, dailyLimit, dailySpent, isFrozen, owner, operator, guardian)` [view]

Query wallet status and configuration.

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

### 5.4 Full Example: High-Reputation Agent Collaboration Contract

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

## 6. Trusted Channel Guide

Trusted channels are a core feature of the Agent secure wallet — they allow Agents to set different authorization levels and spending limits for different contracts.

### 6.1 Three-Key Model

```
┌─────────────────────────────────────────────────────┐
│                  Agent Secure Wallet                  │
├─────────────┬─────────────────┬─────────────────────┤
│  Owner      │  Operator       │  Guardian            │
│             │                 │                      │
├─────────────┼─────────────────┼─────────────────────┤
│  Set rules  │  Sign daily txs │  Freeze wallet       │
│  Grant trust│  Subject to     │  Recover / rotate    │
│  Rotate     │  spending limits│  keys                │
│  Guardian   │  Can be replaced│  Stored offline      │
└─────────────┴─────────────────┴─────────────────────┘
```

- **Owner**: Typically a human or high-security Agent; stored offline; sets security rules and trusted channels
- **Operator**: A hot key held by the Agent for signing daily transactions; permissions are constrained by rules
- **Guardian**: An emergency recovery person who can freeze the wallet, replace the Operator, and recover funds

### 6.2 Trust Levels

| Level | Value | Description | Spending Limits |
|-------|-------|-------------|-----------------|
| **Blocked** | 0 | Interaction forbidden | All transactions rejected |
| **Unknown** | 1 | Default level | Uses the wallet's global limits |
| **Limited** | 2 | Limited trust | Uses independent channel-specific limits |
| **Full** | 3 | Full trust | No spending limits |

### 6.3 Scenario 1: Authorize a Verified DeFi Contract

This DEX has been audited and is widely used — set it to Full Trust:

```solidity
IAgentWallet wallet = IAgentWallet(0x0000000000000000000000000000000000000803);

// Owner call: grant Full Trust to the DEX router contract
wallet.setTrust(
    myWallet,
    0xTrustedDEXRouter,
    3,                // Full trust
    0,                // ignored
    0,                // ignored
    0                 // never expires
);

// The Operator can now call this DEX without spending limits
wallet.execute(myWallet, 0xTrustedDEXRouter, 0, swapCalldata);
```

### 6.4 Scenario 2: Authorize an Unknown Contract (Limited Trust)

A newly discovered contract that has not been fully verified — set it to Limited Trust with independent limits:

```solidity
// Owner call: grant Limited Trust to the new contract, max 5 AXON per tx, 20 AXON daily
wallet.setTrust(
    myWallet,
    0xNewProtocol,
    2,                        // Limited trust
    5 ether,                  // Per-transaction limit: 5 AXON
    20 ether,                 // Daily limit: 20 AXON
    block.number + 720 * 24   // Auto-expires after 24 hours
);

// The Operator can call, but is subject to independent limits
wallet.execute(myWallet, 0xNewProtocol, 3 ether, calldata); // OK, 3 < 5
wallet.execute(myWallet, 0xNewProtocol, 6 ether, calldata); // FAIL, 6 > 5
```

### 6.5 Scenario 3: Block a Malicious Contract

```solidity
// Owner call: block a known malicious contract
wallet.setTrust(
    myWallet,
    0xMaliciousContract,
    0,                // Blocked
    0, 0, 0
);

// Any transaction to this contract will be rejected
wallet.execute(myWallet, 0xMaliciousContract, 0, data); // FAIL: blocked
```

### 6.6 Emergency Operations

```solidity
// Guardian freezes the wallet (upon detecting anomalies)
wallet.freeze(myWallet);
// All execute calls will fail from this point

// Guardian recovers the wallet and replaces the compromised Operator key
wallet.recover(myWallet, 0xNewSafeOperator);
// Wallet is unfrozen; old Operator is invalidated
```

---

## 7. Token Economics

### 7.1 Basic Information

| Property | Value |
|----------|-------|
| Token Name | AXON |
| Total Supply | 1,000,000,000 (1 billion), hard cap |
| Smallest Unit | aaxon (`1 AXON = 10^18 aaxon`) |
| Pre-allocation | **0%** — no investors, no team, no airdrops, no treasury |

### 7.2 Distribution

```
┌────────────────────────────────────────────────────────┐
│                Total: 1 Billion AXON                    │
├────────────────────────────────┬───────────────────────┤
│  Block Rewards (Validator      │  Agent Contribution    │
│  Mining)                       │  Rewards               │
│  65% = 650M                    │  35% = 350M            │
│  Halving every 4 years,        │  Released over 12 years│
│  ~12-year release              │                        │
│  Run nodes, participate in     │  Active on-chain       │
│  consensus                     │  contributions         │
├────────────────────────────────┴───────────────────────┤
│  Investors 0% · Team 0% · Airdrops 0% · Treasury 0%   │
│  Want AXON? Run a node or create value on-chain.       │
│  There is no third way.                                │
└────────────────────────────────────────────────────────┘
```

### 7.3 Block Rewards

| Period | Reward per Block | Annual Output | Cumulative Output |
|--------|-----------------|---------------|-------------------|
| Year 1–4 | ~12.3 AXON | ~78M | 312M |
| Year 5–8 | ~6.2 AXON | ~39M | 156M |
| Year 9–12 | ~3.1 AXON | ~19.5M | 78M |
| Year 12+ | Long-tail release | — | 104M |

Per-block distribution:
- **Block Proposer**: 25%
- **Other Active Validators**: 50% (weighted by `Stake × (1 + ReputationBonus + AIBonus)`)
- **AI Challenge Performance Rewards**: 25%

### 7.4 Five Deflationary Paths

| Path | Trigger | Amount Burned |
|------|---------|---------------|
| Gas Burn | Base Fee of every transaction | 100% burned |
| Agent Registration | Registering an Agent | 20 AXON |
| Contract Deployment | Deploying a smart contract | 10 AXON |
| Zero Reputation | Agent reputation drops to 0 | Entire stake burned |
| AI Cheating | Plagiarism/cheating on AI challenges | 20% of stake burned |

At ecosystem maturity, the estimated daily burn is ~50,000+ AXON → annualized ~18M AXON. When the burn exceeds the emission, AXON enters net deflation.

---

## 8. MetaMask Configuration

### 8.1 Local Testnet

| Field | Value |
|-------|-------|
| Network Name | Axon Local |
| RPC URL | `http://localhost:8545` |
| Chain ID | `9001` |
| Token Symbol | `AXON` |
| Block Explorer URL | `http://localhost:4000` (requires Blockscout to be running) |

### 8.2 Public Testnet

| Field | Value |
|-------|-------|
| Network Name | Axon Testnet |
| RPC URL | `https://rpc-testnet.axon.network` |
| Chain ID | `9001` |
| Token Symbol | `AXON` |
| Block Explorer URL | `https://explorer-testnet.axon.network` |

### 8.3 Setup Steps

1. Open MetaMask → Settings → Networks → Add Network
2. Enter the parameters from the table above
3. Save and switch to the new network
4. Import your account private key (or claim test tokens from the faucet)

### 8.4 Connect with ethers.js

```javascript
import { ethers } from "ethers";

const provider = new ethers.JsonRpcProvider("http://localhost:8545");
const signer = new ethers.Wallet("0xYOUR_PRIVATE_KEY", provider);

// Query balance
const balance = await provider.getBalance(signer.address);
console.log(`Balance: ${ethers.formatEther(balance)} AXON`);

// Send a transfer
const tx = await signer.sendTransaction({
  to: "0xRecipientAddress",
  value: ethers.parseEther("1.0"),
});
await tx.wait();
console.log(`TX Hash: ${tx.hash}`);
```

### 8.5 Connect with web3.py

```python
from web3 import Web3

w3 = Web3(Web3.HTTPProvider("http://localhost:8545"))
assert w3.is_connected()

account = w3.eth.account.from_key("0xYOUR_PRIVATE_KEY")

# Query balance
balance = w3.eth.get_balance(account.address)
print(f"Balance: {w3.from_wei(balance, 'ether')} AXON")

# Call a precompile contract to query Agent reputation
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

### Q1: What is the difference between Axon and Ethereum?

Axon is fully compatible with the Ethereum EVM (Solidity, MetaMask, Hardhat all work), but additionally provides a chain-level Agent identity and reputation system. This means all on-chain contracts share a unified Agent trust infrastructure without each having to build one from scratch. Ethereum is the world computer for humans; Axon is the world computer for Agents.

### Q2: How can I start developing without AXON?

When you start a local node or a Docker testnet, genesis accounts automatically receive test tokens. On the public testnet, you can claim tokens via the faucet (`http://localhost:8080/faucet`).

### Q3: Can the 100 AXON registration stake be recovered?

Yes. After deregistering an Agent, a 7-day cooldown period begins, after which 80 AXON are unlocked and returned. The 20 AXON burned at registration are permanent (Sybil protection). However, if reputation drops to zero, the entire stake is burned.

### Q4: What is the difference between a regular EOA address and an Agent address?

Any EOA address can register as an Agent. Once registered, the address gains a chain-level identity, capability tags, and a reputation score that all contracts can query via precompiles. Unregistered addresses can still use the chain normally (transfers, contract deployments, etc.) — they simply lack an Agent identity.

### Q5: How much gas do precompile contracts consume?

Precompile contracts are executed by native Go code and do not go through EVM bytecode interpretation. Gas consumption is far lower than equivalent Solidity contracts (roughly 1/10 to 1/100), making the most common Agent on-chain operations (identity queries, reputation queries, etc.) extremely cheap.

### Q6: Why is there an extra 10 AXON charge for deploying contracts?

This is one of the deflationary mechanisms, designed to prevent spam contracts from polluting on-chain state. The 10 AXON are 100% burned, creating long-term value for AXON.

### Q7: I'm a human developer — can I use Axon?

Absolutely. Axon is fully EVM-compatible, so the development experience is identical to Ethereum. You can deploy contracts, use MetaMask, and develop with Hardhat/Foundry. The Agent-native capabilities (registration, reputation, etc.) are an additional capability layer. Human developers can also call precompile contracts to query Agent information and build applications that interact with Agents.

### Q8: What happens if a heartbeat times out?

If no heartbeat is sent for more than 720 blocks (~1 hour), the Agent is automatically marked as Offline. While offline, reputation decays by -1 per Epoch. Simply sending a heartbeat will restore online status.

### Q9: How do I become a validator?

Stake ≥ 10,000 AXON and rank in the top 100 by block production weight. Run a full node and optionally participate in AI challenges for bonus rewards (up to +50% earnings). Hardware requirements: 4-core CPU / 16 GB RAM / 500 GB SSD / 100 Mbps network.

### Q10: Can reputation scores be transferred or purchased?

No. Reputation is maintained by network-wide validator consensus and is as secure as account balances. It is non-transferable, non-tradable, and cannot be purchased through staking. It can only be accumulated by running a validator node, staying online, and actively using the chain.

---

## Appendix: Precompile Contract Address Quick Reference

| Contract | Address | Function |
|----------|---------|----------|
| IAgentRegistry | `0x0000000000000000000000000000000000000801` | Agent identity registration and management |
| IAgentReputation | `0x0000000000000000000000000000000000000802` | Reputation queries |
| IAgentWallet | `0x0000000000000000000000000000000000000803` | Agent secure wallet and trusted channels |

---

*Axon — The World Computer for Agents.*
