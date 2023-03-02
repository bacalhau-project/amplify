package run

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/spf13/cobra"
)

func NewRunCommand(config *config.AppConfig, nodeGetter ipldformat.NodeGetter, exec executor.Executor) *cobra.Command {
	c := &cobra.Command{
		Use:     "run",
		Short:   "Orchestrate Amplify workloads from the command line",
		Example: "amplify run job metadata --input=Qz0432... # Run the metadata job on the given CID",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	c.AddCommand(newWorkflowCommand(config, nodeGetter, exec))
	return c
}
