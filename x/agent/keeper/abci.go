package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

func (k Keeper) BeginBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	blockHeight := ctx.BlockHeight()

	if params.EpochLength > 0 && blockHeight > 0 && uint64(blockHeight)%params.EpochLength == 0 {
		k.onEpochStart(ctx, params)
	}

	k.checkHeartbeatTimeouts(ctx, params)
	k.ProcessDeregisterQueue(ctx)
}

func (k Keeper) EndBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	blockHeight := ctx.BlockHeight()

	if params.EpochLength > 0 && blockHeight > 0 && uint64(blockHeight)%params.EpochLength == params.EpochLength-1 {
		k.onEpochEnd(ctx)
	}
}

func (k Keeper) onEpochStart(ctx sdk.Context, params types.Params) {
	epoch := k.GetCurrentEpoch(ctx)
	k.Logger(ctx).Info("new epoch started", "epoch", epoch)

	k.GenerateChallenge(ctx, epoch)

	if epoch > 0 {
		k.DistributeEpochRewards(ctx, epoch-1)
	}
}

func (k Keeper) onEpochEnd(ctx sdk.Context) {
	epoch := k.GetCurrentEpoch(ctx)
	k.Logger(ctx).Info("epoch ending", "epoch", epoch)

	k.EvaluateEpochChallenges(ctx, epoch)
	k.ProcessEpochReputation(ctx, epoch)
}

func (k Keeper) checkHeartbeatTimeouts(ctx sdk.Context, params types.Params) {
	blockHeight := ctx.BlockHeight()

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status == types.AgentStatus_AGENT_STATUS_ONLINE &&
			blockHeight-agent.LastHeartbeat > params.HeartbeatTimeout {
			agent.Status = types.AgentStatus_AGENT_STATUS_OFFLINE
			k.SetAgent(ctx, agent)
			k.UpdateReputation(ctx, agent.Address, types.ReputationLossOffline)

			k.Logger(ctx).Info("agent went offline",
				"address", agent.Address,
				"last_heartbeat", agent.LastHeartbeat,
				"current_block", blockHeight,
			)
		}
		return false
	})
}
