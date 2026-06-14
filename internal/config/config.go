// Package config manages panel.json — the panel-level metadata that lives in
// the same directory as the executable. It deliberately does NOT hold the frp
// configuration itself; that is the .toml file on disk, which frp reads and
// which is the single source of truth for proxy behaviour.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Role is the operating mode chosen during first-run setup.
type Role string

const (
	RoleServer Role = "server" // runs frps
	RoleClient Role = "client" // runs frpc
)

// CurrentVersion is the schema version of panel.json.
const CurrentVersion = 1

// FRPInfo records which frp binaries were downloaded and for what platform.
type FRPInfo struct {
	Version   string `json:"version,omitempty"` // e.g. "0.69.1" (no leading v)
	OS        string `json:"os,omitempty"`      // windows | linux | darwin | freebsd
	Arch      string `json:"arch,omitempty"`    // amd64 | arm64 | arm
	ServerBin string `json:"serverBin,omitempty"`
	ClientBin string `json:"clientBin,omitempty"`
}

// PanelConfig is the on-disk shape of panel.json.
type PanelConfig struct {
	Version      int     `json:"version"`
	Configured   bool    `json:"configured"`
	Role         Role    `json:"role,omitempty"`
	ListenAddr   string  `json:"listenAddr"`
	PasswordHash string  `json:"passwordHash,omitempty"`
	AutoStart    bool    `json:"autoStart"`
	FRP          FRPInfo `json:"frp"`
	CreatedAt    string  `json:"createdAt,omitempty"`
	UpdatedAt    string  `json:"updatedAt,omitempty"`
}

func defaults() *PanelConfig {
	return &PanelConfig{
		Version:    CurrentVersion,
		Configured: false,
		ListenAddr: ":8088",
		AutoStart:  true,
	}
}

// Store provides concurrency-safe access to panel.json and resolves paths that
// sit beside the executable.
type Store struct {
	mu      sync.RWMutex
	baseDir string
	path    string
	cfg     *PanelConfig
}

// NewStore creates a store rooted at baseDir and loads panel.json if present.
func NewStore(baseDir string) (*Store, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base dir: %w", err)
	}
	s := &Store{
		baseDir: abs,
		path:    filepath.Join(abs, "panel.json"),
		cfg:     defaults(),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // fresh install: keep defaults
	}
	if err != nil {
		return fmt.Errorf("read panel.json: %w", err)
	}
	cfg := defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse panel.json: %w", err)
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8088"
	}
	s.cfg = cfg
	return nil
}

// persist writes the current config atomically (temp file + rename).
func (s *Store) persist() error {
	s.cfg.UpdatedAt = time.Now().Format(time.RFC3339)
	if s.cfg.CreatedAt == "" {
		s.cfg.CreatedAt = s.cfg.UpdatedAt
	}
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode panel.json: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write panel.json: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit panel.json: %w", err)
	}
	return nil
}

// Get returns a copy of the current config; safe to read without locking.
func (s *Store) Get() PanelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.cfg
}

// Update applies fn under the write lock and persists the result.
func (s *Store) Update(fn func(*PanelConfig)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.cfg)
	if s.cfg.Version == 0 {
		s.cfg.Version = CurrentVersion
	}
	return s.persist()
}

// IsConfigured reports whether first-run setup has completed.
func (s *Store) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Configured && s.cfg.Role != "" && s.cfg.PasswordHash != ""
}

// BaseDir returns the directory that holds panel.json, the frp binaries and the
// generated .toml files.
func (s *Store) BaseDir() string { return s.baseDir }

// Path joins name onto the base directory.
func (s *Store) Path(name string) string { return filepath.Join(s.baseDir, name) }

// ActiveTOMLPath returns the frp config file for the configured role.
func (s *Store) ActiveTOMLPath() string {
	if s.Get().Role == RoleServer {
		return s.Path("frps.toml")
	}
	return s.Path("frpc.toml")
}

// ActiveBinaryPath returns the absolute path to the frp binary for the role.
func (s *Store) ActiveBinaryPath() string {
	c := s.Get()
	name := c.FRP.ClientBin
	if c.Role == RoleServer {
		name = c.FRP.ServerBin
	}
	if name == "" {
		// fall back to conventional names
		if c.Role == RoleServer {
			name = binName("frps")
		} else {
			name = binName("frpc")
		}
	}
	return s.Path(name)
}

func binName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}
