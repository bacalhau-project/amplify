package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// The environment variable prefix of all environment variables bound to our command line flags.
	envPrefix = "AMPLIFY"
)

// Define all global flag names
const (
	LogLevelFlag   = "log-level"
	ConfigPathFlag = "config"
)

type AppConfig struct {
	LogLevel   zerolog.Level `yaml:"log-level"`
	ConfigPath string        `yaml:"config-path"`
}

func ParseAppConfig(cmd *cobra.Command) *AppConfig {
	logLevel, err := zerolog.ParseLevel(cmd.Flag(LogLevelFlag).Value.String())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse log level")
	}
	return &AppConfig{
		LogLevel:   logLevel,
		ConfigPath: cmd.Flag(ConfigPathFlag).Value.String(),
	}
}

func AddGlobalFlags(cmd *cobra.Command) *cobra.Command {
	// Define cobra flags, the default value has the lowest (least significant) precedence
	cmd.PersistentFlags().String(ConfigPathFlag, "config.yaml", "Path to Amplify config")
	cmd.PersistentFlags().String(LogLevelFlag, "info", "Logging level (debug, info, warning, error)")
	return cmd
}

func InitViper(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()

	defaultConfig := ParseAppConfig(cmd)

	// Search config in directory
	v.AddConfigPath(filepath.Dir(defaultConfig.ConfigPath))
	v.SetConfigName(filepath.Base(defaultConfig.ConfigPath))

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable PREFIX_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return v, nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			err := v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to bind flag to environment variable")
			}
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to set flag value")
			}
		}

		// Apply the flags back to the viper config
		v.Set(f.Name, cmd.Flag(f.Name).Value.String())
	})
}
