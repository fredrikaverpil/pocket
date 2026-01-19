package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/fredrikaverpil/pocket/pk"
)

// Config is the Pocket configuration for this project.
var Config = &pk.Config{
	Root: pk.Serial(
		Hello,
		pk.WithOptions(
			ShowDirMulti,
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

// Hello flags.
var helloFlags = flag.NewFlagSet("hello", flag.ContinueOnError)
var helloName = helloFlags.String("name", "Pocket v2", "name to greet")

// Hello is a demo task that prints a greeting.
// Demonstrates task with CLI flags.
var Hello = pk.NewTaskWithFlags("hello", "print a greeting message", helloFlags, func(ctx context.Context) error {
	fmt.Printf("Hello, %s!\n", *helloName)
	return nil
})

// ShowDirMulti executes "pwd" to show the current directory.
// This proves that tasks are actually executing in the correct directories.
var ShowDirMulti = pk.NewTask("show-dir", "show current working directory", func(ctx context.Context) error {
	path := pk.PathFromContext(ctx)

	// Run pwd using the command helper.
	output, err := pk.RunCommandString(ctx, "pwd")
	if err != nil {
		return err
	}

	fmt.Printf("  [show-dir] context path=%s, actual pwd=%s", path, output)
	return nil
})

// Lint checks code quality.
var Lint = pk.NewTask("lint", "run linters on code", func(ctx context.Context) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [lint] in %s: Linting code...\n", path)
	// TODO: Run actual linter (e.g., golangci-lint).
	return nil
})

// Format formats code.
var Format = pk.NewTask("format", "format source code", func(ctx context.Context) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [format] in %s: Formatting code...\n", path)
	// TODO: Run actual formatter (e.g., gofmt, prettier).
	return nil
})

// Build compiles the project.
var Build = pk.NewTask("build", "compile the project", func(ctx context.Context) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [build] in %s: Building...\n", path)
	// TODO: Run actual build (e.g., go build).
	return nil
})

// Test runs all tests.
var Test = pk.NewTask("test", "run tests", func(ctx context.Context) error {
	path := pk.PathFromContext(ctx)
	fmt.Printf("  [test] in %s: Running tests...\n", path)

	// Example: Using FromGitRoot for file operations (both forms work).
	// configFile1 := pk.FromGitRoot("services/api", "config.json")
	// configFile2 := pk.FromGitRoot("services", "api", "config.json")
	// Both produce: /path/to/repo/services/api/config.json

	// TODO: Run actual tests (e.g., go test).
	return nil
})
