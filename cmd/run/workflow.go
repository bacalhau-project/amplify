package run

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
)

func newWorkflowCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "workflow",
		Short:   "Orchestrate an Amplify workflow",
		Long:    "Start an Amplify workflow, specified in the config file, and run it on the Bacalhau network.",
		Example: "amplify run workflow first QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
				return err
			}
			validWorkflows := getWorkflows(cli.DefaultAppContext(cmd))
			if !util.Contains(validWorkflows, args[0]) {
				return fmt.Errorf("workflow (%s) not found in config, must be one of: %v", args[0], validWorkflows)
			}
			_, err := cid.Parse(args[1])
			if err != nil {
				return fmt.Errorf("invalid CID: %s", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			appContext := cli.DefaultAppContext(cmd)
			return createWorkflowCommand(appContext)(cmd, args)
		},
	}
	return c
}

func createWorkflowCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		ctx, cancelFunc := context.WithCancel(cmd.Context())
		defer cancelFunc()
		execQueue, err := queue.NewGenericQueue(ctx, 1, 10)
		if err != nil {
			return err
		}
		execQueue.Start()
		defer execQueue.Stop()
		taskFactory, err := task.NewTaskFactory(appContext, execQueue)
		if err != nil {
			return err
		}
		wf, err := taskFactory.GetWorkflow(args[0])
		if err != nil {
			return err
		}
		n, err := taskFactory.CreateTask(ctx, []task.Workflow{wf}, args[1])
		if err != nil {
			return err
		}
		n[0].Execute(ctx)
		cancelFunc()

		return nil
	}
}

func getWorkflows(appContext cli.AppContext) []string {
	f, err := task.NewTaskFactory(appContext, nil)
	if err != nil {
		panic(err)
	}
	return f.WorkflowNames()
}
