package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/config"
	bacalhauJob "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
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
	log.Ctx(ctx).Debug().Str("jobId", submittedJob.Metadata.ID).Msg("job complete, waiting for results")

	jobWithInfo, bool, err := b.Client.Get(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return result, fmt.Errorf("getting Bacalhau job info: %s", err)
	}
	if !bool {
		return result, fmt.Errorf("job not found")
	}

	log.Ctx(ctx).Debug().Int("JobState", int(jobWithInfo.State.State)).Str("jobId", submittedJob.Metadata.ID).Int("len", len(jobWithInfo.State.Shards)).Msg("job results retrieved")
	rendered, err := json.Marshal(jobWithInfo)
	if err != nil {
		return result, fmt.Errorf("marshalling Bacalhau job info: %s", err)
	}
	fmt.Println(string(rendered))

	result.ID = submittedJob.Metadata.ID
	for _, s := range jobWithInfo.State.Shards {
		for _, e := range s.Executions {
			log.Ctx(ctx).Trace().Str("PublishedResult", fmt.Sprintf("%#v", e)).Str("jobId", submittedJob.Metadata.ID).Msg("parsing result")
			if e.PublishedResult.CID == "" {
				continue
			}
			log.Ctx(ctx).Debug().Str("cid", e.PublishedResult.CID).Str("jobId", submittedJob.Metadata.ID).Msg("parsing result")
			result.CID, err = cid.Parse(e.PublishedResult.CID)
			if err != nil {
				return result, fmt.Errorf("parsing result CID: %s", err)
			}
			result.StdOut = e.RunOutput.STDOUT
			result.StdErr = e.RunOutput.STDERR
			result.Status = e.State.String()
			break
		}
	}
	if result.CID == cid.Undef {
		return result, fmt.Errorf("no result CID found")
	}
	log.Ctx(ctx).Debug().Str("cid", result.CID.String()).Str("jobId", submittedJob.Metadata.ID).Msg("parsed result")

	return result, nil
}

func (b *BacalhauExecutor) Render(job config.Job, inputs []ExecutorIOSpec, outputs []ExecutorIOSpec) interface{} {
	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}

	ips := make([]model.StorageSpec, len(inputs))
	for idx, i := range inputs {
		ips[idx] = model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			Name:          i.Name,
			CID:           i.Ref,
			Path:          i.Path,
		}
	}

	ops := make([]model.StorageSpec, len(outputs))
	for idx, o := range outputs {
		ops[idx] = model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			Name:          o.Name,
			Path:          o.Path,
		}
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
		Inputs:      ips,
		Outputs:     ops,
		Annotations: []string{amplifyAnnotation},
		NodeSelectors: []model.LabelSelectorRequirement{
			{
				Key:      "owner",
				Operator: selection.Equals,
				Values:   []string{"bacalhau"},
			},
		},
		Deal: model.Deal{
			Concurrency: 1,
		},
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
