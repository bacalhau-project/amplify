// Package composite is a structural pattern that links CIDs from a Merkle tree
// to a directory-like structure for use in Bacalhau.
package composite

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
)

// Composite is a combination of IPLD nodes, filename (if any), and resultant
// CIDs. It can also have children, which are other Composites.
// It is made thread safe by embedding a sync.Mutex.
type Composite struct {
	sync.Mutex
	Name     string
	Node     ipldformat.Node
	Result   cid.Cid
	Children []*Composite
}

// NewComposite creates a new composite from a root IPLD node represented by a
// CID
func NewComposite(ctx context.Context, ng ipldformat.NodeGetter, cid cid.Cid) (*Composite, error) {
	node, err := ng.Get(ctx, cid)
	if err != nil {
		return nil, err
	}
	c := &Composite{
		Node: node,
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
	for _, l := range c.Node.Links() {
		childNode, err := ng.Get(ctx, l.Cid)
		if err != nil {
			return err
		}
		childComposite := &Composite{Name: l.Name, Node: childNode}
		if err := childComposite.build(ctx, ng); err != nil {
			return err
		}
		c.Children = append(c.Children, childComposite)
	}
	return nil
}

// String returns a string representation of the composite
func (c *Composite) String() string {
	c.Lock()
	defer c.Unlock()
	var buf bytes.Buffer
	printRecursive(c, &buf, 0)
	return buf.String()
}

func printRecursive(c *Composite, buf *bytes.Buffer, indent int) {
	if c.Result.Defined() {
		fmt.Fprintf(buf, "%s: %s -- %s\n", c.Name, c.Node.Cid().String(), c.Result.String())
	} else {
		fmt.Fprintf(buf, "%s: %s\n", c.Name, c.Node.Cid().String())
	}
	for _, child := range c.Children {
		for i := 0; i < indent; i++ {
			buf.WriteString("│   ")
		}
		buf.WriteString("└── ")
		printRecursive(child, buf, indent+1)
	}
}
