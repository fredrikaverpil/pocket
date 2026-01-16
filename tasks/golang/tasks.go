// Package golang provides Go development tasks.
// This is a "task" package - it orchestrates tools to do work.
package golang

import (
	"github.com/fredrikaverpil/pocket"
)

// Option configures the golang task group.
type Option func(*config)

type config struct {
	lint LintOptions
	test TestOptions
}

// WithLint sets options for the go-lint task.
func WithLint(opts LintOptions) Option {
	return func(c *config) { c.lint = opts }
}

// WithTest sets options for the go-test task.
func WithTest(opts TestOptions) Option {
	return func(c *config) { c.test = opts }
}

// Tasks returns all Go tasks composed as a Runnable.
// Use this with pocket.RunIn() and pocket.Detect() for auto-detection.
//
// Example:
//
//	pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect()))
//
// Example with options:
//
//	pocket.RunIn(golang.Tasks(
//	    golang.WithLint(golang.LintOptions{Config: ".golangci.yml"}),
//	    golang.WithTest(golang.TestOptions{SkipRace: true}),
//	), pocket.Detect(golang.Detect()))
func Tasks(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// Apply options to tasks
	lintTask := Lint
	if cfg.lint != (LintOptions{}) {
		lintTask = pocket.WithOpts(Lint, cfg.lint)
	}

	testTask := Test
	if cfg.test != (TestOptions{}) {
		testTask = pocket.WithOpts(Test, cfg.test)
	}

	return pocket.Serial(
		Fix,
		Format,
		lintTask,
		pocket.Parallel(testTask, Vulncheck),
	)
}

// Detect returns a detection function for Go modules.
// It finds directories containing go.mod files.
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("go.mod")
	}
}
