package server

import (
	"net/http"

	"frppanel/internal/config"
	"frppanel/internal/frpman"
	"frppanel/internal/tomlcfg"
)

// --- bootstrap & auth ---

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	c := s.app.Store().Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":     s.app.Configured(),
		"authenticated":  s.authed(r),
		"role":           c.Role,
		"frpVersion":     c.FRP.Version,
		"panelVersion":   PanelVersion,
		"downloadActive": s.app.DownloadActive(),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !csrfOK(r) {
		writeError(w, http.StatusForbidden, "CSRF 校验失败")
		return
	}
	if !s.app.Configured() {
		writeError(w, http.StatusConflict, "尚未完成初始化")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if !CheckPassword(s.app.Store().Get().PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "密码错误")
		return
	}
	setSessionCookie(w, s.auth.Create())
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.auth.Revoke(c.Value)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- setup ---

func (s *Server) handlePlatforms(w http.ResponseWriter, r *http.Request) {
	det := frpman.DetectPlatform()
	label := det.OS + "/" + det.Arch
	for _, o := range frpman.SupportedPlatforms() {
		if o.OS == det.OS && o.Arch == det.Arch {
			label = o.Label
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"detected": map[string]any{"os": det.OS, "arch": det.Arch, "label": label},
		"options":  frpman.SupportedPlatforms(),
	})
}

func (s *Server) handleSetupInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string `json:"version"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	plat := frpman.Platform{OS: req.OS, Arch: req.Arch}
	if plat.OS == "" || plat.Arch == "" {
		plat = frpman.DetectPlatform()
	}
	if err := s.app.StartInstall(req.Version, plat); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) handleCancelDownload(w http.ResponseWriter, r *http.Request) {
	s.app.CancelDownload()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleFinalize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if len([]rune(req.Password)) < 6 {
		writeError(w, http.StatusBadRequest, "面板密码至少 6 位")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}
	if err := s.app.FinalizeSetup(config.Role(req.Role), hash); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	setSessionCookie(w, s.auth.Create()) // auto-login
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- configuration ---

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.app.ConfigBundle()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Server *tomlcfg.ServerConfig `json:"server"`
		Client *tomlcfg.ClientConfig `json:"client"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	role := s.app.Store().Get().Role
	var err error
	if role == config.RoleServer {
		if req.Server == nil {
			writeError(w, http.StatusBadRequest, "缺少服务端配置")
			return
		}
		err = s.app.SaveServer(req.Server)
	} else {
		if req.Client == nil {
			writeError(w, http.StatusBadRequest, "缺少客户端配置")
			return
		}
		err = s.app.SaveClient(req.Client)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handlePutConfigRaw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Raw string `json:"raw"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.app.SaveRaw(req.Raw); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// --- frp control ---

func (s *Server) writeFRPStatus(w http.ResponseWriter, status int) {
	c := s.app.Store().Get()
	st := s.app.ProcState()
	writeJSON(w, status, map[string]any{
		"proc":       st,
		"role":       c.Role,
		"frpVersion": c.FRP.Version,
		"autoStart":  c.AutoStart,
	})
}

func (s *Server) handleFRPStatus(w http.ResponseWriter, r *http.Request) {
	s.writeFRPStatus(w, http.StatusOK)
}

func (s *Server) handleFRPStart(w http.ResponseWriter, r *http.Request) {
	if err := s.app.StartFRP(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.writeFRPStatus(w, http.StatusOK)
}

func (s *Server) handleFRPStop(w http.ResponseWriter, r *http.Request) {
	if err := s.app.StopFRP(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeFRPStatus(w, http.StatusOK)
}

func (s *Server) handleFRPRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.app.RestartFRP(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.writeFRPStatus(w, http.StatusOK)
}

func (s *Server) handleFRPReload(w http.ResponseWriter, r *http.Request) {
	if s.app.Store().Get().Role != config.RoleClient {
		writeError(w, http.StatusBadRequest, "仅客户端支持热重载")
		return
	}
	if err := s.app.ReloadClient(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

// --- logs ---

func (s *Server) handleLogsHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"lines": s.app.Logs().History()})
}

func (s *Server) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	s.app.Logs().Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// --- monitoring ---

func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	cli, err := s.app.AdminClient()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	ctx := r.Context()
	if s.app.Store().Get().Role == config.RoleServer {
		info, err := cli.ServerInfo(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		proxies := map[string]any{}
		for _, t := range frpman.ProxyTypes {
			if raw, e := cli.ServerProxiesByType(ctx, t); e == nil {
				proxies[t] = raw
			}
		}
		clients, _ := cli.ServerClients(ctx) // optional on older frps
		writeJSON(w, http.StatusOK, map[string]any{
			"role":       "server",
			"serverInfo": info,
			"proxies":    proxies,
			"clients":    clients,
		})
		return
	}
	status, err := cli.ClientStatus(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": "client", "status": status})
}

// --- updates ---

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	info, err := s.app.CheckUpdate(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleUpdatePerform(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string `json:"version"`
	}
	_ = decodeJSON(r, &req) // body optional; empty → latest
	if err := s.app.StartUpdate(req.Version); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// --- settings & account ---

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	c := s.app.Store().Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"role":       c.Role,
		"autoStart":  c.AutoStart,
		"listenAddr": c.ListenAddr,
		"frp":        c.FRP,
	})
}

func (s *Server) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AutoStart *bool `json:"autoStart"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	err := s.app.Store().Update(func(c *config.PanelConfig) {
		if req.AutoStart != nil {
			c.AutoStart = *req.AutoStart
		}
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.handleSettingsGet(w, r)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if !CheckPassword(s.app.Store().Get().PasswordHash, req.Current) {
		writeError(w, http.StatusUnauthorized, "当前密码错误")
		return
	}
	if len([]rune(req.New)) < 6 {
		writeError(w, http.StatusBadRequest, "新密码至少 6 位")
		return
	}
	hash, err := HashPassword(req.New)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}
	if err := s.app.Store().Update(func(c *config.PanelConfig) { c.PasswordHash = hash }); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auth.RevokeAll()
	setSessionCookie(w, s.auth.Create()) // keep the current operator signed in
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
