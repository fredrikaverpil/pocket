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
	"github.com/fredrikaverpil/pocket/pk/conventionalcommits"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/goreleaser"
)

//go:embed workflows/*.tmpl
var workflowTemplates embed.FS

// PocketConfig holds configuration for the pocket workflow template.
type PocketConfig struct {
	Platforms    string // comma-separated list of platforms
	GitDiff      bool
	CommitsCheck bool
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

// SelfUpdateConfig holds configuration for the self-update workflow template.
type SelfUpdateConfig struct {
	CronSchedule string
}

// DefaultSelfUpdateConfig returns the default self-update workflow configuration.
func DefaultSelfUpdateConfig() SelfUpdateConfig {
	return SelfUpdateConfig{
		CronSchedule: "0 0 1 * *",
	}
}

// ReleaseConfig holds configuration for the release workflow template.
type ReleaseConfig struct {
	IncludeGoreleaser bool
}

// WorkflowFlags holds flags for the Workflows task.
// Pointer bools use nil = "not set" (inherit default), non-nil = explicit override.
type WorkflowFlags struct {
	// CLI flags — each workflow is a *bool toggled on/off.
	ConventionalCommitWorkflow *bool `flag:"conventional-commit-workflow" usage:"conventional commit PR"`
	GhPagesWorkflow            *bool `flag:"gh-pages-workflow"            usage:"GitHub Pages"`
	GoReleaserWorkflow         *bool `flag:"goreleaser-workflow"          usage:"GoReleaser release"`
	PerPocketTaskJob           *bool `flag:"per-pocket-task-job"          usage:"per-task jobs"`
	ReleasePleaseWorkflow      *bool `flag:"release-please-workflow"      usage:"release-please"`
	SelfUpdateWorkflow         *bool `flag:"self-update-workflow"         usage:"self-update cron"`
	StaleWorkflow              *bool `flag:"stale-workflow"               usage:"stale issues"`
	GitDiff                    *bool `flag:"git-diff"                     usage:"check uncommitted changes"`
	CommitsCheck               *bool `flag:"commits-check"                usage:"validate conventional commits"`

	// Programmatic-only fields (no flag tag — set via pk.WithFlags, not CLI).
	Platforms               []Platform
	PerPocketTaskJobOptions map[string]PerPocketTaskJobOption

	// ExternalWorkflows lists workflow filenames not managed by Pocket
	// (e.g. "renovate.yml"). The task validates these files exist in
	// .github/workflows/ and fails if any are missing.
	ExternalWorkflows []string
}

// Workflows bootstraps GitHub workflow files into .github/workflows/.
// Most workflows are included by default. Use flags to include/exclude specific ones.
var Workflows = &pk.Task{
	Name:  "github-workflows",
	Usage: "bootstrap GitHub workflow files",
	Flags: WorkflowFlags{
		ConventionalCommitWorkflow: new(true),
		ReleasePleaseWorkflow:      new(true),
		SelfUpdateWorkflow:         new(true),
		StaleWorkflow:              new(true),
		GitDiff:                    new(true),
		CommitsCheck:               new(true),
		Platforms:                  AllPlatforms(),
	},
	Do: runWorkflows,
}

func runWorkflows(ctx context.Context) error {
	verbose := run.Verbose(ctx)

	// Ensure .github/workflows directory exists
	workflowDir := repopath.FromGitRoot(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return fmt.Errorf("create workflows dir: %w", err)
	}

	if verbose {
		run.Printf(ctx, "  Target directory: %s\n", workflowDir)
	}

	// Define workflows to process
	type workflowDef struct {
		tmplFile string
		outFile  string
		data     any
		include  bool
	}

	f := run.GetFlags[WorkflowFlags](ctx)

	pocketConfig := DefaultPocketConfig()
	if len(f.Platforms) > 0 {
		pocketConfig.Platforms = strings.Join(f.Platforms, ", ")
	}
	pocketConfig.GitDiff = boolVal(f.GitDiff)
	pocketConfig.CommitsCheck = boolVal(f.CommitsCheck)
	staleConfig := DefaultStaleConfig()

	// Build release config.
	releaseConfig := ReleaseConfig{IncludeGoreleaser: boolVal(f.GoReleaserWorkflow)}

	selfUpdateConfig := DefaultSelfUpdateConfig()

	workflowDefs := []workflowDef{
		{"gh-pages.yml.tmpl", "gh-pages.yml", nil, boolVal(f.GhPagesWorkflow)},
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !boolVal(f.PerPocketTaskJob)},
		{
			"pr.yml.tmpl",
			"pr.yml",
			struct{ Types []string }{Types: conventionalcommits.Types},
			boolVal(f.ConventionalCommitWorkflow),
		},
		{"release.yml.tmpl", "release.yml", releaseConfig, boolVal(f.ReleasePleaseWorkflow)},
		{"self-update.yml.tmpl", "self-update.yml", selfUpdateConfig, boolVal(f.SelfUpdateWorkflow)},
		{"stale.yml.tmpl", "stale.yml", staleConfig, boolVal(f.StaleWorkflow)},
	}

	// Also manage pocket-pertask.yml (generated separately via per-task generation)
	allManagedFiles := []string{"pocket-pertask.yml"}
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
			run.Printf(ctx, "  Created %s\n", destPath)
		}
		copied++
	}

	// Write default .goreleaser.yml if goreleaser is enabled and no config exists.
	if boolVal(f.GoReleaserWorkflow) {
		cfgPath, err := goreleaser.WriteDefaultConfig()
		if err != nil {
			return fmt.Errorf("write goreleaser config: %w", err)
		}
		if verbose {
			run.Printf(ctx, "  Goreleaser config: %s\n", cfgPath)
		}
	}

	// Generate per-task workflow if requested.
	if boolVal(f.PerPocketTaskJob) {
		if err := generatePerTaskWorkflow(ctx, workflowDir, verbose); err != nil {
			return fmt.Errorf("generate pocket-pertask workflow: %w", err)
		}
		copied++
	}

	// Validate that declared external workflows exist on disk.
	for _, name := range f.ExternalWorkflows {
		extPath := filepath.Join(workflowDir, name)
		if _, err := os.Stat(extPath); err != nil {
			return fmt.Errorf(
				"external workflow %q not found in %s — remove it from ExternalWorkflows: %w",
				name,
				workflowDir,
				err,
			)
		}
		if verbose {
			run.Printf(ctx, "  Verified external workflow: %s\n", name)
		}
	}

	if verbose && copied > 0 {
		run.Printf(ctx, "  Bootstrapped %d workflow(s)\n", copied)
	}

	return nil
}

// perTaskData holds the data for the per-task workflow template.
type perTaskData struct {
	Jobs []StaticJob
}

func generatePerTaskWorkflow(ctx context.Context, workflowDir string, verbose bool) error {
	plan := pk.PlanFromContext(ctx)
	if plan == nil {
		return fmt.Errorf("plan not available in context")
	}

	flags := run.GetFlags[WorkflowFlags](ctx)
	jobs := GenerateStaticJobs(plan.Tasks(), flags)

	// Read template
	tmplContent, err := workflowTemplates.ReadFile(path.Join("workflows", "pocket-pertask.yml.tmpl"))
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("pocket-pertask.yml.tmpl").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, perTaskData{Jobs: jobs}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Write workflow file
	destPath := filepath.Join(workflowDir, "pocket-pertask.yml")
	if err := os.WriteFile(destPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write workflow: %w", err)
	}

	if verbose {
		run.Printf(ctx, "  Created %s (%d jobs)\n", destPath, len(jobs))
	}

	return nil
}
