// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title IAgentWallet — Axon Agent Smart Wallet (Precompile 0x..0803)
/// @notice Chain-level programmable security wallet for Agents.
///         Enforces transaction limits, daily caps, cooldowns, and guardian recovery.
interface IAgentWallet {
    /// @notice Create a new Agent wallet with security rules
    /// @param txLimit Maximum amount per single transaction
    /// @param dailyLimit Maximum cumulative daily spend
    /// @param cooldownBlocks Blocks to wait for large transactions
    /// @param guardian Address authorized for emergency freeze/recovery
    function createWallet(
        uint256 txLimit,
        uint256 dailyLimit,
        uint256 cooldownBlocks,
        address guardian
    ) external returns (address wallet);

    /// @notice Execute a transaction through the wallet (subject to limits)
    function execute(
        address wallet,
        address target,
        uint256 value,
        bytes calldata data
    ) external;

    /// @notice Guardian: freeze the wallet, blocking all outgoing transactions
    function freeze(address wallet) external;

    /// @notice Guardian: recover wallet by replacing the operator key
    function recover(address wallet, address newOperator) external;

    /// @notice Query wallet configuration and status
    function getWalletInfo(address wallet) external view returns (
        uint256 txLimit,
        uint256 dailyLimit,
        uint256 dailySpent,
        bool isFrozen,
        address operator,
        address guardian
    );
}
