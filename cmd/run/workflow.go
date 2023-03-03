package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/workflow"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newWorkflowCommand(appContext cli.AppContext) *cobra.Command {
	c := &cobra.Command{
		Use:     "workflow",
		Short:   "Orchestrate an Amplify workflow",
		Long:    "Start an Amplify workflow, specified in the config file, and run it on the Bacalhau network.",
		Example: "amplify run workflow first QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
				return err
			}
			validWorkflows := getWorkflows(appContext.Config)
			if !contains(validWorkflows, args[0]) {
				return fmt.Errorf("workflow (%s) not found in config, must be one of: %v", args[0], validWorkflows)
			}
			_, err := cid.Parse(args[1])
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
		jobFactory := job.NewJobFactory(*conf)

		// Workflow factory
		workflowFactory := workflow.NewWorkflowFactory(*conf)

		// Create a composite for the given CID
		comp, err := composite.NewComposite(cmd.Context(), appContext.NodeProvider, cid.MustParse(args[1]))
		if err != nil {
			return err
		}
		fmt.Println(comp.String())

		// For each job in the workflow, run it
		workflow, err := workflowFactory.GetWorkflow(args[0])
		if err != nil {
			return err
		}
		log.Ctx(cmd.Context()).Info().Msgf("Running workflow %s", workflow.Name)
		for _, step := range workflow.Jobs {
			log.Ctx(cmd.Context()).Info().Msgf("Running job %s", step.Name)
			switch step.Job.(type) {
			case job.MapJob:
				err = job.MapJob{
					Executor: appContext.Executor,
					Renderer: &jobFactory,
				}.Run(cmd.Context(), step.Name, comp)
				if err != nil {
					return err
				}
			case job.SingleJob:
				err = job.SingleJob{
					Executor: appContext.Executor,
					Renderer: &jobFactory,
				}.Run(cmd.Context(), step.Name, comp)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown job type")
			}
		}

		// Now we have the results, create a final derivative job to merge all the results
		r := comp.Result()
		fmt.Println(r.StdOut)
		fmt.Println(r.StdErr)
		fmt.Println("Download the derivative result with:")
		fmt.Printf("bacalhau get %s\n", r.ID)
		return nil
	}
}

func getWorkflows(conf *config.AppConfig) []string {
	c, err := config.GetConfig(conf.ConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get config")
	}
	factory := workflow.NewWorkflowFactory(*c)
	return factory.WorkflowNames()
}
