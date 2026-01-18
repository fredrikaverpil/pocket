package main

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket/core"
	"github.com/fredrikaverpil/pocket/pk"
)

// Config is the Pocket configuration for this project.
var Config = &core.Config{
	Root: pk.Serial(
		Hello,
		pk.WithOptions(
			pk.Parallel(
				Lint,
				Format,
			),
			pk.WithIncludePath("services"),
			pk.WithExcludePath("vendor"),
		),
		Build,
		pk.WithOptions(
			Test,
			pk.WithIncludePath("pkg"),
		),
	),
}

// Hello is a demo task that prints a greeting.
var Hello = pk.NewTask("hello", func(ctx context.Context, opts map[string]any) error {
	name := "Pocket v2"
	if n, ok := opts["name"].(string); ok {
		name = n
	}
	fmt.Printf("Hello, %s!\n", name)
	return nil
}).With("name", "Pocket v2")

// Lint checks code quality.
var Lint = pk.NewTask("lint", func(ctx context.Context, opts map[string]any) error {
	fmt.Println("  [lint] Linting code...")
	// TODO: Run actual linter (e.g., golangci-lint)
	return nil
})

// Format formats code.
var Format = pk.NewTask("format", func(ctx context.Context, opts map[string]any) error {
	fmt.Println("  [format] Formatting code...")
	// TODO: Run actual formatter (e.g., gofmt, prettier)
	return nil
})

// Build compiles the project.
var Build = pk.NewTask("build", func(ctx context.Context, opts map[string]any) error {
	fmt.Println("  [build] Building...")
	// TODO: Run actual build (e.g., go build)
	return nil
})

// Test runs all tests.
var Test = pk.NewTask("test", func(ctx context.Context, opts map[string]any) error {
	fmt.Println("  [test] Running tests...")
	// TODO: Run actual tests (e.g., go test)
	return nil
})
