package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/amplify/pkg/api"
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
)

const (
	defaultNumWorkers   = 10
	defaultMaxQueueSize = 1024
)

type runEFunc func(cmd *cobra.Command, args []string) error

func NewRunCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "run",
		Short:   "Run all the nodes in a graph for a given CID",
		Example: "amplify run Qz0432... # Run all nodes for the given CID",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			appContext := cli.DefaultAppContext(cmd)
			return createRunCommand(appContext)(cmd, args)
		},
	}
	return c
}

func createRunCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		ctx, cancelFunc := context.WithCancel(cmd.Context())
		defer cancelFunc()
		// Job Queue
		jobQueue, err := queue.NewGenericQueue(ctx, defaultNumWorkers, defaultMaxQueueSize)
		if err != nil {
			return err
		}
		jobQueue.Start()
		defer jobQueue.Stop()

		nodeFactory, err := dag.NewNodeStore(ctx, db.NewInMemDB(), dag.NewInMemWorkRepository[dag.IOSpec]())
		if err != nil {
			return err
		}
		taskFactory, err := task.NewTaskFactory(appContext, jobQueue, nodeFactory)
		if err != nil {
			return err
		}
		nodeExecutor, err := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
		if err != nil {
			return err
		}

		rootNodes, err := taskFactory.CreateTask(ctx, uuid.New(), args[0])
		if err != nil {
			return err
		}
		for _, rootNode := range rootNodes {
			nodeExecutor.Execute(ctx, uuid.New(), rootNode)
		}
		cancelFunc()
		results := util.Dedup(api.GetLeafOutputs(ctx, rootNodes))
		cmd.Println(strings.Join(results, ", "))
		return nil
	}
}
