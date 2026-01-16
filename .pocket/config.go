package main

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/github"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

// Config is the pocket configuration for this project.
var Config = pocket.Config{
	AutoRun: pocket.Parallel(
		pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),
		pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect()),
		github.Workflows.With(github.WorkflowsOptions{SkipPocket: true}),
	),
	ManualRun: []pocket.Runnable{
		Greet,
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
var Greet = pocket.Task("greet", "print a greeting message", greet).
	With(GreetOptions{Name: "world", Count: 1})

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
