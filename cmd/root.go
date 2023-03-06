package cmd

import (
	"context"
	"io"
	"os"
	"os/signal"

	"github.com/bacalhau-project/amplify/cmd/run"
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type runEFunc func(cmd *cobra.Command, args []string) error

func NewRootCommand() (*cobra.Command, io.Closer) {
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
	ctx := c.Context()

	// Add flags to the root command
	c = config.AddGlobalFlags(c)

	config := initializeConfig(c)

	// IPFS Client
	nodeProvider := cli.NewNodeProvider(ctx)

	// Bacalhau Client
	exec := executor.NewBacalhauExecutor()

	// Wrap all the dependencies in an AppContext
	appContext := cli.AppContext{
		Config:       config,
		NodeProvider: &nodeProvider,
		Executor:     exec,
	}

	c.AddCommand(newServeCommand(appContext))
	c.AddCommand(run.NewRunCommand(appContext))

	return c, &appContext
}

func Execute(ctx context.Context) {
	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	rootCmd, closer := NewRootCommand()
	defer func() {
		if err := closer.Close(); err != nil {
			log.Fatal().Err(err).Msg("Failed to close root command")
		}
	}()

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
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	})
	log.Info().Msg("Log level set to " + finalConfig.LogLevel.String())

	return finalConfig
}
