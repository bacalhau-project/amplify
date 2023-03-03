package cmd

import (
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newServeCommand(appContext cli.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:     "serve",
		Short:   "Start the Amplify daemon",
		Long:    "The serve command starts the Amplify daemon and serves the REST API",
		Example: `amplify serve`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Warn().Msg("Not implemented yet")
			return nil
		},
	}
}
