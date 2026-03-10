package wallet

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	cmn "github.com/cosmos/evm/precompiles/common"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/keeper"
	"github.com/axon-chain/axon/x/agent/types"
)

var (
	address = common.HexToAddress("0x0000000000000000000000000000000000000803")
	_       = vm.PrecompiledContract(&Precompile{})
)

const (
	CreateWalletMethod  = "createWallet"
	ExecuteMethod       = "execute"
	FreezeMethod        = "freeze"
	RecoverMethod       = "recover"
	GetWalletInfoMethod = "getWalletInfo"

	GasCreateWallet  = 50000
	GasExecute       = 30000
	GasFreeze        = 10000
	GasRecover       = 30000
	GasGetWalletInfo = 1000

	// BlocksPerDay at 5s/block
	BlocksPerDay int64 = 17280

	WalletKeyPrefix = "AgentWallet/"
)

// WalletInfo is stored in the x/agent KV store.
type WalletInfo struct {
	Operator       common.Address
	Guardian       common.Address
	TxLimit        *big.Int
	DailyLimit     *big.Int
	CooldownBlocks *big.Int
	DailySpent     *big.Int
	LastResetBlock int64
	IsFrozen       bool
}

type Precompile struct {
	cmn.Precompile
	abi    abi.ABI
	keeper keeper.Keeper
}

func NewPrecompile(k keeper.Keeper) (*Precompile, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse IAgentWallet ABI: %w", err)
	}
	return &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
			ContractAddress:      address,
		},
		abi:    parsed,
		keeper: k,
	}, nil
}

func (Precompile) Address() common.Address { return address }

func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	method, err := p.abi.MethodById(input[:4])
	if err != nil {
		return 0
	}
	switch method.Name {
	case CreateWalletMethod:
		return GasCreateWallet
	case ExecuteMethod:
		return GasExecute
	case FreezeMethod:
		return GasFreeze
	case RecoverMethod:
		return GasRecover
	case GetWalletInfoMethod:
		return GasGetWalletInfo
	default:
		return 0
	}
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.execute(ctx, contract, readonly, evm)
	})
}

func (p Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case CreateWalletMethod, ExecuteMethod, FreezeMethod, RecoverMethod:
		return true
	default:
		return false
	}
}

func (p Precompile) execute(ctx sdk.Context, contract *vm.Contract, readOnly bool, evm *vm.EVM) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.abi, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case CreateWalletMethod:
		return p.createWallet(ctx, contract, method, args)
	case ExecuteMethod:
		return p.executeWallet(ctx, contract, method, args, evm)
	case FreezeMethod:
		return p.freezeWallet(ctx, contract, method, args)
	case RecoverMethod:
		return p.recoverWallet(ctx, contract, method, args)
	case GetWalletInfoMethod:
		return p.getWalletInfo(ctx, method, args)
	default:
		return nil, fmt.Errorf("unknown method: %s", method.Name)
	}
}

func (p Precompile) createWallet(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("createWallet requires 4 arguments")
	}

	txLimit, _ := args[0].(*big.Int)
	dailyLimit, _ := args[1].(*big.Int)
	cooldownBlocks, _ := args[2].(*big.Int)
	guardian, _ := args[3].(common.Address)

	caller := contract.Caller()

	// Wallet address = hash(operator, block_height)
	walletAddr := generateWalletAddress(caller, ctx.BlockHeight())

	wallet := WalletInfo{
		Operator:       caller,
		Guardian:       guardian,
		TxLimit:        txLimit,
		DailyLimit:     dailyLimit,
		CooldownBlocks: cooldownBlocks,
		DailySpent:     big.NewInt(0),
		LastResetBlock: ctx.BlockHeight(),
		IsFrozen:       false,
	}

	p.setWallet(ctx, walletAddr, wallet)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_wallet_created",
		sdk.NewAttribute("wallet", walletAddr.Hex()),
		sdk.NewAttribute("operator", caller.Hex()),
		sdk.NewAttribute("guardian", guardian.Hex()),
	))

	return method.Outputs.Pack(walletAddr)
}

func (p Precompile) executeWallet(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}, evm *vm.EVM) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("execute requires 4 arguments")
	}

	walletAddr, _ := args[0].(common.Address)
	target, _ := args[1].(common.Address)
	value, _ := args[2].(*big.Int)
	// data arg[3] is calldata for the target - unused for simple transfers

	wallet, found := p.getWallet(ctx, walletAddr)
	if !found {
		return nil, fmt.Errorf("wallet not found")
	}

	if wallet.IsFrozen {
		return nil, fmt.Errorf("wallet is frozen")
	}

	if contract.Caller() != wallet.Operator {
		return nil, fmt.Errorf("only operator can execute")
	}

	if wallet.TxLimit != nil && value.Cmp(wallet.TxLimit) > 0 {
		return nil, fmt.Errorf("transaction exceeds per-tx limit")
	}

	// Reset daily spent if a new day has passed
	if ctx.BlockHeight()-wallet.LastResetBlock >= BlocksPerDay {
		wallet.DailySpent = big.NewInt(0)
		wallet.LastResetBlock = ctx.BlockHeight()
	}

	newSpent := new(big.Int).Add(wallet.DailySpent, value)
	if wallet.DailyLimit != nil && newSpent.Cmp(wallet.DailyLimit) > 0 {
		return nil, fmt.Errorf("transaction exceeds daily limit")
	}

	wallet.DailySpent = newSpent
	p.setWallet(ctx, walletAddr, wallet)

	// Execute the transfer via EVM
	if value.Sign() > 0 {
		val, _ := uint256.FromBig(value)
		evm.Context.Transfer(evm.StateDB, walletAddr, target, val)
	}

	return method.Outputs.Pack()
}

func (p Precompile) freezeWallet(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("freeze requires 1 argument")
	}
	walletAddr, _ := args[0].(common.Address)

	wallet, found := p.getWallet(ctx, walletAddr)
	if !found {
		return nil, fmt.Errorf("wallet not found")
	}

	if contract.Caller() != wallet.Guardian {
		return nil, fmt.Errorf("only guardian can freeze")
	}

	wallet.IsFrozen = true
	p.setWallet(ctx, walletAddr, wallet)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_wallet_frozen",
		sdk.NewAttribute("wallet", walletAddr.Hex()),
		sdk.NewAttribute("guardian", contract.Caller().Hex()),
	))

	return method.Outputs.Pack()
}

func (p Precompile) recoverWallet(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("recover requires 2 arguments")
	}
	walletAddr, _ := args[0].(common.Address)
	newOperator, _ := args[1].(common.Address)

	wallet, found := p.getWallet(ctx, walletAddr)
	if !found {
		return nil, fmt.Errorf("wallet not found")
	}

	if contract.Caller() != wallet.Guardian {
		return nil, fmt.Errorf("only guardian can recover")
	}

	wallet.Operator = newOperator
	wallet.IsFrozen = false
	p.setWallet(ctx, walletAddr, wallet)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_wallet_recovered",
		sdk.NewAttribute("wallet", walletAddr.Hex()),
		sdk.NewAttribute("new_operator", newOperator.Hex()),
	))

	return method.Outputs.Pack()
}

func (p Precompile) getWalletInfo(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("getWalletInfo requires 1 argument")
	}
	walletAddr, _ := args[0].(common.Address)

	wallet, found := p.getWallet(ctx, walletAddr)
	if !found {
		return method.Outputs.Pack(
			big.NewInt(0), big.NewInt(0), big.NewInt(0),
			false, common.Address{}, common.Address{},
		)
	}

	// Reset daily spent for display if a new day has passed
	dailySpent := wallet.DailySpent
	if ctx.BlockHeight()-wallet.LastResetBlock >= BlocksPerDay {
		dailySpent = big.NewInt(0)
	}

	return method.Outputs.Pack(
		wallet.TxLimit,
		wallet.DailyLimit,
		dailySpent,
		wallet.IsFrozen,
		wallet.Operator,
		wallet.Guardian,
	)
}

// Storage helpers using x/agent KV store

func walletKey(addr common.Address) []byte {
	return []byte(WalletKeyPrefix + addr.Hex())
}

func (p Precompile) setWallet(ctx sdk.Context, addr common.Address, w WalletInfo) {
	store := ctx.KVStore(p.keeper.StoreKey())
	bz := encodeWallet(w)
	store.Set(walletKey(addr), bz)
}

func (p Precompile) getWallet(ctx sdk.Context, addr common.Address) (WalletInfo, bool) {
	store := ctx.KVStore(p.keeper.StoreKey())
	bz := store.Get(walletKey(addr))
	if bz == nil {
		return WalletInfo{}, false
	}
	return decodeWallet(bz), true
}

func generateWalletAddress(operator common.Address, blockHeight int64) common.Address {
	data := append(operator.Bytes(), types.Uint64ToBytes(uint64(blockHeight))...)
	hash := common.BytesToAddress(data[:20])
	return hash
}

// Simple binary encoding for WalletInfo (no protobuf needed for internal state).
func encodeWallet(w WalletInfo) []byte {
	// Format: operator(20) + guardian(20) + txLimit(32) + dailyLimit(32) +
	//         cooldownBlocks(32) + dailySpent(32) + lastResetBlock(8) + isFrozen(1)
	buf := make([]byte, 0, 177)
	buf = append(buf, w.Operator.Bytes()...)
	buf = append(buf, w.Guardian.Bytes()...)
	buf = append(buf, common.LeftPadBytes(w.TxLimit.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(w.DailyLimit.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(w.CooldownBlocks.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(w.DailySpent.Bytes(), 32)...)
	buf = append(buf, types.Uint64ToBytes(uint64(w.LastResetBlock))...)
	if w.IsFrozen {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	return buf
}

func decodeWallet(bz []byte) WalletInfo {
	if len(bz) < 177 {
		return WalletInfo{}
	}
	w := WalletInfo{}
	w.Operator = common.BytesToAddress(bz[0:20])
	w.Guardian = common.BytesToAddress(bz[20:40])
	w.TxLimit = new(big.Int).SetBytes(bz[40:72])
	w.DailyLimit = new(big.Int).SetBytes(bz[72:104])
	w.CooldownBlocks = new(big.Int).SetBytes(bz[104:136])
	w.DailySpent = new(big.Int).SetBytes(bz[136:168])
	w.LastResetBlock = int64(types.BytesToUint64(bz[168:176]))
	w.IsFrozen = bz[176] == 1
	return w
}

const abiJSON = `[
	{
		"inputs": [
			{"name": "txLimit", "type": "uint256"},
			{"name": "dailyLimit", "type": "uint256"},
			{"name": "cooldownBlocks", "type": "uint256"},
			{"name": "guardian", "type": "address"}
		],
		"name": "createWallet",
		"outputs": [{"name": "wallet", "type": "address"}],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "wallet", "type": "address"},
			{"name": "target", "type": "address"},
			{"name": "value", "type": "uint256"},
			{"name": "data", "type": "bytes"}
		],
		"name": "execute",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [{"name": "wallet", "type": "address"}],
		"name": "freeze",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "wallet", "type": "address"},
			{"name": "newOperator", "type": "address"}
		],
		"name": "recover",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
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
			{"name": "guardian", "type": "address"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
