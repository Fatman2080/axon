package app

import (
	"context"
	"encoding/json"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	agentkeeper "github.com/axon-chain/axon/x/agent/keeper"
	agenttypes "github.com/axon-chain/axon/x/agent/types"
)

var (
	_ module.AppModuleBasic  = AgentAppModule{}
	_ module.HasABCIGenesis  = AgentAppModule{}
	_ module.HasName         = AgentAppModule{}
)

type AgentAppModule struct {
	cdc    codec.Codec
	keeper agentkeeper.Keeper
}

func NewAgentAppModule(cdc codec.Codec, keeper agentkeeper.Keeper) AgentAppModule {
	return AgentAppModule{cdc: cdc, keeper: keeper}
}

func (am AgentAppModule) Name() string { return agenttypes.ModuleName }

func (am AgentAppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	agenttypes.RegisterCodec(cdc)
}

func (am AgentAppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	agenttypes.RegisterInterfaces(registry)
}

func (am AgentAppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(agenttypes.DefaultGenesis())
}

func (am AgentAppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs agenttypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return err
	}
	return gs.Validate()
}

func (am AgentAppModule) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (am AgentAppModule) RegisterServices(cfg module.Configurator) {
	agenttypes.RegisterMsgServer(cfg.MsgServer(), agentkeeper.NewMsgServerImpl(am.keeper))
	agenttypes.RegisterQueryServer(cfg.QueryServer(), agentkeeper.NewQueryServerImpl(am.keeper))
}

func (am AgentAppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs agenttypes.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)
	am.keeper.SetParams(ctx, gs.Params)
	for _, agent := range gs.Agents {
		am.keeper.SetAgent(ctx, agent)
	}
	return nil
}

func (am AgentAppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	params := am.keeper.GetParams(ctx)
	agents := am.keeper.GetAllAgents(ctx)
	gs := agenttypes.GenesisState{
		Params: params,
		Agents: agents,
	}
	return cdc.MustMarshalJSON(&gs)
}

func (am AgentAppModule) ConsensusVersion() uint64 { return 1 }

func (am AgentAppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.BeginBlocker(sdkCtx)
	return nil
}

func (am AgentAppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.EndBlocker(sdkCtx)
	return nil
}

func (am AgentAppModule) IsOnePerModuleType() {}
func (am AgentAppModule) IsAppModule()        {}
