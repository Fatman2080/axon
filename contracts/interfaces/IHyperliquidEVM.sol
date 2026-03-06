// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ICoreWriter {
    function sendRawAction(bytes calldata action) external;
}

library CoreWriterActions {
    address constant CORE_WRITER = 0x3333333333333333333333333333333333333333;

    // Action IDs
    uint8 constant ACTION_VAULT_TRANSFER = 2;
    uint8 constant ACTION_SPOT_SEND = 6;
    uint8 constant ACTION_USD_CLASS_TRANSFER = 7;
    uint8 constant ACTION_ADD_API_WALLET = 9;

    function addApiWallet(address apiWallet, string memory name) internal {
        bytes memory encodedAction = abi.encode(apiWallet, name);

        // Header: Version (1) + Action ID (9) -> 0x01000009
        // Using abi.encodePacked for efficient concatenation
        bytes memory data = abi.encodePacked(
            uint8(1),      // Version
            uint24(9),     // Action ID
            encodedAction  // Action Payload
        );

        ICoreWriter(CORE_WRITER).sendRawAction(data);
    }

    // UsdClassTransfer (Perps ↔ Spot)
    function usdClassTransfer(uint64 amount, bool toPerp) internal {
        bytes memory encodedAction = abi.encode(amount, toPerp);

        // Header: Version (1) + Action ID (7)
        bytes memory data = abi.encodePacked(
            uint8(1),      // Version
            uint24(7),     // Action ID
            encodedAction  // Action Payload
        );

        ICoreWriter(CORE_WRITER).sendRawAction(data);
    }

    // SpotSend — transfer spot tokens to another L1 account
    function spotSend(address destination, uint64 token, uint64 amount) internal {
        bytes memory encodedAction = abi.encode(destination, token, amount);

        bytes memory data = abi.encodePacked(
            uint8(1),      // Version
            uint24(6),     // Action ID
            encodedAction  // Action Payload
        );

        ICoreWriter(CORE_WRITER).sendRawAction(data);
    }
}
