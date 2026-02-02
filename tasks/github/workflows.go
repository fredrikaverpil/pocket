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
	"github.com/fredrikaverpil/pocket/pk/pcontext"
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
	verbose := pcontext.Verbose(ctx)

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
		{"pr.yml.tmpl", "pr.yml", nil, !*skipPR},
		{"release.yml.tmpl", "release.yml", nil, !*skipRelease},
		{"stale.yml.tmpl", "stale.yml", staleConfig, !*skipStale},
	}

	// Also manage pocket-matrix.yml (generated separately via static generation)
	allManagedFiles := []string{"pocket-matrix.yml"}
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

	// Generate static pocket-matrix workflow if requested
	if *includePocketMatrix {
		if err := generateStaticMatrixWorkflow(ctx, workflowDir, verbose); err != nil {
			return fmt.Errorf("generate pocket-matrix workflow: %w", err)
		}
		copied++
	}

	if verbose && copied > 0 {
		pk.Printf(ctx, "  Bootstrapped %d workflow(s)\n", copied)
	}

	return nil
}

// staticMatrixData holds the data for the static matrix workflow template.
type staticMatrixData struct {
	Jobs []StaticJob
}

func generateStaticMatrixWorkflow(ctx context.Context, workflowDir string, verbose bool) error {
	plan := pk.PlanFromContext(ctx)
	if plan == nil {
		return fmt.Errorf("plan not available in context")
	}

	cfg := matrixConfigFromContext(ctx)
	jobs := GenerateStaticJobs(plan.Tasks(), cfg)

	// Read template
	tmplContent, err := workflowTemplates.ReadFile(path.Join("workflows", "pocket-matrix-static.yml.tmpl"))
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("pocket-matrix-static.yml.tmpl").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, staticMatrixData{Jobs: jobs}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Write workflow file
	destPath := filepath.Join(workflowDir, "pocket-matrix.yml")
	if err := os.WriteFile(destPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write workflow: %w", err)
	}

	if verbose {
		pk.Printf(ctx, "  Created %s (%d jobs)\n", destPath, len(jobs))
	}

	return nil
}
