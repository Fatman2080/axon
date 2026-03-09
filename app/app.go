package app

/*
Axon — The World Computer for Agents

This file defines the main application structure for the Axon blockchain.
It integrates Cosmos SDK modules, Cosmos EVM for full Ethereum compatibility,
and the custom x/agent module for Agent-native capabilities.

Architecture:
┌─────────────────────────────────────────────┐
│  EVM Layer (Cosmos EVM)                     │
│  - Full Solidity support                    │
│  - JSON-RPC (eth_*)                         │
│  - EVM Precompiles (Agent Registry,         │
│    Reputation, Wallet)                      │
├─────────────────────────────────────────────┤
│  Agent Native Module (x/agent)              │
│  - Agent identity & registration            │
│  - Reputation system                        │
│  - AI challenge mechanism                   │
│  - Contribution rewards                     │
├─────────────────────────────────────────────┤
│  Cosmos SDK Core Modules                    │
│  - x/bank, x/staking, x/gov, x/auth       │
│  - x/distribution, x/slashing              │
├─────────────────────────────────────────────┤
│  CometBFT (Consensus + P2P)                │
│  - BFT consensus, ~5s block time           │
│  - Instant finality                         │
└─────────────────────────────────────────────┘
*/

import (
	"io"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// Cosmos SDK modules
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	// Axon custom modules
	agentmodule "github.com/axon-chain/axon/x/agent"
	agentkeeper "github.com/axon-chain/axon/x/agent/keeper"
	agenttypes "github.com/axon-chain/axon/x/agent/types"
)

const AppName = "axon"

var (
	DefaultNodeHome string
	ModuleBasics    module.BasicManager
)

func init() {
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		distribution.AppModuleBasic{},
		gov.AppModuleBasic{},
		params.AppModuleBasic{},
		slashing.AppModuleBasic{},
		genutil.AppModuleBasic{},
		consensus.AppModuleBasic{},
		agentmodule.AppModuleBasic{},
	)
}

// AxonApp extends baseapp.BaseApp
type AxonApp struct {
	*baseapp.BaseApp

	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry codectypes.InterfaceRegistry

	// store keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// --- Cosmos SDK keepers ---
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	StakingKeeper   *stakingkeeper.Keeper
	DistrKeeper     distrkeeper.Keeper
	GovKeeper       *govkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	ParamsKeeper    paramskeeper.Keeper
	ConsensusKeeper consensuskeeper.Keeper

	// --- Axon keepers ---
	AgentKeeper agentkeeper.Keeper

	// --- EVM keepers ---
	// EVMKeeper and FeeMarketKeeper will be added when integrating Cosmos EVM
	// EVMKeeper       *evmkeeper.Keeper
	// FeeMarketKeeper feemarketkeeper.Keeper

	mm *module.Manager
}

// NewAxonApp creates and initializes the Axon application.
// This is the constructor called by the node binary (cmd/axond).
func NewAxonApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *AxonApp {
	// TODO: Full app wiring with depinject or manual wiring
	// This skeleton shows the architecture; actual wiring requires
	// generated protobuf types and Cosmos EVM integration.

	appCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	_ = appCodec

	bApp := baseapp.NewApp(AppName, logger, db, nil, baseAppOptions...)
	_ = bApp

	app := &AxonApp{
		keys:  make(map[string]*storetypes.KVStoreKey),
		tkeys: make(map[string]*storetypes.TransientStoreKey),
	}

	// Define store keys
	app.keys = storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		agenttypes.StoreKey,
		// Add more module store keys here
	)

	return app
}

func (app *AxonApp) Name() string { return AppName }

func (app *AxonApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	// TODO: Register REST and gRPC-gateway routes
}

func (app *AxonApp) RegisterNodeService(clientCtx interface{}) {
	// TODO: Register node service
}

// GetStoreKeys returns all KV store keys registered in the app
func (app *AxonApp) GetStoreKeys() map[string]*storetypes.KVStoreKey {
	return app.keys
}

// ChainID returns "axon-1" for mainnet
func ChainID() string {
	return "axon-1"
}

// TokenDenom returns the native token denomination
func TokenDenom() string {
	return "aaxon"
}

// HumanDenom returns the human-readable token name
func HumanDenom() string {
	return "AXON"
}

// InitChainConfig sets up the global SDK config for Axon
func InitChainConfig() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("axon", "axonpub")
	config.SetBech32PrefixForValidator("axonvaloper", "axonvaloperpub")
	config.SetBech32PrefixForConsensusNode("axonvalcons", "axonvalconspub")
	config.Seal()
}
