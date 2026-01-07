// Package tool provides utilities for downloading, extracting, and managing tool binaries.
package tool

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket"
)

// Opt is an option for FromRemote and FromLocal.
type Opt func(*options)

type options struct {
	destDir       string
	untarGz       bool
	untar         bool
	unzip         bool
	skipIfExists  string
	symlinkBinary string
	renameFiles   map[string]string
	httpHeaders   map[string]string
	extractFiles  []string // Only extract these files (by base name), flatten to destDir
}

// WithDestinationDir sets the directory to extract files to.
func WithDestinationDir(dir string) Opt {
	return func(o *options) {
		o.destDir = dir
	}
}

// WithUntarGz extracts a .tar.gz archive.
func WithUntarGz() Opt {
	return func(o *options) {
		o.untarGz = true
	}
}

// WithUntar extracts a .tar archive.
func WithUntar() Opt {
	return func(o *options) {
		o.untar = true
	}
}

// WithUnzip extracts a .zip archive.
func WithUnzip() Opt {
	return func(o *options) {
		o.unzip = true
	}
}

// WithSkipIfFileExists skips the download if the given file already exists.
func WithSkipIfFileExists(path string) Opt {
	return func(o *options) {
		o.skipIfExists = path
	}
}

// WithSymlink creates a symlink to the binary in .pocket/bin.
func WithSymlink(binaryPath string) Opt {
	return func(o *options) {
		o.symlinkBinary = binaryPath
	}
}

// WithRenameFile renames a file during extraction.
func WithRenameFile(src, dst string) Opt {
	return func(o *options) {
		if o.renameFiles == nil {
			o.renameFiles = make(map[string]string)
		}
		o.renameFiles[src] = dst
	}
}

// WithHTTPHeader adds a header to the HTTP request.
func WithHTTPHeader(key, value string) Opt {
	return func(o *options) {
		if o.httpHeaders == nil {
			o.httpHeaders = make(map[string]string)
		}
		o.httpHeaders[key] = value
	}
}

// WithExtractFiles only extracts files matching the given base names,
// flattening them directly into the destination directory.
func WithExtractFiles(names ...string) Opt {
	return func(o *options) {
		o.extractFiles = names
	}
}

// FromRemote downloads and optionally extracts a file from a URL.
func FromRemote(ctx context.Context, url string, opts ...Opt) error {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	// Check if we can skip
	if o.skipIfExists != "" {
		if _, err := os.Stat(o.skipIfExists); err == nil {
			// File exists, just ensure symlink
			if o.symlinkBinary != "" {
				if _, err := CreateSymlink(o.symlinkBinary); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// Create destination directory
	if o.destDir != "" {
		if err := os.MkdirAll(o.destDir, 0o755); err != nil {
			return fmt.Errorf("create destination dir: %w", err)
		}
	}

	// Download to temp file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for k, v := range o.httpHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "pocket-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("download %s: %w", url, err)
	}
	tmpFile.Close()

	// Extract or copy
	if err := processFile(tmpPath, o); err != nil {
		return err
	}

	// Create symlink if requested
	if o.symlinkBinary != "" {
		if _, err := CreateSymlink(o.symlinkBinary); err != nil {
			return err
		}
	}

	return nil
}

// FromLocal extracts a local archive file.
func FromLocal(_ context.Context, path string, opts ...Opt) error {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.skipIfExists != "" {
		if _, err := os.Stat(o.skipIfExists); err == nil {
			if o.symlinkBinary != "" {
				if _, err := CreateSymlink(o.symlinkBinary); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if o.destDir != "" {
		if err := os.MkdirAll(o.destDir, 0o755); err != nil {
			return fmt.Errorf("create destination dir: %w", err)
		}
	}

	if err := processFile(path, o); err != nil {
		return err
	}

	if o.symlinkBinary != "" {
		if _, err := CreateSymlink(o.symlinkBinary); err != nil {
			return err
		}
	}

	return nil
}

func processFile(path string, o *options) error {
	switch {
	case o.untarGz:
		return extractTarGz(path, o.destDir, o.renameFiles, o.extractFiles)
	case o.untar:
		return extractTar(path, o.destDir, o.renameFiles, o.extractFiles)
	case o.unzip:
		return extractZip(path, o.destDir, o.renameFiles, o.extractFiles)
	default:
		// Just copy the file
		if o.destDir != "" {
			dst := filepath.Join(o.destDir, filepath.Base(path))
			return copyFile(path, dst)
		}
		return nil
	}
}

func extractTarGz(src, destDir string, renames map[string]string, extractOnly []string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTarReader(tar.NewReader(gzr), destDir, renames, extractOnly)
}

func extractTar(src, destDir string, renames map[string]string, extractOnly []string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	return extractTarReader(tar.NewReader(f), destDir, renames, extractOnly)
}

func extractTarReader(tr *tar.Reader, destDir string, renames map[string]string, extractOnly []string) error {
	extractSet := make(map[string]bool)
	for _, name := range extractOnly {
		extractSet[name] = true
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := header.Name
		baseName := filepath.Base(name)

		// If extractOnly is set, only extract matching files (flattened)
		if len(extractOnly) > 0 {
			if !extractSet[baseName] {
				continue
			}
			// Flatten: use only the base name
			name = baseName
		} else if newName, ok := renames[name]; ok {
			name = newName
		}

		target := filepath.Join(destDir, name)

		// Security check: ensure we don't escape destDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if len(extractOnly) > 0 {
				continue // Skip directories when extracting specific files
			}
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func extractZip(src, destDir string, renames map[string]string, extractOnly []string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	extractSet := make(map[string]bool)
	for _, name := range extractOnly {
		extractSet[name] = true
	}

	for _, f := range r.File {
		name := f.Name
		baseName := filepath.Base(name)

		// If extractOnly is set, only extract matching files (flattened)
		if len(extractOnly) > 0 {
			if !extractSet[baseName] {
				continue
			}
			name = baseName
		} else if newName, ok := renames[name]; ok {
			name = newName
		}

		target := filepath.Join(destDir, name)

		// Security check
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path: %s", name)
		}

		if f.FileInfo().IsDir() {
			if len(extractOnly) > 0 {
				continue
			}
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o755)
}

// CreateSymlink creates a symlink in .pocket/bin pointing to the given binary.
// On Windows, it copies the file instead since symlinks require admin privileges.
// Returns the path to the symlink (or copy on Windows).
func CreateSymlink(binaryPath string) (string, error) {
	binDir := pocket.FromBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("create bin dir: %w", err)
	}

	name := filepath.Base(binaryPath)
	linkPath := filepath.Join(binDir, name)

	// Remove existing file/symlink if it exists.
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return "", fmt.Errorf("remove existing file: %w", err)
		}
	}

	// On Windows, copy the file instead of creating a symlink.
	if isWindows() {
		if err := copyFile(binaryPath, linkPath); err != nil {
			return "", fmt.Errorf("copy binary: %w", err)
		}
		return linkPath, nil
	}

	// Create relative symlink on Unix.
	relPath, err := filepath.Rel(binDir, binaryPath)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}

	if err := os.Symlink(relPath, linkPath); err != nil {
		return "", fmt.Errorf("create symlink: %w", err)
	}

	return linkPath, nil
}

// isWindows returns true if running on Windows.
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}
