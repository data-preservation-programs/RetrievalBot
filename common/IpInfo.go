package common

import (
	"context"
	"encoding/json"
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
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Bogon    bool   `json:"bogon"`
}

func GetPublicIPInfo(ctx context.Context, ip string, token string) (IPInfo, error) {
	url := "https://ipinfo.io/json"
	if ip != "" {
		url = "https://ipinfo.io/" + ip + "/json"
	}

	if token != "" {
		url = url + "?token=" + token
	}

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

	return ipInfo, nil
}

type LocationResolver struct {
	cache       *ttlcache.Cache[string, IPInfo]
	ipInfoToken string
}

func NewLocationResolver(ipInfoToken string) LocationResolver {
	cache := ttlcache.New[string, IPInfo](
		ttlcache.WithTTL[string, IPInfo](time.Hour*24),
		ttlcache.WithDisableTouchOnHit[string, IPInfo]())
	return LocationResolver{
		cache,
		ipInfoToken,
	}
}

func (l LocationResolver) ResolveIP(ctx context.Context, ip net.IP) (IPInfo, error) {
	ipString := ip.String()
	return GetPublicIPInfo(ctx, ipString, l.ipInfoToken)
}

func (l LocationResolver) ResolveIPStr(ctx context.Context, ip string) (IPInfo, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return IPInfo{}, errors.Errorf("invalid IP address: %s", ip)
	}

	return GetPublicIPInfo(ctx, ip, l.ipInfoToken)
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

type IsHostName = bool
type PortNumber = int
type IpOrHost = string

func DecodeMultiaddr(addr multiaddr.Multiaddr) (IpOrHost, IsHostName, PortNumber, error) {
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
