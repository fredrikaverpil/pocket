// Package update provides the update task for updating pocket dependencies.
package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

// Task returns a task that updates pocket and regenerates files.
func Task(cfg pocket.Config) *pocket.Task {
	return &pocket.Task{
		Name:    "update",
		Usage:   "update pocket dependency and regenerate files",
		Builtin: true,
		Action: func(ctx context.Context, _ map[string]string) error {
			pocketDir := filepath.Join(pocket.FromGitRoot(), pocket.DirName)
			verbose := pocket.IsVerbose(ctx)

			// Update pocket dependency
			if verbose {
				fmt.Println("Updating github.com/fredrikaverpil/pocket@latest")
			}
			cmd := exec.CommandContext(ctx, "go", "get", "-u", "github.com/fredrikaverpil/pocket@latest")
			cmd.Dir = pocketDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go get -u: %w", err)
			}

			// Run go mod tidy
			if verbose {
				fmt.Println("Running go mod tidy")
			}
			cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
			cmd.Dir = pocketDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go mod tidy: %w", err)
			}

			// Regenerate all files
			if verbose {
				fmt.Println("Regenerating files")
			}
			if err := scaffold.GenerateAll(&cfg); err != nil {
				return err
			}

			if verbose {
				fmt.Println("Done!")
			}
			return nil
		},
	}
}
