package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

//go:generate go run github.com/hannahhoward/cbor-gen-for --map-encoding Protocol
type Protocol struct {
	// The name of the transport protocol eg "libp2p" or "http"
	Name string
	// The address of the endpoint in multiaddr format
	Addresses []abi.Multiaddrs
}

type ProtocolName string

const (
	GraphSync ProtocolName = "GraphSync"
	Bitswap   ProtocolName = "bitswap"
	HTTP      ProtocolName = "http"
	HTTPS     ProtocolName = "https"
	Libp2p    ProtocolName = "libp2p"
	WS        ProtocolName = "ws"
	WSS       ProtocolName = "wss"
)
