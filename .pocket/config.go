package main

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tasks/github"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

// matrixConfig is the configuration for the GitHub Actions matrix.
var matrixConfig = github.MatrixConfig{
	DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
	TaskOverrides: map[string]github.TaskOverride{
		"go-lint": {Platforms: []string{"ubuntu-latest"}}, // lint only on linux
	},
	ExcludeTasks: []string{"github-workflows"}, // don't run in CI
}

// Config is the Pocket configuration for this project.
var Config = &pk.Config{
	Auto: pk.Parallel(
		// commits.Validate, // Validate commit messages against conventional commits
		golang.Tasks(),
		markdown.Format, // Format markdown files from root
		pk.WithOptions(
			github.Workflows,
			pk.WithFlag(github.Workflows, "skip-pocket", true),
			pk.WithFlag(github.Workflows, "include-pocket-matrix", true),
		),
	),

	// Manual tasks - only run when explicitly invoked.
	Manual: []pk.Runnable{
		Hello.Manual(),              // ./pok hello -name "World"
		github.Matrix(matrixConfig), // ./pok gha-matrix (for GitHub Actions)
	},

	// Plan configuration: shims, directories, and CI settings.
	Plan: &pk.PlanConfig{
		Shims: pk.AllShimsConfig(), // pok, pok.cmd, pok.ps1
	},
}

// Hello flags.
var (
	helloFlags = flag.NewFlagSet("hello", flag.ContinueOnError)
	helloName  = helloFlags.String("name", "Pocket v2", "name to greet")
)

// Hello is a demo task that prints a greeting.
// Demonstrates task with CLI flags.
var Hello = pk.NewTask("hello", "print a greeting message", helloFlags, pk.Do(func(ctx context.Context) error {
	pk.Printf(ctx, "Hello, %s!\n", *helloName)
	return nil
}))
