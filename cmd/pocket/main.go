package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`pocket - bootstrap pocket in your project

Usage:
  pocket init      Initialize .pocket/ in current directory

Examples:
  go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init`)
}

func runInit() error {
	// Check .pocket doesn't already exist
	if _, err := os.Stat(".pocket"); err == nil {
		return fmt.Errorf(".pocket/ already exists")
	}

	fmt.Println("Initializing pocket...")

	// Create .pocket directory
	if err := os.MkdirAll(".pocket", 0o755); err != nil {
		return fmt.Errorf("creating .pocket/: %w", err)
	}

	// Create go.mod
	fmt.Println("  Creating .pocket/go.mod")
	if err := runCommand(".pocket", "go", "mod", "init", "pok"); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// Get dependencies
	fmt.Println("  Adding github.com/fredrikaverpil/pocket@latest")
	if err := runCommand(".pocket", "go", "get", "github.com/fredrikaverpil/pocket@latest"); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	// Generate all scaffold files (config.go, .gitignore, main.go, shim)
	// Detect platform and generate appropriate shim(s)
	fmt.Println("  Generating scaffold files")
	cfg := &pocket.Config{
		Shim: &pocket.ShimConfig{
			Posix:      runtime.GOOS != "windows",
			Windows:    runtime.GOOS == "windows",
			PowerShell: runtime.GOOS == "windows",
		},
	}
	if _, err := scaffold.GenerateAll(cfg); err != nil {
		return fmt.Errorf("generating scaffold: %w", err)
	}

	// Run go mod tidy (after main.go is created)
	fmt.Println("  Running go mod tidy")
	if err := runCommand(".pocket", "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	fmt.Println()
	fmt.Println("Done! You can now run:")
	if runtime.GOOS == "windows" {
		fmt.Println("  .\\pok -h          # list available tasks")
		fmt.Println("  .\\pok             # run all tasks")
		fmt.Println("  .\\pok update      # update pocket to latest version")
	} else {
		fmt.Println("  ./pok -h          # list available tasks")
		fmt.Println("  ./pok             # run all tasks")
		fmt.Println("  ./pok update      # update pocket to latest version")
	}

	return nil
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
