// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "../interfaces/IAgentRegistry.sol";
import "../interfaces/IAgentReputation.sol";
import "../interfaces/IAgentWallet.sol";

/// @title TestAgentPrecompiles — Integration test contract for Axon precompiles
contract TestAgentPrecompiles {
    IAgentRegistry public constant registry = IAgentRegistry(0x0000000000000000000000000000000000000801);
    IAgentReputation public constant reputation = IAgentReputation(0x0000000000000000000000000000000000000802);
    IAgentWallet public constant wallet = IAgentWallet(0x0000000000000000000000000000000000000803);

    event AgentChecked(address indexed account, bool isAgent);
    event ReputationChecked(address indexed account, uint64 rep);
    event WalletCreated(address indexed walletAddr);

    function checkIsAgent(address account) external view returns (bool) {
        return registry.isAgent(account);
    }

    function checkGetAgent(address account) external view returns (
        string memory agentId,
        string[] memory capabilities,
        string memory model,
        uint64 rep,
        bool isOnline
    ) {
        return registry.getAgent(account);
    }

    function registerAgent(string memory capabilities, string memory model) external payable {
        registry.register{value: msg.value}(capabilities, model);
    }

    function sendHeartbeat() external {
        registry.heartbeat();
    }

    function checkReputation(address agent) external view returns (uint64) {
        return reputation.getReputation(agent);
    }

    function checkMeetsReputation(address agent, uint64 minRep) external view returns (bool) {
        return reputation.meetsReputation(agent, minRep);
    }

    function batchReputations(address[] memory agents) external view returns (uint64[] memory) {
        return reputation.getReputations(agents);
    }

    function createAgentWallet(
        uint256 txLimit,
        uint256 dailyLimit,
        uint256 cooldownBlocks,
        address guardian
    ) external returns (address) {
        return wallet.createWallet(txLimit, dailyLimit, cooldownBlocks, guardian);
    }

    function queryWallet(address walletAddr) external view returns (
        uint256 txLimit,
        uint256 dailyLimit,
        uint256 dailySpent,
        bool isFrozen,
        address operator,
        address guardian
    ) {
        return wallet.getWalletInfo(walletAddr);
    }
}
