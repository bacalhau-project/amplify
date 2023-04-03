package run

import (
	"context"
	"os"
	"path"
	"runtime"
	"testing"

	"time"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

// Reset the working directory of the test to the root of the project
// so templates/config/etc. work
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestRunCommand(t *testing.T) {
	appContext := cli.AppContext{
		Config: &config.AppConfig{
			ConfigPath: "testdata/test-config.yaml",
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
	err := f(rootCmd, []string{"bafybeibt4amyuwvxjgq6rynmchyplbu33nixl63sdcsm7g2nb2gt6vrixu"})
	assert.NilError(t, err)
}

type mockExecutor struct{}

func (*mockExecutor) Execute(context.Context, interface{}) (executor.Result, error) {
	return executor.Result{}, nil
}

func (*mockExecutor) Render(config.Job, []executor.ExecutorIOSpec, []executor.ExecutorIOSpec) interface{} {
	return ""
}
