// Package tomlcfg models the parts of the frp configuration the panel edits and
// converts to/from the on-disk frps.toml / frpc.toml — which remain the single
// source of truth that frp itself reads.
package tomlcfg

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Auth is the [auth] table shared by client and server.
type Auth struct {
	Method string `toml:"method,omitempty" json:"method,omitempty"`
	Token  string `toml:"token,omitempty" json:"token,omitempty"`
}

// WebServer is the [webServer] table that powers the frps dashboard / frpc admin
// API. The panel relies on it for monitoring and hot reload.
type WebServer struct {
	Addr     string `toml:"addr,omitempty" json:"addr,omitempty"`
	Port     int    `toml:"port,omitempty" json:"port,omitempty"`
	User     string `toml:"user,omitempty" json:"user,omitempty"`
	Password string `toml:"password,omitempty" json:"password,omitempty"`
}

// Log is the [log] table.
type Log struct {
	To      string `toml:"to,omitempty" json:"to,omitempty"`
	Level   string `toml:"level,omitempty" json:"level,omitempty"`
	MaxDays int    `toml:"maxDays,omitempty" json:"maxDays,omitempty"`
}

// TLS is transport.tls. Enable is a pointer so the panel can express the tri-state
// of "unset / true / false" (frp defaults to enabled).
type TLS struct {
	Enable *bool `toml:"enable,omitempty" json:"enable,omitempty"`
}

// ClientTransport is the client's [transport] table.
type ClientTransport struct {
	Protocol string `toml:"protocol,omitempty" json:"protocol,omitempty"` // tcp|kcp|quic|websocket|wss
	TLS      *TLS   `toml:"tls,omitempty" json:"tls,omitempty"`
}

// ServerConfig models frps.toml.
type ServerConfig struct {
	BindAddr       string     `toml:"bindAddr,omitempty" json:"bindAddr,omitempty"`
	BindPort       int        `toml:"bindPort" json:"bindPort"`
	KCPBindPort    int        `toml:"kcpBindPort,omitempty" json:"kcpBindPort,omitempty"`
	QUICBindPort   int        `toml:"quicBindPort,omitempty" json:"quicBindPort,omitempty"`
	VhostHTTPPort  int        `toml:"vhostHTTPPort,omitempty" json:"vhostHTTPPort,omitempty"`
	VhostHTTPSPort int        `toml:"vhostHTTPSPort,omitempty" json:"vhostHTTPSPort,omitempty"`
	SubDomainHost  string     `toml:"subDomainHost,omitempty" json:"subDomainHost,omitempty"`
	Auth           *Auth      `toml:"auth,omitempty" json:"auth,omitempty"`
	WebServer      *WebServer `toml:"webServer,omitempty" json:"webServer,omitempty"`
	Log            *Log       `toml:"log,omitempty" json:"log,omitempty"`
}

// Proxy models one [[proxies]] entry in frpc.toml. Fields are a superset across
// proxy types; only the relevant ones are populated per type.
type Proxy struct {
	Name          string   `toml:"name" json:"name"`
	Type          string   `toml:"type" json:"type"` // tcp|udp|http|https|stcp|sudp|xtcp|tcpmux
	LocalIP       string   `toml:"localIP,omitempty" json:"localIP,omitempty"`
	LocalPort     int      `toml:"localPort,omitempty" json:"localPort,omitempty"`
	RemotePort    int      `toml:"remotePort,omitempty" json:"remotePort,omitempty"`
	CustomDomains []string `toml:"customDomains,omitempty" json:"customDomains,omitempty"`
	Subdomain     string   `toml:"subdomain,omitempty" json:"subdomain,omitempty"`
	Locations     []string `toml:"locations,omitempty" json:"locations,omitempty"`
	SecretKey     string   `toml:"secretKey,omitempty" json:"secretKey,omitempty"`
	ServerName    string   `toml:"serverName,omitempty" json:"serverName,omitempty"` // for visitors of stcp/xtcp
}

// ClientConfig models frpc.toml.
type ClientConfig struct {
	ServerAddr string           `toml:"serverAddr" json:"serverAddr"`
	ServerPort int              `toml:"serverPort" json:"serverPort"`
	Auth       *Auth            `toml:"auth,omitempty" json:"auth,omitempty"`
	WebServer  *WebServer       `toml:"webServer,omitempty" json:"webServer,omitempty"`
	Log        *Log             `toml:"log,omitempty" json:"log,omitempty"`
	Transport  *ClientTransport `toml:"transport,omitempty" json:"transport,omitempty"`
	Proxies    []Proxy          `toml:"proxies,omitempty" json:"proxies,omitempty"`
}

// --- encode / decode ---

// EncodeServer renders a ServerConfig to TOML bytes.
func EncodeServer(c *ServerConfig) ([]byte, error) { return marshal(c) }

// EncodeClient renders a ClientConfig to TOML bytes.
func EncodeClient(c *ClientConfig) ([]byte, error) { return marshal(c) }

func marshal(v any) ([]byte, error) {
	var sb strings.Builder
	enc := toml.NewEncoder(&sb)
	enc.SetIndentTables(true)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("生成 TOML 失败: %w", err)
	}
	return []byte(sb.String()), nil
}

// DecodeServer parses frps.toml into a ServerConfig.
func DecodeServer(data []byte) (*ServerConfig, error) {
	var c ServerConfig
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析 frps.toml 失败: %w", err)
	}
	return &c, nil
}

// DecodeClient parses frpc.toml into a ClientConfig.
func DecodeClient(data []byte) (*ClientConfig, error) {
	var c ClientConfig
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析 frpc.toml 失败: %w", err)
	}
	return &c, nil
}

// --- defaults ---

// DefaultServer returns a sensible starting server config with random secrets.
func DefaultServer() *ServerConfig {
	return &ServerConfig{
		BindPort: 7000,
		Auth:     &Auth{Method: "token", Token: RandomHex(16)},
		WebServer: &WebServer{
			Addr:     "0.0.0.0",
			Port:     7500,
			User:     "admin",
			Password: RandomHex(8),
		},
		Log: &Log{To: "console", Level: "info", MaxDays: 3},
	}
}

// DefaultClient returns a sensible starting client config with a random admin
// password for the local admin API.
func DefaultClient() *ClientConfig {
	return &ClientConfig{
		ServerAddr: "127.0.0.1",
		ServerPort: 7000,
		Auth:       &Auth{Method: "token"},
		WebServer: &WebServer{
			Addr:     "127.0.0.1",
			Port:     7400,
			User:     "admin",
			Password: RandomHex(8),
		},
		Log:     &Log{To: "console", Level: "info", MaxDays: 3},
		Proxies: []Proxy{},
	}
}

// RandomHex returns 2n hex chars of cryptographically random data.
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "changeme"
	}
	return hex.EncodeToString(b)
}

// --- validation ---

var validProxyTypes = map[string]bool{
	"tcp": true, "udp": true, "http": true, "https": true,
	"stcp": true, "sudp": true, "xtcp": true, "tcpmux": true,
}

// Validate checks a server config for obvious mistakes.
func (c *ServerConfig) Validate() error {
	if c.BindPort <= 0 || c.BindPort > 65535 {
		return fmt.Errorf("bindPort 必须在 1-65535 之间")
	}
	if c.WebServer != nil && c.WebServer.Port != 0 {
		if c.WebServer.Port == c.BindPort {
			return fmt.Errorf("webServer.port 不能与 bindPort 相同")
		}
	}
	return nil
}

// Validate checks a client config and its proxies.
func (c *ClientConfig) Validate() error {
	if strings.TrimSpace(c.ServerAddr) == "" {
		return fmt.Errorf("serverAddr 不能为空")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("serverPort 必须在 1-65535 之间")
	}
	seen := map[string]bool{}
	for i, p := range c.Proxies {
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("第 %d 个代理缺少名称", i+1)
		}
		if seen[p.Name] {
			return fmt.Errorf("代理名称重复: %s", p.Name)
		}
		seen[p.Name] = true
		if !validProxyTypes[p.Type] {
			return fmt.Errorf("代理 %s 的类型无效: %q", p.Name, p.Type)
		}
		switch p.Type {
		case "tcp", "udp":
			if p.RemotePort <= 0 {
				return fmt.Errorf("代理 %s (%s) 需要 remotePort", p.Name, p.Type)
			}
		case "http", "https":
			if len(p.CustomDomains) == 0 && p.Subdomain == "" {
				return fmt.Errorf("代理 %s (%s) 需要 customDomains 或 subdomain", p.Name, p.Type)
			}
		}
		if p.Type != "stcp" && p.Type != "sudp" && p.Type != "xtcp" && p.LocalPort <= 0 {
			return fmt.Errorf("代理 %s 需要 localPort", p.Name)
		}
	}
	return nil
}
