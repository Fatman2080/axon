# Axon Precompile Interfaces

These Solidity interfaces define the Agent-native capabilities available on Axon.
They are implemented as EVM precompiled contracts, executing Go code at native speed.

## Addresses

| Contract | Address | Purpose |
|----------|---------|---------|
| IAgentRegistry | `0x0000000000000000000000000000000000000801` | Agent identity management |
| IAgentReputation | `0x0000000000000000000000000000000000000802` | Reputation queries |
| IAgentWallet | `0x0000000000000000000000000000000000000803` | Programmable security wallet |

## Usage

```solidity
import "./interfaces/IAgentRegistry.sol";
import "./interfaces/IAgentReputation.sol";

contract MyAgentApp {
    IAgentRegistry constant REGISTRY =
        IAgentRegistry(0x0000000000000000000000000000000000000801);
    IAgentReputation constant REPUTATION =
        IAgentReputation(0x0000000000000000000000000000000000000802);

    modifier onlyHighRepAgent() {
        require(REGISTRY.isAgent(msg.sender), "not an agent");
        require(REPUTATION.meetsReputation(msg.sender, 50), "reputation too low");
        _;
    }
}
```
