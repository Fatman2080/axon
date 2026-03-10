package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// BurnCollectedFees burns a portion of collected gas fees from FeeCollector
// at BeginBlock, implementing the Axon deflationary model:
// - Base Fee portion → 100% burned
// - The remaining fees (priority/tips) are left for distribution to proposer/validators
//
// Since precise per-tx base fee tracking is complex, we burn 50% of all fees
// as a conservative estimate (base fee typically = ~50% of total gas cost).
// When NoBaseFee = true (testnet), fees are still burned at this ratio.
const FeeBurnRatioPercent = 50

func BurnCollectedFees(ctx sdk.Context, bankKeeper bankkeeper.Keeper) {
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	balance := bankKeeper.GetBalance(ctx, feeCollectorAddr, "aaxon")

	if balance.IsZero() || !balance.IsPositive() {
		return
	}

	burnAmount := balance.Amount.QuoRaw(100).MulRaw(int64(FeeBurnRatioPercent))
	if burnAmount.IsZero() {
		return
	}

	burnCoin := sdk.NewCoin("aaxon", burnAmount)
	if err := bankKeeper.BurnCoins(ctx, authtypes.FeeCollectorName, sdk.NewCoins(burnCoin)); err != nil {
		return
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"gas_fee_burned",
		sdk.NewAttribute("amount", burnCoin.String()),
		sdk.NewAttribute("remaining", balance.Sub(burnCoin).String()),
		sdk.NewAttribute("block", fmt.Sprintf("%d", ctx.BlockHeight())),
	))
}
