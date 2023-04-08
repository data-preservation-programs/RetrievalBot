package task

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"time"
)

type Provider struct {
	// In case of attempting retrieval from any miner, this field will be empty
	ID         string   `bson:"id"`
	PeerID     string   `bson:"peer_id,omitempty"`
	Multiaddrs []string `bson:"multiaddrs,omitempty"`
	City       string   `bson:"city,omitempty"`
	Region     string   `bson:"region,omitempty"`
	Country    string   `bson:"country,omitempty"`
	Continent  string   `bson:"continent,omitempty"`
}

func (p Provider) GetPeerAddr() (peer.AddrInfo, error) {
	peerID, err := peer.Decode(p.PeerID)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrap(err, "failed to decode peer id")
	}

	addrs, err := convert.StringArrayToMultiaddrs(p.Multiaddrs)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrap(err, "failed to convert multiaddrs")
	}

	return peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}, nil
}

type ModuleName string

const (
	Stub      ModuleName = "stub"
	GraphSync ModuleName = "graphsync"
	HTTP      ModuleName = "http"
	Bitswap   ModuleName = "bitswap"
)

type Content struct {
	CID string `bson:"cid"`
}

type Task struct {
	Requester string            `bson:"requester"`
	Module    ModuleName        `bson:"module"`
	Metadata  map[string]string `bson:"metadata,omitempty"`
	Provider  Provider          `bson:"provider"`
	Content   Content           `bson:"content"`
	Timeout   time.Duration     `bson:"timeout,omitempty"`
	CreatedAt time.Time         `bson:"created_at"`
}
