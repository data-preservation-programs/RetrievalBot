package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/net"
	"github.com/ipfs/go-cid"
	goCid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	bsclient "github.com/ipfs/go-libipfs/bitswap/client"
	bsmsg "github.com/ipfs/go-libipfs/bitswap/message"
	bsnet "github.com/ipfs/go-libipfs/bitswap/network"
	"github.com/ipfs/go-libipfs/blocks"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"golang.org/x/exp/slices"

	_ "github.com/ipld/go-codec-dagpb"
	ipld "github.com/ipld/go-ipld-prime"
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagjson"
	_ "github.com/ipld/go-ipld-prime/codec/raw"
)

var logger = logging.Logger("spade-test")

type SingleContentRouter struct {
	AddrInfo peer.AddrInfo
}

func (s SingleContentRouter) PutValue(context.Context, string, []byte, ...routing.Option) error {
	return routing.ErrNotSupported
}

func (s SingleContentRouter) GetValue(context.Context, string, ...routing.Option) ([]byte, error) {
	return nil, routing.ErrNotFound
}

func (s SingleContentRouter) SearchValue(ctx context.Context, key string, opts ...routing.Option) (
	<-chan []byte, error) {
	return nil, routing.ErrNotFound
}

func (s SingleContentRouter) Provide(context.Context, goCid.Cid, bool) error {
	return routing.ErrNotSupported
}

func (s SingleContentRouter) FindProvidersAsync(context.Context, goCid.Cid, int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo)
	go func() {
		ch <- s.AddrInfo
		close(ch)
	}()
	return ch
}

func (s SingleContentRouter) FindPeer(context.Context, peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, routing.ErrNotFound
}

func (s SingleContentRouter) Bootstrap(context.Context) error {
	return nil
}

func (s SingleContentRouter) Close() error {
	return nil
}

type MessageReceiver struct {
	BSClient       *bsclient.Client
	MessageHandler func(ctx context.Context, sender peer.ID, incoming bsmsg.BitSwapMessage)
}

func (m MessageReceiver) ReceiveMessage(
	ctx context.Context,
	sender peer.ID,
	incoming bsmsg.BitSwapMessage) {
	m.BSClient.ReceiveMessage(ctx, sender, incoming)
	m.MessageHandler(ctx, sender, incoming)
}

func (m MessageReceiver) ReceiveError(err error) {
	m.BSClient.ReceiveError(err)
}

func (m MessageReceiver) PeerConnected(id peer.ID) {
	m.BSClient.PeerConnected(id)
}
func (m MessageReceiver) PeerDisconnected(id peer.ID) {
	m.BSClient.PeerDisconnected(id)
}

type BitswapClient struct {
	host    host.Host
	timeout time.Duration
}

func NewBitswapClient(host host.Host, timeout time.Duration) BitswapClient {
	return BitswapClient{
		host:    host,
		timeout: timeout,
	}
}

// Attempts to decode the block data into a node and return its links
func FindLinks(ctx context.Context, blk blocks.Block) ([]datamodel.Link, error) {
	decoder, err := cidlink.DefaultLinkSystem().DecoderChooser(cidlink.Link{Cid: blk.Cid()})

	if err != nil {
		return nil, err
	}

	if blk.Cid().Prefix().Codec == goCid.Raw {
		// This can happen at the bottom of the tree
		return nil, errors.New("raw block encountered fetching " + blk.Cid().String())
	}

	node, err := ipld.Decode(blk.RawData(), decoder)
	if err != nil {
		return nil, err
	}

	links, err := traversal.SelectLinks(node)
	if err != nil {
		return nil, err
	}

	return links, nil
}

// Returns the raw block data, the links, and error if any
func (c BitswapClient) SpadeTraversal(
	parent context.Context,
	target peer.AddrInfo,
	cid goCid.Cid) (blocks.Block, error) {
	network := bsnet.NewFromIpfsHost(c.host, SingleContentRouter{
		AddrInfo: target,
	})
	bswap := bsclient.New(parent, network, blockstore.NewBlockstore(datastore.NewMapDatastore()))
	notFound := make(chan struct{})
	network.Start(MessageReceiver{BSClient: bswap, MessageHandler: func(
		ctx context.Context, sender peer.ID, incoming bsmsg.BitSwapMessage) {
		if sender == target.ID && slices.Contains(incoming.DontHaves(), cid) {
			logger.Info("Block not found")
			close(notFound)
		}
	}})
	defer bswap.Close()
	defer network.Stop()
	connectContext, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	logger.Info("Connecting to target peer...")
	err := c.host.Connect(connectContext, target)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target peer, %s", err)
	}

	// startTime := time.Now()
	resultChan := make(chan blocks.Block)
	errChan := make(chan error)
	go func() {
		logger.Debug("Retrieving block...")
		blk, err := bswap.GetBlock(connectContext, cid)
		if err != nil {
			logger.Info(err)
			errChan <- err
		} else {
			resultChan <- blk
		}
	}()
	select {
	case <-notFound:
		return nil, errors.New("DONT_HAVE received from the target peer")

	case blk := <-resultChan:
		return blk, nil

	case err := <-errChan:
		return nil, fmt.Errorf("error received %s", err)
	}
}

func main() {
	ctx := context.Background()
	logging.SetLogLevel("spade-test", "DEBUG")
	logger.Debugf("starting spade-test")

	host, err := net.InitHost(ctx, nil)
	if err != nil {
		fmt.Errorf("failed to init host", err)
		return
	}

	client := NewBitswapClient(host, time.Second*1)

	cidToRetrieve, err := cid.Parse("bafybeib62b4ukyzjcj7d2h4mbzjgg7l6qiz3ma4vb4b2bawmcauf5afvua")
	// cidToRetrieve, err := cid.Parse("bafkreiarcpog7fgb3cvs4iznh6jcqtxgyyk5rbsmk4dvxuty5tylof6qea")
	if err != nil {
		log.Fatalf("unable to parse cid %s", err)
	}

	peerID, err := peer.Decode("12D3KooWNrzJ4aeavdsuxkGpErb33G7Daf2FmX8bJHx9bdE6WFzG")
	if err != nil {
		log.Fatalf("unable to decode peerID %s", err)
	}

	addrs, err := convert.StringArrayToMultiaddrs([]string{"/ip4/127.0.0.1/tcp/4001"})
	if err != nil {
		log.Fatalf("unable to convert multiaddrs %s", err)
	}

	p := peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	blk, err := client.SpadeTraversal(ctx, p, cidToRetrieve)
	if err != nil {
		log.Fatalf("unable to retrieve cid %s", err)
	}

	links, err := FindLinks(ctx, blk)

	if err != nil {
		log.Fatalf("unable to find links %s", err)
	}

	fmt.Println(links)
}
