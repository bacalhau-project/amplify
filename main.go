package main

import (
	"context"

	"github.com/bacalhau-project/amplify/cmd"
)

// bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi // an image
// QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T // A bacalhau result
// bafkreidhrwwuuhipvbqnxi4sgn6ea52psaeria4frzc3js3mkvea53aaq4 // a html file

func main() {
	// Base context
	ctx := context.Background()
	defer ctx.Done()

	cmd.Execute(ctx)
}
