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
	// CLI flags — each workflow is a bool toggled on/off.
	ConventionalCommitWorkflow bool `flag:"conventional-commit-workflow" usage:"conventional commit PR"`
	GhPagesWorkflow            bool `flag:"gh-pages-workflow"            usage:"GitHub Pages"`
	GoReleaserWorkflow         bool `flag:"goreleaser-workflow"          usage:"GoReleaser release"`
	PerPocketTaskJob           bool `flag:"per-pocket-task-job"          usage:"per-task jobs"`
	ReleasePleaseWorkflow      bool `flag:"release-please-workflow"      usage:"release-please"`
	StaleWorkflow              bool `flag:"stale-workflow"               usage:"stale issues"`
	GitDiff                    bool `flag:"git-diff"                     usage:"check uncommitted changes"`

	// Programmatic-only fields (no flag tag — set via pk.WithFlags, not CLI).
	Platforms               []Platform
	PerPocketTaskJobOptions map[string]PerPocketTaskJobOption
}

// Workflows bootstraps GitHub workflow files into .github/workflows/.
// Most workflows are included by default. Use flags to include/exclude specific ones.
var Workflows = &pk.Task{
	Name:  "github-workflows",
	Usage: "bootstrap GitHub workflow files",
	Flags: WorkflowFlags{
		ConventionalCommitWorkflow: true,
		ReleasePleaseWorkflow:      true,
		StaleWorkflow:              true,
		GitDiff:                    true,
		Platforms:                  AllPlatforms(),
	},
	Do: runWorkflows,
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
	if len(f.Platforms) > 0 {
		pocketConfig.Platforms = strings.Join(f.Platforms, ", ")
	}
	staleConfig := DefaultStaleConfig()

	// Build release config.
	releaseConfig := ReleaseConfig{IncludeGoreleaser: f.GoReleaserWorkflow}

	workflowDefs := []workflowDef{
		{"gh-pages.yml.tmpl", "gh-pages.yml", nil, f.GhPagesWorkflow},
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !f.PerPocketTaskJob},
		{"pr.yml.tmpl", "pr.yml", nil, f.ConventionalCommitWorkflow},
		{"release.yml.tmpl", "release.yml", releaseConfig, f.ReleasePleaseWorkflow},
		{"stale.yml.tmpl", "stale.yml", staleConfig, f.StaleWorkflow},
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
	if f.GoReleaserWorkflow {
		cfgPath, err := goreleaser.WriteDefaultConfig()
		if err != nil {
			return fmt.Errorf("write goreleaser config: %w", err)
		}
		if verbose {
			pk.Printf(ctx, "  Goreleaser config: %s\n", cfgPath)
		}
	}

	// Generate per-job workflow if requested.
	if f.PerPocketTaskJob {
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

	flags := pk.GetFlags[WorkflowFlags](ctx)
	jobs := GenerateStaticJobs(plan.Tasks(), flags)

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
