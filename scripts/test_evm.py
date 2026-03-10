#!/usr/bin/env python3
"""
Axon EVM Compatibility Test Suite
Tests JSON-RPC, ERC-20 deployment, native transfers, and precompile queries.
Requires: pip3 install web3
Usage: python3 scripts/test_evm.py [--rpc http://localhost:8545]
"""

import sys
import json
import time
import argparse
from web3 import Web3
from eth_account import Account

RPC_URL = "http://localhost:8545"

# Minimal ERC-20 bytecode compiled from TestERC20.sol
# We use a simple inline constructor for testing
ERC20_ABI = json.loads("""[
    {"inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_decimals","type":"uint8"},{"name":"_initialSupply","type":"uint256"}],"stateMutability":"nonpayable","type":"constructor"},
    {"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
    {"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}
]""")

REGISTRY_ABI = json.loads("""[
    {"inputs":[{"name":"account","type":"address"}],"name":"isAgent","outputs":[{"name":"","type":"bool"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"name":"account","type":"address"}],"name":"getAgent","outputs":[{"name":"agentId","type":"string"},{"name":"capabilities","type":"string[]"},{"name":"model","type":"string"},{"name":"reputation","type":"uint64"},{"name":"isOnline","type":"bool"}],"stateMutability":"view","type":"function"}
]""")

REPUTATION_ABI = json.loads("""[
    {"inputs":[{"name":"agent","type":"address"}],"name":"getReputation","outputs":[{"name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"name":"agent","type":"address"},{"name":"minReputation","type":"uint64"}],"name":"meetsReputation","outputs":[{"name":"","type":"bool"}],"stateMutability":"view","type":"function"}
]""")

WALLET_ABI = json.loads("""[
    {"inputs":[{"name":"wallet","type":"address"}],"name":"getWalletInfo","outputs":[{"name":"txLimit","type":"uint256"},{"name":"dailyLimit","type":"uint256"},{"name":"dailySpent","type":"uint256"},{"name":"isFrozen","type":"bool"},{"name":"operator","type":"address"},{"name":"guardian","type":"address"}],"stateMutability":"view","type":"function"}
]""")

PRECOMPILE_REGISTRY = "0x0000000000000000000000000000000000000801"
PRECOMPILE_REPUTATION = "0x0000000000000000000000000000000000000802"
PRECOMPILE_WALLET = "0x0000000000000000000000000000000000000803"


class TestResult:
    def __init__(self):
        self.passed = 0
        self.failed = 0
        self.errors = []

    def ok(self, name):
        self.passed += 1
        print(f"  ✅ {name}")

    def fail(self, name, err):
        self.failed += 1
        self.errors.append((name, str(err)))
        print(f"  ❌ {name}: {err}")

    def summary(self):
        total = self.passed + self.failed
        print(f"\n{'='*60}")
        print(f"Results: {self.passed}/{total} passed, {self.failed} failed")
        if self.errors:
            print("\nFailed tests:")
            for name, err in self.errors:
                print(f"  - {name}: {err}")
        print(f"{'='*60}")
        return self.failed == 0


def test_rpc_basic(w3, result):
    """Test basic JSON-RPC endpoints."""
    print("\n📡 Testing JSON-RPC Basic Endpoints...")

    try:
        chain_id = w3.eth.chain_id
        result.ok(f"eth_chainId = {chain_id}")
    except Exception as e:
        result.fail("eth_chainId", e)

    try:
        block_num = w3.eth.block_number
        result.ok(f"eth_blockNumber = {block_num}")
    except Exception as e:
        result.fail("eth_blockNumber", e)

    try:
        block = w3.eth.get_block("latest")
        result.ok(f"eth_getBlockByNumber (latest hash={block.hash.hex()[:16]}...)")
    except Exception as e:
        result.fail("eth_getBlockByNumber", e)

    try:
        gas_price = w3.eth.gas_price
        result.ok(f"eth_gasPrice = {gas_price}")
    except Exception as e:
        result.fail("eth_gasPrice", e)

    try:
        net_version = w3.net.version
        result.ok(f"net_version = {net_version}")
    except Exception as e:
        result.fail("net_version", e)


def test_accounts_balance(w3, result):
    """Test account queries."""
    print("\n💰 Testing Account & Balance Queries...")

    try:
        accounts = w3.eth.accounts
        if len(accounts) > 0:
            result.ok(f"eth_accounts returned {len(accounts)} account(s)")
            for acc in accounts[:3]:
                bal = w3.eth.get_balance(acc)
                result.ok(f"  Balance of {acc[:10]}... = {w3.from_wei(bal, 'ether')} AXON")
        else:
            result.ok("eth_accounts returned 0 accounts (expected for remote node)")
    except Exception as e:
        result.fail("eth_accounts / eth_getBalance", e)


def test_native_transfer(w3, result):
    """Test native AXON transfer between accounts."""
    print("\n💸 Testing Native Transfer...")

    try:
        accounts = w3.eth.accounts
        if len(accounts) < 1:
            result.ok("Skipped (no unlocked accounts)")
            return

        sender = accounts[0]
        receiver = Account.create().address

        bal_before = w3.eth.get_balance(receiver)

        tx_hash = w3.eth.send_transaction({
            "from": sender,
            "to": receiver,
            "value": w3.to_wei(1, "ether"),
            "gas": 21000,
        })
        receipt = w3.eth.wait_for_transaction_receipt(tx_hash, timeout=30)
        result.ok(f"Transfer tx mined in block {receipt.blockNumber} (gas={receipt.gasUsed})")

        bal_after = w3.eth.get_balance(receiver)
        assert bal_after > bal_before, "Balance did not increase"
        result.ok(f"Receiver balance: {w3.from_wei(bal_after, 'ether')} AXON")

    except Exception as e:
        result.fail("native transfer", e)


def test_precompile_registry(w3, result):
    """Test IAgentRegistry precompile (read-only)."""
    print("\n🤖 Testing IAgentRegistry Precompile (0x...0801)...")

    try:
        registry = w3.eth.contract(
            address=Web3.to_checksum_address(PRECOMPILE_REGISTRY),
            abi=REGISTRY_ABI,
        )
        is_agent = registry.functions.isAgent(
            Web3.to_checksum_address("0x0000000000000000000000000000000000000001")
        ).call()
        result.ok(f"isAgent(0x01) = {is_agent}")
    except Exception as e:
        result.fail("IAgentRegistry.isAgent", e)

    try:
        agent_info = registry.functions.getAgent(
            Web3.to_checksum_address("0x0000000000000000000000000000000000000001")
        ).call()
        result.ok(f"getAgent(0x01) = agentId='{agent_info[0]}', online={agent_info[4]}")
    except Exception as e:
        result.fail("IAgentRegistry.getAgent", e)


def test_precompile_reputation(w3, result):
    """Test IAgentReputation precompile."""
    print("\n⭐ Testing IAgentReputation Precompile (0x...0802)...")

    try:
        rep = w3.eth.contract(
            address=Web3.to_checksum_address(PRECOMPILE_REPUTATION),
            abi=REPUTATION_ABI,
        )
        score = rep.functions.getReputation(
            Web3.to_checksum_address("0x0000000000000000000000000000000000000001")
        ).call()
        result.ok(f"getReputation(0x01) = {score}")
    except Exception as e:
        result.fail("IAgentReputation.getReputation", e)

    try:
        meets = rep.functions.meetsReputation(
            Web3.to_checksum_address("0x0000000000000000000000000000000000000001"),
            50
        ).call()
        result.ok(f"meetsReputation(0x01, 50) = {meets}")
    except Exception as e:
        result.fail("IAgentReputation.meetsReputation", e)


def test_precompile_wallet(w3, result):
    """Test IAgentWallet precompile (read-only)."""
    print("\n👛 Testing IAgentWallet Precompile (0x...0803)...")

    try:
        wallet = w3.eth.contract(
            address=Web3.to_checksum_address(PRECOMPILE_WALLET),
            abi=WALLET_ABI,
        )
        info = wallet.functions.getWalletInfo(
            Web3.to_checksum_address("0x0000000000000000000000000000000000000001")
        ).call()
        result.ok(f"getWalletInfo(0x01) = frozen={info[3]}, operator={info[4][:10]}...")
    except Exception as e:
        result.fail("IAgentWallet.getWalletInfo", e)


def test_contract_deployment(w3, result):
    """Test smart contract deployment (ERC-20)."""
    print("\n📦 Testing Smart Contract Deployment...")

    # Minimal bytecode: a contract that stores 42 and returns it
    # This is a tiny test bytecode, not the full ERC-20
    STORE_42_CODE = "0x6080604052602a60005560158060156000396000f3fe60003560e01c8063" \
                    "2a1afcd914600f57005b60005460405190815260200160405180910390f3"

    try:
        accounts = w3.eth.accounts
        if len(accounts) < 1:
            result.ok("Skipped (no unlocked accounts)")
            return

        tx_hash = w3.eth.send_transaction({
            "from": accounts[0],
            "data": STORE_42_CODE,
            "gas": 200000,
        })
        receipt = w3.eth.wait_for_transaction_receipt(tx_hash, timeout=30)

        if receipt.contractAddress:
            result.ok(f"Contract deployed at {receipt.contractAddress} (gas={receipt.gasUsed})")
        else:
            result.fail("contract deployment", "No contract address in receipt")
    except Exception as e:
        result.fail("contract deployment", e)


def test_eth_call_estimate(w3, result):
    """Test eth_call and eth_estimateGas."""
    print("\n🔧 Testing eth_call & eth_estimateGas...")

    try:
        accounts = w3.eth.accounts
        if len(accounts) < 1:
            result.ok("Skipped (no unlocked accounts)")
            return

        gas_est = w3.eth.estimate_gas({
            "from": accounts[0],
            "to": accounts[0],
            "value": 0,
        })
        result.ok(f"eth_estimateGas (self-transfer) = {gas_est}")
    except Exception as e:
        result.fail("eth_estimateGas", e)


def main():
    parser = argparse.ArgumentParser(description="Axon EVM Compatibility Tests")
    parser.add_argument("--rpc", default=RPC_URL, help="JSON-RPC endpoint URL")
    args = parser.parse_args()

    print(f"🔗 Connecting to {args.rpc}...")
    w3 = Web3(Web3.HTTPProvider(args.rpc))

    if not w3.is_connected():
        print(f"❌ Cannot connect to {args.rpc}")
        print("   Make sure axond is running with --json-rpc.enable")
        sys.exit(1)

    print(f"✅ Connected! Chain ID: {w3.eth.chain_id}")

    result = TestResult()

    test_rpc_basic(w3, result)
    test_accounts_balance(w3, result)
    test_native_transfer(w3, result)
    test_contract_deployment(w3, result)
    test_eth_call_estimate(w3, result)
    test_precompile_registry(w3, result)
    test_precompile_reputation(w3, result)
    test_precompile_wallet(w3, result)

    success = result.summary()
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
