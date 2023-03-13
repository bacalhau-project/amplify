package run

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/ipfs/go-cid"
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
			validJobs := getJobs(appContext)
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

		wf := task.Workflow{
			Name: args[0],
			Jobs: []task.WorkflowJob{
				{
					Name: args[0],
				},
			},
		}

		t, err := taskFactory.CreateTask(cmd.Context(), wf, args[1])
		if err != nil {
			return err
		}

		t.Execute(cmd.Context())
		return nil
	}
}

func getJobs(appContext cli.AppContext) []string {
	f, err := task.NewTaskFactory(appContext)
	if err != nil {
		panic(err)
	}
	return f.JobNames()
}
