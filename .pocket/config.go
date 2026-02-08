package main

import (
	"context"

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
			pk.WithFlag(github.Workflows, "skip-gh-pages", true),
			pk.WithFlag(github.Workflows, "skip-pocket", true),
			pk.WithFlag(github.Workflows, "include-pocket-perjob", true),
			pk.WithContextValue(github.PerJobConfigKey{}, github.PerJobConfig{
				DefaultPlatforms: []string{"ubuntu-latest"},
				TaskOverrides: map[string]github.TaskOverride{
					"go-test": {Platforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"}},
				},
			}),
		),
	),

	// Manual tasks - only run when explicitly invoked.
	Manual: []pk.Runnable{
		golang.Pprof, // ./pok go-pprof -file cpu.prof
	},

	// Plan configuration: shims, directories, and CI settings.
	Plan: &pk.PlanConfig{
		Shims: pk.AllShimsConfig(), // pok, pok.cmd, pok.ps1
	},
}


