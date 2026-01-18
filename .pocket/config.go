package main

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/github"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

// autoRun defines the tasks that run on ./pok with no arguments.
var autoRun = pocket.Parallel(
	pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect())),
	pocket.RunIn(markdown.Tasks(), pocket.Detect(markdown.Detect())),
	pocket.WithOpts(github.Workflows, github.WorkflowsOptions{SkipPocket: true, IncludePocketMatrix: true}),
)

// matrixConfig configures GitHub Actions matrix generation.
var matrixConfig = github.MatrixConfig{
	DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
	TaskOverrides: map[string]github.TaskOverride{
		"go-lint":      {Platforms: []string{"ubuntu-latest"}},
		"go-vulncheck": {Platforms: []string{"ubuntu-latest"}},
		"md-format":    {Platforms: []string{"ubuntu-latest"}},
	},
	ExcludeTasks: []string{"github-workflows"},
}

// Config is the pocket configuration for this project.
var Config = pocket.Config{
	AutoRun: autoRun,
	ManualRun: []pocket.Runnable{
		Greet,
		github.MatrixTask(autoRun, matrixConfig),
	},
	Shim: &pocket.ShimConfig{
		Posix:      true,
		Windows:    true,
		PowerShell: true,
	},
}

// GreetOptions defines the options for the greet task.
type GreetOptions struct {
	Name  string `arg:"name"  usage:"name to greet"`
	Count int    `arg:"count" usage:"number of times to greet"`
}

// Greet is a demo task that prints a greeting.
var Greet = pocket.Task("greet", "print a greeting message", greet,
	pocket.Opts(GreetOptions{Name: "world", Count: 1}))

func greet(ctx context.Context) error {
	opts := pocket.Options[GreetOptions](ctx)
	name := opts.Name
	if name == "" {
		name = "world"
	}
	count := opts.Count
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		pocket.Printf(ctx, "Hello, %s!\n", name)
	}
	return nil
}
