package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EVMClient is a read-only client for querying on-chain data from the Allocator
// contract and ERC-20 USDC balances.
type EVMClient struct {
	client           *ethclient.Client
	allocatorAddress common.Address
	usdcAddress      common.Address
}

// AgentInfo holds the on-chain status and EVM USDC balance for a single agent.
type AgentInfo struct {
	VaultAddress common.Address
	Status       string  // "inactive", "active", "revoked"
	EVMBalance   float64 // USDC on EVM (6 decimals converted to float)
}

// Agent status string constants matching the contract enum.
const (
	AgentStatusInactive = "inactive"
	AgentStatusActive   = "active"
	AgentStatusRevoked  = "revoked"
)

// Function selectors (first 4 bytes of keccak256 hash).
var (
	selectorAgentVaults = crypto.Keccak256([]byte("agentVaults(address)"))[:4]
	selectorUSDC        = crypto.Keccak256([]byte("USDC()"))[:4]
	selectorGetAgentsInfo = crypto.Keccak256([]byte("getAgentsInfo(address[])"))[:4]
)

// NewEVMClient dials the RPC endpoint, reads the USDC address from the
// Allocator contract and caches it for subsequent balance queries.
func NewEVMClient(rpcURL string, allocatorAddr string) (*EVMClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("evm dial: %w", err)
	}

	ec := &EVMClient{
		client:           client,
		allocatorAddress: common.HexToAddress(allocatorAddr),
	}

	// Read USDC() from Allocator and cache.
	usdcAddr, err := ec.callAddress(ctx, ec.allocatorAddress, selectorUSDC, nil)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("evm read USDC address: %w", err)
	}
	ec.usdcAddress = usdcAddr
	return ec, nil
}

// GetVaultAddress calls Allocator.agentVaults(address) and returns the vault
// proxy address for the given agent public key.
func (ec *EVMClient) GetVaultAddress(agentPubKey common.Address) (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return ec.callAddress(ctx, ec.allocatorAddress, selectorAgentVaults, &agentPubKey)
}

// GetAgentsInfo calls Allocator.getAgentsInfo(address[]) which returns
// (Status[] statuses, uint256[] balances) in a single RPC call.
// It also fetches vault addresses individually (agentVaults mapping).
func (ec *EVMClient) GetAgentsInfo(agents []common.Address) (map[common.Address]AgentInfo, error) {
	result := make(map[common.Address]AgentInfo, len(agents))
	if len(agents) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ABI encode: getAgentsInfo(address[])
	// offset to dynamic array (32 bytes) + length (32 bytes) + N * 32 bytes
	data := make([]byte, 0, 4+32+32+len(agents)*32)
	data = append(data, selectorGetAgentsInfo...)
	data = append(data, common.LeftPadBytes(big.NewInt(32).Bytes(), 32)...) // offset
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(agents))).Bytes(), 32)...) // length
	for _, addr := range agents {
		data = append(data, common.LeftPadBytes(addr.Bytes(), 32)...)
	}

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &ec.allocatorAddress,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getAgentsInfo call: %w", err)
	}

	// Decode ABI response: (Status[], uint256[])
	// Two dynamic arrays, so first 64 bytes are offsets
	n := len(agents)
	infos, err := decodeAgentsInfoResponse(out, n)
	if err != nil {
		return nil, err
	}

	// Fetch vault addresses in parallel context (they're from the mapping, not batch)
	for i, addr := range agents {
		vaultAddr, vErr := ec.callAddress(ctx, ec.allocatorAddress, selectorAgentVaults, &addr)
		if vErr != nil {
			// Non-fatal: use zero address
			vaultAddr = common.Address{}
		}
		result[addr] = AgentInfo{
			VaultAddress: vaultAddr,
			Status:       infos[i].Status,
			EVMBalance:   infos[i].EVMBalance,
		}
	}

	return result, nil
}

// decodeAgentsInfoResponse decodes the ABI response from getAgentsInfo.
// Returns (Status[], uint256[]) as two dynamic arrays.
func decodeAgentsInfoResponse(out []byte, n int) ([]AgentInfo, error) {
	// Minimum size: 2 offsets (64) + 2 lengths (64) + 2*n*32 data
	minLen := 64 + 64 + 2*n*32
	if len(out) < minLen {
		return nil, fmt.Errorf("getAgentsInfo response too short: %d < %d", len(out), minLen)
	}

	// Read offsets to the two dynamic arrays
	offsetStatuses := new(big.Int).SetBytes(out[0:32]).Int64()
	offsetBalances := new(big.Int).SetBytes(out[32:64]).Int64()

	infos := make([]AgentInfo, n)

	// Parse statuses array: [offset] -> length(32) + n*32 elements
	sBase := int(offsetStatuses) + 32 // skip length word
	for i := 0; i < n; i++ {
		pos := sBase + i*32
		if pos+32 > len(out) {
			break
		}
		statusVal := new(big.Int).SetBytes(out[pos : pos+32]).Int64()
		switch statusVal {
		case 0:
			infos[i].Status = AgentStatusInactive
		case 1:
			infos[i].Status = AgentStatusActive
		case 2:
			infos[i].Status = AgentStatusRevoked
		default:
			infos[i].Status = AgentStatusInactive
		}
	}

	// Parse balances array: [offset] -> length(32) + n*32 elements
	bBase := int(offsetBalances) + 32 // skip length word
	for i := 0; i < n; i++ {
		pos := bBase + i*32
		if pos+32 > len(out) {
			break
		}
		bal := new(big.Int).SetBytes(out[pos : pos+32])
		infos[i].EVMBalance = usdcToFloat(bal)
	}

	return infos, nil
}

// callAddress performs an eth_call that returns a single address value.
// If arg is non-nil it is ABI-encoded as the first argument.
func (ec *EVMClient) callAddress(ctx context.Context, to common.Address, selector []byte, arg *common.Address) (common.Address, error) {
	data := make([]byte, len(selector))
	copy(data, selector)
	if arg != nil {
		data = append(data, common.LeftPadBytes(arg.Bytes(), 32)...)
	}

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &to,
		Data: data,
	}, nil)
	if err != nil {
		return common.Address{}, err
	}
	if len(out) < 32 {
		return common.Address{}, nil
	}
	return common.BytesToAddress(out[12:32]), nil
}

// usdcToFloat converts a USDC raw amount (6 decimals) to float64.
func usdcToFloat(amount *big.Int) float64 {
	if amount == nil || amount.Sign() == 0 {
		return 0
	}
	f, _ := new(big.Float).SetInt(amount).Float64()
	return math.Round(f/1e6*100) / 100
}
