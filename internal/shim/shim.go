// Package shim provides generation of the ./pok wrapper scripts.
package shim

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	pocket "github.com/fredrikaverpil/pocket"
)

//go:embed pok.sh.tmpl
var posixTemplate string

//go:embed pok.cmd.tmpl
var windowsTemplate string

//go:embed pok.ps1.tmpl
var powershellTemplate string

// shimData holds the template data for generating a shim.
type shimData struct {
	GoVersion   string
	PocketDir   string
	Context     string
	GoChecksums GoChecksums // SHA256 checksums keyed by "os-arch"
}

// shimType represents a type of shim to generate.
type shimType struct {
	name      string // Template name for errors.
	template  string // Template content.
	extension string // File extension (empty for posix).
}

// Generate creates or updates wrapper scripts for all contexts.
// It generates shims at the root and one in each unique module directory.
// Returns the list of generated shim paths relative to the git root.
func Generate(cfg pocket.Config) ([]string, error) {
	return GenerateWithRoot(cfg, pocket.GitRoot())
}

// GenerateWithRoot creates or updates wrapper scripts for all contexts
// using the specified root directory. This is useful for testing.
// Returns the list of generated shim paths relative to the root directory.
func GenerateWithRoot(cfg pocket.Config, rootDir string) ([]string, error) {
	cfg = cfg.WithDefaults()

	goVersion, err := pocket.GoVersionFromDir(filepath.Join(rootDir, pocket.DirName))
	if err != nil {
		return nil, fmt.Errorf("reading Go version: %w", err)
	}

	// Fetch checksums for Go downloads.
	checksums, err := FetchGoChecksums(context.Background(), goVersion)
	if err != nil {
		return nil, fmt.Errorf("fetching Go checksums: %w", err)
	}

	// Determine which shim types to generate.
	var types []shimType
	if cfg.Shim.Posix {
		types = append(types, shimType{
			name:      "posix",
			template:  posixTemplate,
			extension: "",
		})
	}
	if cfg.Shim.Windows {
		types = append(types, shimType{
			name:      "windows",
			template:  windowsTemplate,
			extension: ".cmd",
		})
	}
	if cfg.Shim.PowerShell {
		types = append(types, shimType{
			name:      "powershell",
			template:  powershellTemplate,
			extension: ".ps1",
		})
	}

	// Collect all module directories from the config.
	var moduleDirs []string
	if cfg.AutoRun != nil {
		moduleDirs = pocket.CollectModuleDirectories(cfg.AutoRun)
	} else {
		moduleDirs = []string{"."}
	}

	// Generate each shim type at each module directory.
	var generatedPaths []string
	for _, st := range types {
		tmpl, err := template.New(st.name).Parse(st.template)
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", st.name, err)
		}

		for _, moduleDir := range moduleDirs {
			shimPath, err := generateShimAt(tmpl, cfg.Shim.Name, st.extension, goVersion, checksums, rootDir, moduleDir)
			if err != nil {
				return nil, fmt.Errorf("generating %s shim at %s: %w", st.name, moduleDir, err)
			}
			generatedPaths = append(generatedPaths, shimPath)
		}
	}

	return generatedPaths, nil
}

// generateShimAt creates a single shim at the specified module directory.
// moduleDir is relative to rootDir (e.g., ".", "proj1", "services/api").
// Returns the generated shim path relative to rootDir.
func generateShimAt(
	tmpl *template.Template,
	shimName, extension, goVersion string,
	checksums GoChecksums,
	rootDir, moduleDir string,
) (string, error) {
	// Calculate relative path from moduleDir back to .pocket.
	// For ".", pocketDir is ".pocket".
	// For "proj1", pocketDir is "../.pocket".
	// For "services/api", pocketDir is "../../.pocket".
	pocketDir := ".pocket"
	if moduleDir != "." {
		// Count the depth and prepend "../" for each level.
		depth := strings.Count(moduleDir, "/") + 1
		pocketDir = strings.Repeat("../", depth) + ".pocket"
	}

	data := shimData{
		GoVersion:   goVersion,
		PocketDir:   pocketDir,
		Context:     moduleDir,
		GoChecksums: checksums,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing shim template: %w", err)
	}

	// Create the shim at moduleDir within rootDir.
	shimPath := filepath.Join(rootDir, moduleDir, shimName+extension)

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(shimPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(shimPath, buf.Bytes(), 0o755); err != nil {
		return "", fmt.Errorf("writing shim: %w", err)
	}

	// Return path relative to rootDir.
	relPath := filepath.Join(moduleDir, shimName+extension)
	return relPath, nil
}
