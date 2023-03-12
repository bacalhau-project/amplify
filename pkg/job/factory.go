package job

import (
	"errors"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

type JobFactory struct {
	conf config.Config
}

var ErrJobNotFound = errors.New("job not found")

// NewJobFactory creates a new JobFactory
func NewJobFactory(conf config.Config) JobFactory {
	return JobFactory{
		conf: conf,
	}
}

// GetJob gets a job config from a job factory
func (f *JobFactory) GetJob(name string) (config.Job, error) {
	for _, job := range f.conf.Jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return config.Job{}, ErrJobNotFound
}

// JobNames returns all the names of the jobs in a job factory
func (f *JobFactory) JobNames() []string {
	var names []string
	for _, job := range f.conf.Jobs {
		names = append(names, job.Name)
	}
	return names
}

// Render renders a job from a job factory
func (f *JobFactory) Render(name string, comp *composite.Composite) interface{} {
	job, err := f.GetJob(name)
	if err != nil {
		panic(err)
	}

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
	rootIntput := model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           comp.Node().Cid().String(), // assume root node is root cid
		Path:          "/inputs",
	}
	j.Spec.Inputs = append(j.Spec.Inputs, rootIntput)

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}
