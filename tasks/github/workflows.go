// Package github provides GitHub-related tasks.
package github

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/goreleaser"
)

//go:embed workflows/*.tmpl
var workflowTemplates embed.FS

// PocketConfig holds configuration for the pocket workflow template.
type PocketConfig struct {
	Platforms string // comma-separated list of platforms
}

// DefaultPocketConfig returns the default pocket workflow configuration.
func DefaultPocketConfig() PocketConfig {
	return PocketConfig{
		Platforms: strings.Join(AllPlatforms(), ", "),
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

// ReleaseConfig holds configuration for the release workflow template.
type ReleaseConfig struct {
	IncludeGoreleaser bool
}

// WorkflowFlags holds flags for the Workflows task.
type WorkflowFlags struct {
	IncludeGoreleaser   bool   `flag:"include-goreleaser" usage:"include goreleaser job in release workflow"`
	IncludePocketPerjob bool   `flag:"include-pocket-perjob" usage:"include pocket-perjob workflow (excluded by default)"`
	Platforms           string `flag:"platforms" usage:"platforms for pocket.yml (comma-separated)"`
	SkipGhPages         bool   `flag:"skip-gh-pages" usage:"exclude GitHub Pages workflow"`
	SkipPocket          bool   `flag:"skip-pocket" usage:"exclude pocket workflow"`
	SkipPR              bool   `flag:"skip-pr" usage:"exclude PR workflow"`
	SkipRelease         bool   `flag:"skip-release" usage:"exclude release-please workflow"`
	SkipStale           bool   `flag:"skip-stale" usage:"exclude stale workflow"`
}

// Workflows bootstraps GitHub workflow files into .github/workflows/.
// Most workflows are included by default. Use flags to include/exclude specific ones.
var Workflows = &pk.Task{
	Name:  "github-workflows",
	Usage: "bootstrap GitHub workflow files",
	Flags: WorkflowFlags{SkipGhPages: true},
	Do:    runWorkflows,
}

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

	f := pk.GetFlags[WorkflowFlags](ctx)

	pocketConfig := DefaultPocketConfig()
	if f.Platforms != "" {
		pocketConfig.Platforms = f.Platforms
	}
	staleConfig := DefaultStaleConfig()

	// Build release config.
	releaseConfig := ReleaseConfig{IncludeGoreleaser: f.IncludeGoreleaser}

	workflowDefs := []workflowDef{
		{"gh-pages.yml.tmpl", "gh-pages.yml", nil, !f.SkipGhPages},
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !f.SkipPocket},
		{"pr.yml.tmpl", "pr.yml", nil, !f.SkipPR},
		{"release.yml.tmpl", "release.yml", releaseConfig, !f.SkipRelease},
		{"stale.yml.tmpl", "stale.yml", staleConfig, !f.SkipStale},
	}

	// Also manage pocket-perjob.yml (generated separately via per-job generation)
	allManagedFiles := []string{"pocket-perjob.yml"}
	for _, wf := range workflowDefs {
		allManagedFiles = append(allManagedFiles, wf.outFile)
	}

	// Clean up managed workflow files before generating.
	// This ensures disabled workflows are removed.
	for _, outFile := range allManagedFiles {
		destPath := filepath.Join(workflowDir, outFile)
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", outFile, err)
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

	// Write default .goreleaser.yml if goreleaser is enabled and no config exists.
	if f.IncludeGoreleaser {
		cfgPath, err := goreleaser.WriteDefaultConfig()
		if err != nil {
			return fmt.Errorf("write goreleaser config: %w", err)
		}
		if verbose {
			pk.Printf(ctx, "  Goreleaser config: %s\n", cfgPath)
		}
	}

	// Generate per-job workflow if requested.
	if f.IncludePocketPerjob {
		if err := generatePerJobWorkflow(ctx, workflowDir, verbose); err != nil {
			return fmt.Errorf("generate pocket-perjob workflow: %w", err)
		}
		copied++
	}

	if verbose && copied > 0 {
		pk.Printf(ctx, "  Bootstrapped %d workflow(s)\n", copied)
	}

	return nil
}

// perJobData holds the data for the per-job workflow template.
type perJobData struct {
	Jobs []StaticJob
}

func generatePerJobWorkflow(ctx context.Context, workflowDir string, verbose bool) error {
	plan := pk.PlanFromContext(ctx)
	if plan == nil {
		return fmt.Errorf("plan not available in context")
	}

	cfg := perJobConfigFromContext(ctx)
	jobs := GenerateStaticJobs(plan.Tasks(), cfg)

	// Read template
	tmplContent, err := workflowTemplates.ReadFile(path.Join("workflows", "pocket-perjob.yml.tmpl"))
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("pocket-perjob.yml.tmpl").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, perJobData{Jobs: jobs}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Write workflow file
	destPath := filepath.Join(workflowDir, "pocket-perjob.yml")
	if err := os.WriteFile(destPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write workflow: %w", err)
	}

	if verbose {
		pk.Printf(ctx, "  Created %s (%d jobs)\n", destPath, len(jobs))
	}

	return nil
}
