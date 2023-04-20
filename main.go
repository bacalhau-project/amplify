package main

import (
	"context"

	"github.com/bacalhau-project/amplify/cmd"
)

func main() {
	ctx := context.Background()
	defer ctx.Done()

	cmd.Execute(ctx)
}
