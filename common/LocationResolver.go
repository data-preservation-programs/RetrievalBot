package common

import (
	"context"
	"encoding/json"
	"github.com/data-preservation-programs/RetrievalBot/common/resources"
	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"net"
	"net/http"
	"strconv"
	"time"
)

type IPInfo struct {
	IP        string `json:"ip"`
	City      string `json:"city"`
	Region    string `json:"region"`
	Country   string `json:"country"`
	Continent string `json:"continent"`
	Loc       string `json:"loc"`
	Org       string `json:"org"`
	Postal    string `json:"postal"`
	Timezone  string `json:"timezone"`
	Bogon     bool   `json:"bogon"`
}

func GetPublicIPInfo(ctx context.Context, ip string, token string) (IPInfo, error) {
	logger := logging.Logger("location_resolver")
	url := "https://ipinfo.io/json"
	if ip != "" {
		url = "https://ipinfo.io/" + ip + "/json"
	}

	if token != "" {
		url = url + "?token=" + token
	}

	logger.Debugf("Getting IP info for %s", ip)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return IPInfo{}, errors.Wrap(err, "failed to create http request")
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return IPInfo{}, errors.Wrap(err, "failed to get IP info")
	}
	defer resp.Body.Close()

	var ipInfo IPInfo
	err = json.NewDecoder(resp.Body).Decode(&ipInfo)
	if err != nil {
		return IPInfo{}, errors.Wrap(err, "failed to decode IP info")
	}

	logger.Debugf("Got IP info for %s: %+v", ip, ipInfo)
	return ipInfo, nil
}

type LocationResolver struct {
	cache          *ttlcache.Cache[string, IPInfo]
	ipInfoToken    string
	countryMapping map[string]string
}

func NewLocationResolver(ipInfoToken string, ttl time.Duration) LocationResolver {
	cache := ttlcache.New[string, IPInfo](
		//nolint:gomnd
		ttlcache.WithTTL[string, IPInfo](ttl),
		ttlcache.WithDisableTouchOnHit[string, IPInfo]())
	countryMapping := make(map[string]string)
	if err := json.Unmarshal(resources.CountryToContinentJSON, &countryMapping); err != nil {
		panic(err)
	}
	return LocationResolver{
		cache,
		ipInfoToken,
		countryMapping,
	}
}

func (l LocationResolver) ResolveIP(ctx context.Context, ip net.IP) (IPInfo, error) {
	logger := logging.Logger("location_resolver")
	ipString := ip.String()
	if ipInfo := l.cache.Get(ipString); ipInfo != nil {
		return ipInfo.Value(), nil
	}

	ipInfo, err := GetPublicIPInfo(ctx, ipString, l.ipInfoToken)
	if continent, ok := l.countryMapping[ipInfo.Country]; ok {
		ipInfo.Continent = continent
	} else {
		logger.Error("Unknown country: " + ipInfo.Country)
	}

	if err != nil {
		return IPInfo{}, errors.Wrap(err, "failed to get IP info")
	}

	if ipInfo.Bogon {
		return IPInfo{}, errors.New("bogon IP address")
	}

	l.cache.Set(ipString, ipInfo, ttlcache.DefaultTTL)
	return ipInfo, nil
}

func (l LocationResolver) ResolveIPStr(ctx context.Context, ip string) (IPInfo, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return IPInfo{}, errors.Errorf("invalid IP address: %s", ip)
	}

	return l.ResolveIP(ctx, parsed)
}

func (l LocationResolver) ResolveMultiaddr(ctx context.Context, addr multiaddr.Multiaddr) (IPInfo, error) {
	host, isHostName, _, err := DecodeMultiaddr(addr)
	if err != nil {
		return IPInfo{}, errors.Wrap(err, "failed to decode multiaddr")
	}

	if isHostName {
		ips, err := net.LookupIP(host)
		if err != nil {
			return IPInfo{}, errors.Wrapf(err, "failed to lookup host %s", host)
		}

		host = ips[0].String()
	}

	return l.ResolveIPStr(ctx, host)
}

func (l LocationResolver) ResolveMultiaddrsBytes(ctx context.Context, bytesAddrs []abi.Multiaddrs) (IPInfo, error) {
	logger := logging.Logger("location_resolver")
	addrs := make([]multiaddr.Multiaddr, 0)
	for _, bytesAddr := range bytesAddrs {
		addr, err := multiaddr.NewMultiaddrBytes(bytesAddr)
		if err != nil {
			logger.With("err", err).Debugf("Failed to decode multiaddr %s", bytesAddr)
			continue
		}

		addrs = append(addrs, addr)
	}

	return l.ResolveMultiaddrs(ctx, addrs)
}

func (l LocationResolver) ResolveMultiaddrs(ctx context.Context, addrs []multiaddr.Multiaddr) (IPInfo, error) {
	logger := logging.Logger("location_resolver")
	for _, addr := range addrs {
		ipInfo, err := l.ResolveMultiaddr(ctx, addr)
		if err != nil {
			logger.With("err", err).Debugf("Failed to resolve multiaddr %s", addr)
			continue
		}

		return ipInfo, nil
	}

	return IPInfo{}, errors.New("failed to resolve any multiaddr")
}

type IsHostName = bool
type PortNumber = int
type IPOrHost = string

func DecodeMultiaddr(addr multiaddr.Multiaddr) (IPOrHost, IsHostName, PortNumber, error) {
	protocols := addr.Protocols()
	isHostName := false
	const expectedProtocolCount = 2

	if len(protocols) != expectedProtocolCount {
		return "", false, 0, errors.New("multiaddr does not contain two protocols")
	}

	if !slices.Contains(
		[]int{
			multiaddr.P_IP4, multiaddr.P_IP6,
			multiaddr.P_DNS4, multiaddr.P_DNS6,
			multiaddr.P_DNS, multiaddr.P_DNSADDR,
		}, protocols[0].Code,
	) {
		return "", false, 0, errors.New("multiaddr does not contain a valid ip or dns protocol")
	}

	if slices.Contains(
		[]int{
			multiaddr.P_DNS, multiaddr.P_DNSADDR,
			multiaddr.P_DNS4, multiaddr.P_DNS6,
		}, protocols[0].Code,
	) {
		isHostName = true
	}

	if protocols[1].Code != multiaddr.P_TCP {
		return "", false, 0, errors.New("multiaddr does not contain a valid tcp protocol")
	}

	splitted := multiaddr.Split(addr)

	component0, ok := splitted[0].(*multiaddr.Component)
	if !ok {
		return "", false, 0, errors.New("failed to cast component")
	}

	host := component0.Value()

	component1, ok := splitted[1].(*multiaddr.Component)
	if !ok {
		return "", false, 0, errors.New("failed to cast component")
	}

	port, err := strconv.Atoi(component1.Value())
	if err != nil {
		return "", false, 0, errors.Wrap(err, "failed to parse port")
	}

	return host, isHostName, port, nil
}
