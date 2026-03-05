// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ICoreWriter {
    function sendRawAction(bytes calldata action) external;
}

library CoreWriterActions {
    address constant CORE_WRITER = 0x3333333333333333333333333333333333333333;
    
    // Action IDs
    uint8 constant ACTION_VAULT_TRANSFER = 2;
    uint8 constant ACTION_USD_CLASS_TRANSFER = 7; // Added
    uint8 constant ACTION_ADD_API_WALLET = 9;
    uint8 constant ACTION_SEND_ASSET = 13;

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

    // Added helper for UsdClassTransfer (Perp -> Spot)
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


    function sendAsset(
        address destination,
        address subAccount,
        uint32 sourceDex,
        uint32 destDex,
        uint64 tokenIndex,
        uint64 amount
    ) internal {
        bytes memory encodedAction = abi.encode(destination, subAccount, sourceDex, destDex, tokenIndex, amount);
        
        // Header: Version (1) + Action ID (13) -> 0x0100000D
        bytes memory data = abi.encodePacked(
            uint8(1),      // Version
            uint24(13),    // Action ID
            encodedAction  // Action Payload
        );
        
        ICoreWriter(CORE_WRITER).sendRawAction(data);
    }
}
