package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
)

const amplifyAnnotation = "amplify"

var (
	ErrCIDIsNotValid    = fmt.Errorf("cid is not valid")
	ErrJobNotSuccessful = fmt.Errorf("job was not successful")
)

func NewBacalhauExecutor() Executor {
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
	return &BacalhauExecutor{Client: getClient("bootstrap.production.bacalhau.org", 1234)}
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
	submittedJob, err := b.Client.Submit(ctx, &j)
	if err != nil {
		return result, fmt.Errorf("submitting Bacalhau job: %s", err)
	}
	log.Ctx(ctx).Debug().Str("jobId", submittedJob.Metadata.ID).Msg("job submitted, waiting for completion")
	err = waitUntilCompleted(ctx, b.Client, submittedJob)
	if err != nil {
		log.Warn().Err(err).Str("jobId", submittedJob.Metadata.ID).Msg("wait for job completion failed")
	}
	log.Ctx(ctx).Debug().Str("jobId", submittedJob.Metadata.ID).Msg("job complete, getting results")
	jobWithInfo, bool, err := b.Client.Get(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return result, fmt.Errorf("getting Bacalhau job info: %s", err)
	}
	if !bool {
		return result, fmt.Errorf("job not found")
	}
	log.Ctx(ctx).Debug().Int("JobState", int(jobWithInfo.State.State)).Str("jobId", submittedJob.Metadata.ID).Int("len(executions)", len(jobWithInfo.State.Executions)).Msg("job results retrieved")
	result, err = parseResult(ctx, jobWithInfo)
	if err != nil {
		return result, fmt.Errorf("parsing result: %s", err)
	}
	if result.Status != model.JobStateCompleted.String() {
		return result, ErrJobNotSuccessful
	}
	if result.CID == "" {
		return result, fmt.Errorf("no result CID found, job may have failed")
	}
	log.Ctx(ctx).Debug().Str("cid", result.CID).Str("jobId", submittedJob.Metadata.ID).Msg("parsed result")

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
		result.CID = c.String()
		result.StdOut = e.RunOutput.STDOUT
		result.StdErr = e.RunOutput.STDERR
		result.Status = e.State.String()
		break
	}
	if result.Status == "" {
		result.Status = jobWithInfo.State.State.String()
		for _, e := range jobWithInfo.State.Executions {
			if e.State != model.ExecutionStateFailed {
				continue
			}
			result.StdErr = e.Status
		}
	}
	return result, nil
}

func (b *BacalhauExecutor) Render(job config.Job, inputs []ExecutorIOSpec, outputs []ExecutorIOSpec) (interface{}, error) {
	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}

	ips := make([]model.StorageSpec, len(inputs))
	for idx, i := range inputs {
		if i.Ref == "" {
			return nil, fmt.Errorf("input CID for %s is blank", i.Name)
		}
		if i.Path == "" {
			return nil, fmt.Errorf("input path for %s is blank", i.Name)
		}
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
	return j, nil
}

func waitUntilCompleted(ctx context.Context, client *publicapi.RequesterAPIClient, submittedJob *model.Job) error {
	timeOutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timeOutCtx.Done():
			return fmt.Errorf("timed out waiting for job to complete")
		case <-ticker.C:
			jobWithInfo, bool, err := client.Get(ctx, submittedJob.Metadata.ID)
			if err != nil {
				return fmt.Errorf("getting Bacalhau job info: %s", err)
			}
			if !bool {
				return fmt.Errorf("job not found")
			}
			log.Ctx(ctx).Debug().Int("JobState", int(jobWithInfo.State.State)).Str("jobId", submittedJob.Metadata.ID).Int("len(executions)", len(jobWithInfo.State.Executions)).Msg("job results retrieved")
			result, err := parseResult(ctx, jobWithInfo)
			if err != nil {
				return fmt.Errorf("parsing result: %s", err)
			}
			if result.Status == model.JobStateCompleted.String() || result.Status == model.JobStateError.String() || result.Status == model.JobStateCancelled.String() {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// TODO: This doesn't seem to work
	// resolver := client.GetJobStateResolver()
	// return resolver.Wait(
	// 	ctx,
	// 	submittedJob.Metadata.ID,
	// 	bacalhauJob.WaitForTerminalStates(),
	// )
}

func getClient(host string, port uint16) *publicapi.RequesterAPIClient {
	client := publicapi.NewRequesterAPIClient(host, port)
	return client
}
