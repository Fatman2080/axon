package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/axon-chain/axon/app"
)

func main() {
	app.InitChainConfig()

	rootCmd := NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "axond",
		Short: "Axon — The World Computer for Agents",
		Long: `Axon is a Layer 1 public blockchain built for AI Agents.

Agents run the network, participate in consensus, earn block rewards,
and build applications using full EVM compatibility.

Key features:
  - Agent-native identity and reputation at chain level
  - PoS + AI capability verification consensus
  - Full EVM compatibility (Solidity, MetaMask, Hardhat)
  - Zero preallocation token economics

Start a node:
  axond start

Register as validator:
  axond tx staking create-validator ...

Register an Agent:
  axond tx agent register ...
`,
	}

	// TODO: Add standard Cosmos SDK server commands
	// server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, exportApp)

	rootCmd.AddCommand(VersionCmd())
	return rootCmd
}

func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("axond v0.1.0-dev")
			fmt.Println("Axon — The World Computer for Agents")
		},
	}
}
