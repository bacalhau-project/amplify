package main

import (
	"context"

	"github.com/bacalhau-project/amplify/cmd"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// const fileCid = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi" // found by cid.contact
// const fileCid = "QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T" // A bacalhau result

func init() {
	// init system configs
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	// Base context
	ctx := context.Background()
	defer ctx.Done()

	cmd.Execute(ctx)
}
