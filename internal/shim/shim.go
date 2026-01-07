// Package shim provides generation of the ./bld wrapper script.
package shim

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fredrikaverpil/bld"
)

//go:embed bld.sh.tmpl
var shimTemplate string

// shimData holds the template data for generating a shim.
type shimData struct {
	GoVersion string
	BldDir    string
	Context   string
}

// Generate creates or updates wrapper scripts for all contexts.
// It generates a shim at the root and one in each unique module directory.
func Generate(cfg bld.Config) error {
	return GenerateWithRoot(cfg, bld.GitRoot())
}

// GenerateWithRoot creates or updates wrapper scripts for all contexts
// using the specified root directory. This is useful for testing.
func GenerateWithRoot(cfg bld.Config, rootDir string) error {
	cfg = cfg.WithDefaults()

	goVersion, err := extractGoVersionFromDir(filepath.Join(rootDir, bld.DirName))
	if err != nil {
		return fmt.Errorf("reading Go version: %w", err)
	}

	tmpl, err := template.New("shim").Parse(shimTemplate)
	if err != nil {
		return fmt.Errorf("parsing shim template: %w", err)
	}

	// Generate shims for all unique module paths.
	for _, context := range cfg.UniqueModulePaths() {
		if err := generateShim(tmpl, cfg.ShimName, goVersion, context, rootDir); err != nil {
			return fmt.Errorf("generating shim for context %q: %w", context, err)
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
func generateShim(tmpl *template.Template, shimName, goVersion, context, rootDir string) error {
	// Calculate the relative path from the shim location to .bld/.
	bldDir := calculateBldDir(context)

	data := shimData{
		GoVersion: goVersion,
		BldDir:    bldDir,
		Context:   context,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing shim template: %w", err)
	}

	// Determine the shim path.
	var shimPath string
	if context == "." {
		shimPath = filepath.Join(rootDir, shimName)
	} else {
		// Ensure the directory exists.
		dir := filepath.Join(rootDir, context)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", context, err)
		}
		shimPath = filepath.Join(dir, shimName)
	}

	if err := os.WriteFile(shimPath, buf.Bytes(), 0o755); err != nil {
		return fmt.Errorf("writing shim: %w", err)
	}

	return nil
}

// calculateBldDir returns the relative path from a context directory to .bld/.
// For "." it returns ".bld", for "tests" it returns "../.bld", etc.
// Always uses forward slashes since the output is used in bash scripts.
func calculateBldDir(context string) string {
	if context == "." {
		return ".bld"
	}

	// Count the depth of the context path.
	// Handle both forward and back slashes for cross-platform compatibility.
	depth := strings.Count(context, "/") + strings.Count(context, "\\") + 1

	// Build the relative path back to root, then to .bld.
	// Use forward slashes since this is for bash scripts.
	parts := make([]string, depth+1)
	for i := range depth {
		parts[i] = ".."
	}
	parts[depth] = ".bld"

	return strings.Join(parts, "/")
}
