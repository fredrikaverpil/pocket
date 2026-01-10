package main

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
	Run: pocket.Parallel(
		pocket.AutoDetect(golang.Tasks()),
		pocket.AutoDetect(markdown.Tasks()),
		greetTask,
	),
	Shim: &pocket.ShimConfig{
		Posix:      true,
		Windows:    true,
		PowerShell: true,
	},
}

// GreetOptions defines the options for the greet task.
type GreetOptions struct {
	Name string
}

// greetTask demonstrates task arguments.
var greetTask = &pocket.Task{
	Name:  "greet",
	Usage: "print a greeting message",
	Options: GreetOptions{Name: "world"},
	Action: func(ctx context.Context, rc *pocket.RunContext) error {
		args := pocket.GetArgs[GreetOptions](rc)
		pocket.Printf(ctx, "Hello, %s!\n", args.Name)
		return nil
	},
}
