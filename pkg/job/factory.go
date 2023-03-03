package job

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

type JobFactory struct {
	conf config.Config
}

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
	return config.Job{}, fmt.Errorf("job %s not found", name)
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

	if job.Inputs.Type == config.StorageTypeIncremental {
		inputNum := 0
		var generateInputsRecursive func(*composite.Composite, string)
		generateInputsRecursive = func(c *composite.Composite, path string) {
			// If this file is a leaf node, add it to the inputs
			if len(c.Children()) == 0 {
				input := model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					CID:           c.Result().CID.String(),
					Path:          fmt.Sprintf("/inputs%d%s", inputNum, path),
				}
				j.Spec.Inputs = append(j.Spec.Inputs, input)
				inputNum++
			}
			// If the child has children, we need to recurse
			for _, child := range c.Children() {
				var childPath string
				if child.Name() != "" {
					childPath = strings.Join([]string{path, child.Name()}, "/")
				} else {
					childPath = strings.Join([]string{path, child.Node().Cid().String()}, "/")
				}
				generateInputsRecursive(child, childPath)
			}
		}
		generateInputsRecursive(comp, "")
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}
