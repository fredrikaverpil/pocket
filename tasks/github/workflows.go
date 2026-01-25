// Package github provides GitHub-related tasks.
package github

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/fredrikaverpil/pocket/pk"
)

//go:embed workflows/*.tmpl
var workflowTemplates embed.FS

// Workflow flags.
var (
	workflowFlags = flag.NewFlagSet("github-workflows", flag.ContinueOnError)

	skipPocket  = workflowFlags.Bool("skip-pocket", false, "exclude pocket workflow")
	skipPR      = workflowFlags.Bool("skip-pr", false, "exclude PR workflow")
	skipRelease = workflowFlags.Bool("skip-release", false, "exclude release-please workflow")
	skipStale   = workflowFlags.Bool("skip-stale", false, "exclude stale workflow")

	includePocketMatrix = workflowFlags.Bool(
		"include-pocket-matrix",
		false,
		"include pocket-matrix workflow (excluded by default)",
	)

	platforms = workflowFlags.String("platforms", "", "platforms for pocket.yml (comma-separated)")
)

// PocketConfig holds configuration for the pocket workflow template.
type PocketConfig struct {
	Platforms string // comma-separated list of platforms
}

// DefaultPocketConfig returns the default pocket workflow configuration.
func DefaultPocketConfig() PocketConfig {
	return PocketConfig{
		Platforms: "ubuntu-latest, macos-latest, windows-latest",
	}
}

// StaleConfig holds configuration for the stale workflow template.
type StaleConfig struct {
	DaysBeforeStale int
	DaysBeforeClose int
	ExemptLabels    string
}

// DefaultStaleConfig returns the default stale workflow configuration.
func DefaultStaleConfig() StaleConfig {
	return StaleConfig{
		DaysBeforeStale: 30,
		DaysBeforeClose: 30,
		ExemptLabels:    "dependencies,pinned,bug",
	}
}

// Workflows bootstraps GitHub workflow files into .github/workflows/.
// By default, all workflows are copied. Use flags to select specific ones.
var Workflows = pk.NewTask(
	"github-workflows",
	"bootstrap GitHub workflow files",
	workflowFlags,
	pk.Do(runWorkflows),
)

func runWorkflows(ctx context.Context) error {
	verbose := pk.Verbose(ctx)

	// Ensure .github/workflows directory exists
	workflowDir := pk.FromGitRoot(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return fmt.Errorf("create workflows dir: %w", err)
	}

	if verbose {
		pk.Printf(ctx, "  Target directory: %s\n", workflowDir)
	}

	// Define workflows to process
	type workflowDef struct {
		tmplFile string
		outFile  string
		data     any
		include  bool
	}

	pocketConfig := DefaultPocketConfig()
	if *platforms != "" {
		pocketConfig.Platforms = *platforms
	}
	staleConfig := DefaultStaleConfig()

	workflowDefs := []workflowDef{
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !*skipPocket},
		{"pocket-matrix.yml.tmpl", "pocket-matrix.yml", nil, *includePocketMatrix},
		{"pr.yml.tmpl", "pr.yml", nil, !*skipPR},
		{"release.yml.tmpl", "release.yml", nil, !*skipRelease},
		{"stale.yml.tmpl", "stale.yml", staleConfig, !*skipStale},
	}

	// Clean up managed workflow files before generating.
	// This ensures disabled workflows are removed.
	for _, wf := range workflowDefs {
		destPath := filepath.Join(workflowDir, wf.outFile)
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", wf.outFile, err)
		}
	}

	copied := 0
	for _, wf := range workflowDefs {
		if !wf.include {
			continue
		}

		destPath := filepath.Join(workflowDir, wf.outFile)

		// Read and parse template
		// NOTE: Use path.Join (not filepath.Join) because embed.FS always uses forward slashes.
		tmplContent, err := workflowTemplates.ReadFile(path.Join("workflows", wf.tmplFile))
		if err != nil {
			return fmt.Errorf("read template %s: %w", wf.tmplFile, err)
		}

		var content []byte
		if wf.data != nil {
			// Render template with data
			tmpl, err := template.New(wf.tmplFile).Parse(string(tmplContent))
			if err != nil {
				return fmt.Errorf("parse template %s: %w", wf.tmplFile, err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, wf.data); err != nil {
				return fmt.Errorf("execute template %s: %w", wf.tmplFile, err)
			}
			content = buf.Bytes()
		} else {
			// No templating needed, use as-is
			content = tmplContent
		}

		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", wf.outFile, err)
		}

		if verbose {
			pk.Printf(ctx, "  Created %s\n", destPath)
		}
		copied++
	}

	if verbose && copied > 0 {
		pk.Printf(ctx, "  Bootstrapped %d workflow(s)\n", copied)
	}

	return nil
}
