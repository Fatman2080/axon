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
	VaultAddress   common.Address
	Status         string  // "inactive", "active"
	EVMBalance     float64 // USDC on EVM (6 decimals converted to float)
	InitialCapital float64 // initial capital deposited (6 decimals converted to float)
}

// Agent status: active (vault user matches agent) or inactive (otherwise).
const (
	AgentStatusInactive = "inactive"
	AgentStatusActive   = "active"
)

// Function selectors (first 4 bytes of keccak256 hash).
var (
	selectorUsdc             = crypto.Keccak256([]byte("usdc()"))[:4]
	selectorUserVault        = crypto.Keccak256([]byte("userVault(address)"))[:4]
	selectorGetVaultsInfo    = crypto.Keccak256([]byte("getVaultsInfo(address[])"))[:4]
	selectorVaultCount       = crypto.Keccak256([]byte("vaultCount()"))[:4]
	selectorGetVaultsByRange = crypto.Keccak256([]byte("getVaultsByRange(uint256,uint256)"))[:4]
	selectorOwner            = crypto.Keccak256([]byte("owner()"))[:4]
	selectorBalanceOf        = crypto.Keccak256([]byte("balanceOf(address)"))[:4]
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

	// Read usdc() from Allocator and cache.
	usdcAddr, err := ec.callAddress(ctx, ec.allocatorAddress, selectorUsdc, nil)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("evm read usdc address: %w", err)
	}
	ec.usdcAddress = usdcAddr
	return ec, nil
}

// GetVaultAddress calls Allocator.userVault(address) and returns the vault
// proxy address for the given agent user address.
func (ec *EVMClient) GetVaultAddress(agentPubKey common.Address) (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return ec.callAddress(ctx, ec.allocatorAddress, selectorUserVault, &agentPubKey)
}

// GetVaultCount calls Allocator.vaultCount() and returns the total number of vaults.
func (ec *EVMClient) GetVaultCount() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &ec.allocatorAddress,
		Data: selectorVaultCount,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("vaultCount call: %w", err)
	}
	if len(out) < 32 {
		return 0, fmt.Errorf("vaultCount response too short: %d", len(out))
	}
	count := new(big.Int).SetBytes(out[:32])
	return int(count.Int64()), nil
}

// GetVaultsByRange calls Allocator.getVaultsByRange(uint256,uint256) and returns
// vault addresses in the specified range.
func (ec *EVMClient) GetVaultsByRange(start, count int) ([]common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data := make([]byte, 0, 4+64)
	data = append(data, selectorGetVaultsByRange...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(start)).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(count)).Bytes(), 32)...)

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &ec.allocatorAddress,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getVaultsByRange call: %w", err)
	}

	// ABI: returns address[] — offset(32) + length(32) + n*32
	if len(out) < 64 {
		return nil, fmt.Errorf("getVaultsByRange response too short: %d", len(out))
	}
	offset := new(big.Int).SetBytes(out[0:32]).Int64()
	if int(offset)+32 > len(out) {
		return nil, fmt.Errorf("getVaultsByRange invalid offset: %d", offset)
	}
	n := new(big.Int).SetBytes(out[offset : offset+32]).Int64()
	base := int(offset) + 32

	addrs := make([]common.Address, 0, n)
	for i := 0; i < int(n); i++ {
		pos := base + i*32
		if pos+32 > len(out) {
			break
		}
		addrs = append(addrs, common.BytesToAddress(out[pos+12:pos+32]))
	}
	return addrs, nil
}

// GetVaultsInfo queries vault info for a batch of vault addresses.
// Returns (users, balances, valids, capitals) decoded into vaultInfo slices.
func (ec *EVMClient) GetVaultsInfo(vaultAddrs []common.Address) ([]vaultInfo, error) {
	if len(vaultAddrs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data := make([]byte, 0, 4+32+32+len(vaultAddrs)*32)
	data = append(data, selectorGetVaultsInfo...)
	data = append(data, common.LeftPadBytes(big.NewInt(32).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(vaultAddrs))).Bytes(), 32)...)
	for _, addr := range vaultAddrs {
		data = append(data, common.LeftPadBytes(addr.Bytes(), 32)...)
	}

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &ec.allocatorAddress,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getVaultsInfo call: %w", err)
	}
	return decodeVaultsInfoResponse(out, len(vaultAddrs))
}

// GetAgentsInfo queries agent status and EVM balance for each agent address.
// Flow: userVault(agent) → vault address, then getVaultsInfo(vaults[]) → (users[], balances[], valids[]).
// Status is "active" if the vault's current user matches the agent, "inactive" otherwise.
func (ec *EVMClient) GetAgentsInfo(agents []common.Address) (map[common.Address]AgentInfo, error) {
	result := make(map[common.Address]AgentInfo, len(agents))
	if len(agents) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Step 1: Get vault address for each agent via userVault(address)
	type agentVault struct {
		agent common.Address
		vault common.Address
	}
	var withVaults []agentVault

	for _, agent := range agents {
		vaultAddr, err := ec.callAddress(ctx, ec.allocatorAddress, selectorUserVault, &agent)
		if err != nil || vaultAddr == (common.Address{}) {
			result[agent] = AgentInfo{Status: AgentStatusInactive}
			continue
		}
		withVaults = append(withVaults, agentVault{agent: agent, vault: vaultAddr})
	}

	if len(withVaults) == 0 {
		return result, nil
	}

	// Step 2: Batch getVaultsInfo(address[]) for all vault addresses
	vaultAddrs := make([]common.Address, len(withVaults))
	for i, av := range withVaults {
		vaultAddrs[i] = av.vault
	}

	data := make([]byte, 0, 4+32+32+len(vaultAddrs)*32)
	data = append(data, selectorGetVaultsInfo...)
	data = append(data, common.LeftPadBytes(big.NewInt(32).Bytes(), 32)...)                     // offset
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(vaultAddrs))).Bytes(), 32)...) // length
	for _, addr := range vaultAddrs {
		data = append(data, common.LeftPadBytes(addr.Bytes(), 32)...)
	}

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &ec.allocatorAddress,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getVaultsInfo call: %w", err)
	}

	vInfos, err := decodeVaultsInfoResponse(out, len(vaultAddrs))
	if err != nil {
		return nil, err
	}

	// Step 3: Build results — vault's user must match agent to be "active"
	for i, av := range withVaults {
		info := AgentInfo{
			VaultAddress:   av.vault,
			EVMBalance:     vInfos[i].balance,
			InitialCapital: vInfos[i].capital,
		}
		if vInfos[i].valid && vInfos[i].user == av.agent {
			info.Status = AgentStatusActive
		} else {
			info.Status = AgentStatusInactive
		}
		result[av.agent] = info
	}

	return result, nil
}

// vaultInfo holds decoded data from a single entry in getVaultsInfo response.
type vaultInfo struct {
	user    common.Address
	balance float64
	valid   bool
	capital float64
}

// decodeVaultsInfoResponse decodes the ABI response from getVaultsInfo.
// Contract returns (address[] users, uint256[] balances, bool[] valids, uint256[] capitals).
func decodeVaultsInfoResponse(out []byte, n int) ([]vaultInfo, error) {
	// 4 offset words (users, balances, valids, capitals) = 128 bytes minimum
	if len(out) < 128 {
		return nil, fmt.Errorf("getVaultsInfo response too short: %d", len(out))
	}

	offsetUsers := new(big.Int).SetBytes(out[0:32]).Int64()
	offsetBalances := new(big.Int).SetBytes(out[32:64]).Int64()
	offsetValids := new(big.Int).SetBytes(out[64:96]).Int64()
	offsetCapitals := new(big.Int).SetBytes(out[96:128]).Int64()

	// Minimum size check against the last array
	minLen := int(offsetCapitals) + 32 + n*32
	if len(out) < minLen {
		return nil, fmt.Errorf("getVaultsInfo response too short: %d < %d", len(out), minLen)
	}

	infos := make([]vaultInfo, n)

	// Parse users array
	uBase := int(offsetUsers) + 32 // skip length word
	for i := 0; i < n; i++ {
		pos := uBase + i*32
		if pos+32 > len(out) {
			break
		}
		infos[i].user = common.BytesToAddress(out[pos+12 : pos+32])
	}

	// Parse balances array
	bBase := int(offsetBalances) + 32
	for i := 0; i < n; i++ {
		pos := bBase + i*32
		if pos+32 > len(out) {
			break
		}
		bal := new(big.Int).SetBytes(out[pos : pos+32])
		infos[i].balance = usdcToFloat(bal)
	}

	// Parse valids array
	vBase := int(offsetValids) + 32
	for i := 0; i < n; i++ {
		pos := vBase + i*32
		if pos+32 > len(out) {
			break
		}
		infos[i].valid = new(big.Int).SetBytes(out[pos:pos+32]).Sign() != 0
	}

	// Parse capitals array
	cBase := int(offsetCapitals) + 32
	for i := 0; i < n; i++ {
		pos := cBase + i*32
		if pos+32 > len(out) {
			break
		}
		cap := new(big.Int).SetBytes(out[pos : pos+32])
		infos[i].capital = usdcToFloat(cap)
	}

	return infos, nil
}

// GetAllocatorOwner calls Allocator.owner() and returns the owner address.
func (ec *EVMClient) GetAllocatorOwner() (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return ec.callAddress(ctx, ec.allocatorAddress, selectorOwner, nil)
}

// GetUSDCBalance calls USDC.balanceOf(addr) and returns the balance as float64.
func (ec *EVMClient) GetUSDCBalance(addr common.Address) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	data := make([]byte, 0, 4+32)
	data = append(data, selectorBalanceOf...)
	data = append(data, common.LeftPadBytes(addr.Bytes(), 32)...)

	to := ec.usdcAddress
	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &to,
		Data: data,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("balanceOf call: %w", err)
	}
	if len(out) < 32 {
		return 0, fmt.Errorf("balanceOf response too short: %d", len(out))
	}
	bal := new(big.Int).SetBytes(out[:32])
	return usdcToFloat(bal), nil
}

// GetERC20Balance calls token.balanceOf(userAddr) and returns the balance as float64 (assuming 6 decimals).
func (ec *EVMClient) GetERC20Balance(tokenAddr, userAddr common.Address) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	data := make([]byte, 0, 4+32)
	data = append(data, selectorBalanceOf...)
	data = append(data, common.LeftPadBytes(userAddr.Bytes(), 32)...)

	out, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("balanceOf call: %w", err)
	}
	if len(out) < 32 {
		return 0, fmt.Errorf("balanceOf response too short: %d", len(out))
	}
	bal := new(big.Int).SetBytes(out[:32])
	return usdcToFloat(bal), nil
}

// GetNativeBalance calls eth_getBalance(addr, "latest") and returns the balance as float64 (assuming 18 decimals for Hyperliquid EVM native USDC).
func (ec *EVMClient) GetNativeBalance(addr common.Address) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	bal, err := ec.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return 0, fmt.Errorf("eth_getBalance call: %w", err)
	}

	f, _ := new(big.Float).SetInt(bal).Float64()
	return math.Round(f/1e16) / 100, nil // 1e18 -> 100
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
