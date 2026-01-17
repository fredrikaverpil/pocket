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

// Shell represents a shell configuration for the CI matrix.
type Shell string

const (
	ShellBash       Shell = "bash"
	ShellZsh        Shell = "zsh"
	ShellFish       Shell = "fish"
	ShellPowershell Shell = "pwsh"
	ShellCmd        Shell = "cmd"
)

// Platform represents a GitHub Actions runner.
type Platform string

const (
	PlatformUbuntu  Platform = "ubuntu-latest"
	PlatformMacOS   Platform = "macos-latest"
	PlatformWindows Platform = "windows-latest"
)

// MatrixEntry represents a single OS/shell combination in the matrix.
type MatrixEntry struct {
	OS    Platform
	Shell Shell
	Pok   string // The pok command to use (./pok, ./pok.ps1, etc.)
}

// CIOptions configures the github-ci task.
type CIOptions struct {
	// SplitTasks runs each task as a separate parallel job.
	SplitTasks bool `arg:"split-tasks" usage:"run each task as a separate parallel job"`
	// FailFast stops all jobs if one fails.
	FailFast bool `arg:"fail-fast" usage:"stop all jobs if one fails"`
	// Tasks is a comma-separated list of tasks to run (only with split-tasks).
	Tasks string `arg:"tasks" usage:"comma-separated list of tasks to run in parallel jobs"`
	// Platforms is a comma-separated list of platforms.
	Platforms string `arg:"platforms" usage:"platforms: ubuntu-latest,macos-latest,windows-latest"`
	// Shells is a comma-separated list of shells to test.
	Shells string `arg:"shells" usage:"shells: bash,zsh,pwsh (platform-appropriate defaults)"`
	// InstallShells installs additional shells (zsh, fish) on Linux.
	InstallShells bool `arg:"install-shells" usage:"install zsh and fish on Linux"`
}

// CIConfig holds the processed configuration for the CI workflow template.
type CIConfig struct {
	Matrix        []MatrixEntry
	Tasks         []string
	SplitTasks    bool
	FailFast      bool
	InstallShells bool
}

// DefaultCIConfig returns the default CI workflow configuration.
// It creates a matrix with sensible defaults for each platform.
func DefaultCIConfig() CIConfig {
	return CIConfig{
		Matrix: []MatrixEntry{
			{OS: PlatformUbuntu, Shell: ShellBash, Pok: "./pok"},
			{OS: PlatformMacOS, Shell: ShellBash, Pok: "./pok"},
			{OS: PlatformWindows, Shell: ShellPowershell, Pok: "./pok.ps1"},
		},
		SplitTasks:    false,
		FailFast:      false,
		InstallShells: false,
	}
}

// CI generates a smart GitHub Actions workflow for Pocket.
// It supports multiple platforms, shells, and can run tasks in parallel.
//
// Example usage:
//
//	./pok github-ci                          # Default: 3 platforms, default shells
//	./pok github-ci -split-tasks -tasks="go-lint,go-test"  # Split into parallel jobs
//	./pok github-ci -shells=bash,zsh         # Test with multiple shells
//	./pok github-ci -platforms=ubuntu-latest # Single platform
var CI = pocket.Task("github-ci", "generate smart GitHub Actions workflow",
	ciCmd(),
	pocket.Opts(CIOptions{}),
)

func ciCmd() pocket.Runnable {
	return pocket.Do(runCI)
}

func runCI(ctx context.Context) error {
	opts := pocket.Options[CIOptions](ctx)
	verbose := pocket.Verbose(ctx)

	config := buildCIConfig(opts)

	// Ensure .github/workflows directory exists
	workflowDir := pocket.FromGitRoot(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return fmt.Errorf("create workflows dir: %w", err)
	}

	if verbose {
		pocket.Printf(ctx, "  Target directory: %s\n", workflowDir)
		pocket.Printf(ctx, "  Matrix entries: %d\n", len(config.Matrix))
		if config.SplitTasks {
			pocket.Printf(ctx, "  Tasks: %v\n", config.Tasks)
		}
	}

	// Read and parse template
	tmplContent, err := workflowTemplates.ReadFile("workflows/pocket-ci.yml.tmpl")
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	tmpl, err := template.New("pocket-ci.yml.tmpl").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	destPath := filepath.Join(workflowDir, "pocket-ci.yml")
	if err := os.WriteFile(destPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write pocket-ci.yml: %w", err)
	}

	pocket.Printf(ctx, "  Created %s\n", destPath)

	return nil
}

func buildCIConfig(opts CIOptions) CIConfig {
	config := DefaultCIConfig()
	config.SplitTasks = opts.SplitTasks
	config.FailFast = opts.FailFast
	config.InstallShells = opts.InstallShells

	// Parse tasks if provided
	if opts.Tasks != "" {
		config.Tasks = splitAndTrim(opts.Tasks)
	}

	// Build custom matrix if platforms or shells are specified
	if opts.Platforms != "" || opts.Shells != "" {
		config.Matrix = buildMatrix(opts)
	}

	return config
}

func buildMatrix(opts CIOptions) []MatrixEntry {
	// Parse platforms
	platforms := []Platform{PlatformUbuntu, PlatformMacOS, PlatformWindows}
	if opts.Platforms != "" {
		platforms = nil
		for _, p := range splitAndTrim(opts.Platforms) {
			platforms = append(platforms, Platform(p))
		}
	}

	// Parse shells or use platform defaults
	var matrix []MatrixEntry

	if opts.Shells != "" {
		// User specified shells - apply to all platforms where valid
		shells := splitAndTrim(opts.Shells)
		for _, platform := range platforms {
			for _, shell := range shells {
				if entry, ok := createMatrixEntry(platform, Shell(shell)); ok {
					matrix = append(matrix, entry)
				}
			}
		}
	} else {
		// Use platform defaults
		for _, platform := range platforms {
			matrix = append(matrix, defaultEntryForPlatform(platform))
		}
	}

	return matrix
}

func createMatrixEntry(platform Platform, shell Shell) (MatrixEntry, bool) {
	entry := MatrixEntry{OS: platform, Shell: shell}

	// Determine pok command based on shell
	switch shell {
	case ShellBash, ShellZsh, ShellFish:
		if platform == PlatformWindows {
			// Git Bash on Windows
			entry.Pok = "./pok"
		} else {
			entry.Pok = "./pok"
		}
	case ShellPowershell:
		entry.Pok = "./pok.ps1"
	case ShellCmd:
		if platform != PlatformWindows {
			return MatrixEntry{}, false // cmd only on Windows
		}
		entry.Pok = "./pok.cmd"
	}

	// Validate shell availability on platform
	switch platform {
	case PlatformWindows:
		// Windows supports: pwsh, cmd, bash (Git Bash)
		if shell == ShellZsh || shell == ShellFish {
			return MatrixEntry{}, false
		}
	case PlatformUbuntu, PlatformMacOS:
		// Unix supports: bash, zsh, fish (with install)
		if shell == ShellCmd {
			return MatrixEntry{}, false
		}
	}

	return entry, true
}

func defaultEntryForPlatform(platform Platform) MatrixEntry {
	switch platform {
	case PlatformWindows:
		return MatrixEntry{OS: platform, Shell: ShellPowershell, Pok: "./pok.ps1"}
	default:
		return MatrixEntry{OS: platform, Shell: ShellBash, Pok: "./pok"}
	}
}

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range bytes.Split([]byte(s), []byte(",")) {
		trimmed := bytes.TrimSpace(part)
		if len(trimmed) > 0 {
			result = append(result, string(trimmed))
		}
	}
	return result
}
