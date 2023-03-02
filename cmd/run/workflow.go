package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newWorkflowCommand(config *config.AppConfig, nodeGetter ipldformat.NodeGetter, exec executor.Executor) *cobra.Command {
	c := &cobra.Command{
		Use:   "workflow",
		Short: "Orchestrate an Amplify workflow",
		Long:  "Start an Amplify workflow, specified in the config file, and run it on the Bacalhau network.",
		Args:  cobra.ExactArgs(1),
		RunE:  createWorkflowCommand(config, nodeGetter, exec),
	}
	return c
}

type RunEFunc func(cmd *cobra.Command, args []string) error

func createWorkflowCommand(conf *config.AppConfig, nodeGetter ipldformat.NodeGetter, exec executor.Executor) RunEFunc {
	return func(cmd *cobra.Command, args []string) error {
		// Workflow Config
		conf, err := config.GetConfig(conf.ConfigPath)
		if err != nil {
			return err
		}

		// Job Factory
		factory := job.NewJobFactory(*conf)

		// Create a composite for the given CID
		comp, err := composite.NewComposite(cmd.Context(), nodeGetter, cid.MustParse(args[0]))
		if err != nil {
			return err
		}
		fmt.Println(comp.String())

		// For each CID in the composite, start a tika job to infer the data type
		err = job.MapJob{
			Executor: exec,
			Renderer: &factory,
		}.Run(cmd.Context(), "metadata", comp)
		if err != nil {
			return err
		}
		log.Info().Msg("Finished metadata for all CIDs")
		fmt.Println(comp.String())

		// Now we have the results, create a final derivative job to merge all the results
		err = job.SingleJob{
			Executor: exec,
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
