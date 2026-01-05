package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	case "update":
		if err := runUpdate(); err != nil {
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
	fmt.Println(`bld - bootstrap and update bld in your project

Usage:
  bld init      Initialize .bld/ in current directory
  bld update    Update bld dependency and wrapper script

Examples:
  go run github.com/fredrikaverpil/bld/cmd/bld@latest init
  go run github.com/fredrikaverpil/bld/cmd/bld@latest update`)
}

func runInit() error {
	// Check we're in a Go module
	moduleName, err := getModuleName()
	if err != nil {
		return fmt.Errorf("not in a Go module (no go.mod found): %w", err)
	}

	// Check .bld doesn't already exist
	if _, err := os.Stat(".bld"); err == nil {
		return fmt.Errorf(".bld/ already exists")
	}

	fmt.Println("Initializing bld...")

	// Create .bld directory
	if err := os.MkdirAll(".bld", 0o755); err != nil {
		return fmt.Errorf("creating .bld/: %w", err)
	}

	// Create go.mod
	buildModule := moduleName + "-build"
	fmt.Printf("  Creating .bld/go.mod (%s)\n", buildModule)
	if err := runCommand(".bld", "go", "mod", "init", buildModule); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// Get dependencies
	deps := []string{
		"github.com/fredrikaverpil/bld@latest",
		"github.com/goyek/goyek/v3@latest",
		"github.com/goyek/x@latest",
	}
	for _, dep := range deps {
		fmt.Printf("  Adding %s\n", dep)
		if err := runCommand(".bld", "go", "get", dep); err != nil {
			return fmt.Errorf("go get %s: %w", dep, err)
		}
	}

	// Create config.go
	fmt.Println("  Creating .bld/config.go")
	if err := os.WriteFile(".bld/config.go", []byte(configTemplate), 0o644); err != nil {
		return fmt.Errorf("creating config.go: %w", err)
	}

	// Create main.go
	fmt.Println("  Creating .bld/main.go")
	if err := os.WriteFile(".bld/main.go", []byte(mainTemplate), 0o644); err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}

	// Create .gitignore
	fmt.Println("  Creating .bld/.gitignore")
	if err := os.WriteFile(".bld/.gitignore", []byte(gitignoreTemplate), 0o644); err != nil {
		return fmt.Errorf("creating .gitignore: %w", err)
	}

	// Run go mod tidy
	fmt.Println("  Running go mod tidy")
	if err := runCommand(".bld", "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Create wrapper script
	fmt.Println("  Creating ./bld (wrapper script)")
	if err := os.WriteFile("bld", []byte(wrapperBashTemplate), 0o755); err != nil {
		return fmt.Errorf("creating bld wrapper: %w", err)
	}

	fmt.Println()
	fmt.Println("Done! You can now run:")
	fmt.Println("  ./bld -h          # list available tasks")
	fmt.Println("  ./bld go-fmt      # format Go code")
	fmt.Println("  ./bld go-test     # run tests")
	fmt.Println("  ./bld update      # generate CI workflows")

	return nil
}

func getModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runUpdate() error {
	// Check .bld exists
	if _, err := os.Stat(".bld"); os.IsNotExist(err) {
		return fmt.Errorf(".bld/ not found - run 'bld init' first")
	}

	fmt.Println("Updating bld...")

	// Update bld dependency
	fmt.Println("  Updating github.com/fredrikaverpil/bld@latest")
	if err := runCommand(".bld", "go", "get", "-u", "github.com/fredrikaverpil/bld@latest"); err != nil {
		return fmt.Errorf("go get -u: %w", err)
	}

	// Run go mod tidy
	fmt.Println("  Running go mod tidy")
	if err := runCommand(".bld", "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Update wrapper script
	fmt.Println("  Updating ./bld (wrapper script)")
	if err := os.WriteFile("bld", []byte(wrapperBashTemplate), 0o755); err != nil {
		return fmt.Errorf("updating bld wrapper: %w", err)
	}

	fmt.Println("Done!")
	return nil
}

const configTemplate = `package main

import "github.com/fredrikaverpil/bld"

var Config = bld.Config{
	Go: &bld.GoConfig{
		Modules: map[string]bld.GoModuleOptions{
			".": {},
		},
	},
	GitHub: &bld.GitHubConfig{},
}
`

const mainTemplate = `package main

import (
	"os"
	"os/exec"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tasks/golang"
	"github.com/fredrikaverpil/bld/workflows"
	"github.com/goyek/goyek/v3"
	"github.com/goyek/x/boot"
)

var tasks = golang.NewTasks(Config)

var update = goyek.Define(goyek.Task{
	Name:  "update",
	Usage: "update bld and generate CI workflows",
	Action: func(a *goyek.A) {
		// Update bld dependency and wrapper script
		cmd := exec.CommandContext(a.Context(), "go", "run", "github.com/fredrikaverpil/bld/cmd/bld@latest", "update")
		cmd.Dir = bld.FromGitRoot()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			a.Fatalf("bld update: %v", err)
		}

		// Generate workflows
		if err := workflows.Generate(Config); err != nil {
			a.Fatal(err)
		}
		a.Log("Generated workflows in .github/workflows/")
	},
})

func main() {
	goyek.SetDefault(tasks.All)
	boot.Main()
}
`

const gitignoreTemplate = `# Downloaded tool binaries
bin/
tools/
`

const wrapperBashTemplate = `#!/bin/bash
set -e
go run -C .bld . -v "$@"
`
