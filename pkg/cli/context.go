package cli

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/spf13/cobra"
)

// AppContext is a wapper that encapsulates amplifies needs when running the CLI
type AppContext struct {
	Config   *config.AppConfig
	Executor map[string]executor.Executor
}

// Implement the io.Closer interface
func (a *AppContext) Close() error {
	return nil
}

func DefaultAppContext(cmd *cobra.Command) AppContext {
	defaultExecutor := executor.NewBacalhauExecutor()
	return AppContext{
		Config: InitializeConfig(cmd),
		Executor: map[string]executor.Executor{
			"internal": executor.NewInternalExecutor(),
			"bacalhau": defaultExecutor,
			"":         defaultExecutor,
		},
	}
}
