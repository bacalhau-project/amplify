package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
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
			if !util.Contains(validWorkflows, args[0]) {
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
		taskFactory, err := task.NewTaskFactory(appContext)
		if err != nil {
			return err
		}

		n, err := taskFactory.CreateWorkflowTask(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		n.Execute(cmd.Context())

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
