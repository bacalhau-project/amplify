// Dag provides a way to describe a directed acyclic graph of work to be done.
// It starts with a root node, then you add nodes to it. Outputs are
// automatically connected to subsequent inputs.
package dag

import (
	"context"
)

type any comparable

func FilterForRootNodes(ctx context.Context, dags []Node[IOSpec]) ([]Node[IOSpec], error) {
	rootNodes := []Node[IOSpec]{}
	for _, node := range dags {
		n, err := node.Get(ctx)
		if err != nil {
			return nil, err
		}
		for _, i := range n.Inputs {
			if i.IsRoot() {
				rootNodes = append(rootNodes, node)
				break
			}
		}
	}
	return rootNodes, nil
}

func NodeMapToList[T any](dags map[T]Node[IOSpec]) (nodes []Node[IOSpec]) {
	for _, node := range dags {
		nodes = append(nodes, node)
	}
	return
}
