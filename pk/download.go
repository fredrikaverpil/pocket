package pk

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
	format       string // "tar.gz", "tar", "zip", "gz", "" (raw copy)
	extractOpts  []ExtractOpt
	symlink      bool
	skipIfExists string
	outputName   string // for "gz" format: the output filename
}

func newDownloadConfig(opts []DownloadOpt) *downloadConfig {
	cfg := &downloadConfig{}
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

// WithExtract adds extraction options.
func WithExtract(opt ExtractOpt) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.extractOpts = append(cfg.extractOpts, opt)
	}
}

// WithSymlink creates a symlink in .pocket/bin/ after extraction.
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

// WithOutputName sets the output filename for "gz" format extraction.
func WithOutputName(name string) DownloadOpt {
	return func(cfg *downloadConfig) {
		cfg.outputName = name
	}
}

// Download creates a Runnable that fetches a URL and optionally extracts it.
func Download(url string, opts ...DownloadOpt) Runnable {
	return Do(func(ctx context.Context) error {
		return download(ctx, url, opts...)
	})
}

func download(ctx context.Context, url string, opts ...DownloadOpt) error {
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

	Printf(ctx, "  Downloading %s\n", url)

	// Download to temp file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
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

// processFile extracts or copies a file based on configuration.
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
	case "gz":
		if cfg.outputName == "" {
			return "", fmt.Errorf("gz format requires WithOutputName option")
		}
		if err := ExtractGz(path, destDir, cfg.outputName); err != nil {
			return "", fmt.Errorf("extract gz: %w", err)
		}
		firstFile = filepath.Join(destDir, cfg.outputName)
	default:
		// Raw copy.
		dst := filepath.Join(destDir, filepath.Base(path))
		if err := CopyFile(path, dst); err != nil {
			return "", fmt.Errorf("copy file: %w", err)
		}
		firstFile = dst
	}

	return firstFile, nil
}

func findFirstExtractedFile(destDir string, opts []ExtractOpt) string {
	cfg := newExtractConfig(opts)

	for _, destName := range cfg.renameMap {
		return filepath.Join(destDir, destName)
	}

	return ""
}
