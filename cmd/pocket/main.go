// Command pocket bootstraps a new Pocket project.
//
// Usage:
//
//	go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fredrikaverpil/pocket/internal/scaffold"
	"github.com/fredrikaverpil/pocket/internal/shim"
)

const usage = `pocket - initialize a Pocket task runner

Usage:
    pocket init    Initialize .pocket/ in the current directory
    pocket help    Show this help message

Examples:
    go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := runInit(ctx)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", cmd, usage)
		os.Exit(1)
	}
}

func runInit(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	pocketDir := filepath.Join(cwd, ".pocket")

	// Check if .pocket already exists.
	if _, err := os.Stat(pocketDir); err == nil {
		return fmt.Errorf(".pocket/ already exists - remove it first to reinitialize")
	}

	fmt.Println("Initializing Pocket...")

	// Create .pocket directory.
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		return fmt.Errorf("creating .pocket directory: %w", err)
	}

	// Initialize Go module.
	fmt.Println("  Creating Go module...")
	initCmd := exec.CommandContext(ctx, "go", "mod", "init", "pocket")
	initCmd.Dir = pocketDir
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("initializing Go module: %w", err)
	}

	// Add pocket dependency.
	fmt.Println("  Adding pocket dependency...")
	getCmd := exec.CommandContext(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest")
	getCmd.Dir = pocketDir
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	if err := getCmd.Run(); err != nil {
		return fmt.Errorf("adding pocket dependency: %w", err)
	}

	// Generate scaffold files.
	fmt.Println("  Generating scaffold files...")
	if err := scaffold.GenerateAll(pocketDir); err != nil {
		return fmt.Errorf("generating scaffold: %w", err)
	}

	// Run go mod tidy.
	fmt.Println("  Running go mod tidy...")
	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = pocketDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("running go mod tidy: %w", err)
	}

	// Generate shims.
	fmt.Println("  Generating shim scripts...")

	shimCfg := shim.Config{
		Name:       "pok",
		Posix:      runtime.GOOS != "windows",
		Windows:    runtime.GOOS == "windows",
		PowerShell: runtime.GOOS == "windows",
	}

	shimPaths, err := shim.GenerateShims(ctx, cwd, pocketDir, nil, shimCfg)
	if err != nil {
		return fmt.Errorf("generating shims: %w", err)
	}

	// Success message.
	fmt.Println()
	fmt.Println("Pocket initialized successfully!")
	fmt.Println()
	fmt.Println("Generated files:")
	fmt.Println("  .pocket/config.go  - Edit this to define your tasks")
	fmt.Println("  .pocket/main.go    - Auto-generated entry point")
	for _, p := range shimPaths {
		fmt.Printf("  %s\n", p)
	}
	fmt.Println()

	// Platform-specific instructions.
	if runtime.GOOS == "windows" {
		fmt.Println("Run your tasks with:")
		fmt.Println("  .\\pok.cmd")
		fmt.Println("  # or")
		fmt.Println("  .\\pok.ps1")
	} else {
		fmt.Println("Run your tasks with:")
		fmt.Println("  ./pok")
	}

	return nil
}
