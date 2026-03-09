package wallet

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/axon-chain/axon/x/agent/keeper"
)

// Address: 0x0000000000000000000000000000000000000803
var Address = common.HexToAddress("0x0000000000000000000000000000000000000803")

const (
	GasCreateWallet  = 100000
	GasExecute       = 50000
	GasFreeze        = 10000
	GasRecover       = 50000
	GasGetWalletInfo = 1000
)

type Precompile struct {
	keeper keeper.Keeper
	abi    abi.ABI
}

func NewPrecompile(keeper keeper.Keeper) (*Precompile, error) {
	parsed, err := abi.JSON(strings.NewReader(ABI))
	if err != nil {
		return nil, err
	}
	return &Precompile{
		keeper: keeper,
		abi:    parsed,
	}, nil
}

func (p *Precompile) Address() common.Address {
	return Address
}

func (p *Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	method, err := p.abi.MethodById(input[:4])
	if err != nil {
		return 0
	}
	switch method.Name {
	case "createWallet":
		return GasCreateWallet
	case "execute":
		return GasExecute
	case "freeze":
		return GasFreeze
	case "recover":
		return GasRecover
	case "getWalletInfo":
		return GasGetWalletInfo
	default:
		return 0
	}
}

func (p *Precompile) Run(evm *vm.EVM, input []byte, caller common.Address, value *big.Int, readOnly bool) ([]byte, error) {
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	method, err := p.abi.MethodById(input[:4])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	switch method.Name {
	case "getWalletInfo":
		// read-only
		return p.getWalletInfo(input[4:])
	default:
		if readOnly {
			return nil, vm.ErrWriteProtection
		}
		// TODO: implement state-changing wallet operations
		return nil, vm.ErrExecutionReverted
	}
}

func (p *Precompile) getWalletInfo(input []byte) ([]byte, error) {
	// TODO: query wallet info from keeper
	return nil, vm.ErrExecutionReverted
}

const ABI = `[
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
