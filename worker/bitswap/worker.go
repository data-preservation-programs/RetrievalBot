package bitswap

import (
	"context"
	"strconv"

	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/net"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

type Worker struct{}

var logger = logging.Logger("bitswap_worker")

func (e Worker) DoWork(tsk task.Task) (*task.RetrievalResult, error) {
	ctx := context.Background()

	host, err := net.InitHost(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init host")
	}

	client := net.NewBitswapClient(host, tsk.Timeout)

	// First, check if the provider is using boost
	protocolProvider := resolver.ProtocolResolver(host, tsk.Timeout)
	addrInfo, err := tsk.Provider.GetPeerAddr()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get peer addr")
	}
	contentCID := cid.MustParse(tsk.Content.CID)
	isBoost, err := protocolProvider.IsBoostProvider(context.Background(), addrInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if provider is boost")
	}

	if !isBoost {
		return task.NewErrorRetrievalResult(
			task.ProtocolNotSupported,
			errors.New("Provider is not using boost")), nil
	}

	// If so, find the Bitswap endpoint
	protocols, err := protocolProvider.GetRetrievalProtocols(ctx, addrInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get retrieval protocols")
	}

	var peerID peer.ID
	addrs := make([]multiaddr.Multiaddr, 0)
	for _, protocol := range protocols {
		if protocol.Name != string(model.Bitswap) {
			continue
		}
		addr, err := convert.AbiToMultiaddr(protocol.Addresses[0])
		if err != nil {
			return task.NewErrorRetrievalResult(task.ProtocolNotSupported, err), nil
		}

		remain, last := multiaddr.SplitLast(addr)
		if last.Protocol().Code == multiaddr.P_P2P {
			newPeerID, err := peer.IDFromBytes(last.RawValue())
			if err != nil {
				return task.NewErrorRetrievalResult(task.ProtocolNotSupported, err), nil
			}
			if peerID == "" || peerID == newPeerID {
				peerID = newPeerID
				addrs = append(addrs, remain)
			} else {
				logger.With("name", protocol.Name, "addr", addr.String()).Warn("Found multiple peer IDs for Bitswap")
			}
		}
	}

	if peerID == "" || len(addrs) == 0 {
		return task.NewErrorRetrievalResult(task.ProtocolNotSupported, errors.New("No bitswap multiaddr available")), nil
	}

	if tsk.Metadata["retrieve_type"] == string(task.Spade) {
		depth, err := strconv.ParseUint(tsk.Metadata["max_traverse_depth"], 10, 32)

		if err != nil {
			return nil, errors.Wrap(err, "failed to parse max_traverse_depth")
		}

		return client.SpadeTraversal(ctx, peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		}, contentCID, uint(depth),
		)
	}

	//nolint:wrapcheck
	return client.Retrieve(ctx, peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}, contentCID)
}
