package ipfs

import (
	"context"
	"encoding/base64"
	"net/http"

	drc "github.com/ipfs/go-delegated-routing/client"
	drclient "github.com/ipfs/go-libipfs/routing/http/client"
	"github.com/ipfs/go-libipfs/routing/http/contentrouter"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multicodec"
)

const MaxProvideBatchSize = 100
const MaxProvideConcurrency = 100

var CurrentCommit string

const CurrentVersionNumber = "0.0.0-dev"

// MakeHttpRouter creates a new http routing wrapper.
// Code adapted from: github.com/ipfs/kubo/routing
func MakeHttpRouter(endpoint string, peerID string, addrs []string, privKey string) (routing.Routing, error) {
	// Increase per-host connection pool since we are making lots of concurrent requests.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 500
	transport.MaxIdleConnsPerHost = 100

	delegateHTTPClient := &http.Client{
		Transport: &drclient.ResponseBodyLimitedTransport{
			RoundTripper: transport,
			LimitBytes:   1 << 20,
		},
	}

	key, err := decodePrivKey(privKey)
	if err != nil {
		return nil, err
	}

	addrInfo, err := createAddrInfo(peerID, addrs)
	if err != nil {
		return nil, err
	}

	cli, err := drclient.New(
		endpoint,
		drclient.WithHTTPClient(delegateHTTPClient),
		drclient.WithIdentity(key),
		drclient.WithProviderInfo(addrInfo.ID, addrInfo.Addrs),
		drclient.WithUserAgent(getUserAgentVersion()),
	)
	if err != nil {
		return nil, err
	}

	cr := contentrouter.NewContentRoutingClient(
		cli,
		contentrouter.WithMaxProvideBatchSize(MaxProvideBatchSize),
		contentrouter.WithMaxProvideConcurrency(MaxProvideConcurrency),
	)

	return &httpRoutingWrapper{
		ContentRouting:    cr,
		ProvideManyRouter: cr,
	}, nil
}

// httpRoutingWrapper is a wrapper needed to construct the routing.Routing interface from
// http delegated routing.
type httpRoutingWrapper struct {
	routing.ContentRouting
	routinghelpers.ProvideManyRouter
}

func (c *httpRoutingWrapper) Bootstrap(ctx context.Context) error {
	return nil
}

func (c *httpRoutingWrapper) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, routing.ErrNotSupported
}

func (c *httpRoutingWrapper) PutValue(context.Context, string, []byte, ...routing.Option) error {
	return routing.ErrNotSupported
}

func (c *httpRoutingWrapper) GetValue(context.Context, string, ...routing.Option) ([]byte, error) {
	return nil, routing.ErrNotSupported
}

func (c *httpRoutingWrapper) SearchValue(context.Context, string, ...routing.Option) (<-chan []byte, error) {
	out := make(chan []byte)
	close(out)
	return out, routing.ErrNotSupported
}

func decodePrivKey(keyB64 string) (ic.PrivKey, error) {
	pk, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, err
	}

	return ic.UnmarshalPrivateKey(pk)
}

func createAddrInfo(peerID string, addrs []string) (peer.AddrInfo, error) {
	pID, err := peer.Decode(peerID)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	var mas []ma.Multiaddr
	for _, a := range addrs {
		m, err := ma.NewMultiaddr(a)
		if err != nil {
			return peer.AddrInfo{}, err
		}

		mas = append(mas, m)
	}

	return peer.AddrInfo{
		ID:    pID,
		Addrs: mas,
	}, nil
}

func createProvider(peerID string, addrs []string) (*drc.Provider, error) {
	addrInfo, err := createAddrInfo(peerID, addrs)
	if err != nil {
		return nil, err
	}
	return &drc.Provider{
		Peer: addrInfo,
		ProviderProto: []drc.TransferProtocol{
			{Codec: multicodec.TransportBitswap},
		},
	}, nil
}

// GetUserAgentVersion is the libp2p user agent used by go-ipfs.
//
// Note: This will end in `/` when no commit is available. This is expected.
func getUserAgentVersion() string {
	userAgent := "amplify/" + CurrentVersionNumber + "/" + CurrentCommit
	return userAgent
}
