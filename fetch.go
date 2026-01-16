package pocket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadOpt configures download and extraction behavior.
type DownloadOpt func(*downloadConfig)

type downloadConfig struct {
	destDir      string
	format       string // "tar.gz", "tar", "zip", "" (raw copy)
	extractOpts  []ExtractOpt
	symlink      bool
	skipIfExists string
	httpHeaders  map[string]string
}

func newDownloadConfig(opts []DownloadOpt) *downloadConfig {
	cfg := &downloadConfig{
		httpHeaders: make(map[string]string),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithDestDir sets the destination directory for extraction.
func WithDestDir(dir string) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.destDir = dir
	}
}

// WithFormat sets the archive format.
// Supported formats: "tar.gz", "tar", "zip", or "" for raw copy.
func WithFormat(format string) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.format = format
	}
}

// WithExtract adds extraction options from the extract package.
// Use this to pass WithRenameFile, WithExtractFile, or WithFlatten options.
//
// Example:
//
//	Download(ctx, url,
//	    WithExtract(WithRenameFile("tool-1.0.0/tool", "tool")),
//	    WithExtract(WithExtractFile("LICENSE")),
//	)
func WithExtract(opt ExtractOpt) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.extractOpts = append(cfg.extractOpts, opt)
	}
}

// WithSymlink creates a symlink in .pocket/bin/ after extraction.
// The symlink points to the first extracted file.
func WithSymlink() DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.symlink = true
	}
}

// WithSkipIfExists skips the download if the specified file exists.
func WithSkipIfExists(path string) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.skipIfExists = path
	}
}

// WithHTTPHeader adds an HTTP header to the download request.
// Multiple calls accumulate headers.
func WithHTTPHeader(key, value string) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.httpHeaders[key] = value
	}
}

// Download creates a Runnable that fetches a URL and optionally extracts it.
// Progress and status messages are written to the context's output.
//
// Example:
//
//	var Install = pocket.Task("install:tool", "install tool",
//	    pocket.Download(url,
//	        pocket.WithDestDir(binDir),
//	        pocket.WithFormat("tar.gz"),
//	        pocket.WithExtract(pocket.WithRenameFile("tool-1.0.0/tool", "tool")),
//	        pocket.WithSymlink(),
//	    ),
//	).Hidden()
func Download(url string, opts ...DownloadOpt) Runnable {
	return Do(func(ctx context.Context) error {
		return download(ctx, url, opts...)
	})
}

// download is the internal implementation of Download.
func download(ctx context.Context, url string, opts ...DownloadOpt) error {
	cfg := newDownloadConfig(opts)

	// Check if we can skip.
	if cfg.skipIfExists != "" {
		if _, err := os.Stat(cfg.skipIfExists); err == nil {
			// Already exists, just ensure symlink if requested.
			if cfg.symlink {
				if _, err := CreateSymlink(cfg.skipIfExists); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// Create destination directory.
	if cfg.destDir != "" {
		if err := os.MkdirAll(cfg.destDir, 0o755); err != nil {
			return fmt.Errorf("create destination dir: %w", err)
		}
	}

	Printf(ctx, "  Downloading %s\n", url)

	// Download to temp file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for k, v := range cfg.httpHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "pocket-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("download: %w", err)
	}
	tmpFile.Close()

	// Process the downloaded file.
	binaryPath, err := processFile(tmpPath, cfg)
	if err != nil {
		return err
	}

	// Create symlink if requested.
	if cfg.symlink && binaryPath != "" {
		if _, err := CreateSymlink(binaryPath); err != nil {
			return err
		}
	}

	return nil
}

// FromLocal creates a Runnable that processes a local file (extract/copy).
// Useful for processing pre-downloaded or bundled archives.
func FromLocal(path string, opts ...DownloadOpt) Runnable {
	return Do(func(ctx context.Context) error {
		return fromLocal(ctx, path, opts...)
	})
}

// fromLocal is the internal implementation of FromLocal.
func fromLocal(ctx context.Context, path string, opts ...DownloadOpt) error {
	cfg := newDownloadConfig(opts)

	// Check if we can skip.
	if cfg.skipIfExists != "" {
		if _, err := os.Stat(cfg.skipIfExists); err == nil {
			if cfg.symlink {
				if _, err := CreateSymlink(cfg.skipIfExists); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// Create destination directory.
	if cfg.destDir != "" {
		if err := os.MkdirAll(cfg.destDir, 0o755); err != nil {
			return fmt.Errorf("create destination dir: %w", err)
		}
	}

	Printf(ctx, "  Processing %s\n", path)

	// Process the file.
	binaryPath, err := processFile(path, cfg)
	if err != nil {
		return err
	}

	// Create symlink if requested.
	if cfg.symlink && binaryPath != "" {
		if _, err := CreateSymlink(binaryPath); err != nil {
			return err
		}
	}

	return nil
}

// processFile extracts or copies a file based on configuration.
// Returns the path to the first extracted/copied file (for symlink creation).
func processFile(path string, cfg *downloadConfig) (string, error) {
	destDir := cfg.destDir
	if destDir == "" {
		destDir = "."
	}

	var firstFile string

	switch cfg.format {
	case "tar.gz":
		if err := ExtractTarGz(path, destDir, cfg.extractOpts...); err != nil {
			return "", fmt.Errorf("extract tar.gz: %w", err)
		}
		firstFile = findFirstExtractedFile(destDir, cfg.extractOpts)
	case "tar":
		if err := ExtractTar(path, destDir, cfg.extractOpts...); err != nil {
			return "", fmt.Errorf("extract tar: %w", err)
		}
		firstFile = findFirstExtractedFile(destDir, cfg.extractOpts)
	case "zip":
		if err := ExtractZip(path, destDir, cfg.extractOpts...); err != nil {
			return "", fmt.Errorf("extract zip: %w", err)
		}
		firstFile = findFirstExtractedFile(destDir, cfg.extractOpts)
	default:
		// Raw copy - use base name of source file.
		dst := filepath.Join(destDir, filepath.Base(path))
		if err := CopyFile(path, dst); err != nil {
			return "", fmt.Errorf("copy file: %w", err)
		}
		firstFile = dst
	}

	return firstFile, nil
}

// findFirstExtractedFile determines the path to the first extracted file.
// This is used for symlink creation.
func findFirstExtractedFile(destDir string, opts []ExtractOpt) string {
	// Build extract config to find renamed files.
	cfg := newExtractConfig(opts)

	// If we have rename mappings, use the first destination name.
	for _, destName := range cfg.renameMap {
		return filepath.Join(destDir, destName)
	}

	// Otherwise, we can't determine which file was extracted.
	return ""
}
