package pocket

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractOpt configures extraction behavior.
type ExtractOpt func(*extractConfig)

type extractConfig struct {
	// renameMap maps source paths (or base names) to destination names.
	// If a file matches a key, it's extracted with the corresponding value as name.
	renameMap map[string]string
	// flatten extracts all files to the root of destDir, ignoring directory structure.
	flatten bool
}

func newExtractConfig(opts []ExtractOpt) *extractConfig {
	cfg := &extractConfig{
		renameMap: make(map[string]string),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithRenameFile extracts a specific file and optionally renames it.
// srcPath can be a full path within the archive or just the base name.
// If destName is empty, the original base name is preserved.
// Multiple calls accumulate files to extract.
//
// Example:
//
//	ExtractTarGz(src, dest,
//	    WithRenameFile("golangci-lint-1.55.0-linux-amd64/golangci-lint", "golangci-lint"),
//	)
func WithRenameFile(srcPath, destName string) ExtractOpt {
	return func(cfg *extractConfig) {
		if destName == "" {
			destName = filepath.Base(srcPath)
		}
		cfg.renameMap[srcPath] = destName
	}
}

// WithExtractFile extracts only the specified file (by base name).
// The file is extracted with its original name.
// Multiple calls accumulate files to extract.
func WithExtractFile(name string) ExtractOpt {
	return func(cfg *extractConfig) {
		cfg.renameMap[name] = name
	}
}

// WithFlatten flattens directory structure, extracting all files to destDir root.
// File names are preserved but directory paths are discarded.
func WithFlatten() ExtractOpt {
	return func(cfg *extractConfig) {
		cfg.flatten = true
	}
}

// ExtractTarGz extracts a .tar.gz archive to destDir.
// If no options are provided, all files are extracted preserving directory structure.
// Use WithRenameFile or WithExtractFile to limit extraction to specific files.
func ExtractTarGz(src, destDir string, opts ...ExtractOpt) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	return extractTarReader(tar.NewReader(gzr), destDir, newExtractConfig(opts))
}

// ExtractTar extracts a .tar archive to destDir.
// If no options are provided, all files are extracted preserving directory structure.
// Use WithRenameFile or WithExtractFile to limit extraction to specific files.
func ExtractTar(src, destDir string, opts ...ExtractOpt) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	return extractTarReader(tar.NewReader(f), destDir, newExtractConfig(opts))
}

// ExtractZip extracts a .zip archive to destDir.
// If no options are provided, all files are extracted preserving directory structure.
// Use WithRenameFile or WithExtractFile to limit extraction to specific files.
func ExtractZip(src, destDir string, opts ...ExtractOpt) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	cfg := newExtractConfig(opts)

	for _, f := range r.File {
		name := f.Name
		baseName := filepath.Base(name)

		// Determine output name based on config.
		outputName, shouldExtract := resolveOutputName(name, baseName, cfg)
		if !shouldExtract {
			continue
		}

		target := filepath.Join(destDir, outputName)

		// Security check: ensure we don't escape destDir.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path: %s", name)
		}

		if f.FileInfo().IsDir() {
			if cfg.flatten || len(cfg.renameMap) > 0 {
				continue
			}
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}

		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open file in archive: %w", err)
	}
	defer rc.Close()

	outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func extractTarReader(tr *tar.Reader, destDir string, cfg *extractConfig) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		name := header.Name
		baseName := filepath.Base(name)

		// Determine output name based on config.
		outputName, shouldExtract := resolveOutputName(name, baseName, cfg)
		if !shouldExtract {
			continue
		}

		target := filepath.Join(destDir, outputName)

		// Security check: ensure we don't escape destDir.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if cfg.flatten || len(cfg.renameMap) > 0 {
				continue
			}
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent directory: %w", err)
			}
			if err := extractTarFile(tr, target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractTarFile(tr *tar.Reader, target string, mode os.FileMode) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, tr); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// resolveOutputName determines the output file name based on extraction config.
// Returns the output name and whether the file should be extracted.
func resolveOutputName(fullPath, baseName string, cfg *extractConfig) (string, bool) {
	// If no rename map, extract all (optionally flattening).
	if len(cfg.renameMap) == 0 {
		if cfg.flatten {
			return baseName, true
		}
		return fullPath, true
	}

	// Check full path first, then base name.
	if destName, ok := cfg.renameMap[fullPath]; ok {
		return destName, true
	}
	if destName, ok := cfg.renameMap[baseName]; ok {
		return destName, true
	}

	return "", false
}
