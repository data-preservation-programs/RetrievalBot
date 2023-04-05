package common

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

type ProviderResolver struct {
	cache             *ttlcache.Cache[string, api.MinerInfo]
	lotusClient       api.FullNode
	lotusClientCloser jsonrpc.ClientCloser
}

func NewProviderResolver(url string, token string, ttl time.Duration) (*ProviderResolver, error) {
	cache := ttlcache.New[string, api.MinerInfo](
		//nolint:gomnd
		ttlcache.WithTTL[string, api.MinerInfo](ttl),
		ttlcache.WithDisableTouchOnHit[string, api.MinerInfo]())
	headers := http.Header{}
	if token != "" {
		headers.Add("Authorization", "Bearer "+token)
	}
	lotus, closer, err := client.NewFullNodeRPCV1(context.Background(), url, headers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create lotus client")
	}

	return &ProviderResolver{
		cache:             cache,
		lotusClient:       lotus,
		lotusClientCloser: closer,
	}, nil
}

func (p *ProviderResolver) Close() {
	p.lotusClientCloser()
}

func (p *ProviderResolver) ResolveProvider(ctx context.Context, provider string) (api.MinerInfo, error) {
	logger := logging.Logger("location_resolver")
	if minerInfo := p.cache.Get(provider); minerInfo != nil {
		return minerInfo.Value(), nil
	}

	addr, err := address.NewFromString(provider)
	if err != nil {
		return api.MinerInfo{}, errors.Wrap(err, "failed to parse address")
	}

	logger.With("provider", provider).Debug("Getting miner info")
	minerInfo, err := p.lotusClient.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return api.MinerInfo{}, errors.Wrap(err, "failed to get miner info")
	}

	logger.With("provider", provider).Debug("Got miner info")

	p.cache.Set(provider, minerInfo, ttlcache.DefaultTTL)

	return minerInfo, nil
}
