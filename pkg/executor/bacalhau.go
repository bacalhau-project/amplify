package executor

import (
	"encoding/json"
	"fmt"

	bacalhauJob "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

func NewBacalhauExecutor() *BacalhauExecutor {
	return &BacalhauExecutor{Client: getClient("bootstrap.production.bacalhau.org", "1234")}
}

type BacalhauExecutor struct {
	Client *publicapi.RequesterAPIClient
}

func (b *BacalhauExecutor) Execute(ctx context.Context, rawJob interface{}) (Result, error) {
	result := Result{}
	j, ok := rawJob.(model.Job)
	if !ok {
		return result, fmt.Errorf("invalid job type for Bacalhau executor")
	}
	marshalledJob, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshalling Bacalhau job: %s", err)
	}
	log.Ctx(ctx).Debug().Msgf("Executing job:\n%s\n", marshalledJob)
	submittedJob, err := b.Client.Submit(ctx, &j)
	if err != nil {
		return result, fmt.Errorf("submitting Bacalhau job: %s", err)
	}
	log.Ctx(ctx).Info().Msgf("bacalhau describe %s", submittedJob.Metadata.ID)
	err = waitUntilCompleted(ctx, b.Client, submittedJob)
	if err != nil {
		return result, fmt.Errorf("waiting until completed: %s", err)
	}
	jobState, err := b.Client.GetJobState(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return result, fmt.Errorf("getting Bacalhau job state: %s", err)
	}

	result.ID = submittedJob.Metadata.ID
	for _, s := range jobState.Shards {
		for _, e := range s.Executions {
			if e.PublishedResult.CID == "" {
				continue
			}
			result.CID, err = cid.Parse(e.PublishedResult.CID)
			if err != nil {
				return result, fmt.Errorf("parsing result CID: %s", err)
			}
			result.StdOut = e.RunOutput.STDOUT
			result.StdErr = e.RunOutput.STDERR
			break
		}
		if result.CID != cid.Undef {
			break
		}
	}

	return result, nil
}

func waitUntilCompleted(ctx context.Context, client *publicapi.RequesterAPIClient, submittedJob *model.Job) error {
	resolver := client.GetJobStateResolver()
	return resolver.Wait(
		ctx,
		submittedJob.Metadata.ID,
		bacalhauJob.WaitForSuccessfulCompletion(),
	)
}

func getClient(host, port string) *publicapi.RequesterAPIClient {
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://%s:%s", host, port))
	return client
}
