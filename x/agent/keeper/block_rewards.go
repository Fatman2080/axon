package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

const (
	// BlocksPerYear at 5s/block = 6,307,200
	BlocksPerYear int64 = 6_307_200

	// HalvingInterval = 4 years in blocks
	HalvingInterval int64 = BlocksPerYear * 4

	// BaseBlockReward in aaxon: 12.3 AXON = 12.3e18 aaxon
	// Precisely: 650_000_000 AXON / (4 * 6_307_200 blocks) ≈ 25.757 AXON/block for 65%
	// But whitepaper says ~12.3 AXON/block for year 1-4 → 78M/year
	// 78M * 1e18 / 6_307_200 ≈ 12.367e18 aaxon/block
	// We use 12_367_000_000_000_000_000 (12.367 AXON)
	BaseBlockRewardStr = "12367000000000000000"

	// ProposerShare = 25%
	ProposerSharePercent = 25
	// ValidatorPoolShare = 50%
	ValidatorPoolSharePercent = 50
	// AIPerformanceShare = 25%
	AIPerformanceSharePercent = 25
)

// DistributeBlockRewards mints and distributes block rewards.
// Called every block in BeginBlocker.
func (k Keeper) DistributeBlockRewards(ctx sdk.Context) {
	blockHeight := ctx.BlockHeight()
	if blockHeight <= 1 {
		return
	}

	reward := calculateBlockReward(blockHeight)
	if reward.IsZero() {
		return
	}

	rewardCoin := sdk.NewCoin("aaxon", reward)

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(rewardCoin)); err != nil {
		k.Logger(ctx).Error("failed to mint block rewards", "error", err)
		return
	}

	proposerReward := reward.Mul(sdkmath.NewInt(ProposerSharePercent)).Quo(sdkmath.NewInt(100))
	validatorReward := reward.Mul(sdkmath.NewInt(ValidatorPoolSharePercent)).Quo(sdkmath.NewInt(100))
	aiReward := reward.Sub(proposerReward).Sub(validatorReward)

	k.distributeProposerReward(ctx, proposerReward)
	k.distributeValidatorRewards(ctx, validatorReward)
	k.distributeAIPerformanceRewards(ctx, aiReward)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"block_rewards",
		sdk.NewAttribute("height", fmt.Sprintf("%d", blockHeight)),
		sdk.NewAttribute("total", rewardCoin.String()),
		sdk.NewAttribute("proposer", proposerReward.String()),
		sdk.NewAttribute("validators", validatorReward.String()),
		sdk.NewAttribute("ai_performance", aiReward.String()),
	))
}

// calculateBlockReward returns the per-block reward accounting for halvings.
func calculateBlockReward(blockHeight int64) sdkmath.Int {
	baseReward, ok := new(big.Int).SetString(BaseBlockRewardStr, 10)
	if !ok {
		return sdkmath.ZeroInt()
	}

	halvings := blockHeight / HalvingInterval
	if halvings >= 64 {
		return sdkmath.ZeroInt()
	}

	// Right-shift to apply halving: reward = base >> halvings
	reward := new(big.Int).Rsh(baseReward, uint(halvings))
	if reward.Sign() <= 0 {
		return sdkmath.ZeroInt()
	}

	return sdkmath.NewIntFromBigInt(reward)
}

// distributeProposerReward sends 25% to the block proposer.
func (k Keeper) distributeProposerReward(ctx sdk.Context, amount sdkmath.Int) {
	if amount.IsZero() {
		return
	}

	proposerAddr := ctx.BlockHeader().ProposerAddress
	if len(proposerAddr) == 0 {
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", amount))
		return
	}

	proposerAccAddr := sdk.AccAddress(proposerAddr)
	coins := sdk.NewCoins(sdk.NewCoin("aaxon", amount))

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, proposerAccAddr, coins); err != nil {
		k.Logger(ctx).Error("failed to send proposer reward", "error", err)
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", amount))
	}
}

// distributeValidatorRewards distributes 50% to active validators by weight.
// Weight = Stake × (100 + Reputation + AIBonus).
func (k Keeper) distributeValidatorRewards(ctx sdk.Context, totalAmount sdkmath.Int) {
	if totalAmount.IsZero() {
		return
	}

	type validatorWeight struct {
		address string
		weight  *big.Int
	}

	var validators []validatorWeight
	totalWeight := new(big.Int)

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status != types.AgentStatus_AGENT_STATUS_ONLINE {
			return false
		}

		stake := agent.StakeAmount.Amount.BigInt()
		repBonus := int64(agent.Reputation)
		aiBonus := k.GetAIBonus(ctx, agent.Address)
		multiplier := int64(100) + repBonus + aiBonus
		if multiplier < 10 {
			multiplier = 10
		}

		w := new(big.Int).Mul(stake, big.NewInt(multiplier))
		totalWeight.Add(totalWeight, w)
		validators = append(validators, validatorWeight{address: agent.Address, weight: w})
		return false
	})

	if totalWeight.Sign() <= 0 || len(validators) == 0 {
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", totalAmount))
		return
	}

	totalBig := totalAmount.BigInt()
	distributed := sdkmath.ZeroInt()

	for _, v := range validators {
		share := new(big.Int).Mul(totalBig, v.weight)
		share.Div(share, totalWeight)
		reward := sdkmath.NewIntFromBigInt(share)
		if reward.IsZero() {
			continue
		}

		addr, err := sdk.AccAddressFromBech32(v.address)
		if err != nil {
			continue
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("aaxon", reward))); err != nil {
			k.Logger(ctx).Error("failed to send validator reward", "address", v.address, "error", err)
			continue
		}
		distributed = distributed.Add(reward)
	}

	remainder := totalAmount.Sub(distributed)
	if remainder.IsPositive() {
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", remainder))
	}
}

// distributeAIPerformanceRewards distributes 25% by AI challenge scores.
// Stored per-epoch; agents with better AI scores get more.
func (k Keeper) distributeAIPerformanceRewards(ctx sdk.Context, totalAmount sdkmath.Int) {
	if totalAmount.IsZero() {
		return
	}

	epoch := k.GetCurrentEpoch(ctx)

	type aiWeight struct {
		address string
		bonus   int64
	}

	var agents []aiWeight
	totalBonus := int64(0)

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status != types.AgentStatus_AGENT_STATUS_ONLINE {
			return false
		}
		bonus := k.GetAIBonus(ctx, agent.Address)
		if bonus <= 0 {
			return false
		}
		agents = append(agents, aiWeight{address: agent.Address, bonus: bonus})
		totalBonus += bonus
		return false
	})

	if totalBonus <= 0 || len(agents) == 0 {
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", totalAmount))
		return
	}

	totalBig := totalAmount.BigInt()
	distributed := sdkmath.ZeroInt()

	for _, a := range agents {
		share := new(big.Int).Mul(totalBig, big.NewInt(a.bonus))
		share.Div(share, big.NewInt(totalBonus))
		reward := sdkmath.NewIntFromBigInt(share)
		if reward.IsZero() {
			continue
		}

		addr, err := sdk.AccAddressFromBech32(a.address)
		if err != nil {
			continue
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("aaxon", reward))); err != nil {
			k.Logger(ctx).Error("failed to send AI performance reward", "address", a.address, "error", err)
			continue
		}
		distributed = distributed.Add(reward)
	}

	_ = epoch
	remainder := totalAmount.Sub(distributed)
	if remainder.IsPositive() {
		k.AddToRewardPool(ctx, sdk.NewCoin("aaxon", remainder))
	}
}
