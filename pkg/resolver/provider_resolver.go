package resolver

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc/v3"
)

type ProviderResolver struct {
	cache       *ttlcache.Cache[string, MinerInfo]
	lotusClient jsonrpc.RPCClient
}

type MinerInfo struct {
	//nolint:stylecheck
	PeerId string
	//nolint:tagliatelle
	MultiaddrsBase64Encoded []string `json:"Multiaddrs"`
	Multiaddrs              []abi.Multiaddrs
}

func NewProviderResolver(url string, token string, ttl time.Duration) (*ProviderResolver, error) {
	cache := ttlcache.New[string, MinerInfo](
		//nolint:gomnd
		ttlcache.WithTTL[string, MinerInfo](ttl),
		ttlcache.WithDisableTouchOnHit[string, MinerInfo]())
	var lotusClient jsonrpc.RPCClient
	if token == "" {
		lotusClient = jsonrpc.NewClient(url)
	} else {
		lotusClient = jsonrpc.NewClientWithOpts(url, &jsonrpc.RPCClientOpts{
			CustomHeaders: map[string]string{
				"Authorization": "Bearer " + token,
			},
		})
	}
	return &ProviderResolver{
		cache:       cache,
		lotusClient: lotusClient,
	}, nil
}

func (p *ProviderResolver) ResolveProvider(ctx context.Context, provider string) (MinerInfo, error) {
	logger := logging.Logger("location_resolver")
	if minerInfo := p.cache.Get(provider); minerInfo != nil && !minerInfo.IsExpired() {
		return minerInfo.Value(), nil
	}

	logger.With("provider", provider).Debug("Getting miner info")
	minerInfo := new(MinerInfo)
	err := p.lotusClient.CallFor(ctx, minerInfo, "Filecoin.StateMinerInfo", provider, nil)
	if err != nil {
		return MinerInfo{}, errors.Wrap(err, "failed to get miner info")
	}

	logger.With("provider", provider, "minerinfo", minerInfo).Debug("Got miner info")
	minerInfo.Multiaddrs = make([]abi.Multiaddrs, len(minerInfo.MultiaddrsBase64Encoded))
	for i, multiaddr := range minerInfo.MultiaddrsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(multiaddr)
		if err != nil {
			return MinerInfo{}, errors.Wrap(err, "failed to decode multiaddr")
		}
		minerInfo.Multiaddrs[i] = decoded
	}
	p.cache.Set(provider, *minerInfo, ttlcache.DefaultTTL)

	return *minerInfo, nil
}
