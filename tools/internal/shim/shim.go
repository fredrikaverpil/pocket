package shim

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

//go:embed templates/pok.sh.tmpl
var posixTemplate string

//go:embed templates/pok.cmd.tmpl
var windowsTemplate string

//go:embed templates/pok.ps1.tmpl
var powershellTemplate string

// shimData holds the template data for generating a shim.
type shimData struct {
	GoVersion   string
	PocketDir   string
	Context     string
	GoChecksums GoChecksums
}

// shimType represents a type of shim to generate.
type shimType struct {
	name      string // Template name for errors.
	template  string // Template content.
	extension string // File extension (empty for posix).
}

// Config configures which shims to generate.
type Config struct {
	Name       string // Shim filename (default "pok").
	Posix      bool   // Generate POSIX shell script.
	Windows    bool   // Generate Windows batch file.
	PowerShell bool   // Generate PowerShell script.
}

// GenerateShims creates wrapper scripts in root and module directories.
// It reads the Go version from pocketDir/go.mod and fetches checksums.
// Returns the list of generated shim paths relative to rootDir.
//
// Parameters:
//   - ctx: Context for HTTP requests (checksum fetching).
//   - rootDir: Git repository root (absolute path).
//   - pocketDir: Path to .pocket directory (absolute path).
//   - moduleDirs: Directories where shims should be generated (relative to rootDir).
//     If empty, shims are only generated at rootDir.
//   - cfg: Configuration specifying which shim types to generate.
func GenerateShims(ctx context.Context, rootDir, pocketDir string, moduleDirs []string, cfg Config) ([]string, error) {
	// Apply defaults.
	if cfg.Name == "" {
		cfg.Name = "pok"
	}

	// Read Go version from pocketDir/go.mod.
	goVersion, err := goVersionFromMod(pocketDir)
	if err != nil {
		return nil, fmt.Errorf("reading Go version: %w", err)
	}

	// Fetch checksums for Go downloads.
	checksums, err := fetchGoChecksums(ctx, goVersion)
	if err != nil {
		return nil, fmt.Errorf("fetching Go checksums: %w", err)
	}

	// Collect directories (always include root as ".").
	dirSet := make(map[string]bool)
	dirSet["."] = true
	for _, dir := range moduleDirs {
		if dir != "" {
			dirSet[dir] = true
		}
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)

	// Determine which shim types to generate.
	var types []shimType
	if cfg.Posix {
		types = append(types, shimType{
			name:      "posix",
			template:  posixTemplate,
			extension: "",
		})
	}
	if cfg.Windows {
		types = append(types, shimType{
			name:      "windows",
			template:  windowsTemplate,
			extension: ".cmd",
		})
	}
	if cfg.PowerShell {
		types = append(types, shimType{
			name:      "powershell",
			template:  powershellTemplate,
			extension: ".ps1",
		})
	}

	// Generate each shim type at each directory.
	var generatedPaths []string
	for _, st := range types {
		tmpl, err := template.New(st.name).Parse(st.template)
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", st.name, err)
		}

		for _, dir := range dirs {
			shimPath, err := generateShimAt(tmpl, cfg.Name, st.extension, goVersion, checksums, rootDir, dir)
			if err != nil {
				return nil, fmt.Errorf("generating %s shim at %s: %w", st.name, dir, err)
			}
			generatedPaths = append(generatedPaths, shimPath)
		}
	}

	return generatedPaths, nil
}

// generateShimAt creates a single shim at the specified directory.
// dir is relative to rootDir (e.g., ".", "proj1", "services/api").
// Returns the generated shim path relative to rootDir.
func generateShimAt(
	tmpl *template.Template,
	shimName, extension, goVersion string,
	checksums GoChecksums,
	rootDir, dir string,
) (string, error) {
	// Calculate relative path from dir back to .pocket.
	// For ".", pocketDir is ".pocket".
	// For "proj1", pocketDir is "../.pocket".
	// For "services/api", pocketDir is "../../.pocket".
	pocketDir := ".pocket"
	if dir != "." {
		// Count the depth and prepend "../" for each level.
		depth := strings.Count(dir, "/") + 1
		pocketDir = strings.Repeat("../", depth) + ".pocket"
	}

	data := shimData{
		GoVersion:   goVersion,
		PocketDir:   pocketDir,
		Context:     dir,
		GoChecksums: checksums,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing shim template: %w", err)
	}

	// Create the shim at dir within rootDir.
	shimPath := filepath.Join(rootDir, dir, shimName+extension)

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(shimPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(shimPath, buf.Bytes(), 0o755); err != nil {
		return "", fmt.Errorf("writing shim: %w", err)
	}

	// Return path relative to rootDir.
	relPath := filepath.Join(dir, shimName+extension)
	return relPath, nil
}
