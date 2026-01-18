// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"strings"

	"github.com/fredrikaverpil/pocket"
)

// pythonVersionToRuff converts a Python version (e.g., "3.9") to ruff's format (e.g., "py39").
func pythonVersionToRuff(version string) string {
	// Handle formats like "3.9", "3.10", "3.9.1"
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return "py" + parts[0] + parts[1]
	}
	return "py" + strings.ReplaceAll(version, ".", "")
}

// Option configures the python task group.
type Option func(*config)

type config struct {
	pythonVersion string
	format        FormatOptions
	lint          LintOptions
	typecheck     TypecheckOptions
	test          TestOptions
}

// WithPythonVersion sets the Python version for uv commands.
func WithPythonVersion(version string) Option {
	return func(c *config) { c.pythonVersion = version }
}

// WithFormat sets options for the py-format task.
func WithFormat(opts FormatOptions) Option {
	return func(c *config) { c.format = opts }
}

// WithLint sets options for the py-lint task.
func WithLint(opts LintOptions) Option {
	return func(c *config) { c.lint = opts }
}

// WithTypecheck sets options for the py-typecheck task.
func WithTypecheck(opts TypecheckOptions) Option {
	return func(c *config) { c.typecheck = opts }
}

// WithTest sets options for the py-test task.
func WithTest(opts TestOptions) Option {
	return func(c *config) { c.test = opts }
}

// Tasks returns a Runnable that executes all Python tasks.
// Use pocket.RunIn(python.Tasks(), pocket.Detect(python.Detect())) to enable path filtering.
//
// Execution order: format, lint, typecheck, then test (serial since format/lint modify files).
//
// Example with options:
//
//	pocket.RunIn(python.Tasks(
//	    python.WithFormat(python.FormatOptions{RuffConfig: "ruff.toml"}),
//	    python.WithTest(python.TestOptions{SkipCoverage: true}),
//	), pocket.Detect(python.Detect()))
func Tasks(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// Build options for each task, merging pythonVersion with any explicit options
	syncOpts := SyncOptions{PythonVersion: cfg.pythonVersion}

	formatOpts := cfg.format
	if cfg.pythonVersion != "" && formatOpts.PythonVersion == "" {
		formatOpts.PythonVersion = cfg.pythonVersion
	}

	lintOpts := cfg.lint
	if cfg.pythonVersion != "" && lintOpts.PythonVersion == "" {
		lintOpts.PythonVersion = cfg.pythonVersion
	}

	typecheckOpts := cfg.typecheck
	if cfg.pythonVersion != "" && typecheckOpts.PythonVersion == "" {
		typecheckOpts.PythonVersion = cfg.pythonVersion
	}

	testOpts := cfg.test
	if cfg.pythonVersion != "" && testOpts.PythonVersion == "" {
		testOpts.PythonVersion = cfg.pythonVersion
	}

	// Run sync first, then format, lint, typecheck, test (serial since format/lint modify files)
	return pocket.Serial(
		pocket.WithOpts(Sync, syncOpts),
		pocket.WithOpts(Format, formatOpts),
		pocket.WithOpts(Lint, lintOpts),
		pocket.WithOpts(Typecheck, typecheckOpts),
		pocket.WithOpts(Test, testOpts),
	)
}

// Detect returns a detection function that finds Python projects.
// It detects directories containing pyproject.toml, setup.py, or setup.cfg.
//
// Usage:
//
//	pocket.RunIn(python.Tasks(), pocket.Detect(python.Detect()))
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
	}
}
