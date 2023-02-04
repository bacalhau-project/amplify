package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/ipfs"
	"github.com/bacalhau-project/amplify/pkg/job"
	bacalhauJob "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
)

// const fileCid = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi" // found by cid.contact
const fileCid = "QmabskAjK5ePM1fTYoUzDTk51LkGdTn2rt26FBj1Q9Qv7T" // A bacalhau result

func init() {
	// init system configs
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	// Context
	ctx := context.Background()
	defer ctx.Done()

	// IPFS Session
	session, err := ipfs.NewIPFSSession(ctx)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Bacalhau Client
	client := getClient("bootstrap.production.bacalhau.org", "1234")

	// Create a composite for the given CID
	comp, err := composite.NewComposite(ctx, session.NodeGetter, cid.MustParse(fileCid))
	if err != nil {
		panic(err)
	}
	fmt.Println(comp.String())

	// For each CID in the composite, start a tika job to infer the data type
	var jobRecurse func(context.Context, *composite.Composite)
	jobRecurse = func(ctx context.Context, c *composite.Composite) {
		// Only run on leaf nodes. I.e. if it has children, don't run.
		if len(c.Children) == 0 {
			aj := job.MetadataExtractionJob{CID: c.Node.Cid()}
			j := aj.Spec()
			fmt.Printf("Submitting job for CID %s\n", j.Spec.Inputs[0].CID)

			submittedJob, err := client.Submit(ctx, &j)
			if err != nil {
				panic(err)
			}
			err = waitUntilCompleted(ctx, client, submittedJob)
			if err != nil {
				panic(fmt.Errorf("waiting until completed: %s", err))
			}
			jobState, err := client.GetJobState(ctx, submittedJob.Metadata.ID)
			if err != nil {
				panic(fmt.Errorf("getting job state: %s", err))
			}

			results := []string{}
			for _, s := range jobState.Shards {
				for _, e := range s.Executions {
					if e.PublishedResult.CID == "" {
						continue
					}
					results = append(results, e.PublishedResult.CID)
				}
			}

			if len(results) == 0 {
				panic(fmt.Errorf("no results found"))
			}

			if len(results) > 1 {
				panic(fmt.Errorf("too many results found (expected 1, got %d)", len(results)))
			}

			c.Result = cid.MustParse(results[0])
			fmt.Println(c.String())
		}
		for _, child := range c.Children {
			jobRecurse(ctx, child)
		}
	}
	jobRecurse(ctx, comp)

	log.Println("Finished processing all CIDs")
	fmt.Println(comp.String())

	// Now we have the results, create a final derivative job to merge all the results
	mj := job.MergeJob{Composite: comp}
	job := mj.Spec()
	jobBytes, _ := json.MarshalIndent(job, "", "    ")
	fmt.Println(string(jobBytes))

	submittedJob, err := client.Submit(ctx, &job)
	if err != nil {
		panic(err)
	}
	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		panic(fmt.Errorf("waiting until completed: %s", err))
	}
	fmt.Printf("bacalhau describe %s\n", submittedJob.Metadata.ID)
	jobState, err := client.GetJobState(ctx, submittedJob.Metadata.ID)
	if err != nil {
		panic(fmt.Errorf("getting job state: %s", err))
	}
	for _, s := range jobState.Shards {
		for _, e := range s.Executions {
			if e.RunOutput != nil {
				fmt.Println(e.RunOutput.STDOUT)
				fmt.Println(e.RunOutput.STDERR)
			}
		}
	}
	fmt.Println("Success. Download the derivative result with:")
	fmt.Printf("bacalhau get %s\n", submittedJob.Metadata.ID)
}

func getClient(host, port string) *publicapi.RequesterAPIClient {
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://%s:%s", host, port))
	return client
}

func waitUntilCompleted(ctx context.Context, client *publicapi.RequesterAPIClient, submittedJob *model.Job) error {
	resolver := client.GetJobStateResolver()
	return resolver.Wait(
		ctx,
		submittedJob.Metadata.ID,
		bacalhauJob.WaitForSuccessfulCompletion(),
	)
}
