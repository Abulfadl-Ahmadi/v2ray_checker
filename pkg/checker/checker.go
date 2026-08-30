package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"v2ray_checker/pkg/parser"
)

type CheckResult struct {
	IsAlive  bool
	PingMs   int64
	ServerIP string
	Country  string
	Error    string
}

type Checker struct {
	probeURLs  []string
	timeoutSec int
}

func NewChecker(probeURLs []string, timeoutSec int) *Checker {
	if len(probeURLs) == 0 {
		probeURLs = []string{"https://1.1.1.1/cdn-cgi/trace"}
	}
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	return &Checker{
		probeURLs:  probeURLs,
		timeoutSec: timeoutSec,
	}
}

// TestNode performs a fast and robust real-latency handshake test on the proxy node
func (c *Checker) TestNode(cfg *parser.ParsedConfig) CheckResult {
	res := CheckResult{}
	if cfg.Server == "" || cfg.Port == 0 {
		res.Error = "invalid server address or port"
		return res
	}

	// 1. Resolve Server IP
	ips, err := net.LookupIP(cfg.Server)
	if err == nil && len(ips) > 0 {
		res.ServerIP = ips[0].String()
	} else {
		res.ServerIP = cfg.Server
	}

	// 2. Perform connection & Handshake probe
	targetAddr := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	timeout := time.Duration(c.timeoutSec) * time.Second

	start := time.Now()

	// If TLS / Reality is specified, execute full TLS handshake
	if strings.Contains(cfg.Security, "tls") || strings.Contains(cfg.Security, "reality") {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		if cfg.SNI != "" {
			tlsConfig.ServerName = cfg.SNI
		} else if cfg.Host != "" {
			tlsConfig.ServerName = cfg.Host
		} else {
			tlsConfig.ServerName = cfg.Server
		}

		dialer := &net.Dialer{Timeout: timeout}
		conn, err := tls.DialWithDialer(dialer, "tcp", targetAddr, tlsConfig)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		conn.Close()
	} else {
		// Plain TCP Handshake
		conn, err := net.DialTimeout("tcp", targetAddr, timeout)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		conn.Close()
	}

	elapsed := time.Since(start).Milliseconds()
	if elapsed == 0 {
		elapsed = 1
	}

	res.IsAlive = true
	res.PingMs = elapsed
	return res
}

// PingWithHTTPContext checks end-to-end HTTP connectivity via a local socks/http proxy if active
func PingWithHTTPContext(ctx context.Context, proxyURL, probeURL string) (int64, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return 0, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(u),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   6 * time.Second,
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", probeURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return time.Since(start).Milliseconds(), nil
	}
	return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
