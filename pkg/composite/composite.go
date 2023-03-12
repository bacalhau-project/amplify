// Package composite links a CID to a result
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

// Composite is a combination of an IPLD node and resultant CIDs.
// It is made thread safe by embedding a sync.Mutex and hiding the fields
type Composite struct {
	sync.Mutex
	node   ipldformat.Node
	result executor.Result
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
	return c, nil
}

// String returns a string representation of the composite
func (c *Composite) String() string {
	var buf bytes.Buffer
	print(c, &buf, 0)
	return buf.String()
}

// Node returns the Node of a composite
func (c *Composite) Node() ipldformat.Node {
	c.Lock()
	defer c.Unlock()
	return c.node
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

func print(c *Composite, buf *bytes.Buffer, indent int) {
	c.Lock()
	defer c.Unlock()
	if c.result.CID.Defined() {
		fmt.Fprintf(buf, "%s --> %s\n", c.node.Cid().String(), c.result.CID.String())
	} else {
		fmt.Fprintf(buf, "%s\n", c.node.Cid().String())
	}
}
