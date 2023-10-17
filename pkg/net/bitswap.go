package net

import (
	"context"
	"math/rand"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	bsclient "github.com/ipfs/go-libipfs/bitswap/client"
	bsmsg "github.com/ipfs/go-libipfs/bitswap/message"
	bsnet "github.com/ipfs/go-libipfs/bitswap/network"
	"github.com/ipfs/go-libipfs/blocks"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	_ "github.com/ipld/go-codec-dagpb"
	ipld "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/traversal"

	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagjson"
	_ "github.com/ipld/go-ipld-prime/codec/raw"
)

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

func (s SingleContentRouter) Provide(context.Context, cid.Cid, bool) error {
	return routing.ErrNotSupported
}

func (s SingleContentRouter) FindProvidersAsync(context.Context, cid.Cid, int) <-chan peer.AddrInfo {
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

func (c BitswapClient) Retrieve(
	parent context.Context,
	target peer.AddrInfo,
	cid cid.Cid) (*task.RetrievalResult, error) {
	logger := logging.Logger("bitswap_client").With("cid", cid).With("target", target)
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
		logger.With("err", err).Info("Failed to connect to target peer")
		return task.NewErrorRetrievalResultWithErrorResolution(task.CannotConnect, err), nil
	}

	startTime := time.Now()
	resultChan := make(chan blocks.Block)
	errChan := make(chan error)
	go func() {
		logger.Info("Retrieving block...")
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
		return task.NewErrorRetrievalResult(
			task.NotFound, errors.New("DONT_HAVE received from the target peer")), nil
	case blk := <-resultChan:
		elapsed := time.Since(startTime)
		var size = int64(len(blk.RawData()))
		logger.With("size", size).With("elapsed", elapsed).Info("Retrieved block")
		return task.NewSuccessfulRetrievalResult(elapsed, size, elapsed), nil
	case err := <-errChan:
		return task.NewErrorRetrievalResultWithErrorResolution(task.RetrievalFailure, err), nil
	}
}

// Starts with the root CID, then fetches a random CID from the children and grandchildren nodes, until it reaches `traverseDepth` or hits a node with no links
// Note: the root CID is considered depth `0`, so passing `traverseDepth=0` will only fetch the root CID
// Returns a `SuccessfulRetrievalResult` if *all* retrievals were successful, `ErrorRetrievalResult` if any failed
func (c BitswapClient) SpadeTraversal(parent context.Context, target peer.AddrInfo, startingCid cid.Cid, maxTraverseDepth uint) (*task.RetrievalResult, error) {
	logger := logging.Logger("bitswap_client_spade").With("cid", startingCid).With("target", target)
	cidToRetrieve := startingCid

	// Initialize hosts and clients required to do all the retrieval tests
	network := bsnet.NewFromIpfsHost(c.host, SingleContentRouter{
		AddrInfo: target,
	})
	bswap := bsclient.New(parent, network, blockstore.NewBlockstore(datastore.NewMapDatastore()))

	defer bswap.Close()
	defer network.Stop()

	startTime := time.Now()

	// support structures such as: https://github.com/filecoin-project/go-dagaggregator-unixfs#grouping-unixfs-structure
	i := uint(0)
	for {
		// Retrieval
		logger.Infof("retrieving %s\n", cidToRetrieve.String())
		blk, err := c.RetrieveBlock(parent, target, network, bswap, cidToRetrieve)

		// Compute the CID of the block (we can verify that it matches after this)
		// c2, err := cidToRetrieve.Prefix().Sum(blk.RawData())

		if err != nil {
			return task.NewErrorRetrievalResultWithErrorResolution(task.RetrievalFailure, err), nil
		}

		if i == maxTraverseDepth {
			var size = int64(len(blk.RawData()))
			elapsed := time.Since(startTime)
			logger.With("size", size).With("elapsed", elapsed).Info("Retrieved block")

			// we've reached the requested depth of the tree
			return task.NewSuccessfulRetrievalResult(elapsed, size, elapsed), nil
		}

		// if not at bottom of the tree, keep going down the links until we reach it or hit a dead end
		links, err := FindLinks(parent, blk)
		if err != nil {
			return task.NewErrorRetrievalResultWithErrorResolution(task.CannotDecodeLinks, err), nil
		}

		logger.Debugf("cid %s has %d links\n", cidToRetrieve.String(), len(links))

		if len(links) == 0 {
			var size = int64(len(blk.RawData()))
			elapsed := time.Since(startTime)
			logger.With("size", size).With("elapsed", elapsed).Info("Retrieved block")

			return task.NewSuccessfulRetrievalResult(elapsed, size, elapsed), nil
		}

		// randomly pick a link to go down
		rand.Seed(time.Now().UnixNano())
		nextIndex := rand.Intn(len(links))

		cidToRetrieve, err = cid.Parse(links[nextIndex].String())
		if err != nil {
			return task.NewErrorRetrievalResultWithErrorResolution(task.CIDCodecNotSupported, err), nil
		}

		i++ // To the next layer of the tree
	}
}

// Returns the raw block data, the links, and error if any
// Takes in `network` and `bswap` client, so that it can be used in a loop for traversals / multiple retrievals
func (c BitswapClient) RetrieveBlock(
	parent context.Context,
	target peer.AddrInfo,
	network bsnet.BitSwapNetwork,
	bswap *bsclient.Client,
	targetCid cid.Cid) (blocks.Block, error) {
	logger := logging.Logger("bitswap_retrieve_block").With("cid", targetCid).With("target", target)

	notFound := make(chan struct{})
	network.Start(MessageReceiver{BSClient: bswap, MessageHandler: func(
		ctx context.Context, sender peer.ID, incoming bsmsg.BitSwapMessage) {
		if sender == target.ID && slices.Contains(incoming.DontHaves(), targetCid) {
			logger.Info("Block not found")
			close(notFound)
		}
	}})
	connectContext, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	logger.Info("Connecting to target peer...")
	err := c.host.Connect(connectContext, target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to target peer")
	}

	resultChan := make(chan blocks.Block)
	errChan := make(chan error)
	go func() {
		logger.Debug("Retrieving block...")
		blk, err := bswap.GetBlock(connectContext, targetCid)
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
		return nil, errors.Wrap(err, "error received %s")
	}
}

// Attempts to decode the block data into a node and return its links
func FindLinks(ctx context.Context, blk blocks.Block) ([]datamodel.Link, error) {
	if blk.Cid().Prefix().Codec == cid.Raw {
		// Note: this case will happen at the bottom of the tree
		return []datamodel.Link{}, nil
	}

	decoder, err := cidlink.DefaultLinkSystem().DecoderChooser(cidlink.Link{Cid: blk.Cid()})

	if err != nil {
		return nil, err
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
