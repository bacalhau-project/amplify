package run

import (
	"github.com/spf13/cobra"
)

type runEFunc func(cmd *cobra.Command, args []string) error

func NewRunCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "run",
		Short:   "Orchestrate Amplify workloads from the command line",
		Example: "amplify run job metadata --input=Qz0432... # Run the metadata job on the given CID",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	c.AddCommand(newJobCommand())
	c.AddCommand(newWorkflowCommand())
	return c
}
