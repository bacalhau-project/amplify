package run

import (
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/spf13/cobra"
)

type runEFunc func(cmd *cobra.Command, args []string) error

func NewRunCommand(appContext cli.AppContext) *cobra.Command {
	c := &cobra.Command{
		Use:     "run",
		Short:   "Orchestrate Amplify workloads from the command line",
		Example: "amplify run job metadata --input=Qz0432... # Run the metadata job on the given CID",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	c.AddCommand(newJobCommand(appContext))
	c.AddCommand(newWorkflowCommand(appContext))
	return c
}
