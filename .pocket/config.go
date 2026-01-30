package main

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tasks/github"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

// Config is the Pocket configuration for this project.
var Config = &pk.Config{
	Auto: pk.Parallel(
		// commits.Validate, // Validate commit messages against conventional commits
		golang.Tasks(),
		markdown.Format, // Format markdown files from root
		pk.WithOptions(
			github.Tasks(),
			pk.WithFlag(github.Workflows, "skip-pocket", true),
			pk.WithFlag(github.Workflows, "include-pocket-matrix", true),
			pk.WithContextValue(github.MatrixConfigKey{}, github.MatrixConfig{
				DefaultPlatforms: []string{"ubuntu-latest"},
				TaskOverrides: map[string]github.TaskOverride{
					"go-test": {Platforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"}},
				},
			}),
		),
	),

	// Manual tasks - only run when explicitly invoked.
	Manual: []pk.Runnable{
		Hello.Manual(), // ./pok hello -name "World"
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
