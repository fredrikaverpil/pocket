// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"github.com/fredrikaverpil/pocket"
)

// Option configures the python task group.
type Option func(*config)

type config struct {
	format FormatOptions
	lint   LintOptions
	test   TestOptions
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

// Workflow returns a Runnable that executes all Python tasks.
// Use pocket.Paths(python.Workflow()).DetectBy(python.Detect()) to enable path filtering.
//
// Execution order: format, lint, typecheck, then test (serial since format/lint modify files).
//
// Example with options:
//
//	pocket.Paths(python.Workflow(
//	    python.WithFormat(python.FormatOptions{RuffConfig: "ruff.toml"}),
//	    python.WithTest(python.TestOptions{SkipCoverage: true}),
//	)).DetectBy(python.Detect())
func Workflow(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	formatTask := Format
	if cfg.format != (FormatOptions{}) {
		formatTask = Format.With(cfg.format)
	}

	lintTask := Lint
	if cfg.lint != (LintOptions{}) {
		lintTask = Lint.With(cfg.lint)
	}

	testTask := Test
	if cfg.test != (TestOptions{}) {
		testTask = Test.With(cfg.test)
	}

	// Run format and lint in serial since both modify files
	return pocket.Serial(formatTask, lintTask, Typecheck, testTask)
}

// Detect returns a detection function that finds Python projects.
// It detects directories containing pyproject.toml, setup.py, or setup.cfg.
//
// Usage:
//
//	pocket.Paths(python.Workflow()).DetectBy(python.Detect())
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
	}
}
