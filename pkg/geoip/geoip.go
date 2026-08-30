package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GeoInfo struct {
	CountryCode string `json:"countryCode"`
	Country     string `json:"country"`
	Query       string `json:"query"`
	ISP         string `json:"isp"`
}

var (
	cache   = make(map[string]string)
	cacheMu sync.RWMutex
	client  = &http.Client{Timeout: 3 * time.Second}
)

// GetCountry returns the 2-letter ISO country code (e.g. "IR", "DE", "US") for a given IP or domain
func GetCountry(ipOrHost string) string {
	ipOrHost = strings.TrimSpace(ipOrHost)
	if ipOrHost == "" {
		return "XX"
	}

	cacheMu.RLock()
	if c, found := cache[ipOrHost]; found {
		cacheMu.RUnlock()
		return c
	}
	cacheMu.RUnlock()

	// Query ip-api.com
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=countryCode,status", ipOrHost)
	resp, err := client.Get(apiURL)
	if err != nil {
		return "UN"
	}
	defer resp.Body.Close()

	var res struct {
		Status      string `json:"status"`
		CountryCode string `json:"countryCode"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || res.Status != "success" {
		return "UN"
	}

	country := res.CountryCode
	if country == "" {
		country = "UN"
	}

	cacheMu.Lock()
	cache[ipOrHost] = country
	cacheMu.Unlock()

	return country
}
