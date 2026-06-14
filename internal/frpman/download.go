package frpman

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Phase enumerates the stages of an install/update.
type Phase string

const (
	PhaseResolve  Phase = "resolving"
	PhaseDownload Phase = "downloading"
	PhaseVerify   Phase = "verifying"
	PhaseExtract  Phase = "extracting"
	PhaseDone     Phase = "done"
	PhaseError    Phase = "error"
)

// Progress is one update emitted during a download. It is JSON-encoded straight
// onto the SSE stream the browser consumes.
type Progress struct {
	Phase      Phase   `json:"phase"`
	Message    string  `json:"message"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    float64 `json:"percent"`
	Version    string  `json:"version,omitempty"`
	Done       bool    `json:"done"`
	Error      string  `json:"error,omitempty"`
}

// ProgressFunc receives progress updates. It may be nil.
type ProgressFunc func(Progress)

func (f ProgressFunc) emit(p Progress) {
	if f != nil {
		f(p)
	}
}

// DownloadOptions controls an install.
type DownloadOptions struct {
	Version  string // "" or "latest" for the newest release
	Platform Platform
	DestDir  string
}

// DownloadResult describes what landed on disk.
type DownloadResult struct {
	Version   string   `json:"version"`
	Platform  Platform `json:"platform"`
	ServerBin string   `json:"serverBin"`
	ClientBin string   `json:"clientBin"`
}

// Downloader fetches and installs frp binaries.
type Downloader struct {
	gh   *GitHub
	http *http.Client
}

// NewDownloader returns a downloader. The HTTP client has no overall timeout so
// large archives are bounded by the caller's context instead.
func NewDownloader() *Downloader {
	return &Downloader{
		gh:   NewGitHub(),
		http: &http.Client{},
	}
}

// ReleaseByTag fetches a specific release. version may be with or without "v".
func (g *GitHub) ReleaseByTag(ctx context.Context, version string) (*Release, error) {
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", g.BaseURL, frpOwner, frpRepo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query release %s: %w", tag, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("frp release %s not found", tag)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

// Download resolves the requested release, downloads the matching archive,
// verifies its sha256 against the release checksums file, and extracts the
// frps/frpc binaries into DestDir. Progress is streamed through report.
func (d *Downloader) Download(ctx context.Context, opts DownloadOptions, report ProgressFunc) (result *DownloadResult, err error) {
	defer func() {
		if err != nil {
			report.emit(Progress{Phase: PhaseError, Message: "失败", Error: err.Error(), Done: true})
		}
	}()

	// 1. Resolve the release.
	report.emit(Progress{Phase: PhaseResolve, Message: "正在查询 frp 版本信息…"})
	var rel *Release
	if opts.Version == "" || opts.Version == "latest" {
		rel, err = d.gh.LatestRelease(ctx)
	} else {
		rel, err = d.gh.ReleaseByTag(ctx, opts.Version)
	}
	if err != nil {
		return nil, err
	}
	version := rel.Version()

	assetName := opts.Platform.AssetName(version)
	asset, ok := rel.FindAsset(assetName)
	if !ok {
		return nil, fmt.Errorf("frp %s 没有提供 %s/%s 的构建 (%s)", version, opts.Platform.OS, opts.Platform.Arch, assetName)
	}

	if err = os.MkdirAll(opts.DestDir, 0o755); err != nil {
		return nil, fmt.Errorf("create dest dir: %w", err)
	}

	// 2. Download to a temp file beside the destination.
	report.emit(Progress{Phase: PhaseDownload, Message: "正在下载 " + assetName, Total: asset.Size})
	tmp, err := os.CreateTemp(opts.DestDir, "frp-download-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	sum, derr := d.downloadTo(ctx, tmp, asset, report)
	tmp.Close()
	if derr != nil {
		return nil, derr
	}

	// 3. Verify checksum when the release ships one.
	report.emit(Progress{Phase: PhaseVerify, Message: "正在校验文件完整性…", Total: asset.Size, Downloaded: asset.Size, Percent: 100})
	if csAsset, ok := rel.FindAsset("frp_sha256_checksums.txt"); ok {
		expected, cerr := d.fetchExpectedSum(ctx, csAsset.URL, assetName)
		if cerr != nil {
			return nil, cerr
		}
		if expected != "" && !strings.EqualFold(expected, sum) {
			return nil, fmt.Errorf("校验失败: 期望 %s, 实际 %s", expected, sum)
		}
	}

	// 4. Extract binaries.
	report.emit(Progress{Phase: PhaseExtract, Message: "正在解压…", Total: asset.Size, Downloaded: asset.Size, Percent: 100})
	ex, err := extractFRP(tmpPath, opts.DestDir, opts.Platform)
	if err != nil {
		return nil, err
	}

	result = &DownloadResult{
		Version:   version,
		Platform:  opts.Platform,
		ServerBin: ex.serverBin,
		ClientBin: ex.clientBin,
	}
	report.emit(Progress{Phase: PhaseDone, Message: "frp " + version + " 安装完成", Version: version, Percent: 100, Total: asset.Size, Downloaded: asset.Size, Done: true})
	return result, nil
}

// downloadTo streams the asset into w, reporting progress, and returns the
// lowercase hex sha256 of the bytes written.
func (d *Downloader) downloadTo(ctx context.Context, w io.Writer, asset Asset, report ProgressFunc) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %s", resp.Status)
	}

	total := asset.Size
	if total <= 0 {
		total = resp.ContentLength
	}
	hasher := sha256.New()
	pr := &progressReader{
		r:     resp.Body,
		total: total,
		report: func(read, total int64) {
			pct := 0.0
			if total > 0 {
				pct = float64(read) / float64(total) * 100
			}
			report.emit(Progress{Phase: PhaseDownload, Message: "正在下载…", Downloaded: read, Total: total, Percent: pct})
		},
	}
	if _, err := io.Copy(io.MultiWriter(w, hasher), pr); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fetchExpectedSum downloads the checksums file and returns the sha256 for
// assetName, or "" if the file lists no matching entry.
func (d *Downloader) fetchExpectedSum(ctx context.Context, url, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch checksums returned %s", resp.Status)
	}
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && fields[1] == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", sc.Err()
}

// progressReader wraps a reader and reports cumulative bytes read, throttled to
// avoid flooding the SSE stream.
type progressReader struct {
	r          io.Reader
	total      int64
	read       int64
	lastReport time.Time
	report     func(read, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	now := time.Now()
	if err != nil || now.Sub(pr.lastReport) >= 120*time.Millisecond {
		pr.lastReport = now
		if pr.report != nil {
			pr.report(pr.read, pr.total)
		}
	}
	return n, err
}
