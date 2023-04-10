package net

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	datatransfer "github.com/filecoin-project/go-data-transfer/v2"
	retrievaltypes "github.com/filecoin-project/go-retrieval-types"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lassie/pkg/net/client"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log/v2"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/storage/memstore"
	selectorparse "github.com/ipld/go-ipld-prime/traversal/selector/parse"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

type GraphsyncClient struct {
	host    host.Host
	timeout time.Duration
	counter *TimeCounter
}

func NewGraphsyncClient(host host.Host, timeout time.Duration) GraphsyncClient {
	return GraphsyncClient{
		host:    host,
		timeout: timeout,
		counter: NewTimeCounter(),
	}
}

type TimeCounter struct {
	counter uint64
}

func NewTimeCounter() *TimeCounter {
	return &TimeCounter{counter: uint64(time.Now().UnixNano())}
}

func (tc *TimeCounter) Next() uint64 {
	counter := atomic.AddUint64(&tc.counter, 1)
	return counter
}

func (c GraphsyncClient) Retrieve(
	parent context.Context,
	target peer.AddrInfo,
	cid cid.Cid) (*task.RetrievalResult, error) {
	logger := logging.Logger("graphsync_client").With("cid", cid, "target", target)
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	datastore := sync.MutexWrap(datastore.NewMapDatastore())
	retrievalClient, err := client.NewClient(ctx, datastore, c.host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create graphsync retrieval client")
	}
	if err := retrievalClient.AwaitReady(); err != nil {
		return nil, errors.Wrap(err, "failed to wait for graphsync retrieval client to be ready")
	}
	err = retrievalClient.Connect(ctx, target)
	if err != nil {
		return task.NewErrorRetrievalResultWithErrorResolution(task.CannotConnect, err), nil
	}

	shutDown := make(chan struct{})
	go func() {
		time.Sleep(c.timeout)
		shutDown <- struct{}{}
	}()

	selector := selectorparse.CommonSelector_MatchPoint
	params, err := retrievaltypes.NewParamsV1(big.Zero(), 0, 0, selector, nil, big.Zero())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create retrieval params")
	}

	linkSystem := cidlink.DefaultLinkSystem()
	storage := &memstore.Store{}
	linkSystem.SetWriteStorage(storage)
	linkSystem.SetReadStorage(storage)

	stats, err := retrievalClient.RetrieveFromPeer(
		ctx,
		linkSystem,
		target.ID,
		&retrievaltypes.DealProposal{
			PayloadCID: cid,
			ID:         retrievaltypes.DealID(c.counter.Next()),
			Params:     params,
		},
		selector,
		func(event datatransfer.Event, channelState datatransfer.ChannelState) {
			logger.With("event", event, "channelState", channelState).Debug("received data transfer event")
		},
		shutDown,
	)

	if err != nil {
		logger.Info(err)
		return task.NewErrorRetrievalResultWithErrorResolution(task.RetrievalFailure, err), nil
	}

	return task.NewSuccessfulRetrievalResult(stats.TimeToFirstByte, int64(stats.Size), stats.Duration), nil
}
