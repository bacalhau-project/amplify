package run

import (
	"context"
	"os"
	"testing"

	"time"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestRunCommand(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: my-foo-job
graph:
- id: my-foo-node
  job_id: my-foo-job
  inputs:
  - root: true # Identifies that this is a root node
`), 0644)
	assert.NilError(t, err)

	appContext := cli.AppContext{
		Config: &config.AppConfig{
			ConfigPath: tempFile,
			Port:       9999,
		},
		Executor: map[string]executor.Executor{"": &mockExecutor{}},
	}

	f := createRunCommand(appContext)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	rootCmd := &cobra.Command{
		Use:  "root",
		RunE: f,
	}
	rootCmd.SetContext(ctx)
	err = f(rootCmd, []string{"bafybeibt4amyuwvxjgq6rynmchyplbu33nixl63sdcsm7g2nb2gt6vrixu"})
	assert.NilError(t, err)
}

type mockExecutor struct{}

func (*mockExecutor) Execute(context.Context, config.Job, interface{}) (executor.Result, error) {
	return executor.Result{}, nil
}

func (*mockExecutor) Render(config.Job, []executor.ExecutorIOSpec, []executor.ExecutorIOSpec) (interface{}, error) {
	return "", nil
}
