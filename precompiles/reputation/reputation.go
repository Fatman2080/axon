package reputation

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/axon-chain/axon/x/agent/keeper"
)

// Address: 0x0000000000000000000000000000000000000802
var Address = common.HexToAddress("0x0000000000000000000000000000000000000802")

const (
	GasGetReputation  = 200
	GasGetReputations = 500
	GasMeetsReputation = 200
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
	case "getReputation":
		return GasGetReputation
	case "getReputations":
		return GasGetReputations
	case "meetsReputation":
		return GasMeetsReputation
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

	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	switch method.Name {
	case "getReputation":
		return p.getReputation(args)
	case "meetsReputation":
		return p.meetsReputation(args)
	default:
		return nil, vm.ErrExecutionReverted
	}
}

func (p *Precompile) getReputation(args []interface{}) ([]byte, error) {
	// TODO: query keeper for reputation of given address
	return p.abi.Methods["getReputation"].Outputs.Pack(uint64(0))
}

func (p *Precompile) meetsReputation(args []interface{}) ([]byte, error) {
	// TODO: check reputation threshold
	return p.abi.Methods["meetsReputation"].Outputs.Pack(false)
}

const ABI = `[
	{
		"inputs": [{"name": "agent", "type": "address"}],
		"name": "getReputation",
		"outputs": [{"name": "", "type": "uint64"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "agents", "type": "address[]"}],
		"name": "getReputations",
		"outputs": [{"name": "", "type": "uint64[]"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "agent", "type": "address"},
			{"name": "minReputation", "type": "uint64"}
		],
		"name": "meetsReputation",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "view",
		"type": "function"
	}
]`
