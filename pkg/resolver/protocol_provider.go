package resolver

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multistream"
	"golang.org/x/exp/slices"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

const RetrievalProtocolName = "/fil/retrieval/transports/1.0.0"

type ProtocolProvider struct {
	host    host.Host
	timeout time.Duration
}

func NewProtocolProvider(host host.Host, timeout time.Duration) ProtocolProvider {
	return ProtocolProvider{
		host:    host,
		timeout: timeout,
	}
}

func (p ProtocolProvider) IsBoostProvider(ctx context.Context, minerInfo peer.AddrInfo) (bool, error) {
	protocols, err := p.getLibp2pProtocols(ctx, minerInfo)
	if err != nil {
		return false, errors.Wrap(err, "failed to get libp2p protocols")
	}

	return slices.Contains(protocols, RetrievalProtocolName), nil
}

func (p ProtocolProvider) getLibp2pProtocols(
	parent context.Context,
	minerInfo peer.AddrInfo) ([]protocol.ID, error) {
	ctx, cancel := context.WithTimeout(parent, p.timeout)
	defer cancel()
	p.host.Peerstore().AddAddrs(minerInfo.ID, minerInfo.Addrs, peerstore.PermanentAddrTTL)
	if err := p.host.Connect(ctx, minerInfo); err != nil {
		return nil, &requesterror.CannotConnectError{
			PeerID: minerInfo.ID,
			Err:    err,
		}
	}

	protocols, err := p.host.Peerstore().GetProtocols(minerInfo.ID)
	if err != nil {
		return nil, &requesterror.StreamError{
			Err: err,
		}
	}

	return protocols, nil
}

func (p ProtocolProvider) GetRetrievalProtocols(
	parent context.Context,
	minerInfo peer.AddrInfo,
) ([]model.Protocol, error) {
	ctx, cancel := context.WithTimeout(parent, p.timeout)
	defer cancel()

	if err := p.host.Connect(ctx, minerInfo); err != nil {
		return nil, &requesterror.CannotConnectError{
			PeerID: minerInfo.ID,
			Err:    err,
		}
	}

	stream, err := p.host.NewStream(ctx, minerInfo.ID, RetrievalProtocolName)
	if errors.Is(err, multistream.ErrNotSupported[protocol.ID]{}) {
		addrs := make([]abi.Multiaddrs, len(minerInfo.Addrs))
		for i, addr := range minerInfo.Addrs {
			addrs[i] = addr.Bytes()
		}
		return []model.Protocol{
			{
				Name:      "libp2p",
				Addresses: addrs,
			},
		}, nil
	}

	if err != nil {
		return nil, &requesterror.StreamError{
			Err: err,
		}
	}

	//nolint: errcheck
	defer stream.Close()

	_ = stream.SetReadDeadline(time.Now().Add(p.timeout))
	//nolint: errcheck
	defer stream.SetReadDeadline(time.Time{})

	queryResponse := new(model.QueryResponse)
	err = cborutil.ReadCborRPC(stream, queryResponse)
	if err != nil {
		return nil, &requesterror.StreamError{
			Err: err,
		}
	}

	return queryResponse.Protocols, nil
}
