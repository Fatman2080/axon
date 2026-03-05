// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./interfaces/IHyperliquidEVM.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

// Minimal ERC20 Interface
interface IERC20 {
    function transfer(address recipient, uint256 amount) external returns (bool);
    function transferFrom(address sender, address recipient, uint256 amount) external returns (bool);
    function approve(address spender, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

interface ICoreDepositWallet {
    function deposit(uint256 amount, uint32 destination) external;
}

// Simple Agent Vault that holds funds and delegates to an API wallet (the agent)
contract AgentVault {
    // Inactive: Initial state or after rotation/reset, waiting for funds/activation
    // Active: Authorized and trading
    // Revoked: Trading stopped by Risk Manager
    enum Status { Inactive, Active, Revoked }

    address public factory;
    address public owner; // The Allocator
    address public agent; // The trading agent (API Wallet)
    bool public initialized;
    Status public status;
    
    // Hyperliquid Mainnet Constants
    // USDC Token Address (EVM)
    address public USDC;
    // Core Deposit Bridge Address (EVM)
    address public CORE_DEPOSIT;
    // USDC System Address (L1) - Index 0
    address constant USDC_SYSTEM = 0x2000000000000000000000000000000000000000;

    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }

    function initialize(address _owner, address _agent, address _usdc, address _coreDeposit) external {
        require(!initialized, "Already initialized");
        initialized = true;
        factory = msg.sender;
        owner = _owner;
        agent = _agent;
        USDC = _usdc;
        CORE_DEPOSIT = _coreDeposit;
        status = Status.Inactive;
        
        // Removed automatic delegation to prevent initialization deadlock
        // Delegation will happen upon first bridge/deposit
    }

    // Bridge funds from EVM to Exchange L1 (Perps Account)
    function bridgeToL1(uint256 amount) external onlyOwner {
        // 1. Approve CoreDepositWallet
        IERC20(USDC).approve(CORE_DEPOSIT, amount);
        
        // 2. Deposit to Perps (destination = 0)
        // This moves funds from this contract's EVM balance to its Exchange L1 balance
        // Note: This action creates the L1 account if it doesn't exist.
        ICoreDepositWallet(CORE_DEPOSIT).deposit(amount, 0);

        // 3. Re-authorize Agent
        // Since the L1 account might have just been created by the deposit, 
        // we must ensure the agent is authorized NOW.
        CoreWriterActions.addApiWallet(agent, "AgentVault");
        status = Status.Active;
    }

    // Withdraw from Perps to Spot (L1 internal transfer)
    // Only owner (Allocator) can call this
    function withdrawFromPerp(uint64 amount) external onlyOwner {
        // Move amount from Perps (false) -> Spot
        // toPerp = false means Perp -> Spot
        CoreWriterActions.usdClassTransfer(amount, false);
    }

    // Manual Re-authorization helper
    function authorize() external onlyOwner {
        CoreWriterActions.addApiWallet(agent, "AgentVault");
        status = Status.Active;
    }

    // Bridge funds from Exchange L1 (Spot) back to EVM
    function bridgeToEVM(uint64 amount) external onlyOwner {
        // Send USDC from L1 Spot (self) to USDC System Address
        // This triggers the bridge to credit this contract on EVM
        // Note: L1 USDC has 8 decimals, EVM USDC has 6.
        // We receive `amount` in EVM units (6 dec). We must scale to L1 units (8 dec).
        uint64 l1Amount = amount * 100;

        CoreWriterActions.sendAsset(
            USDC_SYSTEM, // destination (System Address)
            address(0),  // subAccount (0 = default)
            4294967295,  // sourceDex (Spot = max uint32)
            4294967295,  // destDex (Spot = max uint32)
            0,           // tokenIndex (USDC = 0)
            l1Amount     // amount (L1 USDC units)
        );
    }

    // Allow owner (Allocator) to withdraw funds
    function withdraw(address token, uint256 amount) external onlyOwner {
        // Transfer logic (standard ERC20 transfer)
        IERC20(token).transfer(owner, amount);
    }

    // Explicitly revoke agent permission
    function revokeDelegate() external onlyOwner {
        // Delegate to the Vault itself (this contract address) to freeze trading.
        CoreWriterActions.addApiWallet(address(this), "Revoked");
        status = Status.Revoked;
    }

    // Set new agent (rotate key)
    function setAgent(address newAgent) external onlyOwner {
        agent = newAgent;
        // Authorize new agent
        CoreWriterActions.addApiWallet(newAgent, "AgentVault");
        // Reset to Inactive, waiting for funds or explicit activation if needed.
        // Or keep Active if we assume rotation means immediate trading.
        // Given user request "replenish then re-allocate", Inactive is safer.
        status = Status.Inactive;
    }

    // Allow owner to execute arbitrary actions (like removing delegate)
    function execute(address target, bytes calldata data) external onlyOwner {
        (bool success, ) = target.call(data);
        require(success, "Execution failed");
    }
}

contract Allocator is Ownable {
    UpgradeableBeacon public beacon;
    
    // Configurable addresses
    address public immutable USDC;
    address public immutable CORE_DEPOSIT;
    
    // Risk Managers whitelist
    mapping(address => bool) public isRiskManager;

    // Mapping from Agent EOA to their Vault contract
    mapping(address => address) public agentVaults;
    address[] public allAgents;

    event AgentAllocated(address indexed agent, address indexed vault, uint256 amount);
    event AgentDeallocated(address indexed agent, address indexed vault, uint256 amount);
    event AgentBridged(address indexed agent, address indexed vault, uint256 amount);
    event AgentRevoked(address indexed agent, address indexed vault);
    event Withdrawal(address indexed token, uint256 amount);
    event RiskManagerUpdated(address indexed manager, bool status);
    event ImplementationUpdated(address indexed newImplementation);

    modifier onlyRiskManagerOrOwner() {
        require(msg.sender == owner() || isRiskManager[msg.sender], "Not authorized");
        _;
    }

    constructor(address _implementation, address _usdc, address _coreDeposit) Ownable(msg.sender) {
        // 1. Deploy Beacon, pointing to initial logic
        beacon = new UpgradeableBeacon(_implementation, msg.sender);
        USDC = _usdc;
        CORE_DEPOSIT = _coreDeposit;
    }

    // --- Administration ---
    
    // Upgrade all Vaults to new implementation
    function upgradeTo(address newImplementation) external onlyOwner {
        beacon.upgradeTo(newImplementation);
        emit ImplementationUpdated(newImplementation);
    }

    // --- Risk Management ---
    function setRiskManager(address manager, bool status) external onlyOwner {
        isRiskManager[manager] = status;
        emit RiskManagerUpdated(manager, status);
    }

    // Create a new sub-vault for an agent if it doesn't exist
    function getOrCreateAgentVault(address agent) public returns (address) {
        if (agentVaults[agent] == address(0)) {
            // Deploy BeaconProxy instead of Clones.clone
            // BeaconProxy will ask beacon "where is the implementation?"
            BeaconProxy proxy = new BeaconProxy(
                address(beacon),
                "" // Don't initialize in constructor, call manually later
            );
            
            address vault = address(proxy);
            
            // Initialize Proxy with configured addresses
            AgentVault(vault).initialize(address(this), agent, USDC, CORE_DEPOSIT);
            
            agentVaults[agent] = vault;
            allAgents.push(agent);
        }
        return agentVaults[agent];
    }

    // Batch create vaults

    // Batch create vaults
    function batchCreate(address[] calldata agents) external returns (address[] memory vaults) {
        vaults = new address[](agents.length);
        for (uint256 i = 0; i < agents.length; i++) {
            vaults[i] = getOrCreateAgentVault(agents[i]);
        }
    }

    // Allocate funds to an agent (pull from user -> Allocator -> AgentVault)
    // User must approve Allocator first!
    function allocate(address agent, address token, uint256 amount) external {
        address vault = getOrCreateAgentVault(agent);
        IERC20(token).transferFrom(msg.sender, vault, amount);
        emit AgentAllocated(agent, vault, amount);
    }

    // Batch allocate funds (User -> Allocator -> Vaults)
    // User must approve Allocator for total amount first
    function batchAllocate(address[] calldata agents, address token, uint256[] calldata amounts) external {
        require(agents.length == amounts.length, "Length mismatch");
        for (uint256 i = 0; i < agents.length; i++) {
            // Re-use logic inline to save gas on external calls? Or just call getOrCreate
            address vault = getOrCreateAgentVault(agents[i]);
            IERC20(token).transferFrom(msg.sender, vault, amounts[i]);
            emit AgentAllocated(agents[i], vault, amounts[i]);
        }
    }

    // Bridge funds from AgentVault EVM to Exchange L1
    function bridge(address agent, uint256 amount) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");
        
        AgentVault(vault).bridgeToL1(amount);
        emit AgentBridged(agent, vault, amount);
    }

    // Batch bridge to L1
    function batchBridge(address[] calldata agents, uint256[] calldata amounts) external onlyRiskManagerOrOwner {
        require(agents.length == amounts.length, "Length mismatch");
        for (uint256 i = 0; i < agents.length; i++) {
            address vault = agentVaults[agents[i]];
            require(vault != address(0), "Agent vault not found");
            AgentVault(vault).bridgeToL1(amounts[i]);
            emit AgentBridged(agents[i], vault, amounts[i]);
        }
    }

    // Manually authorize agent (Fix for "User not found" issue)
    function authorizeAgent(address agent) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");
        
        AgentVault(vault).authorize();
        // Emit an event? reusing AgentAllocated implies success, maybe just silent or new event.
    }

    // Withdraw from Perp to Spot (Fix for stuck funds)
    function withdrawFromPerp(address agent, uint64 amount) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");
        
        // Call AgentVault.withdrawFromPerp
        AgentVault(vault).withdrawFromPerp(amount);
    }

    // Rotate agent key for a vault
    function rotateAgent(address oldAgent, address newAgent) external onlyRiskManagerOrOwner {
        address vault = agentVaults[oldAgent];
        require(vault != address(0), "Agent vault not found");
        require(agentVaults[newAgent] == address(0), "New agent already has a vault");

        // 1. Update Vault contract state and authorize new agent on L1
        AgentVault(vault).setAgent(newAgent);

        // 2. Update Allocator mapping
        agentVaults[newAgent] = vault;
        agentVaults[oldAgent] = address(0); 

        // 3. Update allAgents list
        allAgents.push(newAgent);
        
        // 4. Emit event
        emit AgentAllocated(newAgent, vault, 0); 
    }

    // Bridge funds from Exchange L1 back to AgentVault EVM
    function bridgeBack(address agent, uint64 amount) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");
        
        AgentVault(vault).bridgeToEVM(amount);
        emit AgentBridged(agent, vault, amount); 
    }

    // Deallocate funds (pull back from agent vault to Allocator, then to Owner)
    // Authorized: Owner OR Risk Manager
    function deallocate(address agent, address token, uint256 amount) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");

        // 1. Pull from Vault to Allocator
        AgentVault(vault).withdraw(token, amount);
        
        // 2. Send from Allocator to Owner (NOT to msg.sender)
        // This ensures Risk Manager can only save funds to cold wallet, not steal them.
        IERC20(token).transfer(owner(), amount);
        
        emit AgentDeallocated(agent, vault, amount);
    }

    // Force revoke agent permission without moving funds
    // Authorized: Owner OR Risk Manager
    function revokeAgent(address agent) external onlyRiskManagerOrOwner {
        address vault = agentVaults[agent];
        require(vault != address(0), "Agent vault not found");
        
        AgentVault(vault).revokeDelegate();
        emit AgentRevoked(agent, vault);
    }

    // Emergency Withdraw from Allocator itself
    // Authorized: Only Owner (Risk Manager cannot touch Allocator's own balance)
    function withdraw(address token, uint256 amount) external onlyOwner {
        IERC20(token).transfer(owner(), amount);
        emit Withdrawal(token, amount);
    }

    // View helper
    function getAgentVault(address agent) external view returns (address) {
        return agentVaults[agent];
    }

    function getAgentStatus(address agent) external view returns (AgentVault.Status) {
        address vault = agentVaults[agent];
        if (vault == address(0)) {
            return AgentVault.Status.Inactive; // Default
        }
        return AgentVault(vault).status();
    }

    // Batch view helper
    function getAgentsInfo(address[] calldata agents) external view returns (AgentVault.Status[] memory statuses, uint256[] memory balances) {
        statuses = new AgentVault.Status[](agents.length);
        balances = new uint256[](agents.length);
        
        for (uint256 i = 0; i < agents.length; i++) {
            address vault = agentVaults[agents[i]];
            if (vault != address(0)) {
                statuses[i] = AgentVault(vault).status();
                // Get EVM balance of the vault
                balances[i] = IERC20(USDC).balanceOf(vault);
            } else {
                statuses[i] = AgentVault.Status.Inactive;
                balances[i] = 0;
            }
        }
    }
}
