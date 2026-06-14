package frpman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ProxyTypes are the frps proxy categories the monitor aggregates.
var ProxyTypes = []string{"tcp", "udp", "http", "https", "tcpmux", "stcp", "sudp", "xtcp"}

// AdminClient talks to a running frpc/frps webServer (admin) API over loopback.
// We always dial 127.0.0.1 regardless of the configured bind address, since the
// panel runs on the same host as frp.
type AdminClient struct {
	http *http.Client
	base string
	user string
	pass string
}

// NewAdminClient builds a client for the admin API on the given local port.
func NewAdminClient(port int, user, pass string) *AdminClient {
	return &AdminClient{
		http: &http.Client{Timeout: 8 * time.Second},
		base: "http://127.0.0.1:" + strconv.Itoa(port),
		user: user,
		pass: pass,
	}
}

func (c *AdminClient) do(ctx context.Context, method, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if c.user != "" || c.pass != "" {
		req.SetBasicAuth(c.user, c.pass)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("无法连接 frp 管理端口 (是否已启用 webServer 且 frp 正在运行?): %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return resp.StatusCode, body, fmt.Errorf("管理 API 鉴权失败,请检查 webServer 用户名/密码")
	}
	return resp.StatusCode, body, nil
}

func (c *AdminClient) getJSON(ctx context.Context, path string) (json.RawMessage, error) {
	status, body, err := c.do(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("frp 管理 API 返回 %d: %s", status, strings.TrimSpace(string(body)))
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("frp 管理 API 返回了非 JSON 响应")
	}
	return json.RawMessage(body), nil
}

// --- frpc (client) ---

// ClientStatus returns the proxy status report (GET /api/status).
func (c *AdminClient) ClientStatus(ctx context.Context) (json.RawMessage, error) {
	return c.getJSON(ctx, "/api/status")
}

// ClientReload triggers a hot reload of the client config (GET /api/reload).
func (c *AdminClient) ClientReload(ctx context.Context) error {
	status, body, err := c.do(ctx, http.MethodGet, "/api/reload")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("重载失败 (%d): %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// --- frps (server) ---

// ServerInfo returns server-wide stats (GET /api/serverinfo).
func (c *AdminClient) ServerInfo(ctx context.Context) (json.RawMessage, error) {
	return c.getJSON(ctx, "/api/serverinfo")
}

// ServerProxiesByType lists proxies of one type (GET /api/proxy/{type}).
func (c *AdminClient) ServerProxiesByType(ctx context.Context, typ string) (json.RawMessage, error) {
	return c.getJSON(ctx, "/api/proxy/"+url.PathEscape(typ))
}

// ServerClients lists connected clients (GET /api/clients). Available in newer
// frps; callers should tolerate an error on older versions.
func (c *AdminClient) ServerClients(ctx context.Context) (json.RawMessage, error) {
	return c.getJSON(ctx, "/api/clients")
}

// ServerTraffic returns traffic history for a proxy (GET /api/traffic/{name}).
func (c *AdminClient) ServerTraffic(ctx context.Context, name string) (json.RawMessage, error) {
	return c.getJSON(ctx, "/api/traffic/"+url.PathEscape(name))
}
