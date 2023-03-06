package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
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
		// Config
		conf, err := config.GetConfig(appContext.Config.ConfigPath)
		if err != nil {
			return err
		}

		// Job Factory
		factory := job.NewJobFactory(*conf)

		// Create a composite for the given CID
		comp, err := composite.NewComposite(cmd.Context(), appContext.NodeProvider, cid.MustParse(args[1]))
		if err != nil {
			return err
		}
		fmt.Println(comp.String())

		// Start a simple job using the given CID
		err = job.SingleJob{
			Executor: appContext.Executor,
			Renderer: &factory,
		}.Run(cmd.Context(), args[0], comp)
		if err != nil {
			return err
		}
		r := comp.Result()
		fmt.Println(r.StdOut)
		fmt.Println(r.StdErr)
		fmt.Println("Download the derivative result with:")
		fmt.Printf("bacalhau get %s\n", r.ID)
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
