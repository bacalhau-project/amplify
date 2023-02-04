// Package job provides an abstraction of a job that runs on Bacalhau.
//
// A job is part of an Amplify workflow that performs some kind of
// data related task. It's unit of work is a Bacalhau job.
package job

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/ipfs/go-cid"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

type BacalhauJob interface {
	Spec() model.Job
}

// MetadataExtractionJob constructs a Bacalhau job that extracts metadata from
// a CID
type MetadataExtractionJob struct {
	CID cid.Cid
}

func (p *MetadataExtractionJob) Spec() model.Job {
	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ghcr.io/bacalhau-project/amplify/tika:latest@sha256:266a93a391f139e6d23cff2cd9e4b555e49a58f001c20e9082a62a6859a02a50",
			Entrypoint: []string{
				"extract-metadata",
				"/inputs",
				"/outputs",
			},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "inputs",
				CID:           p.CID.String(),
				Path:          "/inputs",
			},
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
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

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}

// MergeJob takes an Amplify composite then carefully structures
// the job inputs so that a simple copy job can produce the outputs.
type MergeJob struct {
	Composite *composite.Composite
}

func (p *MergeJob) Spec() model.Job {
	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}

	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			// TODO: There's a lot going on here, and we should encapsulate it in code/container.
			Entrypoint: []string{
				"bash",
				"-c",
				// Remember that the original data might be a blob, or it might have a folder structure
				"if [ -d /inputs ] ; then cp -r /inputs/* /outputs ; else cp /inputs /outputs/blob ; fi && " + // First copy all the original data
					"find / -iwholename '/inputs*metadata.json' | while read line ; do " + // Find all metadata.json files
					"result=$(" + // Store the result in a variable
					"echo $line | " +
					"sed 's/\\(.*\\)outputs\\//\\1/g' | " + // Remove the /outputs post fix from the metadata results
					"sed 's/\\/inputs[0-9]*//g' | " + // Remove the /inputs prefix from the metadata results
					"sed -r 's/(.+)\\//\\1./g' " + // Remove the last / from the metadata results so it becomes file.metadata.json, except for blobs
					") ; " + // End store the result in a variable
					"output=/outputs${result} ; " + // Build the output directory
					"mkdir -p $(dirname ${output}) ; " + // Create the output directory
					"echo \"Copying $line to $output\" ; " +
					"cp $line $output " + // Copy the metadata file to the output
					"; done",
			},
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
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
		CID:           p.Composite.Node.Cid().String(),
		Path:          "/inputs",
	}
	j.Spec.Inputs = append(j.Spec.Inputs, rootIntput)

	inputNum := 0
	var generateInputsRecursive func(*composite.Composite, string)
	generateInputsRecursive = func(c *composite.Composite, path string) {
		// If this file is a leaf node, add it to the inputs
		if len(c.Children) == 0 {
			input := model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				CID:           c.Result.String(),
				Path:          fmt.Sprintf("/inputs%d%s", inputNum, path),
			}
			j.Spec.Inputs = append(j.Spec.Inputs, input)
			inputNum++
		}
		// If the child has children, we need to recurse
		for _, child := range c.Children {
			var childPath string
			if child.Name != "" {
				childPath = strings.Join([]string{path, child.Name}, "/")
			} else {
				childPath = strings.Join([]string{path, child.Node.Cid().String()}, "/")
			}
			generateInputsRecursive(child, childPath)
		}
	}
	generateInputsRecursive(p.Composite, "")

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}
