"""Axon Agent Client — Python SDK for the Axon blockchain."""

from typing import Optional, Tuple, List, Any, Dict
from web3 import Web3
from eth_account import Account

from axon.precompiles import (
    REGISTRY_ADDRESS, REPUTATION_ADDRESS, WALLET_ADDRESS,
    REGISTRY_ABI, REPUTATION_ABI, WALLET_ABI,
)

AXON_DECIMALS = 18
ONE_AXON = 10 ** AXON_DECIMALS


class AgentClient:
    """High-level client for interacting with the Axon chain."""

    def __init__(self, rpc_url: str = "http://localhost:8545"):
        self.w3 = Web3(Web3.HTTPProvider(rpc_url))
        if not self.w3.is_connected():
            raise ConnectionError(f"Cannot connect to {rpc_url}")

        self._account = None
        self._registry = self.w3.eth.contract(
            address=Web3.to_checksum_address(REGISTRY_ADDRESS), abi=REGISTRY_ABI
        )
        self._reputation = self.w3.eth.contract(
            address=Web3.to_checksum_address(REPUTATION_ADDRESS), abi=REPUTATION_ABI
        )
        self._wallet = self.w3.eth.contract(
            address=Web3.to_checksum_address(WALLET_ADDRESS), abi=WALLET_ABI
        )

    # ---- Connection info ----

    @property
    def chain_id(self) -> int:
        return self.w3.eth.chain_id

    @property
    def block_number(self) -> int:
        return self.w3.eth.block_number

    @property
    def address(self) -> Optional[str]:
        return self._account.address if self._account else None

    def balance(self, address: Optional[str] = None) -> float:
        addr = address or self.address
        if not addr:
            raise ValueError("No address specified")
        wei = self.w3.eth.get_balance(Web3.to_checksum_address(addr))
        return wei / ONE_AXON

    # ---- Account management ----

    def set_account(self, private_key: str):
        self._account = Account.from_key(private_key)

    def create_account(self) -> Tuple[str, str]:
        acct = Account.create()
        self._account = acct
        return acct.address, acct.key.hex()

    # ---- Agent Registry (0x...0801) ----

    def is_agent(self, address: str) -> bool:
        return self._registry.functions.isAgent(
            Web3.to_checksum_address(address)
        ).call()

    def get_agent(self, address: str) -> Dict[str, Any]:
        result = self._registry.functions.getAgent(
            Web3.to_checksum_address(address)
        ).call()
        return {
            "agent_id": result[0],
            "capabilities": result[1],
            "model": result[2],
            "reputation": result[3],
            "is_online": result[4],
        }

    def register_agent(
        self, capabilities: str, model: str, stake_axon: float = 100
    ) -> str:
        self._require_account()
        stake_wei = int(stake_axon * ONE_AXON)
        tx = self._registry.functions.register(capabilities, model).build_transaction(
            self._tx_params(value=stake_wei)
        )
        return self._send_tx(tx)

    def update_agent(self, capabilities: str, model: str) -> str:
        self._require_account()
        tx = self._registry.functions.updateAgent(
            capabilities, model
        ).build_transaction(self._tx_params())
        return self._send_tx(tx)

    def heartbeat(self) -> str:
        self._require_account()
        tx = self._registry.functions.heartbeat().build_transaction(self._tx_params())
        return self._send_tx(tx)

    def deregister(self) -> str:
        self._require_account()
        tx = self._registry.functions.deregister().build_transaction(self._tx_params())
        return self._send_tx(tx)

    # ---- Reputation (0x...0802) ----

    def get_reputation(self, address: str) -> int:
        return self._reputation.functions.getReputation(
            Web3.to_checksum_address(address)
        ).call()

    def get_reputations(self, addresses: List[str]) -> List[int]:
        addrs = [Web3.to_checksum_address(a) for a in addresses]
        return self._reputation.functions.getReputations(addrs).call()

    def meets_reputation(self, address: str, min_rep: int) -> bool:
        return self._reputation.functions.meetsReputation(
            Web3.to_checksum_address(address), min_rep
        ).call()

    # ---- Wallet (0x...0803) ----

    def create_wallet(
        self,
        tx_limit_axon: float,
        daily_limit_axon: float,
        cooldown_blocks: int,
        guardian: str,
    ) -> str:
        self._require_account()
        tx = self._wallet.functions.createWallet(
            int(tx_limit_axon * ONE_AXON),
            int(daily_limit_axon * ONE_AXON),
            cooldown_blocks,
            Web3.to_checksum_address(guardian),
        ).build_transaction(self._tx_params())
        return self._send_tx(tx)

    def get_wallet_info(self, wallet_address: str) -> Dict[str, Any]:
        result = self._wallet.functions.getWalletInfo(
            Web3.to_checksum_address(wallet_address)
        ).call()
        return {
            "tx_limit": result[0] / ONE_AXON,
            "daily_limit": result[1] / ONE_AXON,
            "daily_spent": result[2] / ONE_AXON,
            "is_frozen": result[3],
            "operator": result[4],
            "guardian": result[5],
        }

    # ---- Smart Contract deployment ----

    def deploy_contract(self, bytecode: str, **kwargs) -> Tuple[str, str]:
        self._require_account()
        tx = {
            **self._tx_params(),
            "data": bytecode if bytecode.startswith("0x") else f"0x{bytecode}",
            "gas": kwargs.get("gas", 3_000_000),
        }
        tx_hash = self._send_tx(tx)
        receipt = self.w3.eth.wait_for_transaction_receipt(tx_hash, timeout=60)
        return tx_hash, receipt.contractAddress

    def call_contract(
        self, address: str, abi: list, method: str, args: list = None, value: int = 0
    ) -> Any:
        contract = self.w3.eth.contract(
            address=Web3.to_checksum_address(address), abi=abi
        )
        fn = getattr(contract.functions, method)
        call_args = args or []
        return fn(*call_args).call()

    def send_contract_tx(
        self, address: str, abi: list, method: str, args: list = None, value: int = 0
    ) -> str:
        self._require_account()
        contract = self.w3.eth.contract(
            address=Web3.to_checksum_address(address), abi=abi
        )
        fn = getattr(contract.functions, method)
        call_args = args or []
        tx = fn(*call_args).build_transaction(self._tx_params(value=value))
        return self._send_tx(tx)

    # ---- Transfer ----

    def transfer(self, to: str, amount_axon: float) -> str:
        self._require_account()
        tx = {
            **self._tx_params(),
            "to": Web3.to_checksum_address(to),
            "value": int(amount_axon * ONE_AXON),
            "gas": 21000,
        }
        return self._send_tx(tx)

    # ---- Internals ----

    def _require_account(self):
        if not self._account:
            raise ValueError("No account set. Call set_account() or create_account() first.")

    def _tx_params(self, value: int = 0) -> dict:
        return {
            "from": self._account.address,
            "nonce": self.w3.eth.get_transaction_count(self._account.address),
            "gas": 500_000,
            "gasPrice": self.w3.eth.gas_price or 0,
            "chainId": self.chain_id,
            "value": value,
        }

    def _send_tx(self, tx: dict) -> str:
        signed = self._account.sign_transaction(tx)
        tx_hash = self.w3.eth.send_raw_transaction(signed.raw_transaction)
        return tx_hash.hex()

    def wait_for_tx(self, tx_hash: str, timeout: int = 30) -> dict:
        receipt = self.w3.eth.wait_for_transaction_receipt(tx_hash, timeout=timeout)
        return dict(receipt)
