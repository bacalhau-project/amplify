// Package cli provides helpers for the CLI
package cli

import (
	"context"
	"sync"

	"github.com/bacalhau-project/amplify/pkg/ipfs"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog/log"
)

// NodeProvider implements the ipldformat.NodeGetter interface and lazily
// loads an IPFS client to get nodes.
type NodeProvider interface {
	Get(context.Context, cid.Cid) (ipldformat.Node, error)
	GetMany(context.Context, []cid.Cid) <-chan *ipldformat.NodeOption
	Close() error
}

type IPFSNodeProvider struct {
	sync.Mutex
	session *ipfs.IPFSSession
}

func NewNodeProvider(ctx context.Context) IPFSNodeProvider {
	return IPFSNodeProvider{session: nil}
}

// Get lazily loads the IPFS connection and proxies NodeGetter.Get
func (p *IPFSNodeProvider) Get(ctx context.Context, c cid.Cid) (ipldformat.Node, error) {
	p.initIPFS(ctx)
	return p.session.NodeGetter.Get(ctx, c)
}

// GetMany lazily loads the IPFS connection and proxies NodeGetter.GetMany
func (p *IPFSNodeProvider) GetMany(ctx context.Context, c []cid.Cid) <-chan *ipldformat.NodeOption {
	p.initIPFS(ctx)
	return p.session.NodeGetter.GetMany(ctx, c)
}

// Close closes the underlying IPFS session if it exists
func (p *IPFSNodeProvider) Close() error {
	if p.session != nil {
		p.session.Close()
	}
	return nil
}

func (p *IPFSNodeProvider) initIPFS(ctx context.Context) {
	if p.session == nil {
		p.Lock()
		defer p.Unlock()
		session, err := ipfs.NewIPFSSession(ctx)
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("Failed to initialize IPFS session")
		}
		p.session = session
	}
}
