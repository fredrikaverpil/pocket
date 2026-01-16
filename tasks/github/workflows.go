// Package github provides GitHub-related tasks.
package github

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fredrikaverpil/pocket"
)

//go:embed workflows/*.tmpl
var workflowTemplates embed.FS

// WorkflowsOptions configures which workflows to bootstrap.
type WorkflowsOptions struct {
	SkipPocket  bool `arg:"skip-pocket"  usage:"exclude pocket workflow"`
	SkipPR      bool `arg:"skip-pr"      usage:"exclude PR workflow"`
	SkipRelease bool `arg:"skip-release" usage:"exclude release-please workflow"`
	SkipStale   bool `arg:"skip-stale"   usage:"exclude stale workflow"`
	SkipSync    bool `arg:"skip-sync"    usage:"exclude sync workflow"`
}

// PocketConfig holds configuration for the pocket workflow template.
type PocketConfig struct {
	Platforms string // comma-separated list of platforms (e.g., "ubuntu-latest, macos-latest")
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
var Workflows = pocket.Task("github-workflows", "bootstrap GitHub workflow files",
	workflowsCmd(),
	pocket.Opts(WorkflowsOptions{}),
)

func workflowsCmd() pocket.Runnable {
	return pocket.Do(runWorkflows)
}

func runWorkflows(ctx context.Context) error {
	opts := pocket.Options[WorkflowsOptions](ctx)
	verbose := pocket.Verbose(ctx)

	// Include all workflows by default, use Skip* to exclude specific ones

	// Ensure .github/workflows directory exists
	workflowDir := pocket.FromGitRoot(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return fmt.Errorf("create workflows dir: %w", err)
	}

	if verbose {
		pocket.Printf(ctx, "  Target directory: %s\n", workflowDir)
	}

	// Define workflows to process
	type workflowDef struct {
		tmplFile string
		outFile  string
		data     any
		include  bool
	}

	pocketConfig := DefaultPocketConfig()
	staleConfig := DefaultStaleConfig()

	workflowDefs := []workflowDef{
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !opts.SkipPocket},
		{"pr.yml.tmpl", "pr.yml", nil, !opts.SkipPR},
		{"release.yml.tmpl", "release.yml", nil, !opts.SkipRelease},
		{"stale.yml.tmpl", "stale.yml", staleConfig, !opts.SkipStale},
		{"sync.yml.tmpl", "sync.yml", nil, !opts.SkipSync},
	}

	copied := 0
	for _, wf := range workflowDefs {
		if !wf.include {
			continue
		}

		destPath := filepath.Join(workflowDir, wf.outFile)

		// Read and parse template
		tmplContent, err := workflowTemplates.ReadFile(filepath.Join("workflows", wf.tmplFile))
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

		pocket.Printf(ctx, "  Created %s\n", destPath)
		copied++
	}

	if copied > 0 {
		pocket.Printf(ctx, "  Bootstrapped %d workflow(s)\n", copied)
	}

	return nil
}
