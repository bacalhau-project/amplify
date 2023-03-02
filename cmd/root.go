package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/bacalhau-project/amplify/cmd/run"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewRootCommand(nodeGetter ipldformat.NodeGetter, exec executor.Executor) *cobra.Command {
	c := &cobra.Command{
		Use:   "amplify",
		Short: "Amplify enriches your data",
		Long: "Amplify is a data enhancement tool that uses the " +
			"https://bacalhau.org compute network to run decentralised jobs " +
			"that automatically describe, augment, and enrich you data.",
		Example: "amplify serve # start the Amplify daemon",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	// Add flags to the root command
	c = config.AddGlobalFlags(c)

	config := initializeConfig(c)

	c.AddCommand(newServeCommand(config))
	c.AddCommand(run.NewRunCommand(config, nodeGetter, exec))

	return c
}

func Execute(ctx context.Context) {
	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// IPFS Client
	session, err := ipfs.NewIPFSSession(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize IPFS session")
	}
	defer session.Close()

	// Bacalhau Client
	exec := executor.NewBacalhauExecutor()

	rootCmd := NewRootCommand(session.NodeGetter, exec)
	rootCmd.SetContext(ctx)
	rootCmd.SetOut(system.Stdout)
	rootCmd.SetErr(system.Stderr)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute root command")
	}
}

func initializeConfig(cmd *cobra.Command) *config.AppConfig {
	// Initialize viper
	_, err := config.InitViper(cmd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize config")
	}

	// Parse final config for log level settings
	finalConfig := config.ParseAppConfig(cmd)

	// Set log level
	zerolog.SetGlobalLevel(finalConfig.LogLevel)

	return finalConfig
}
