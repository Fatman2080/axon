package registry

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/axon-chain/axon/x/agent/keeper"
)

// Address is the precompile address for IAgentRegistry: 0x0000000000000000000000000000000000000801
var Address = common.HexToAddress("0x0000000000000000000000000000000000000801")

// Gas costs
const (
	GasIsAgent   = 200
	GasGetAgent  = 1000
	GasRegister  = 50000
	GasUpdate    = 10000
	GasHeartbeat = 5000
)

// Precompile implements the IAgentRegistry EVM precompile.
// It bridges Solidity calls to the x/agent Cosmos SDK module.
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

	methodID := input[:4]
	method, err := p.abi.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case "isAgent":
		return GasIsAgent
	case "getAgent":
		return GasGetAgent
	case "register":
		return GasRegister
	case "updateAgent":
		return GasUpdate
	case "heartbeat":
		return GasHeartbeat
	default:
		return 0
	}
}

func (p *Precompile) Run(evm *vm.EVM, input []byte, caller common.Address, value *big.Int, readOnly bool) ([]byte, error) {
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	methodID := input[:4]
	method, err := p.abi.MethodById(methodID)
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	switch method.Name {
	case "isAgent":
		return p.isAgent(args)
	case "getAgent":
		return p.getAgent(args)
	default:
		if readOnly {
			return nil, vm.ErrWriteProtection
		}
		return nil, vm.ErrExecutionReverted
	}
}

func (p *Precompile) isAgent(args []interface{}) ([]byte, error) {
	// TODO: extract address from args, query keeper, pack result
	return p.abi.Methods["isAgent"].Outputs.Pack(false)
}

func (p *Precompile) getAgent(args []interface{}) ([]byte, error) {
	// TODO: extract address, query keeper, pack full agent data
	return nil, vm.ErrExecutionReverted
}

// ABI is the Solidity ABI for the IAgentRegistry precompile
const ABI = `[
	{
		"inputs": [{"name": "account", "type": "address"}],
		"name": "isAgent",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "account", "type": "address"}],
		"name": "getAgent",
		"outputs": [
			{"name": "agentId", "type": "string"},
			{"name": "capabilities", "type": "string[]"},
			{"name": "model", "type": "string"},
			{"name": "reputation", "type": "uint64"},
			{"name": "isOnline", "type": "bool"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "capabilities", "type": "string"},
			{"name": "model", "type": "string"}
		],
		"name": "register",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "capabilities", "type": "string"},
			{"name": "model", "type": "string"}
		],
		"name": "updateAgent",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "heartbeat",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "deregister",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
