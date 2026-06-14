package frpman

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// maxBinarySize caps a single extracted file to guard against decompression
// bombs. frp binaries are well under this.
const maxBinarySize = 256 << 20 // 256 MiB

type extractResult struct {
	serverBin string // on-disk name, e.g. "frps.exe"
	clientBin string
}

// wantedNames maps the basenames we extract to their output filename. For frp
// the output name equals the archive basename (frps / frps.exe).
func wantedNames(plat Platform) map[string]string {
	srv := plat.BinaryName("frps")
	cli := plat.BinaryName("frpc")
	return map[string]string{srv: srv, cli: cli}
}

// extractFRP pulls the frps and frpc executables out of a downloaded archive
// into destDir, ignoring example configs and docs. Entry basenames are used as
// the destination filename, so archive path traversal cannot escape destDir.
func extractFRP(archivePath, destDir string, plat Platform) (extractResult, error) {
	if plat.OS == "windows" || strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZip(archivePath, destDir, plat)
	}
	return extractTarGz(archivePath, destDir, plat)
}

func extractZip(archivePath, destDir string, plat Platform) (extractResult, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return extractResult{}, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	want := wantedNames(plat)
	var res extractResult
	for _, f := range r.File {
		base := path.Base(filepath.ToSlash(f.Name))
		out, ok := want[base]
		if !ok || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return extractResult{}, fmt.Errorf("read %s: %w", base, err)
		}
		err = writeBinary(filepath.Join(destDir, out), rc)
		rc.Close()
		if err != nil {
			return extractResult{}, err
		}
		assignBinary(&res, base, out, plat)
	}
	return ensureFound(res)
}

func extractTarGz(archivePath, destDir string, plat Platform) (extractResult, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return extractResult{}, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return extractResult{}, fmt.Errorf("gunzip: %w", err)
	}
	defer gz.Close()

	want := wantedNames(plat)
	tr := tar.NewReader(gz)
	var res extractResult
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extractResult{}, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := path.Base(filepath.ToSlash(hdr.Name))
		out, ok := want[base]
		if !ok {
			continue
		}
		if err := writeBinary(filepath.Join(destDir, out), tr); err != nil {
			return extractResult{}, err
		}
		assignBinary(&res, base, out, plat)
	}
	return ensureFound(res)
}

func assignBinary(res *extractResult, base, out string, plat Platform) {
	switch base {
	case plat.BinaryName("frps"):
		res.serverBin = out
	case plat.BinaryName("frpc"):
		res.clientBin = out
	}
}

func ensureFound(res extractResult) (extractResult, error) {
	if res.serverBin == "" && res.clientBin == "" {
		return res, fmt.Errorf("archive did not contain frps/frpc binaries")
	}
	return res, nil
}

// writeBinary copies src to a temp file then atomically renames it into place,
// so an interrupted extraction never leaves a half-written (corrupt) binary.
// Capped at maxBinarySize to guard against decompression bombs.
func writeBinary(dst string, src io.Reader) error {
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(dst), err)
	}
	n, copyErr := io.Copy(out, io.LimitReader(src, maxBinarySize+1))
	closeErr := out.Close() // must close before rename (Windows locks open files)
	if copyErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("write %s: %w", filepath.Base(dst), copyErr)
	}
	if closeErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("write %s: %w", filepath.Base(dst), closeErr)
	}
	if n > maxBinarySize {
		os.Remove(tmp)
		return fmt.Errorf("%s exceeds size limit", filepath.Base(dst))
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("install %s: %w", filepath.Base(dst), err)
	}
	return nil
}
