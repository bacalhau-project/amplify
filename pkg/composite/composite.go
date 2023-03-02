// Package composite is a structural pattern that links CIDs from a Merkle tree
// to a directory-like structure for use in Bacalhau.
package composite

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
)

// Composite is a combination of IPLD nodes, filename (if any), and resultant
// CIDs. It can also have children, which are other Composites.
// It is made thread safe by embedding a sync.Mutex and hiding the fields
type Composite struct {
	sync.Mutex
	name     string
	node     ipldformat.Node
	result   executor.Result
	children []*Composite
}

// NewComposite creates a new composite from a root IPLD node represented by a
// CID
func NewComposite(ctx context.Context, ng ipldformat.NodeGetter, cid cid.Cid) (*Composite, error) {
	node, err := ng.Get(ctx, cid)
	if err != nil {
		return nil, err
	}
	c := &Composite{
		node: node,
	}
	err = c.build(ctx, ng)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// build recursively walks a Merkel tree to form a composite
func (c *Composite) build(ctx context.Context, ng ipldformat.NodeGetter) error {
	c.Lock()
	defer c.Unlock()
	// Get the root node
	for _, l := range c.node.Links() {
		childNode, err := ng.Get(ctx, l.Cid)
		if err != nil {
			return err
		}
		childComposite := &Composite{name: l.Name, node: childNode}
		if err := childComposite.build(ctx, ng); err != nil {
			return err
		}
		c.children = append(c.children, childComposite)
	}
	return nil
}

// String returns a string representation of the composite
func (c *Composite) String() string {
	var buf bytes.Buffer
	printRecursive(c, &buf, 0)
	return buf.String()
}

// Name returns the Name of a composite node
func (c *Composite) Name() string {
	c.Lock()
	defer c.Unlock()
	return c.name
}

// Node returns the Node of a composite
func (c *Composite) Node() ipldformat.Node {
	c.Lock()
	defer c.Unlock()
	return c.node
}

// Children returns all the children of a composite
func (c *Composite) Children() []*Composite {
	c.Lock()
	defer c.Unlock()
	return c.children
}

// Result gets the result of a composite
func (c *Composite) Result() executor.Result {
	c.Lock()
	defer c.Unlock()
	return c.result
}

// SetResult sets the result of a composite
func (c *Composite) SetResult(result executor.Result) {
	c.Lock()
	defer c.Unlock()
	c.result = result
}

func printRecursive(c *Composite, buf *bytes.Buffer, indent int) {
	c.Lock()
	defer c.Unlock()
	if c.result.CID.Defined() {
		fmt.Fprintf(buf, "%s: %s -- %s\n", c.name, c.node.Cid().String(), c.result.CID.String())
	} else {
		fmt.Fprintf(buf, "%s: %s\n", c.name, c.node.Cid().String())
	}
	for _, child := range c.children {
		for i := 0; i < indent; i++ {
			buf.WriteString("│   ")
		}
		buf.WriteString("└── ")
		printRecursive(child, buf, indent+1)
	}
}
