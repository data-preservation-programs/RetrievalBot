package http

import (
	"context"
	"fmt"
	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/net"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	net2 "net"
	"net/url"
	"strconv"
)

type Worker struct{}

func ToURL(ma multiaddr.Multiaddr) (*url.URL, error) {
	// host should be either the dns name or the IP
	_, host, err := manet.DialArgs(ma)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dial args")
	}
	if ip := net2.ParseIP(host); ip != nil {
		if !ip.To4().Equal(ip) {
			// raw v6 IPs need `[ip]` encapsulation.
			host = fmt.Sprintf("[%s]", host)
		}
	}

	protos := ma.Protocols()
	pm := make(map[int]string, len(protos))
	for _, p := range protos {
		v, err := ma.ValueForProtocol(p.Code)
		if err == nil {
			pm[p.Code] = v
		}
	}

	scheme := model.HTTP
	//nolint:nestif
	if _, ok := pm[multiaddr.P_HTTPS]; ok {
		scheme = model.HTTPS
	} else if _, ok = pm[multiaddr.P_HTTP]; ok {
		// /tls/http == /https
		if _, ok = pm[multiaddr.P_TLS]; ok {
			scheme = model.HTTPS
		}
	} else if _, ok = pm[multiaddr.P_WSS]; ok {
		scheme = model.WSS
	} else if _, ok = pm[multiaddr.P_WS]; ok {
		scheme = model.WS
		// /tls/ws == /wss
		if _, ok = pm[multiaddr.P_TLS]; ok {
			scheme = model.WSS
		}
	}

	path := ""
	if pb, ok := pm[0x300200]; ok {
		path, err = url.PathUnescape(pb)
		if err != nil {
			path = ""
		}
	}

	//nolint:exhaustruct
	out := url.URL{
		Scheme: string(scheme),
		Host:   host,
		Path:   path,
	}
	return &out, nil
}

func (e Worker) DoWork(tsk task.Task) (*task.RetrievalResult, error) {
	ctx := context.Background()
	client := net.NewHTTPClient(tsk.Timeout)

	host, err := net.InitHost(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init host")
	}

	// First, check if the provider is using boost
	protocolProvider := resolver.NewProtocolProvider(host, tsk.Timeout)
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

	// If so, find the HTTP endpoint
	protocols, err := protocolProvider.GetRetrievalProtocols(ctx, addrInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get retrieval protocols")
	}

	var urlString string
	for _, protocol := range protocols {
		if (protocol.Name == string(model.HTTP) || protocol.Name == string(model.HTTPS)) && len(protocol.Addresses) > 0 {
			addr, err := convert.AbiToMultiaddr(protocol.Addresses[0])
			if err != nil {
				return task.NewErrorRetrievalResult(task.ProtocolNotSupported, err), nil
			}

			url, err := ToURL(addr)
			if err != nil {
				return task.NewErrorRetrievalResult(
					task.ProtocolNotSupported,
					errors.Wrap(err, "Cannot convert multiaddr to URL")), nil
			}

			urlString = url.String()
		}
	}

	if urlString == "" {
		return task.NewErrorRetrievalResult(
			task.ProtocolNotSupported,
			errors.New("No HTTP endpoint found")), nil
	}

	size := 1024 * 1024
	if sizeStr, ok := tsk.Metadata["retrieve_size"]; ok {
		size, err = strconv.Atoi(sizeStr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert retrieve_size to int")
		}
	}

	// Finally, retrieve the file
	//nolint:wrapcheck
	return client.RetrievePiece(ctx, urlString, contentCID, int64(size))
}
