package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/bacalhau-project/amplify/pkg/config"
	bacalhauJob "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
)

const (
	amplifyAnnotation        = "amplify"
	bacalhauApiHost          = "bootstrap.production.bacalhau.org"
	bacalhauApiPort          = uint16(1234)
	maxUInt16         uint16 = 0xFFFF
	minUInt16         uint16 = 0x0000
)

func NewBacalhauExecutor() Executor {
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}

	var apiHost = bacalhauApiHost
	var apiPort = bacalhauApiPort

	// configurable target api to support localhost, devstack, etc.
	if h := os.Getenv("BACALHAU_API_HOST"); h != "" {
		apiHost = h
	}
	if p := os.Getenv("BACALHAU_API_PORT"); p != "" {
		// apiPort = uint16(p)
		port, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			panic(fmt.Sprintf("BACALHAU_API_PORT must be uint16 (%d-%d): %s", minUInt16, maxUInt16, p))
		}
		apiPort = uint16(port)

	}

	return &BacalhauExecutor{Client: createClient(apiHost, apiPort)}
}

type BacalhauExecutor struct {
	Client *publicapi.RequesterAPIClient
}

// func (b *BacalhauExecutor) GetClient() *publicapi.RequesterAPIClient {
// 	return b.Client
// }

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
		jobWithInfo, bool, err := b.Client.Get(ctx, submittedJob.Metadata.ID)
		if err != nil {
			return result, fmt.Errorf("getting Bacalhau job info: %s", err)
		}
		if !bool {
			return result, fmt.Errorf("job not found")
		}
		result, err = parseResult(ctx, jobWithInfo)
		if err != nil {
			return result, fmt.Errorf("parsing result: %s", err)
		}
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

	log.Ctx(ctx).Debug().Int("JobState", int(jobWithInfo.State.State)).Str("jobId", submittedJob.Metadata.ID).Int("len(executions)", len(jobWithInfo.State.Executions)).Msg("job results retrieved")
	rendered, err := json.Marshal(jobWithInfo)
	if err != nil {
		return result, fmt.Errorf("marshalling Bacalhau job info: %s", err)
	}
	fmt.Println(string(rendered))

	result, err = parseResult(ctx, jobWithInfo)
	if err != nil {
		return result, fmt.Errorf("parsing result: %s", err)
	}
	if result.CID == cid.Undef {
		return result, fmt.Errorf("no result CID found")
	}
	log.Ctx(ctx).Debug().Str("cid", result.CID.String()).Str("jobId", submittedJob.Metadata.ID).Msg("parsed result")

	return result, nil
}

func parseResult(ctx context.Context, jobWithInfo *model.JobWithInfo) (Result, error) {
	result := Result{}
	result.ID = jobWithInfo.Job.ID()
	for _, e := range jobWithInfo.State.Executions {
		log.Ctx(ctx).Trace().Str("PublishedResult", fmt.Sprintf("%#v", e)).Str("jobId", result.ID).Msg("parsing result")
		if e.PublishedResult.CID == "" {
			continue
		}
		log.Ctx(ctx).Debug().Str("cid", e.PublishedResult.CID).Str("jobId", result.ID).Msg("parsing result")
		c, err := cid.Parse(e.PublishedResult.CID)
		if err != nil {
			return result, fmt.Errorf("parsing result CID: %s", err)
		}
		result.CID = c
		result.StdOut = e.RunOutput.STDOUT
		result.StdErr = e.RunOutput.STDERR
		result.Status = e.State.String()
		break
	}
	if result.Status == "" {
		result.Status = jobWithInfo.State.State.String()
	}
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
				Operator: "=",
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

func createClient(host string, port uint16) *publicapi.RequesterAPIClient {
	client := publicapi.NewRequesterAPIClient(host, port)
	return client
}
