// Package core holds the panel's business logic: it ties the config store, the
// frp process manager, the downloader and the admin-API client together behind
// a small surface the HTTP layer calls into.
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"frppanel/internal/config"
	"frppanel/internal/frpman"
	"frppanel/internal/tomlcfg"
)

// App is the central coordinator. One instance lives for the program lifetime.
type App struct {
	store    *config.Store
	logs     *frpman.LogHub
	proc     *frpman.ProcessManager
	down     *frpman.Downloader
	gh       *frpman.GitHub
	progress *ProgressHub

	dlMu     sync.Mutex
	dlActive bool
	dlCancel context.CancelFunc
}

// New builds an App around a loaded config store.
func New(store *config.Store) *App {
	hub := frpman.NewLogHub(2000)
	return &App{
		store:    store,
		logs:     hub,
		proc:     frpman.NewProcessManager(hub),
		down:     frpman.NewDownloader(),
		gh:       frpman.NewGitHub(),
		progress: NewProgressHub(),
	}
}

// Accessors.
func (a *App) Store() *config.Store            { return a.store }
func (a *App) Logs() *frpman.LogHub            { return a.logs }
func (a *App) Proc() *frpman.ProcessManager    { return a.proc }
func (a *App) Progress() *ProgressHub          { return a.progress }
func (a *App) ProcState() frpman.ProcState     { return a.proc.State() }
func (a *App) Configured() bool                { return a.store.IsConfigured() }

// --- frp process control ---

// StartFRP launches the frp binary for the configured role.
func (a *App) StartFRP() error {
	c := a.store.Get()
	if !c.Configured {
		return errors.New("尚未完成初始化")
	}
	tomlPath := a.store.ActiveTOMLPath()
	if _, err := os.Stat(tomlPath); err != nil {
		return errors.New("配置文件不存在,请先在「配置」页保存")
	}
	bin := a.store.ActiveBinaryPath()
	return a.proc.Start(bin, []string{"-c", tomlPath}, a.store.BaseDir())
}

// StopFRP stops the frp process.
func (a *App) StopFRP() error { return a.proc.Stop() }

// RestartFRP restarts the frp process from the current config.
func (a *App) RestartFRP() error {
	if a.proc.Running() {
		if err := a.proc.Stop(); err != nil {
			return err
		}
	}
	return a.StartFRP()
}

// AutoStartIfNeeded starts frp on launch when configured and enabled.
func (a *App) AutoStartIfNeeded() {
	c := a.store.Get()
	if c.Configured && c.AutoStart {
		if err := a.StartFRP(); err != nil {
			a.logs.Append("[panel] 自动启动失败: " + err.Error())
		}
	}
}

// Shutdown cancels any download and stops frp. Called on program exit.
func (a *App) Shutdown() {
	a.CancelDownload()
	_ = a.proc.Stop()
}

// --- admin API / monitoring ---

// AdminClient builds a client for the running frp's admin API from the current
// config's webServer settings.
func (a *App) AdminClient() (*frpman.AdminClient, error) {
	c := a.store.Get()
	data, err := os.ReadFile(a.store.ActiveTOMLPath())
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	var ws *tomlcfg.WebServer
	if c.Role == config.RoleServer {
		sc, derr := tomlcfg.DecodeServer(data)
		if derr != nil {
			return nil, derr
		}
		ws = sc.WebServer
	} else {
		cc, derr := tomlcfg.DecodeClient(data)
		if derr != nil {
			return nil, derr
		}
		ws = cc.WebServer
	}
	if ws == nil || ws.Port == 0 {
		return nil, errors.New("未启用管理端口 (webServer),无法获取监控数据")
	}
	return frpman.NewAdminClient(ws.Port, ws.User, ws.Password), nil
}

// ReloadClient hot-reloads the frpc config via its admin API.
func (a *App) ReloadClient(ctx context.Context) error {
	cli, err := a.AdminClient()
	if err != nil {
		return err
	}
	return cli.ClientReload(ctx)
}

// --- configuration ---

// ConfigBundle is the response for GET /api/config.
type ConfigBundle struct {
	Role   config.Role           `json:"role"`
	Server *tomlcfg.ServerConfig `json:"server,omitempty"`
	Client *tomlcfg.ClientConfig `json:"client,omitempty"`
	Raw    string                `json:"raw"`
}

// ConfigBundle reads the active .toml (creating a default if missing) and
// returns both the structured model and the raw text.
func (a *App) ConfigBundle() (*ConfigBundle, error) {
	c := a.store.Get()
	tomlPath := a.store.ActiveTOMLPath()
	data, err := os.ReadFile(tomlPath)
	if errors.Is(err, os.ErrNotExist) {
		if c.Role == config.RoleServer {
			data, _ = tomlcfg.EncodeServer(tomlcfg.DefaultServer())
		} else {
			data, _ = tomlcfg.EncodeClient(tomlcfg.DefaultClient())
		}
		if werr := writeFileAtomic(tomlPath, data, 0o600); werr != nil {
			return nil, werr
		}
	} else if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	b := &ConfigBundle{Role: c.Role, Raw: string(data)}
	if c.Role == config.RoleServer {
		sc, derr := tomlcfg.DecodeServer(data)
		if derr != nil {
			return nil, derr
		}
		b.Server = sc
	} else {
		cc, derr := tomlcfg.DecodeClient(data)
		if derr != nil {
			return nil, derr
		}
		b.Client = cc
	}
	return b, nil
}

// SaveServer validates and writes a server config.
func (a *App) SaveServer(sc *tomlcfg.ServerConfig) error {
	if a.store.Get().Role != config.RoleServer {
		return errors.New("当前为客户端模式,无法保存服务端配置")
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	data, err := tomlcfg.EncodeServer(sc)
	if err != nil {
		return err
	}
	return writeFileAtomic(a.store.Path("frps.toml"), data, 0o600)
}

// SaveClient validates and writes a client config.
func (a *App) SaveClient(cc *tomlcfg.ClientConfig) error {
	if a.store.Get().Role != config.RoleClient {
		return errors.New("当前为服务端模式,无法保存客户端配置")
	}
	if err := cc.Validate(); err != nil {
		return err
	}
	data, err := tomlcfg.EncodeClient(cc)
	if err != nil {
		return err
	}
	return writeFileAtomic(a.store.Path("frpc.toml"), data, 0o600)
}

// SaveRaw validates raw TOML (it must parse and pass model validation) and
// writes it verbatim, preserving comments and formatting.
func (a *App) SaveRaw(raw string) error {
	c := a.store.Get()
	if c.Role == config.RoleServer {
		sc, err := tomlcfg.DecodeServer([]byte(raw))
		if err != nil {
			return err
		}
		if err := sc.Validate(); err != nil {
			return err
		}
	} else {
		cc, err := tomlcfg.DecodeClient([]byte(raw))
		if err != nil {
			return err
		}
		if err := cc.Validate(); err != nil {
			return err
		}
	}
	return writeFileAtomic(a.store.ActiveTOMLPath(), []byte(raw), 0o600)
}

// --- setup & updates ---

// UpdateInfo is the response for GET /api/update/check.
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"hasUpdate"`
	URL       string `json:"url"`
	Notes     string `json:"notes"`
	Published string `json:"published"`
}

// CheckUpdate queries GitHub for the newest frp release.
func (a *App) CheckUpdate(ctx context.Context) (*UpdateInfo, error) {
	rel, err := a.gh.LatestRelease(ctx)
	if err != nil {
		return nil, err
	}
	cur := a.store.Get().FRP.Version
	latest := rel.Version()
	return &UpdateInfo{
		Current:   cur,
		Latest:    latest,
		HasUpdate: frpman.CompareVersions(latest, cur) > 0,
		URL:       rel.HTMLURL,
		Notes:     rel.Body,
		Published: rel.PublishedAt,
	}, nil
}

// DownloadActive reports whether an install/update is in progress.
func (a *App) DownloadActive() bool {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	return a.dlActive
}

// CancelDownload aborts an in-flight install/update.
func (a *App) CancelDownload() {
	a.dlMu.Lock()
	if a.dlCancel != nil {
		a.dlCancel()
	}
	a.dlMu.Unlock()
}

// startDownload runs an install/update in the background, streaming progress.
// pre runs before the download (e.g. stop frp); post runs after with the result
// or error (e.g. record version, restart frp).
func (a *App) startDownload(version string, plat frpman.Platform, pre func(), post func(*frpman.DownloadResult, error)) error {
	a.dlMu.Lock()
	if a.dlActive {
		a.dlMu.Unlock()
		return errors.New("已有下载任务进行中")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	a.dlActive = true
	a.dlCancel = cancel
	a.dlMu.Unlock()

	a.progress.Reset()
	go func() {
		defer func() {
			a.dlMu.Lock()
			a.dlActive = false
			a.dlCancel = nil
			a.dlMu.Unlock()
			cancel()
		}()
		if pre != nil {
			pre()
		}
		res, err := a.down.Download(ctx, frpman.DownloadOptions{
			Version:  version,
			Platform: plat,
			DestDir:  a.store.BaseDir(),
		}, func(p frpman.Progress) { a.progress.Publish(p) })
		if post != nil {
			post(res, err)
		}
	}()
	return nil
}

// StartInstall downloads frp during first-run setup, recording the platform and
// version. Setup is finalized separately once the user sets a password.
func (a *App) StartInstall(version string, plat frpman.Platform) error {
	return a.startDownload(version, plat, nil, func(res *frpman.DownloadResult, err error) {
		if err != nil || res == nil {
			return
		}
		_ = a.store.Update(func(c *config.PanelConfig) {
			c.FRP = config.FRPInfo{
				Version:   res.Version,
				OS:        res.Platform.OS,
				Arch:      res.Platform.Arch,
				ServerBin: res.ServerBin,
				ClientBin: res.ClientBin,
			}
		})
	})
}

// StartUpdate downloads a newer frp, stopping frp first (required on Windows to
// replace a running binary) and restarting it afterwards if it had been running.
func (a *App) StartUpdate(version string) error {
	if !a.Configured() {
		return errors.New("尚未完成初始化")
	}
	c := a.store.Get()
	plat := frpman.Platform{OS: c.FRP.OS, Arch: c.FRP.Arch}
	if plat.OS == "" || plat.Arch == "" {
		plat = frpman.DetectPlatform()
	}
	wasRunning := a.proc.Running()
	return a.startDownload(version, plat,
		func() {
			if wasRunning {
				a.progress.Publish(frpman.Progress{Phase: frpman.PhaseResolve, Message: "正在停止 frp 以便替换二进制…"})
				_ = a.proc.Stop()
			}
		},
		func(res *frpman.DownloadResult, err error) {
			if err == nil && res != nil {
				_ = a.store.Update(func(cc *config.PanelConfig) {
					cc.FRP.Version = res.Version
					if res.ServerBin != "" {
						cc.FRP.ServerBin = res.ServerBin
					}
					if res.ClientBin != "" {
						cc.FRP.ClientBin = res.ClientBin
					}
				})
				a.logs.Append("[panel] frp 已更新到 " + res.Version)
			}
			// Bring frp back regardless of update outcome.
			if wasRunning {
				if serr := a.StartFRP(); serr != nil {
					a.logs.Append("[panel] 更新后重启失败: " + serr.Error())
				}
			}
		})
}

// FinalizeSetup records the chosen role and panel password, writes a default
// config if none exists, and marks the panel configured. passwordHash must be a
// bcrypt hash produced by the caller.
func (a *App) FinalizeSetup(role config.Role, passwordHash string) error {
	if role != config.RoleServer && role != config.RoleClient {
		return errors.New("无效的角色")
	}
	if passwordHash == "" {
		return errors.New("缺少面板密码")
	}
	c := a.store.Get()

	// Resolve the expected binary name and ensure it is present.
	base := "frpc"
	if role == config.RoleServer {
		base = "frps"
	}
	binName := c.FRP.ClientBin
	if role == config.RoleServer {
		binName = c.FRP.ServerBin
	}
	if binName == "" {
		plat := frpman.Platform{OS: c.FRP.OS, Arch: c.FRP.Arch}
		if plat.OS == "" {
			plat = frpman.DetectPlatform()
		}
		binName = plat.BinaryName(base)
	}
	if _, err := os.Stat(a.store.Path(binName)); err != nil {
		return errors.New("frp 可执行文件不存在,请先完成下载")
	}

	// Write a default config if the role's .toml does not yet exist.
	tomlName := "frpc.toml"
	if role == config.RoleServer {
		tomlName = "frps.toml"
	}
	tomlPath := a.store.Path(tomlName)
	if _, err := os.Stat(tomlPath); errors.Is(err, os.ErrNotExist) {
		var data []byte
		if role == config.RoleServer {
			data, _ = tomlcfg.EncodeServer(tomlcfg.DefaultServer())
		} else {
			data, _ = tomlcfg.EncodeClient(tomlcfg.DefaultClient())
		}
		if werr := writeFileAtomic(tomlPath, data, 0o600); werr != nil {
			return werr
		}
	}

	return a.store.Update(func(cc *config.PanelConfig) {
		cc.Role = role
		cc.PasswordHash = passwordHash
		cc.Configured = true
		if role == config.RoleServer && cc.FRP.ServerBin == "" {
			cc.FRP.ServerBin = binName
		}
		if role == config.RoleClient && cc.FRP.ClientBin == "" {
			cc.FRP.ClientBin = binName
		}
	})
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("提交 %s 失败: %w", path, err)
	}
	return nil
}
