package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-libipfs/bitswap/client"
	"github.com/ipfs/go-libipfs/bitswap/network"
	"github.com/ipfs/go-libipfs/blocks"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/kubo/routing"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Idea
// Inspecting what a CID is is hard. It might be a CAR, it might be a file. It might be some other directory-like format.
// For amplify we need to introspect files to obtain metadata. But that only works well for files.
// So this is an experiment to try to "iterate" of the IPLD graph of a CID and see what we can find and print out all the info about it.

// const fileCid = "QmP86GD3S6HZjaG49mW6z9xdXaTzSS9LijUArtwZYGWHZC" // not found by cid.contact
// const fileCid = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi" // found by cid.contact
// const fileCid = "bafykbzaceduyw3zbo3tkqzf56uxo2g42cvdooloog4n2zuxsmx4eprkiylcdw" // 176MB file
// const fileCid = "bafykbzaceanbnp4bhicw5egtdppcyv6utgoykpnquobrjwdqqhe2nsqjarnlq" // 32GB file from common crawl -- times out, ipfs get doesn't work either
const fileCid = "QmSsAZE92hcshQLMXtzziSFttFCPLyA7H6G6sNWzQDqfnm" // A bacalhau result

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

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Enable logging of IPFS subsystems -- it get's noisy, try to isolate what you want to see.
	logging.SetLogLevel("blockservice", "debug")
	logging.SetLogLevel("bitswap-client", "debug")
	// logging.SetLogLevel("bitswap", "debug")
	// logging.SetDebugLogging()

	// Make an ID
	privateKey, err := makeIdentity()
	if err != nil {
		log.Fatal(err)
	}

	// Make a host
	h, err := makeHost(0, privateKey)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close()

	// This function periodically prints the connected peers.
	// go periodicFunc(ctx, func() {
	// 	peers := h.Peerstore().Peers()
	// 	log.Println("Host stats:")
	// 	log.Printf("  - %d peers\n", len(peers))
	// 	log.Printf("  - %s\n", peers.String())
	// })

	fullAddr := getHostAddress(h)
	log.Printf("I am %s\n", fullAddr)

	log.Println("Connecting to public IPFS peers")
	for _, addr := range publicIPFSPeers {
		err = connectToPeers(ctx, h, addr)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("downloading UnixFS file with CID: %s\n", fileCid)
	_, err = runClient(ctx, h, privateKey, cid.MustParse(fileCid))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("found the data")
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

func getHostAddress(h host.Host) string {
	// Build host multiaddress
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().String()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := h.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

type CustomBlockReceivedNotifier struct{}

func (c *CustomBlockReceivedNotifier) ReceivedBlocks(p peer.ID, blks []blocks.Block) {
	log.Printf("received %d blocks from peer %s", len(blks), p.String())
}

func runClient(ctx context.Context, h host.Host, privateKey crypto.PrivKey, c cid.Cid) ([]byte, error) {
	// The datastore for this node
	datastore := datastore.NewNullDatastore() // i.e. don't cache or store anything
	bs := blockstore.NewBlockstore(datastore)

	// Create a DHT client, which is a content routing client that uses the DHT
	dhtRouter := dht.NewDHTClient(ctx, h, datastore)

	// Create HTTP client, which routes via contact.cid
	privkeyb, err := crypto.MarshalPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	httpRouter, err := routing.ConstructHTTPRouter("https://cid.contact", h.ID().Pretty(), []string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/udp/4001/quic"}, base64.StdEncoding.EncodeToString(privkeyb))
	if err != nil {
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
	n := network.NewFromIpfsHost(h, router)
	// Create a notifier to announce when a block has been received
	blockNotifier := client.WithBlockReceivedNotifier(&CustomBlockReceivedNotifier{})
	// Now create a bitswap client and start the bitswap service. This allows us to make requests.
	bswap := client.New(ctx, n, bs, blockNotifier)
	n.Start(bswap)
	defer bswap.Close()

	// Check if the CID is already in the local block store (if using a NullDatastore, this will always fail)
	_, err = bs.Get(ctx, c)
	if ipldformat.IsNotFound(err) {
		log.Println("cid is not found in the local blockstore, will ask peers for it")
	} else if err != nil {
		return nil, err
	}

	// This periodically prints the bitswap stats
	go periodicFunc(ctx, func() {
		log.Println("Bitswap stats:")
		log.Printf("  IsOnline: %t", bswap.IsOnline())
		log.Printf("  Wantlist [keys: %d]\n", len(bswap.GetWantlist()))
		log.Printf("  Wantlist [want-haves: %d]\n", len(bswap.GetWantHaves()))
		log.Printf("  Wantlist [want-blocks: %d]\n", len(bswap.GetWantBlocks()))
		for _, c := range bswap.GetWantlist() {
			log.Printf("  %s - %d\n", c.String(), c.Type())
		}
	})

	// Now we can create a new block service and a DAG service, which manages block requests and navigation
	blockService := blockservice.New(bs, bswap)
	nodeGetter := merkledag.NewDAGService(blockService)
	// A DAG session ensures that if multiple blocks are requested (a directory-based CID, for example)
	// they are managed in a single request
	nodeGetterSession := merkledag.NewSession(ctx, nodeGetter)

	// And finally, get the CID, called a node.
	node, err := nodeGetterSession.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	// Lots of information about the node
	log.Printf("found the node: %s", node.Cid().String())
	size, err := node.Size()
	if err != nil {
		return nil, err
	}
	stat, err := node.Stat()
	if err != nil {
		return nil, err
	}
	log.Printf("node stat: %+v", stat)
	log.Printf("node size: %d", size)
	log.Println("node tree: ")
	for _, s := range node.Tree("", -1) {
		log.Printf("   %s", s)
	}
	log.Println("node links: ")
	for _, l := range node.Links() {
		log.Printf("   %s %d - %s", l.Name, l.Size, l.Cid.String())
	}

	// // Get the file from the node
	// dserv := merkledag.NewReadOnlyDagService(nodeGetterSession)
	// nd, err := dserv.Get(ctx, c)
	// if err != nil {
	// 	return nil, err
	// }

	// unixFSNode, err := unixfile.NewUnixfsFile(ctx, dserv, nd)
	// if err != nil {
	// 	return nil, err
	// }

	// var buf bytes.Buffer
	// if f, ok := unixFSNode.(files.File); ok {
	// 	if _, err := io.Copy(&buf, f); err != nil {
	// 		return nil, err
	// 	}
	// }

	return nil, nil
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
