package app

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	agentkeeper "github.com/axon-chain/axon/x/agent/keeper"
)

const (
	// DeployBurnAmount is 10 AXON = 10 * 10^18 aaxon
	DeployBurnDenom  = "aaxon"
	DeployBurnAxon   = 10
	DeployBurnExp    = 18
)

var _ evmtypes.EvmHooks = DeployBurnHook{}

// DeployBurnHook burns 10 AXON from the deployer when a contract is created,
// and tracks the deployment for contribution rewards.
type DeployBurnHook struct {
	bankKeeper  bankkeeper.Keeper
	agentKeeper agentkeeper.Keeper
}

func NewDeployBurnHook(bk bankkeeper.Keeper, ak agentkeeper.Keeper) DeployBurnHook {
	return DeployBurnHook{bankKeeper: bk, agentKeeper: ak}
}

func (h DeployBurnHook) PostTxProcessing(
	ctx sdk.Context,
	sender common.Address,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	if receipt.ContractAddress == (common.Address{}) {
		return nil
	}

	// 10 AXON = 10 * 10^18 aaxon
	burnAmount := sdkmath.NewInt(DeployBurnAxon).Mul(sdkmath.NewIntWithDecimal(1, DeployBurnExp))
	burnCoin := sdk.NewCoin(DeployBurnDenom, burnAmount)

	deployer := sdk.AccAddress(sender.Bytes())

	balance := h.bankKeeper.GetBalance(ctx, deployer, DeployBurnDenom)
	if balance.Amount.LT(burnAmount) {
		return fmt.Errorf("insufficient balance for contract deployment burn: need %s %s, have %s", burnAmount.String(), DeployBurnDenom, balance.Amount.String())
	}

	if err := h.bankKeeper.SendCoinsFromAccountToModule(ctx, deployer, evmtypes.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return err
	}
	if err := h.bankKeeper.BurnCoins(ctx, evmtypes.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return err
	}

	// Track deployment for contribution rewards
	if h.agentKeeper.IsAgent(ctx, deployer.String()) {
		h.agentKeeper.IncrementDeployCount(ctx, deployer.String())
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"contract_deploy_burn",
		sdk.NewAttribute("deployer", deployer.String()),
		sdk.NewAttribute("contract", receipt.ContractAddress.Hex()),
		sdk.NewAttribute("burned", burnCoin.String()),
	))

	return nil
}
