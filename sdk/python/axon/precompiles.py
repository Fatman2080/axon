"""Axon precompile contract addresses and ABIs."""

REGISTRY_ADDRESS = "0x0000000000000000000000000000000000000801"
REPUTATION_ADDRESS = "0x0000000000000000000000000000000000000802"
WALLET_ADDRESS = "0x0000000000000000000000000000000000000803"

REGISTRY_ABI = [
    {
        "inputs": [{"name": "account", "type": "address"}],
        "name": "isAgent",
        "outputs": [{"name": "", "type": "bool"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [{"name": "account", "type": "address"}],
        "name": "getAgent",
        "outputs": [
            {"name": "agentId", "type": "string"},
            {"name": "capabilities", "type": "string[]"},
            {"name": "model", "type": "string"},
            {"name": "reputation", "type": "uint64"},
            {"name": "isOnline", "type": "bool"},
        ],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [
            {"name": "capabilities", "type": "string"},
            {"name": "model", "type": "string"},
        ],
        "name": "register",
        "outputs": [],
        "stateMutability": "payable",
        "type": "function",
    },
    {
        "inputs": [
            {"name": "capabilities", "type": "string"},
            {"name": "model", "type": "string"},
        ],
        "name": "updateAgent",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [],
        "name": "heartbeat",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [],
        "name": "deregister",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
]

REPUTATION_ABI = [
    {
        "inputs": [{"name": "agent", "type": "address"}],
        "name": "getReputation",
        "outputs": [{"name": "", "type": "uint64"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [{"name": "agents", "type": "address[]"}],
        "name": "getReputations",
        "outputs": [{"name": "", "type": "uint64[]"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [
            {"name": "agent", "type": "address"},
            {"name": "minReputation", "type": "uint64"},
        ],
        "name": "meetsReputation",
        "outputs": [{"name": "", "type": "bool"}],
        "stateMutability": "view",
        "type": "function",
    },
]

WALLET_ABI = [
    {
        "inputs": [
            {"name": "txLimit", "type": "uint256"},
            {"name": "dailyLimit", "type": "uint256"},
            {"name": "cooldownBlocks", "type": "uint256"},
            {"name": "guardian", "type": "address"},
        ],
        "name": "createWallet",
        "outputs": [{"name": "wallet", "type": "address"}],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [
            {"name": "wallet", "type": "address"},
            {"name": "target", "type": "address"},
            {"name": "value", "type": "uint256"},
            {"name": "data", "type": "bytes"},
        ],
        "name": "execute",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [{"name": "wallet", "type": "address"}],
        "name": "freeze",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [
            {"name": "wallet", "type": "address"},
            {"name": "newOperator", "type": "address"},
        ],
        "name": "recover",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
    {
        "inputs": [{"name": "wallet", "type": "address"}],
        "name": "getWalletInfo",
        "outputs": [
            {"name": "txLimit", "type": "uint256"},
            {"name": "dailyLimit", "type": "uint256"},
            {"name": "dailySpent", "type": "uint256"},
            {"name": "isFrozen", "type": "bool"},
            {"name": "operator", "type": "address"},
            {"name": "guardian", "type": "address"},
        ],
        "stateMutability": "view",
        "type": "function",
    },
]
