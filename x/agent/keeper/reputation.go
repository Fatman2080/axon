package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

// SetAIBonus stores the AIBonus percentage for a validator.
func (k Keeper) SetAIBonus(ctx sdk.Context, address string, bonus int64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(bonus))
	store.Set(types.KeyAIBonus(address), bz)
}

// GetAIBonus returns the AIBonus percentage for a validator.
func (k Keeper) GetAIBonus(ctx sdk.Context, address string) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyAIBonus(address))
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

// DeleteAIBonus removes the AIBonus entry for an address.
func (k Keeper) DeleteAIBonus(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.KeyAIBonus(address))
}

// IncrementEpochActivity increments the transaction count for an agent in the current epoch.
func (k Keeper) IncrementEpochActivity(ctx sdk.Context, address string) {
	epoch := k.GetCurrentEpoch(ctx)
	store := ctx.KVStore(k.storeKey)
	key := types.KeyEpochActivity(epoch, address)

	count := uint64(0)
	bz := store.Get(key)
	if bz != nil {
		count = binary.BigEndian.Uint64(bz)
	}
	count++

	bz = make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(key, bz)
}

// GetEpochActivity returns the transaction count for an agent in a given epoch.
func (k Keeper) GetEpochActivity(ctx sdk.Context, epoch uint64, address string) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEpochActivity(epoch, address))
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// ProcessEpochReputation updates reputation for all agents at epoch boundaries.
func (k Keeper) ProcessEpochReputation(ctx sdk.Context, epoch uint64) {
	params := k.GetParams(ctx)

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status == types.AgentStatus_AGENT_STATUS_SUSPENDED {
			return false
		}

		delta := int64(0)

		// Online for the full epoch → +1
		if agent.Status == types.AgentStatus_AGENT_STATUS_ONLINE {
			delta += types.ReputationGainEpochOnline
		}

		// Offline (missed heartbeats) → -1
		if agent.Status == types.AgentStatus_AGENT_STATUS_OFFLINE {
			delta += types.ReputationLossNoHeartbeatEpoch
		}

		// Active in the epoch (≥10 tx) → +1
		activity := k.GetEpochActivity(ctx, epoch, agent.Address)
		if activity >= 10 {
			delta += types.ReputationGainActiveEpoch
		}

		if delta != 0 {
			k.UpdateReputation(ctx, agent.Address, delta)
		}

		// If reputation hits 0, burn all remaining stake
		updatedAgent, found := k.GetAgent(ctx, agent.Address)
		if found && updatedAgent.Reputation == 0 {
			k.handleZeroReputation(ctx, updatedAgent, params)
		}

		return false
	})
}

// handleZeroReputation burns remaining stake and suspends the agent.
func (k Keeper) handleZeroReputation(ctx sdk.Context, agent types.Agent, params types.Params) {
	burnedAtRegister := sdk.NewInt64Coin("aaxon", int64(params.RegisterBurnAmount)*1e18)
	remaining := agent.StakeAmount.Sub(burnedAtRegister)

	if remaining.IsPositive() {
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(remaining)); err != nil {
			k.Logger(ctx).Error("failed to burn stake for zero reputation", "address", agent.Address, "error", err)
			return
		}
	}

	agent.Status = types.AgentStatus_AGENT_STATUS_SUSPENDED
	agent.StakeAmount = sdk.NewInt64Coin("aaxon", 0)
	k.SetAgent(ctx, agent)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_stake_burned",
		sdk.NewAttribute("address", agent.Address),
		sdk.NewAttribute("burned", remaining.String()),
		sdk.NewAttribute("reason", "reputation_zero"),
	))
}
