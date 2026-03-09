package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

// BeginBlocker runs at the beginning of each block.
// Handles epoch transitions, AI challenge generation, and reputation decay.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	blockHeight := ctx.BlockHeight()

	if params.EpochLength > 0 && blockHeight%int64(params.EpochLength) == 0 {
		k.onEpochStart(ctx, params)
	}

	k.checkHeartbeatTimeouts(ctx, params)
}

// EndBlocker runs at the end of each block.
// Handles AI challenge evaluation at epoch boundaries.
func (k Keeper) EndBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	blockHeight := ctx.BlockHeight()

	if params.EpochLength > 0 && blockHeight%int64(params.EpochLength) == int64(params.EpochLength)-1 {
		k.onEpochEnd(ctx)
	}
}

func (k Keeper) onEpochStart(ctx sdk.Context, params types.Params) {
	epoch := k.GetCurrentEpoch(ctx)
	k.Logger(ctx).Info("new epoch started", "epoch", epoch)

	// TODO: Generate new AI challenge from challenge pool
	// TODO: Broadcast challenge hash to validators
	// TODO: Distribute Agent contribution rewards for previous epoch
}

func (k Keeper) onEpochEnd(ctx sdk.Context) {
	epoch := k.GetCurrentEpoch(ctx)
	k.Logger(ctx).Info("epoch ending", "epoch", epoch)

	// TODO: Evaluate AI challenge responses
	// TODO: Update AIBonus for validators
	// TODO: Update reputation scores based on epoch activity
}

func (k Keeper) checkHeartbeatTimeouts(ctx sdk.Context, params types.Params) {
	blockHeight := ctx.BlockHeight()

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status == types.AgentStatus_AGENT_STATUS_ONLINE &&
			blockHeight-agent.LastHeartbeat > params.HeartbeatTimeout {
			agent.Status = types.AgentStatus_AGENT_STATUS_OFFLINE
			k.UpdateReputation(ctx, agent.Address, -1)
			k.SetAgent(ctx, agent)

			k.Logger(ctx).Info("agent went offline due to heartbeat timeout",
				"address", agent.Address)
		}
		return false
	})
}
