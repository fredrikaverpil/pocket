// Package shim provides generation of the ./pok wrapper scripts.
package shim

import (
	"bytes"
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
	GoVersion string
	PocketDir string
	Context   string
}

// shimType represents a type of shim to generate.
type shimType struct {
	name      string // Template name for errors.
	template  string // Template content.
	extension string // File extension (empty for posix).
	pathSep   string // Path separator to use in template output.
}

// Generate creates or updates wrapper scripts for all contexts.
// It generates shims at the root and one in each unique module directory.
func Generate(cfg pocket.Config) error {
	return GenerateWithRoot(cfg, pocket.GitRoot())
}

// GenerateWithRoot creates or updates wrapper scripts for all contexts
// using the specified root directory. This is useful for testing.
func GenerateWithRoot(cfg pocket.Config, rootDir string) error {
	cfg = cfg.WithDefaults()

	goVersion, err := extractGoVersionFromDir(filepath.Join(rootDir, pocket.DirName))
	if err != nil {
		return fmt.Errorf("reading Go version: %w", err)
	}

	// Determine which shim types to generate.
	var types []shimType
	if cfg.Shim.Posix {
		types = append(types, shimType{
			name:      "posix",
			template:  posixTemplate,
			extension: "",
			pathSep:   "/",
		})
	}
	if cfg.Shim.Windows {
		types = append(types, shimType{
			name:      "windows",
			template:  windowsTemplate,
			extension: ".cmd",
			pathSep:   "\\",
		})
	}
	if cfg.Shim.PowerShell {
		types = append(types, shimType{
			name:      "powershell",
			template:  powershellTemplate,
			extension: ".ps1",
			pathSep:   "\\",
		})
	}

	// Generate each shim type for all contexts.
	for _, st := range types {
		tmpl, err := template.New(st.name).Parse(st.template)
		if err != nil {
			return fmt.Errorf("parsing %s template: %w", st.name, err)
		}

		for _, context := range pocket.AllModulePaths(cfg) {
			if err := generateShim(tmpl, cfg.Shim.Name, st.extension, st.pathSep, goVersion, context, rootDir); err != nil {
				return fmt.Errorf("generating %s shim for context %q: %w", st.name, context, err)
			}
		}
	}

	return nil
}

// extractGoVersionFromDir reads a go.mod file from the given directory
// and returns the Go version specified in the "go" directive.
func extractGoVersionFromDir(dir string) (string, error) {
	gomodPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	// Parse the go directive from the file.
	// Look for a line starting with "go " followed by the version.
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "go "); ok {
			version := after
			return strings.TrimSpace(version), nil
		}
	}

	return "", fmt.Errorf("no go directive in %s", gomodPath)
}

// generateShim creates a single shim for the given context.
func generateShim(tmpl *template.Template, shimName, extension, pathSep, goVersion, context, rootDir string) error {
	// Calculate the relative path from the shim location to .pocket/.
	pocketDir := calculatePocketDir(context, pathSep)

	data := shimData{
		GoVersion: goVersion,
		PocketDir: pocketDir,
		Context:   context,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing shim template: %w", err)
	}

	// Build the shim filename.
	shimFilename := shimName + extension

	// Determine the shim path.
	var shimPath string
	if context == "." {
		shimPath = filepath.Join(rootDir, shimFilename)
	} else {
		// Ensure the directory exists.
		dir := filepath.Join(rootDir, context)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", context, err)
		}
		shimPath = filepath.Join(dir, shimFilename)
	}

	if err := os.WriteFile(shimPath, buf.Bytes(), 0o755); err != nil {
		return fmt.Errorf("writing shim: %w", err)
	}

	return nil
}

// calculatePocketDir returns the relative path from a context directory to .pocket/.
// For "." it returns ".pocket", for "tests" it returns "../.pocket", etc.
// Uses the provided path separator for the output.
func calculatePocketDir(context, pathSep string) string {
	if context == "." {
		return ".pocket"
	}

	// Count the depth of the context path.
	// Handle both forward and back slashes for cross-platform compatibility.
	depth := strings.Count(context, "/") + strings.Count(context, "\\") + 1

	// Build the relative path back to root, then to .pocket.
	parts := make([]string, depth+1)
	for i := range depth {
		parts[i] = ".."
	}
	parts[depth] = ".pocket"

	return strings.Join(parts, pathSep)
}
