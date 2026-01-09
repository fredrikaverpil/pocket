package main

import (
	"context"
	"fmt"

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

// greetTask demonstrates task arguments.
var greetTask = &pocket.Task{
	Name:  "greet",
	Usage: "print a greeting message",
	Args: []pocket.ArgDef{
		{Name: "name", Usage: "who to greet", Default: "world"},
	},
	Action: func(_ context.Context, opts *pocket.RunContext) error {
		fmt.Printf("Hello, %s!\n", opts.Args["name"])
		return nil
	},
}
