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

// Flag names for the Workflows task.
const (
	FlagIncludePocketPerjob = "include-pocket-perjob"
	FlagPlatforms           = "platforms"
	FlagSkipGhPages         = "skip-gh-pages"
	FlagSkipPocket          = "skip-pocket"
	FlagSkipPR              = "skip-pr"
	FlagSkipRelease         = "skip-release"
	FlagSkipStale           = "skip-stale"
)

// Workflows bootstraps GitHub workflow files into .github/workflows/.
// By default, all workflows are copied. Use flags to select specific ones.
var Workflows = &pk.Task{
	Name:  "github-workflows",
	Usage: "bootstrap GitHub workflow files",
	Flags: map[string]pk.FlagDef{
		FlagIncludePocketPerjob: {Default: false, Usage: "include pocket-perjob workflow (excluded by default)"},
		FlagPlatforms:           {Default: "", Usage: "platforms for pocket.yml (comma-separated)"},
		FlagSkipGhPages:         {Default: false, Usage: "exclude GitHub Pages workflow"},
		FlagSkipPocket:          {Default: false, Usage: "exclude pocket workflow"},
		FlagSkipPR:              {Default: false, Usage: "exclude PR workflow"},
		FlagSkipRelease:         {Default: false, Usage: "exclude release-please workflow"},
		FlagSkipStale:           {Default: false, Usage: "exclude stale workflow"},
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

	pocketConfig := DefaultPocketConfig()
	if p := pk.GetFlag[string](ctx, FlagPlatforms); p != "" {
		pocketConfig.Platforms = p
	}
	staleConfig := DefaultStaleConfig()

	workflowDefs := []workflowDef{
		{"gh-pages.yml.tmpl", "gh-pages.yml", nil, !pk.GetFlag[bool](ctx, FlagSkipGhPages)},
		{"pocket.yml.tmpl", "pocket.yml", pocketConfig, !pk.GetFlag[bool](ctx, FlagSkipPocket)},
		{"pr.yml.tmpl", "pr.yml", nil, !pk.GetFlag[bool](ctx, FlagSkipPR)},
		{"release.yml.tmpl", "release.yml", nil, !pk.GetFlag[bool](ctx, FlagSkipRelease)},
		{"stale.yml.tmpl", "stale.yml", staleConfig, !pk.GetFlag[bool](ctx, FlagSkipStale)},
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

	// Generate per-job workflow if requested
	if pk.GetFlag[bool](ctx, FlagIncludePocketPerjob) {
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
