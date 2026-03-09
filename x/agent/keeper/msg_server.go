package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Register(goCtx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	if k.IsAgent(ctx, msg.Sender) {
		return nil, types.ErrAgentAlreadyRegistered
	}

	minStake := sdk.NewInt64Coin("aaxon", int64(params.MinRegisterStake)*1e18)
	if msg.Stake.IsLT(minStake) {
		return nil, types.ErrInsufficientStake
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	stakeCoins := sdk.NewCoins(msg.Stake)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, stakeCoins); err != nil {
		return nil, err
	}

	burnAmount := sdk.NewInt64Coin("aaxon", int64(params.RegisterBurnAmount)*1e18)
	burnCoins := sdk.NewCoins(burnAmount)
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return nil, err
	}

	capabilities := strings.Split(msg.Capabilities, ",")

	agent := types.Agent{
		Address:       msg.Sender,
		AgentId:       generateAgentID(msg.Sender, ctx.BlockHeight()),
		Capabilities:  capabilities,
		Model:         msg.Model,
		Reputation:    params.InitialReputation,
		Status:        types.AgentStatus_AGENT_STATUS_ONLINE,
		StakeAmount:   msg.Stake,
		RegisteredAt:  ctx.BlockHeight(),
		LastHeartbeat: ctx.BlockHeight(),
	}

	k.SetAgent(ctx, agent)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_registered",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("agent_id", agent.AgentId),
		sdk.NewAttribute("reputation", fmt.Sprintf("%d", agent.Reputation)),
	))

	return &types.MsgRegisterResponse{AgentId: agent.AgentId}, nil
}

func (k msgServer) UpdateAgent(goCtx context.Context, msg *types.MsgUpdateAgent) (*types.MsgUpdateAgentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	agent, found := k.GetAgent(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	if msg.Capabilities != "" {
		agent.Capabilities = strings.Split(msg.Capabilities, ",")
	}
	if msg.Model != "" {
		agent.Model = msg.Model
	}

	k.SetAgent(ctx, agent)
	return &types.MsgUpdateAgentResponse{}, nil
}

func (k msgServer) Heartbeat(goCtx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	agent, found := k.GetAgent(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	if ctx.BlockHeight()-agent.LastHeartbeat < params.HeartbeatInterval {
		return nil, types.ErrHeartbeatTooFrequent
	}

	agent.LastHeartbeat = ctx.BlockHeight()
	agent.Status = types.AgentStatus_AGENT_STATUS_ONLINE
	k.SetAgent(ctx, agent)

	return &types.MsgHeartbeatResponse{}, nil
}

func (k msgServer) Deregister(goCtx context.Context, msg *types.MsgDeregister) (*types.MsgDeregisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	agent, found := k.GetAgent(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	recipientAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	refundCoins := sdk.NewCoins(agent.StakeAmount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, refundCoins); err != nil {
		return nil, err
	}

	k.DeleteAgent(ctx, msg.Sender)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"agent_deregistered",
		sdk.NewAttribute("address", msg.Sender),
	))

	return &types.MsgDeregisterResponse{}, nil
}

func (k msgServer) SubmitAIChallengeResponse(goCtx context.Context, msg *types.MsgSubmitAIChallengeResponse) (*types.MsgSubmitAIChallengeResponseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsAgent(ctx, msg.Sender) {
		return nil, types.ErrAgentNotFound
	}

	store := ctx.KVStore(k.storeKey)
	key := types.KeyAIResponse(msg.Epoch, msg.Sender)
	if store.Has(key) {
		return nil, types.ErrAlreadySubmitted
	}

	response := types.AIResponse{
		ValidatorAddress: msg.Sender,
		Epoch:            msg.Epoch,
		CommitHash:       msg.CommitHash,
		Evaluated:        false,
	}

	bz := k.cdc.MustMarshal(&response)
	store.Set(key, bz)

	return &types.MsgSubmitAIChallengeResponseResponse{}, nil
}

func (k msgServer) RevealAIChallengeResponse(goCtx context.Context, msg *types.MsgRevealAIChallengeResponse) (*types.MsgRevealAIChallengeResponseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	key := types.KeyAIResponse(msg.Epoch, msg.Sender)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrChallengeNotActive
	}

	var response types.AIResponse
	k.cdc.MustUnmarshal(bz, &response)

	revealHash := sha256.Sum256([]byte(msg.RevealData))
	if hex.EncodeToString(revealHash[:]) != response.CommitHash {
		return nil, types.ErrInvalidReveal
	}

	response.RevealData = msg.RevealData
	bz = k.cdc.MustMarshal(&response)
	store.Set(key, bz)

	return &types.MsgRevealAIChallengeResponseResponse{}, nil
}

func generateAgentID(address string, blockHeight int64) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", address, blockHeight)))
	return fmt.Sprintf("agent-%s", hex.EncodeToString(hash[:8]))
}
