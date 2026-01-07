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
	cfg = cfg.WithDefaults()

	goVersion, err := bld.ExtractGoVersion(bld.DirName)
	if err != nil {
		return fmt.Errorf("reading Go version: %w", err)
	}

	tmpl, err := template.New("shim").Parse(shimTemplate)
	if err != nil {
		return fmt.Errorf("parsing shim template: %w", err)
	}

	// Generate shims for all unique module paths.
	for _, context := range cfg.UniqueModulePaths() {
		if err := generateShim(tmpl, cfg.ShimName, goVersion, context); err != nil {
			return fmt.Errorf("generating shim for context %q: %w", context, err)
		}
	}

	return nil
}

// generateShim creates a single shim for the given context.
func generateShim(tmpl *template.Template, shimName, goVersion, context string) error {
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
		shimPath = bld.FromGitRoot(shimName)
	} else {
		// Ensure the directory exists.
		dir := bld.FromGitRoot(context)
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
func calculateBldDir(context string) string {
	if context == "." {
		return ".bld"
	}

	// Count the depth of the context path.
	depth := strings.Count(context, string(filepath.Separator)) + 1

	// Build the relative path back to root, then to .bld.
	// Allocate depth+1 for the ".." entries plus ".bld".
	parts := make([]string, depth+1)
	for i := range depth {
		parts[i] = ".."
	}
	parts[depth] = ".bld"

	return filepath.Join(parts...)
}
