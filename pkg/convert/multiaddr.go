package convert

import (
	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

func MultiaddrToAbi(addr multiaddr.Multiaddr) abi.Multiaddrs {
	return addr.Bytes()
}

func AbiToMultiaddr(addr abi.Multiaddrs) (multiaddr.Multiaddr, error) {
	//nolint:wrapcheck
	return multiaddr.NewMultiaddrBytes(addr)
}

func MultiaddrsToAbi(addrs []multiaddr.Multiaddr) []abi.Multiaddrs {
	abiAddrs := make([]abi.Multiaddrs, len(addrs))
	for i, addr := range addrs {
		abiAddrs[i] = MultiaddrToAbi(addr)
	}
	return abiAddrs
}

func AbiToMultiaddrs(addrs []abi.Multiaddrs) ([]multiaddr.Multiaddr, error) {
	multiAddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		multiAddr, err := AbiToMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		multiAddrs[i] = multiAddr
	}
	return multiAddrs, nil
}

func AbiToMultiaddrsSkippingError(addrs []abi.Multiaddrs) []multiaddr.Multiaddr {
	multiAddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		multiAddr, err := AbiToMultiaddr(addr)
		if err != nil {
			logging.Logger("convert").With("err", err, "addr", addr).Debug("Failed to decode multiaddr")
			continue
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}
	return multiAddrs
}

func MultiaddrsBytesToStringArraySkippingError(addrs []abi.Multiaddrs) []string {
	maddrs := AbiToMultiaddrsSkippingError(addrs)
	strs := make([]string, len(maddrs))
	for i, maddr := range maddrs {
		strs[i] = maddr.String()
	}
	return strs
}

func StringArrayToMultiaddrsSkippingError(addrs []string) []multiaddr.Multiaddr {
	maddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			logging.Logger("convert").With("err", err, "addr", addr).Debug("Failed to decode multiaddr")
			continue
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs
}

func StringArrayToMultiaddrs(addrs []string) ([]multiaddr.Multiaddr, error) {
	maddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode multiaddr")
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs, nil
}
