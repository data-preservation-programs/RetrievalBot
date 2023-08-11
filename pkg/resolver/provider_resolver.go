package resolver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc/v3"
)

type ProviderResolver struct {
	localCache  *ttlcache.Cache[string, MinerInfo]
	lotusClient jsonrpc.RPCClient
	remoteTTL   int
}

type MinerInfo struct {
	//nolint:stylecheck
	PeerId string
	//nolint:tagliatelle
	MultiaddrsBase64Encoded []string `json:"Multiaddrs"`
	Multiaddrs              []abi.Multiaddrs
}

type ProviderCachePayload struct {
	Provider        string `json:"provider"`
	ProviderPayload []byte `json:"providerPayload"`
	TTL             int    `json:"ttl"`
}

func NewProviderResolver(url string, token string, localTTL time.Duration, remoteTTL int) (*ProviderResolver, error) {
	localCache := ttlcache.New[string, MinerInfo](
		//nolint:gomnd
		ttlcache.WithTTL[string, MinerInfo](localTTL),
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
		localCache:  localCache,
		lotusClient: lotusClient,
		remoteTTL:   remoteTTL,
	}, nil
}

func (p *ProviderResolver) ResolveProvider(ctx context.Context, provider string) (MinerInfo, error) {
	logger := logging.Logger("location_resolver")

	if minerInfo := p.localCache.Get(provider); minerInfo != nil && !minerInfo.IsExpired() {
		return minerInfo.Value(), nil
	}

	if os.Getenv("PROVIDER_CACHE_URL") != "" {
		response, _ := http.NewRequest(http.MethodGet,
			os.Getenv("PROVIDER_CACHE_URL")+"/getProviderInfo?provider="+provider, nil)

		var minerInfo MinerInfo
		if response.Response.StatusCode == http.StatusOK && response.Body != nil {
			_ = json.NewDecoder(response.Body).Decode(&minerInfo)
			return minerInfo, nil
		}
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

	minerJSON, err := json.Marshal(minerInfo)
	if err != nil {
		return MinerInfo{}, errors.Wrap(err, "failed to get IP info")
	}

	p.localCache.Set(provider, *minerInfo, ttlcache.DefaultTTL)

	if os.Getenv("PROVIDER_CACHE_URL") != "" {
		requestBody, err := json.Marshal(LocationCachePayload{provider, minerJSON, p.remoteTTL})
		if err != nil {
			return MinerInfo{}, errors.Wrap(err, "Could not serialize MinerInfo")
		}

		_, _ = http.NewRequest(
			http.MethodPost,
			os.Getenv("PROVIDER_CACHE_URL"),
			bytes.NewReader(requestBody))
	}

	return *minerInfo, nil
}
