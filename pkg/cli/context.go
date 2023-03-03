package cli

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
)

// AppContext is a wapper that encapsulates amplifies needs when running the CLI
type AppContext struct {
	Config       *config.AppConfig
	NodeProvider NodeProvider
	Executor     executor.Executor
}

// Implemnet the io.Closer interface
func (a *AppContext) Close() error {
	if a.NodeProvider != nil {
		return a.NodeProvider.Close()
	}
	return nil
}
