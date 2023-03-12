package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newJobCommand(appContext cli.AppContext) *cobra.Command {
	c := &cobra.Command{
		Use:     "job [job name] [CID]",
		Short:   "Run a single Amplify job on a CID",
		Long:    "Run a single Amplify job on the Bacalhau network. Useful for testing and developing new jobs. [job name] must exist within the config file.",
		Example: "amplify run job metadata bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
				return err
			}
			validJobs := getJobs(appContext.Config)
			if !util.Contains(validJobs, args[0]) {
				return fmt.Errorf("job (%s) not found in config, must be one of: %v", args[0], validJobs)
			}
			_, err := cid.Parse(args[1])
			if err != nil {
				return fmt.Errorf("invalid CID: %s", err)
			}
			return nil
		},
		RunE: createJobCommand(appContext),
	}
	return c
}

func createJobCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		taskFactory, err := task.NewTaskFactory(appContext)
		if err != nil {
			return err
		}

		callable, err := taskFactory.CreateJobTask(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}

		err = callable(cmd.Context())
		if err != nil {
			return err
		}
		return nil
	}
}

func getJobs(conf *config.AppConfig) []string {
	c, err := config.GetConfig(conf.ConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get config")
	}
	factory := job.NewJobFactory(*c)
	return factory.JobNames()
}
