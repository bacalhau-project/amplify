package cli

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
)

// AppContext is a wapper that encapsulates amplifies needs when running the CLI
type AppContext struct {
	Config   *config.AppConfig
	Executor executor.Executor
}

// Implement the io.Closer interface
func (a *AppContext) Close() error {
	return nil
}
