package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newWorkflowCommand(appContext cli.AppContext) *cobra.Command {
	c := &cobra.Command{
		Use:     "workflow",
		Short:   "Orchestrate an Amplify workflow",
		Long:    "Start an Amplify workflow, specified in the config file, and run it on the Bacalhau network.",
		Example: "amplify run workflow QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}
			_, err := cid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid CID: %s", err)
			}
			return nil
		},
		RunE: createWorkflowCommand(appContext),
	}
	return c
}

func createWorkflowCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		// Workflow Config
		conf, err := config.GetConfig(appContext.Config.ConfigPath)
		if err != nil {
			return err
		}

		// Job Factory
		factory := job.NewJobFactory(*conf)

		// Create a composite for the given CID
		comp, err := composite.NewComposite(cmd.Context(), appContext.NodeProvider, cid.MustParse(args[0]))
		if err != nil {
			return err
		}
		fmt.Println(comp.String())

		// For each CID in the composite, start a tika job to infer the data type
		err = job.MapJob{
			Executor: appContext.Executor,
			Renderer: &factory,
		}.Run(cmd.Context(), "metadata", comp)
		if err != nil {
			return err
		}
		log.Info().Msg("Finished metadata for all CIDs")
		fmt.Println(comp.String())

		// Now we have the results, create a final derivative job to merge all the results
		err = job.SingleJob{
			Executor: appContext.Executor,
			Renderer: &factory,
		}.Run(cmd.Context(), "merge", comp)
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
