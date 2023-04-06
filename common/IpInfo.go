package common

import (
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
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
}

func GetPublicIPInfo() (IPInfo, error) {
	//nolint noctx
	resp, err := http.Get("https://ipinfo.io/json")
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
