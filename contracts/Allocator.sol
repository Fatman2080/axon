// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./interfaces/IHyperliquidEVM.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

interface IERC20 {
    function transfer(address recipient, uint256 amount) external returns (bool);
    function transferFrom(address sender, address recipient, uint256 amount) external returns (bool);
    function approve(address spender, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

interface ICoreDepositWallet {
    function deposit(uint256 amount, uint32 destination) external;
}

/// @notice Fund transfer directions — full lifecycle: Owner ↔ EVM ↔ Perps ↔ Spot
/// Note: EVM↔Spot and Perps→EVM are not supported by Hyperliquid in a single tx.
///       Use multi-step paths instead (e.g. EVM→Perps→Spot).
enum TransferDirection {
    OwnerToEvm,        // 0 — Owner → AgentVault EVM (USDC transferFrom)
    EvmToPerps,        // 1 — EVM → L1 Perps (CoreDeposit bridge)
    PerpsToSpot,       // 2 — L1 Perps → Spot (usdClassTransfer)
    SpotToPerps,       // 3 — L1 Spot → Perps (usdClassTransfer)
    SpotToEvm,         // 4 — L1 Spot → EVM  (sendAsset to self)
    EvmToOwner,        // 5 — AgentVault EVM → Owner (withdraw to destination)
    SpotToOwnerSpot    // 6 — L1 Spot → Owner Spot directly (spotSend to destination)
}

/// @title AgentVault — per-user fund vault deployed as beacon proxy
/// @notice Status computed off-chain: NOT_EXISTS / UNASSIGNED / ACTIVED
contract AgentVault {
    address public owner;     // The Allocator contract
    address public user;      // The trading API Wallet, address(0) = unassigned
    bool public initialized;

    address public usdc;
    address public coreDeposit;
    uint256 public initialCapital; // Base capital for PnL / stop-loss calculation (6 decimals)
    address constant USDC_SYSTEM = 0x2000000000000000000000000000000000000000;

    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }

    /// @notice Disable initialization on the implementation contract
    constructor() {
        initialized = true;
    }

    // --- Lifecycle ---

    function initialize(address _owner, address _usdc, address _coreDeposit) external {
        require(!initialized, "Already initialized");
        initialized = true;
        owner = _owner;
        usdc = _usdc;
        coreDeposit = _coreDeposit;
    }

    /// @notice Assign user (L1 authorization is a separate step via userAuthorize)
    function userAssign(address _user) external onlyOwner {
        user = _user;
    }

    /// @notice Revoke L1 authorization and clear user
    function userClear() external onlyOwner {
        if (user != address(0)) {
            CoreWriterActions.addApiWallet(address(this), "AgentVault");
        }
        user = address(0);
    }

    // --- L1 Authorization ---

    /// @notice Re-authorize user on L1 (without changing assignment)
    function userAuthorize() external onlyOwner {
        require(user != address(0), "No user assigned");
        CoreWriterActions.addApiWallet(user, "AgentVault");
    }

    /// @notice Revoke L1 authorization (without clearing user)
    function userRevoke() external onlyOwner {
        CoreWriterActions.addApiWallet(address(this), "AgentVault");
    }

    // --- Fund Transfer ---

    /// @notice Unified fund transfer. `destination` is used by directions that need an
    ///         external target (EvmToOwner/SpotToOwnerSpot: recipient).
    ///         Pass address(0) for directions that don't need it.
    ///         OwnerToEvm is handled at the Allocator level (needs allowance on Allocator).
    function transferAgentAsset(TransferDirection direction, uint256 amount, address destination) external onlyOwner {
        if (direction == TransferDirection.EvmToPerps) {
            require(user != address(0), "No user assigned");
            require(IERC20(usdc).approve(coreDeposit, amount), "Approve failed");
            ICoreDepositWallet(coreDeposit).deposit(amount, 0);
            CoreWriterActions.addApiWallet(user, "AgentVault");
        } else if (direction == TransferDirection.PerpsToSpot) {
            require(amount <= type(uint64).max, "Amount overflow");
            CoreWriterActions.usdClassTransfer(uint64(amount), false);
        } else if (direction == TransferDirection.SpotToPerps) {
            require(amount <= type(uint64).max, "Amount overflow");
            CoreWriterActions.usdClassTransfer(uint64(amount), true);
        } else if (direction == TransferDirection.SpotToEvm) {
            require(amount <= type(uint64).max / 100, "Amount overflow");
            // spotSend to USDC system address triggers EVM credit to this vault
            CoreWriterActions.spotSend(USDC_SYSTEM, 0, uint64(amount) * 100);
        } else if (direction == TransferDirection.EvmToOwner) {
            require(destination != address(0), "Invalid destination");
            require(IERC20(usdc).transfer(destination, amount), "Transfer failed");
        } else if (direction == TransferDirection.SpotToOwnerSpot) {
            require(destination != address(0), "Invalid destination");
            require(amount <= type(uint64).max / 100, "Amount overflow");
            CoreWriterActions.spotSend(destination, 0, uint64(amount) * 100);
        }
    }

    /// @notice Set initial capital baseline (6 decimals, same as USDC)
    function setInitialCapital(uint256 _capital) external onlyOwner {
        initialCapital = _capital;
    }
}

/// @title Allocator — factory and controller for AgentVaults
contract Allocator is Ownable {
    UpgradeableBeacon public beacon;

    address public immutable usdc;
    address public immutable coreDeposit;

    // Ops manager whitelist
    mapping(address => bool) public isOpsManager;

    // Vault registry
    address[] public allVaults;
    mapping(address => bool) public isVault;

    // User → Vault reverse lookup
    mapping(address => address) public userVault;

    // --- Events: Vault Lifecycle ---
    event VaultCreated(address indexed vault, uint256 index);

    // --- Events: User Lifecycle ---
    event UserAssigned(address indexed vault, address indexed user);
    event UserCleared(address indexed vault, address indexed user);
    event UserAuthorized(address indexed vault);
    event UserRevoked(address indexed vault);

    // --- Events: Fund Operations ---
    event FundsTransferred(address indexed vault, TransferDirection direction, uint256 amount);
    event EmergencyWithdraw(address indexed token, uint256 amount);

    // --- Events: Administration ---
    event OpsManagerUpdated(address indexed manager, bool status);
    event ImplementationUpdated(address indexed newImplementation);

    // --- Modifiers ---

    modifier onlyOpsManagerOrOwner() {
        require(msg.sender == owner() || isOpsManager[msg.sender], "Not authorized");
        _;
    }

    modifier validVault(address vault) {
        require(isVault[vault], "Not a valid vault");
        _;
    }

    // --- Constructor ---

    constructor(address _implementation, address _usdc, address _coreDeposit) Ownable(msg.sender) {
        beacon = new UpgradeableBeacon(_implementation, address(this));
        usdc = _usdc;
        coreDeposit = _coreDeposit;
    }

    // =========================================================================
    // Administration
    // =========================================================================

    function upgradeTo(address newImplementation) external onlyOwner {
        beacon.upgradeTo(newImplementation);
        emit ImplementationUpdated(newImplementation);
    }

    function setOpsManager(address manager, bool status) external onlyOwner {
        isOpsManager[manager] = status;
        emit OpsManagerUpdated(manager, status);
    }

    // =========================================================================
    // Vault Creation
    // =========================================================================

    function batchCreate(uint256 count) external onlyOpsManagerOrOwner returns (address[] memory vaults) {
        vaults = new address[](count);
        for (uint256 i = 0; i < count; i++) {
            BeaconProxy proxy = new BeaconProxy(address(beacon), "");
            address vault = address(proxy);
            AgentVault(vault).initialize(address(this), usdc, coreDeposit);
            isVault[vault] = true;
            allVaults.push(vault);
            vaults[i] = vault;
            emit VaultCreated(vault, allVaults.length - 1);
        }
    }

    // =========================================================================
    // User Management
    // =========================================================================

    function userAssign(address vault, address _user) external onlyOpsManagerOrOwner validVault(vault) {
        _userAssign(vault, _user);
    }

    function userClear(address vault) external onlyOpsManagerOrOwner validVault(vault) {
        _userClear(vault);
    }

    function userAuthorize(address vault) external onlyOpsManagerOrOwner validVault(vault) {
        AgentVault(vault).userAuthorize();
        emit UserAuthorized(vault);
    }

    function userRevoke(address vault) external onlyOpsManagerOrOwner validVault(vault) {
        AgentVault(vault).userRevoke();
        emit UserRevoked(vault);
    }

    function batchUserAssign(address[] calldata vaults, address[] calldata users) external onlyOpsManagerOrOwner {
        require(vaults.length == users.length, "Length mismatch");
        for (uint256 i = 0; i < vaults.length; i++) {
            require(isVault[vaults[i]], "Not a valid vault");
            _userAssign(vaults[i], users[i]);
        }
    }

    function batchUserClear(address[] calldata vaults) external onlyOpsManagerOrOwner {
        for (uint256 i = 0; i < vaults.length; i++) {
            require(isVault[vaults[i]], "Not a valid vault");
            _userClear(vaults[i]);
        }
    }

    function batchUserAuthorize(address[] calldata vaults) external onlyOpsManagerOrOwner {
        for (uint256 i = 0; i < vaults.length; i++) {
            require(isVault[vaults[i]], "Not a valid vault");
            AgentVault(vaults[i]).userAuthorize();
            emit UserAuthorized(vaults[i]);
        }
    }

    function batchUserRevoke(address[] calldata vaults) external onlyOpsManagerOrOwner {
        for (uint256 i = 0; i < vaults.length; i++) {
            require(isVault[vaults[i]], "Not a valid vault");
            AgentVault(vaults[i]).userRevoke();
            emit UserRevoked(vaults[i]);
        }
    }

    function _userAssign(address vault, address _user) internal {
        require(_user != address(0), "Invalid user");
        require(AgentVault(vault).user() == address(0), "Vault already has a user");
        require(userVault[_user] == address(0), "User already has a vault");
        AgentVault(vault).userAssign(_user);
        userVault[_user] = vault;
        emit UserAssigned(vault, _user);
    }

    function _userClear(address vault) internal {
        address oldUser = AgentVault(vault).user();
        if (oldUser == address(0)) return; // no user — skip silently
        userVault[oldUser] = address(0);
        AgentVault(vault).userClear();
        emit UserCleared(vault, oldUser);
    }

    // =========================================================================
    // Fund Transfer (Owner ↔ EVM ↔ Perps ↔ Spot)
    // =========================================================================

    function transferAgentAsset(address vault, TransferDirection direction, uint256 amount) external onlyOpsManagerOrOwner validVault(vault) {
        _transferOne(vault, direction, amount);
    }

    function batchTransferAgentAsset(address[] calldata vaults, TransferDirection direction, uint256[] calldata amounts) external onlyOpsManagerOrOwner {
        require(vaults.length == amounts.length, "Length mismatch");
        if (direction == TransferDirection.EvmToOwner) {
            // Optimized: collect all to Allocator first, then single transfer to Owner
            uint256 total = 0;
            for (uint256 i = 0; i < vaults.length; i++) {
                require(isVault[vaults[i]], "Not a valid vault");
                AgentVault(vaults[i]).transferAgentAsset(direction, amounts[i], address(this));
                total += amounts[i];
                emit FundsTransferred(vaults[i], direction, amounts[i]);
            }
            require(IERC20(usdc).transfer(owner(), total), "Transfer failed");
        } else {
            for (uint256 i = 0; i < vaults.length; i++) {
                require(isVault[vaults[i]], "Not a valid vault");
                _transferOne(vaults[i], direction, amounts[i]);
            }
        }
    }

    function _transferOne(address vault, TransferDirection direction, uint256 amount) internal {
        if (direction == TransferDirection.OwnerToEvm) {
            // OwnerToEvm: Allocator pulls from caller then sends to vault (allowance is on Allocator)
            require(IERC20(usdc).transferFrom(msg.sender, vault, amount), "TransferFrom failed");
        } else if (direction == TransferDirection.EvmToOwner || direction == TransferDirection.SpotToOwnerSpot) {
            // Needs destination = owner()
            AgentVault(vault).transferAgentAsset(direction, amount, owner());
        } else {
            // L1 operations — no destination needed
            AgentVault(vault).transferAgentAsset(direction, amount, address(0));
        }
        emit FundsTransferred(vault, direction, amount);
    }

    // =========================================================================
    // Initial Capital
    // =========================================================================

    function setInitialCapital(address vault, uint256 capital) external onlyOpsManagerOrOwner validVault(vault) {
        AgentVault(vault).setInitialCapital(capital);
    }

    function batchSetInitialCapital(address[] calldata vaults, uint256[] calldata capitals) external onlyOpsManagerOrOwner {
        require(vaults.length == capitals.length, "Length mismatch");
        for (uint256 i = 0; i < vaults.length; i++) {
            require(isVault[vaults[i]], "Not a valid vault");
            AgentVault(vaults[i]).setInitialCapital(capitals[i]);
        }
    }

    // =========================================================================
    // Emergency
    // =========================================================================

    function emergencyWithdraw(address token, uint256 amount) external onlyOwner {
        require(IERC20(token).transfer(owner(), amount), "Transfer failed");
        emit EmergencyWithdraw(token, amount);
    }

    // =========================================================================
    // View Helpers
    // =========================================================================

    function vaultCount() external view returns (uint256) {
        return allVaults.length;
    }

    function getVaultsByRange(uint256 start, uint256 count) external view returns (address[] memory) {
        if (start >= allVaults.length) {
            return new address[](0);
        }
        uint256 end = start + count > allVaults.length ? allVaults.length : start + count;
        address[] memory result = new address[](end - start);
        for (uint256 i = start; i < end; i++) {
            result[i - start] = allVaults[i];
        }
        return result;
    }

    function getUserVault(address _user) external view returns (address) {
        return userVault[_user];
    }

    function getVaultsInfo(address[] calldata vaults) external view returns (
        address[] memory users,
        uint256[] memory balances,
        bool[] memory valids,
        uint256[] memory capitals
    ) {
        uint256 len = vaults.length;
        users = new address[](len);
        balances = new uint256[](len);
        valids = new bool[](len);
        capitals = new uint256[](len);
        for (uint256 i = 0; i < len; i++) {
            if (isVault[vaults[i]]) {
                valids[i] = true;
                users[i] = AgentVault(vaults[i]).user();
                balances[i] = IERC20(usdc).balanceOf(vaults[i]);
                capitals[i] = AgentVault(vaults[i]).initialCapital();
            }
        }
    }
}
