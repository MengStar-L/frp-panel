// Package frpman owns everything about the frp binaries: discovering releases on
// GitHub, downloading and extracting them next to the program, running frps/frpc
// as a managed subprocess, and proxying their admin APIs for monitoring.
package frpman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	frpOwner  = "fatedier"
	frpRepo   = "frp"
	userAgent = "frp-panel"
)

// Asset is one downloadable file attached to a GitHub release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Release is the subset of the GitHub release payload we use.
type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Version returns the tag without a leading "v" (frp asset names omit it).
func (r Release) Version() string { return strings.TrimPrefix(r.TagName, "v") }

// FindAsset returns the asset with the given exact name.
func (r Release) FindAsset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// Platform identifies an frp build target.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// DetectPlatform returns the platform the panel is running on.
func DetectPlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// IsHost reports whether this platform matches the running host (i.e. the
// downloaded binaries will actually be runnable here).
func (p Platform) IsHost() bool {
	return p.OS == runtime.GOOS && p.Arch == runtime.GOARCH
}

// AssetName builds the frp release asset filename for a version, e.g.
// "frp_0.69.1_windows_amd64.zip".
func (p Platform) AssetName(version string) string {
	ext := "tar.gz"
	if p.OS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("frp_%s_%s_%s.%s", version, p.OS, p.Arch, ext)
}

// BinaryName returns the on-disk name of an frp binary for this platform.
func (p Platform) BinaryName(base string) string {
	if p.OS == "windows" {
		return base + ".exe"
	}
	return base
}

// PlatformOption is a selectable target shown in the setup wizard.
type PlatformOption struct {
	OS    string `json:"os"`
	Arch  string `json:"arch"`
	Label string `json:"label"`
}

// SupportedPlatforms is the curated matrix the UI offers. frp publishes more,
// but these cover the realistic targets a user runs the panel on.
func SupportedPlatforms() []PlatformOption {
	return []PlatformOption{
		{"windows", "amd64", "Windows x64"},
		{"windows", "arm64", "Windows ARM64"},
		{"linux", "amd64", "Linux x64"},
		{"linux", "arm64", "Linux ARM64"},
		{"linux", "arm", "Linux ARM (32-bit)"},
		{"darwin", "amd64", "macOS (Intel)"},
		{"darwin", "arm64", "macOS (Apple Silicon)"},
	}
}

// GitHub is a tiny client for the releases API.
type GitHub struct {
	HTTP    *http.Client
	BaseURL string
}

// NewGitHub returns a client with sane timeouts.
func NewGitHub() *GitHub {
	return &GitHub{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		BaseURL: "https://api.github.com",
	}
}

// LatestRelease fetches the newest non-draft release of fatedier/frp.
func (g *GitHub) LatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", g.BaseURL, frpOwner, frpRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("github rate limit reached, please retry later")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

// CompareVersions returns -1, 0, or 1 for a<b, a==b, a>b using dotted numeric
// segments. Non-numeric segments compare as 0. Leading "v" is ignored.
func CompareVersions(a, b string) int {
	as := splitVersion(a)
	bs := splitVersion(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(as) {
			x = as[i]
		}
		if i < len(bs) {
			y = bs[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		// stop at any pre-release suffix like "1-rc1"
		if idx := strings.IndexAny(p, "-+"); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
