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
		ShowDir, // Runs at root (no WithOptions)
		pk.WithOptions(
			ShowDir,
			pk.WithIncludePath("services"),
			pk.WithIncludePath("pkg"),
		),
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

// ShowDir executes "pwd" to show the current directory.
// This proves that tasks are actually executing in the correct directories.
var ShowDir = pk.NewTask("show-dir", func(ctx context.Context, opts map[string]any) error {
	path := pk.PathFromContext(ctx)

	// Run pwd using the command helper
	output, err := pk.RunCommandString(ctx, "pwd")
	if err != nil {
		return err
	}

	fmt.Printf("  [show-dir] context path=%s, actual pwd=%s", path, output)
	return nil
})

// Lint checks code quality.
var Lint = pk.NewTask("lint", func(ctx context.Context, opts map[string]any) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [lint] in %s: Linting code...\n", path)
	// TODO: Run actual linter (e.g., golangci-lint)
	return nil
})

// Format formats code.
var Format = pk.NewTask("format", func(ctx context.Context, opts map[string]any) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [format] in %s: Formatting code...\n", path)
	// TODO: Run actual formatter (e.g., gofmt, prettier)
	return nil
})

// Build compiles the project.
var Build = pk.NewTask("build", func(ctx context.Context, opts map[string]any) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [build] in %s: Building...\n", path)
	// TODO: Run actual build (e.g., go build)
	return nil
})

// Test runs all tests.
var Test = pk.NewTask("test", func(ctx context.Context, opts map[string]any) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [test] in %s: Running tests...\n", path)
	// TODO: Run actual tests (e.g., go test)
	return nil
})
