// Package server exposes the panel's HTTP API and serves the embedded Vue SPA.
package server

import (
	"context"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"frppanel/internal/core"
)

// PanelVersion is the panel's own version (distinct from the frp version).
const PanelVersion = "1.0.0"

// Server wires the App to an HTTP mux.
type Server struct {
	app     *core.App
	auth    *AuthManager
	web     fs.FS
	mux     *http.ServeMux
	httpSrv *http.Server
	baseCtx context.Context
	cancel  context.CancelFunc
}

// New constructs a server. web is the embedded frontend filesystem rooted at
// the SPA's index.html.
func New(app *core.App, web fs.FS) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		app:     app,
		auth:    NewAuthManager(),
		web:     web,
		mux:     http.NewServeMux(),
		baseCtx: ctx,
		cancel:  cancel,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	m := s.mux

	// Bootstrap / auth (always available).
	m.HandleFunc("GET /api/state", s.handleState)
	m.HandleFunc("POST /api/login", s.handleLogin)
	m.HandleFunc("POST /api/logout", s.handleLogout)

	// First-run setup (only before the panel is configured).
	m.Handle("GET /api/setup/platforms", s.setupOnly(s.handlePlatforms))
	m.Handle("POST /api/setup/install", s.setupOnly(s.handleSetupInstall))
	m.Handle("GET /api/setup/progress", s.setupOnly(s.handleProgressStream))
	m.Handle("POST /api/setup/cancel", s.setupOnly(s.handleCancelDownload))
	m.Handle("POST /api/setup/finalize", s.setupOnly(s.handleFinalize))

	// Configuration.
	m.Handle("GET /api/config", s.protected(s.handleGetConfig))
	m.Handle("PUT /api/config", s.protected(s.handlePutConfig))
	m.Handle("PUT /api/config/raw", s.protected(s.handlePutConfigRaw))

	// frp process control.
	m.Handle("GET /api/frp/status", s.protected(s.handleFRPStatus))
	m.Handle("POST /api/frp/start", s.protected(s.handleFRPStart))
	m.Handle("POST /api/frp/stop", s.protected(s.handleFRPStop))
	m.Handle("POST /api/frp/restart", s.protected(s.handleFRPRestart))
	m.Handle("POST /api/frp/reload", s.protected(s.handleFRPReload))

	// Logs.
	m.Handle("GET /api/logs", s.protected(s.handleLogsHistory))
	m.Handle("GET /api/logs/stream", s.protected(s.handleLogStream))
	m.Handle("DELETE /api/logs", s.protected(s.handleLogsClear))

	// Monitoring.
	m.Handle("GET /api/monitor", s.protected(s.handleMonitor))

	// Updates.
	m.Handle("GET /api/update/check", s.protected(s.handleUpdateCheck))
	m.Handle("POST /api/update/perform", s.protected(s.handleUpdatePerform))
	m.Handle("POST /api/update/cancel", s.protected(s.handleCancelDownload))
	m.Handle("GET /api/progress", s.protected(s.handleProgressStream))

	// Settings & account.
	m.Handle("GET /api/settings", s.protected(s.handleSettingsGet))
	m.Handle("PUT /api/settings", s.protected(s.handleSettingsPut))
	m.Handle("POST /api/account/password", s.protected(s.handleChangePassword))

	// Static SPA (catch-all).
	m.HandleFunc("/", s.serveStatic)
}

// --- middleware ---

func isMutating(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	}
	return false
}

// csrfOK requires a custom header that browsers will not attach to cross-site
// form/navigation requests. Combined with SameSite=Strict cookies this blocks
// CSRF without a token dance.
func csrfOK(r *http.Request) bool {
	return r.Header.Get("X-Panel-CSRF") != ""
}

func (s *Server) authed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return s.auth.Valid(c.Value)
}

// protected requires the panel to be configured and the caller authenticated.
func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.app.Configured() {
			writeError(w, http.StatusConflict, "尚未完成初始化")
			return
		}
		if !s.authed(r) {
			writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
			return
		}
		if isMutating(r) && !csrfOK(r) {
			writeError(w, http.StatusForbidden, "CSRF 校验失败")
			return
		}
		h(w, r)
	})
}

// setupOnly allows a handler only before the panel is configured.
func (s *Server) setupOnly(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.app.Configured() {
			writeError(w, http.StatusConflict, "面板已完成初始化")
			return
		}
		if isMutating(r) && !csrfOK(r) {
			writeError(w, http.StatusForbidden, "CSRF 校验失败")
			return
		}
		h(w, r)
	})
}

// --- static SPA serving ---

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	fileServer := http.FileServerFS(s.web)
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	if _, err := fs.Stat(s.web, p); err != nil {
		// Unknown path → let the SPA router handle it.
		r.URL.Path = "/index.html"
		w.Header().Set("Cache-Control", "no-cache")
	} else if strings.HasPrefix(p, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	fileServer.ServeHTTP(w, r)
}

// --- lifecycle ---

// Run starts serving on addr and blocks until shutdown.
func (s *Server) Run(addr string) error {
	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s.httpSrv.ListenAndServe()
}

// Shutdown cancels live streams and gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}
