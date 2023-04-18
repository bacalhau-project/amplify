package cli

import (
	"os"

	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func InitializeConfig(cmd *cobra.Command) *config.AppConfig {
	// Initialize viper
	_, err := config.InitViper(cmd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize config")
	}

	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	})

	// Parse final config for log level settings
	finalConfig := config.ParseAppConfig(cmd)

	// Set log level
	zerolog.SetGlobalLevel(finalConfig.LogLevel)

	return finalConfig
}
