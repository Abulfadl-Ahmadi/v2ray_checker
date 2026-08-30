package parser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type ParsedConfig struct {
	Protocol    string `json:"protocol"`
	Server      string `json:"server"`
	Port        int    `json:"port"`
	UUID        string `json:"uuid,omitempty"`
	Password    string `json:"password,omitempty"`
	Security    string `json:"security,omitempty"`
	Network     string `json:"network,omitempty"`
	Path        string `json:"path,omitempty"`
	Host        string `json:"host,omitempty"`
	SNI         string `json:"sni,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
	ShortID     string `json:"short_id,omitempty"`
	SpiderX     string `json:"spider_x,omitempty"`
	RawLink     string `json:"raw_link"`
	Hash        string `json:"hash"`
}

func Parse(rawLink string) (*ParsedConfig, error) {
	rawLink = strings.TrimSpace(rawLink)
	if strings.HasPrefix(rawLink, "vless://") {
		return parseVless(rawLink)
	} else if strings.HasPrefix(rawLink, "vmess://") {
		return parseVmess(rawLink)
	} else if strings.HasPrefix(rawLink, "trojan://") {
		return parseTrojan(rawLink)
	} else if strings.HasPrefix(rawLink, "ss://") {
		return parseShadowsocks(rawLink)
	}
	return nil, fmt.Errorf("unsupported protocol scheme: %s", rawLink)
}

func GenerateHash(proto, server string, port int, id string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", proto, server, port, id)))
	return hex.EncodeToString(sum[:16])
}

func parseVless(raw string) (*ParsedConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()

	p := &ParsedConfig{
		Protocol:    "vless",
		Server:      u.Hostname(),
		Port:        port,
		UUID:        u.User.Username(),
		Security:    q.Get("security"),
		Network:     q.Get("type"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		SNI:         q.Get("sni"),
		Fingerprint: q.Get("fp"),
		PublicKey:   q.Get("pbk"),
		ShortID:     q.Get("sid"),
		SpiderX:     q.Get("spx"),
		RawLink:     raw,
	}
	if p.Network == "" {
		p.Network = "tcp"
	}
	p.Hash = GenerateHash("vless", p.Server, p.Port, p.UUID)
	return p, nil
}

func parseTrojan(raw string) (*ParsedConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()

	p := &ParsedConfig{
		Protocol:    "trojan",
		Server:      u.Hostname(),
		Port:        port,
		Password:    u.User.Username(),
		Security:    q.Get("security"),
		Network:     q.Get("type"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		SNI:         q.Get("sni"),
		Fingerprint: q.Get("fp"),
		RawLink:     raw,
	}
	if p.Security == "" {
		p.Security = "tls"
	}
	p.Hash = GenerateHash("trojan", p.Server, p.Port, p.Password)
	return p, nil
}

type vmessJSON struct {
	Add  string      `json:"add"`
	Port any         `json:"port"`
	ID   string      `json:"id"`
	Net  string      `json:"net"`
	Type string      `json:"type"`
	Host string      `json:"host"`
	Path string      `json:"path"`
	TLS  string      `json:"tls"`
	SNI  string      `json:"sni"`
	Fp   string      `json:"fp"`
	Scy  string      `json:"scy"`
}

func parseVmess(raw string) (*ParsedConfig, error) {
	b64 := strings.TrimPrefix(raw, "vmess://")
	// Clean standard / URL base64 padding
	b64 = strings.TrimSpace(b64)
	if idx := strings.Index(b64, "#"); idx != -1 {
		b64 = b64[:idx]
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(b64)
		if err != nil {
			decoded, err = base64.URLEncoding.DecodeString(b64)
			if err != nil {
				return nil, fmt.Errorf("failed to decode vmess base64: %w", err)
			}
		}
	}

	var v vmessJSON
	if err := json.Unmarshal(decoded, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vmess json: %w", err)
	}

	var port int
	switch p := v.Port.(type) {
	case float64:
		port = int(p)
	case string:
		port, _ = strconv.Atoi(p)
	}

	p := &ParsedConfig{
		Protocol:    "vmess",
		Server:      v.Add,
		Port:        port,
		UUID:        v.ID,
		Security:    v.TLS,
		Network:     v.Net,
		Path:        v.Path,
		Host:        v.Host,
		SNI:         v.SNI,
		Fingerprint: v.Fp,
		RawLink:     raw,
	}
	p.Hash = GenerateHash("vmess", p.Server, p.Port, p.UUID)
	return p, nil
}

func parseShadowsocks(raw string) (*ParsedConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	auth := u.User.String()

	// If encoded in base64
	if server == "" && u.Host != "" {
		rawB64 := u.Host
		if dec, err := base64.RawURLEncoding.DecodeString(rawB64); err == nil {
			parts := strings.Split(string(dec), "@")
			if len(parts) == 2 {
				auth = parts[0]
				hostPort := strings.Split(parts[1], ":")
				if len(hostPort) == 2 {
					server = hostPort[0]
					port, _ = strconv.Atoi(hostPort[1])
				}
			}
		}
	}

	p := &ParsedConfig{
		Protocol: "ss",
		Server:   server,
		Port:     port,
		Password: auth,
		RawLink:  raw,
	}
	p.Hash = GenerateHash("ss", p.Server, p.Port, auth)
	return p, nil
}
