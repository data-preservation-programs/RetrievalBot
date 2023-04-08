package requesterror

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
)

type InvalidIPError struct {
	IP string
}

type BogonIPError struct {
	IP string
}

type HostLookupError struct {
	Host string
	Err  error
}

type CannotConnectError struct {
	PeerID peer.ID
	Err    error
}

type StreamError struct {
	Err error
}

type NoValidMultiAddrError struct {
}

func (e NoValidMultiAddrError) Error() string {
	return "no valid multiaddr"
}

func (e HostLookupError) Error() string {
	return fmt.Sprintf("failed to lookup host %s: %s", e.Host, e.Err)
}

func (e InvalidIPError) Error() string {
	return fmt.Sprintf("invalid IP: %s", e.IP)
}

func (e BogonIPError) Error() string {
	return fmt.Sprintf("bogon IP: %s", e.IP)
}

func (e CannotConnectError) Error() string {
	return fmt.Sprintf("failed to connect to peer %s: %s", e.PeerID.String(), e.Err)
}

func (e StreamError) Error() string {
	return fmt.Sprintf("failed to get supported protocols from peer: %s", e.Err)
}
