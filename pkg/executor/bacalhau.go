package executor

import (
	"encoding/json"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/config"
	bacalhauJob "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

func NewBacalhauExecutor() Executor {
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
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
	for _, e := range jobState.Executions {
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

	return result, nil
}

func (b *BacalhauExecutor) Render(job config.Job, cids []string) interface{} {
	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}

	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: job.Image,
			// TODO: There's a lot going on here, and we should encapsulate it in code/container.
			Entrypoint: job.Entrypoint,
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          job.Outputs.Path,
			},
		},
		Annotations: []string{amplifyAnnotation},
		NodeSelectors: []model.LabelSelectorRequirement{
			{
				Key:      "owner",
				Operator: selection.Equals,
				Values:   []string{"bacalhau"},
			},
		},
	}

	// The root node in the composite is the original data
	for i, c := range cids {
		input := model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           c,
			Path:          fmt.Sprintf("/inputs%d", i),
		}
		j.Spec.Inputs = append(j.Spec.Inputs, input)
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
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
