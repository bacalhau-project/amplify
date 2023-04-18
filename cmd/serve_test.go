package cmd

import (
	"context"
	"net/http"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

// Reset the working directory of the test to the root of the project
// so templates/config/etc. work
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

// Test that the serve command actually starts a server and responds to requests
// Don't test the actual functionality of the server here, test that in the
// API if you want to with httptest.NewRecorder. It doesn't actually spin
// up a server and is much faster.
func TestServeCommand(t *testing.T) {
	appContext := cli.AppContext{
		Config: &config.AppConfig{
			ConfigPath:          "config.yaml",
			Port:                8080,
			NodeConcurrency:     1,
			WorkflowConcurrency: 1,
			MaxWaitingWorkflows: 1,
		},
		Executor: map[string]executor.Executor{"": &mockExecutor{}},
	}
	f := executeServeCommand(appContext)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	rootCmd := &cobra.Command{
		Use:  "root",
		RunE: f,
	}
	rootCmd.SetContext(ctx)
	errChan := make(chan error)
	responseChan := make(chan *http.Response)
	go func(cmd *cobra.Command) {
		errChan <- cmd.Execute()
	}(rootCmd)

	client := &http.Client{}
	go func(c *http.Client) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if r, err := client.Get("http://localhost:8080/api/v0"); err == nil {
					log.Ctx(ctx).Info().Msg("response received")
					responseChan <- r
					return
				} else {
					log.Ctx(ctx).Info().Err(err).Msg("server not ready yet")
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
	}(client)
	for {
		select {
		case err := <-errChan:
			assert.NilError(t, err)
			return
		case <-ctx.Done():
			t.Fatal("context timed out, no response from server")
			return
		case response := <-responseChan:
			assert.Assert(t, response != nil)
			assert.Equal(t, response.StatusCode, http.StatusOK)
			return
		}
	}
}

type mockExecutor struct{}

func (*mockExecutor) Execute(context.Context, interface{}) (executor.Result, error) {
	return executor.Result{}, nil
}

func (*mockExecutor) Render(config.Job, []executor.ExecutorIOSpec, []executor.ExecutorIOSpec) (interface{}, error) {
	return "", nil
}
