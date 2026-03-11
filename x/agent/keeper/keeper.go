package keeper

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/log/v2"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
	return nil
}

func (k Keeper) GetAgent(ctx sdk.Context, address string) (types.Agent, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyAgent(address))
	if bz == nil {
		return types.Agent{}, false
	}
	var agent types.Agent
	k.cdc.MustUnmarshal(bz, &agent)
	return agent, true
}

func (k Keeper) SetAgent(ctx sdk.Context, agent types.Agent) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&agent)
	store.Set(types.KeyAgent(agent.Address), bz)
}

func (k Keeper) DeleteAgent(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.KeyAgent(address))
}

func (k Keeper) IterateAgents(ctx sdk.Context, cb func(agent types.Agent) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte(types.AgentKeyPrefix))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var agent types.Agent
		k.cdc.MustUnmarshal(iterator.Value(), &agent)
		if cb(agent) {
			break
		}
	}
}

func (k Keeper) GetAllAgents(ctx sdk.Context) []types.Agent {
	var agents []types.Agent
	k.IterateAgents(ctx, func(agent types.Agent) bool {
		agents = append(agents, agent)
		return false
	})
	return agents
}

func (k Keeper) IsAgent(ctx sdk.Context, address string) bool {
	_, found := k.GetAgent(ctx, address)
	return found
}

// Contract deployer tracking for contribution rewards

func (k Keeper) SetContractDeployer(ctx sdk.Context, contractAddr, deployerAddr string) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte("ContractDeployer/"+contractAddr), []byte(deployerAddr))
}

func (k Keeper) GetContractDeployer(ctx sdk.Context, contractAddr string) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte("ContractDeployer/" + contractAddr))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) ExportContractDeployers(ctx sdk.Context) map[string]string {
	result := make(map[string]string)
	store := ctx.KVStore(k.storeKey)
	prefix := []byte("ContractDeployer/")
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		contractAddr := string(iterator.Key()[len(prefix):])
		deployerAddr := string(iterator.Value())
		result[contractAddr] = deployerAddr
	}
	return result
}

func (k Keeper) ImportContractDeployers(ctx sdk.Context, deployers map[string]string) {
	for contractAddr, deployerAddr := range deployers {
		k.SetContractDeployer(ctx, contractAddr, deployerAddr)
	}
}

func (k Keeper) GetReputation(ctx sdk.Context, address string) uint64 {
	agent, found := k.GetAgent(ctx, address)
	if !found {
		return 0
	}
	return agent.Reputation
}

func (k Keeper) UpdateReputation(ctx sdk.Context, address string, delta int64) {
	agent, found := k.GetAgent(ctx, address)
	if !found {
		return
	}

	params := k.GetParams(ctx)
	newRep := int64(agent.Reputation) + delta
	if newRep < 0 {
		newRep = 0
	}
	if newRep > int64(params.MaxReputation) {
		newRep = int64(params.MaxReputation)
	}
	agent.Reputation = uint64(newRep)
	k.SetAgent(ctx, agent)
}

func (k Keeper) GetCurrentEpoch(ctx sdk.Context) uint64 {
	params := k.GetParams(ctx)
	if params.EpochLength == 0 {
		return 0
	}
	return uint64(ctx.BlockHeight()) / params.EpochLength
}

const walletKVPrefix = "AgentWallet/"

func (k Keeper) ExportWalletData(ctx sdk.Context) map[string]string {
	result := make(map[string]string)
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(walletKVPrefix)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keyHex := hex.EncodeToString(iterator.Key())
		valHex := hex.EncodeToString(iterator.Value())
		result[keyHex] = valHex
	}
	return result
}

func (k Keeper) ImportWalletData(ctx sdk.Context, data map[string]string) {
	store := ctx.KVStore(k.storeKey)
	for keyHex, valHex := range data {
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			continue
		}
		val, err := hex.DecodeString(valHex)
		if err != nil {
			continue
		}
		store.Set(key, val)
	}
}
