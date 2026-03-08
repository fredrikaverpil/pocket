package main

import (
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
			pk.WithFlags(github.WorkflowFlags{
				Platforms:        []github.Platform{github.Ubuntu},
				PerPocketTaskJob: new(true),
				PerPocketTaskJobOptions: map[string]github.PerPocketTaskJobOption{
					golang.Test.Name: {Platforms: github.AllPlatforms()},
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
