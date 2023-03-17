package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/bacalhau-project/amplify/cmd/run"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type runEFunc func(cmd *cobra.Command, args []string) error

func NewRootCommand() *cobra.Command {
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

	c.AddCommand(newServeCommand())
	c.AddCommand(run.NewRunCommand())

	return c
}

func Execute(ctx context.Context) {
	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	rootCmd := NewRootCommand()

	// Add flags to the root command
	config.AddGlobalFlags(rootCmd)

	rootCmd.SetContext(ctx)
	rootCmd.SetOut(system.Stdout)
	rootCmd.SetErr(system.Stderr)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute root command")
	}
}
