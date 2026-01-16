// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"github.com/fredrikaverpil/pocket"
)

// Option configures the python task group.
type Option func(*config)

type config struct {
	pythonVersion string
	format        FormatOptions
	lint          LintOptions
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

	// Sync task with Python version
	syncTask := Sync
	if cfg.pythonVersion != "" {
		syncTask = pocket.WithOpts(Sync, SyncOptions{PythonVersion: cfg.pythonVersion})
	}

	formatTask := Format
	if cfg.format != (FormatOptions{}) {
		formatTask = pocket.WithOpts(Format, cfg.format)
	}

	lintTask := Lint
	if cfg.lint != (LintOptions{}) {
		lintTask = pocket.WithOpts(Lint, cfg.lint)
	}

	testTask := Test
	if cfg.test != (TestOptions{}) {
		testTask = pocket.WithOpts(Test, cfg.test)
	}

	// Run sync first, then format, lint, typecheck, test (serial since format/lint modify files)
	return pocket.Serial(syncTask, formatTask, lintTask, Typecheck, testTask)
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
