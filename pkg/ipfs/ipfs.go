// Package ipfs provides a session to the IPFS network.
package ipfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-libipfs/bitswap/client"
	"github.com/ipfs/go-libipfs/bitswap/network"
	"github.com/ipfs/go-libipfs/blocks"
	"github.com/ipfs/go-merkledag"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

var bacalhauIPFSPeers = []string{
	"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}

// See https://docs.ipfs.tech/how-to/peering-with-content-providers/#content-provider-list
var publicIPFSPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

type IPFSSession struct {
	Host          host.Host
	BitswapClient *client.Client
	NodeGetter    ipldformat.NodeGetter
}

// Close closes the IPFSSession
func (s *IPFSSession) Close() {
	if s.BitswapClient != nil {
		err := s.BitswapClient.Close()
		if err != nil {
			log.Err(err).Msg("error closing bitswap client")
		}
	}
	if s.Host != nil {
		err := s.Host.Close()
		if err != nil {
			log.Err(err).Msg("error closing host")
		}
	}
}

func NewIPFSSession(ctx context.Context) (*IPFSSession, error) {
	// Enable logging of IPFS subsystems -- it get's noisy, try to isolate what you want to see.
	// logging.SetLogLevel("blockservice", "debug")
	// logging.SetLogLevel("bitswap-client", "debug")
	// logging.SetLogLevel("bitswap", "debug")
	// logging.SetDebugLogging()
	ipfsSession := &IPFSSession{}

	// Make an ID
	privateKey, err := makeIdentity()
	if err != nil {
		ipfsSession.Close()
		return nil, err
	}

	// Make a host
	ipfsSession.Host, err = makeHost(0, privateKey)
	if err != nil {
		ipfsSession.Close()
		return nil, err
	}

	// This function periodically prints the connected peers.
	// go periodicFunc(ctx, func() {
	// 	peers := h.Peerstore().Peers()
	// 	log.Println("Host stats:")
	// 	log.Printf("  - %d peers\n", len(peers))
	// 	log.Printf("  - %s\n", peers.String())
	// })

	log.Ctx(ctx).Info().Msg("Connecting to public IPFS peers")
	for _, addr := range publicIPFSPeers {
		err = connectToPeers(ctx, ipfsSession.Host, addr)
		if err != nil {
			ipfsSession.Close()
			return nil, err
		}
	}
	log.Ctx(ctx).Info().Msg("Connecting to bacalhau IPFS peers")
	for _, addr := range bacalhauIPFSPeers {
		err = connectToPeers(ctx, ipfsSession.Host, addr)
		if err != nil {
			ipfsSession.Close()
			return nil, err
		}
	}

	// The datastore for this node
	datastore := datastore.NewNullDatastore() // i.e. don't cache or store anything
	bs := blockstore.NewBlockstore(datastore)

	// Create a DHT client, which is a content routing client that uses the DHT
	dhtRouter := dht.NewDHTClient(ctx, ipfsSession.Host, datastore)

	// Create HTTP client, which routes via contact.cid
	privkeyb, err := crypto.MarshalPrivateKey(privateKey)
	if err != nil {
		ipfsSession.Close()
		return nil, err
	}

	httpRouter, err := MakeHttpRouter("https://cid.contact", ipfsSession.Host.ID().Pretty(), []string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/udp/4001/quic"}, base64.StdEncoding.EncodeToString(privkeyb))
	if err != nil {
		ipfsSession.Close()
		return nil, err
	}

	// Create a bitswap router, which contacts various routers in parallel
	router := routinghelpers.NewComposableParallel([]*routinghelpers.ParallelRouter{
		{
			Timeout:     5 * time.Minute,
			IgnoreError: false,
			Router:      dhtRouter,
		},
		{
			Timeout:     5 * time.Minute,
			IgnoreError: false,
			Router:      httpRouter,
		},
	})

	// Create a new bitswap network. This is the thing that actually sends and receives bitswap messages over libp2p.
	n := network.NewFromIpfsHost(ipfsSession.Host, router)
	// Create a notifier to announce when a block has been received
	blockNotifier := client.WithBlockReceivedNotifier(&CustomBlockReceivedNotifier{})
	// Now create a bitswap client and start the bitswap service. This allows us to make requests.
	ipfsSession.BitswapClient = client.New(ctx, n, bs, blockNotifier)
	n.Start(ipfsSession.BitswapClient)

	// This periodically prints the bitswap stats
	go periodicFunc(ctx, func() {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Bitswap stats:")
		fmt.Fprintf(&buf, " IsOnline: %t,", ipfsSession.BitswapClient.IsOnline())
		fmt.Fprintf(&buf, " Wantlist [keys: %d],", len(ipfsSession.BitswapClient.GetWantlist()))
		fmt.Fprintf(&buf, " Wantlist [want-haves: %d],", len(ipfsSession.BitswapClient.GetWantHaves()))
		fmt.Fprintf(&buf, " Wantlist [want-blocks: %d],", len(ipfsSession.BitswapClient.GetWantBlocks()))
		for _, c := range ipfsSession.BitswapClient.GetWantlist() {
			fmt.Fprintf(&buf, " %s - %d,", c.String(), c.Type())
		}
		log.Ctx(ctx).Debug().Msg(buf.String())
	})

	// Now we can create a new block service and a DAG service, which manages block requests and navigation
	blockService := blockservice.New(bs, ipfsSession.BitswapClient)
	nodeGetter := merkledag.NewDAGService(blockService)
	// A DAG session ensures that if multiple blocks are requested (a directory-based CID, for example)
	// they are managed in a single request
	ipfsSession.NodeGetter = merkledag.NewSession(ctx, nodeGetter)

	return ipfsSession, nil

	// log.Printf("downloading UnixFS file with CID: %s\n", fileCid)
	// _, err = runClient(ctx, h, privateKey, cid.MustParse(fileCid))
	// if err != nil {
	// 	log.Fatal(err)
	// }
}

func makeIdentity() (crypto.PrivKey, error) {
	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func makeHost(listenPort int, privateKey crypto.PrivKey) (host.Host, error) {
	// Some basic libp2p options, see the go-libp2p docs for more details
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)), // port we are listening on, limiting to a single interface and protocol for simplicity
		libp2p.Identity(privateKey),
	}

	return libp2p.New(opts...)
}

type CustomBlockReceivedNotifier struct{}

func (c *CustomBlockReceivedNotifier) ReceivedBlocks(p peer.ID, blks []blocks.Block) {
	// log.Printf("received %d blocks from peer %s", len(blks), p.String())
}

func periodicFunc(ctx context.Context, f func()) {
	f()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f()
		}
	}
}

func connectToPeers(ctx context.Context, h host.Host, targetPeer string) error {
	// Turn the targetPeer into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(targetPeer)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	// Directly connect to the peer that we know has the content
	// Generally this peer will come from whatever content routing system is provided, however go-bitswap will also
	// ask peers it is connected to for content so this will work
	if err := h.Connect(ctx, *info); err != nil {
		return err
	}
	return nil
}
