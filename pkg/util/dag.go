package util

import "github.com/bacalhau-project/amplify/pkg/dag"

// GetLeafOutputs returns the outputs of the leaf nodes in the DAG
func GetLeafOutputs(dag []*dag.Node[dag.IOSpec]) []string {
	var results []string
	for _, child := range dag {
		if len(child.Children()) == 0 {
			for _, output := range child.Outputs() {
				results = append(results, output.CID())
			}
		} else {
			results = append(results, GetLeafOutputs(child.Children())...)
		}
	}
	return results
}
