package main

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
	// AutoRun: tasks that run on ./pok (no arguments).
	AutoRun: pocket.Parallel(
		pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),
		pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect()),
	),
	// ManualRun: tasks that require ./pok <taskname>.
	ManualRun: []pocket.Runnable{
		greetTask,
	},
	Shim: &pocket.ShimConfig{
		Posix:      true,
		Windows:    true,
		PowerShell: true,
	},
}

// GreetOptions defines the options for the greet task.
type GreetOptions struct {
	Name string `usage:"name to greet"`
}

// greetAction is the action for the greet task.
func greetAction(_ context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[GreetOptions](tc)
	name := opts.Name
	if name == "" {
		name = "world"
	}
	tc.Out.Printf("Hello, %s!\n", name)
	return nil
}

// greetTask demonstrates the new task pattern.
var greetTask = pocket.NewTask("greet", "print a greeting message", greetAction).
	WithOptions(GreetOptions{Name: "world"})
