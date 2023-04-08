package net

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	bsclient "github.com/ipfs/go-libipfs/bitswap/client"
	bsnet "github.com/ipfs/go-libipfs/bitswap/network"
	logging "github.com/ipfs/go-log/v2"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

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
	network := bsnet.NewFromIpfsHost(c.host, routinghelpers.Null{})
	bswap := bsclient.New(parent, network, blockstore.NewBlockstore(datastore.NewNullDatastore()))
	network.Start(bswap)
	defer bswap.Close()
	defer network.Stop()
	connectContext, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	logger.Info("Connecting to target peer...")
	err := c.host.Connect(connectContext, target)
	if err != nil {
		logger.With("err", err).Error("Failed to connect to target peer")
		return task.NewErrorRetrievalResult(task.CannotConnect, err), nil
	}

	startTime := time.Now()
	logger.Info("Retrieving block...")
	blk, err := bswap.GetBlock(connectContext, cid)
	if err != nil {
		logger.With("err", err).Error("Failed to retrieve block")
		return task.NewErrorRetrievalResult(task.RetrievalFailure, err), nil
	}

	elapsed := time.Since(startTime)
	var size = int64(len(blk.RawData()))
	logger.With("size", size).With("elapsed", elapsed).Info("Retrieved block")
	return task.NewSuccessfulRetrievalResult(elapsed, size, elapsed), nil
}
