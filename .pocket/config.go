package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tasks/golang"
)

// Config is the Pocket configuration for this project.
var Config = &pk.Config{
	Root: pk.Serial(
		golang.Tasks(), // Auto-detects go.mod directories
	),

	// Manual tasks - only run when explicitly invoked.
	Manual: []pk.Runnable{
		Hello.Manual(), // ./pok hello -name "World"
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
	fmt.Printf("Hello, %s!\n", *helloName)
	return nil
}))
