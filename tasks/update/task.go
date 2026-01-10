// Package update provides the update task for updating pocket dependencies.
package update

import (
	"context"
	"fmt"
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
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			pocketDir := filepath.Join(pocket.FromGitRoot(), pocket.DirName)
			verbose := rc.Verbose

			// Update pocket dependency
			if verbose {
				fmt.Println("Updating github.com/fredrikaverpil/pocket@latest")
			}
			cmd := pocket.Command(ctx, "go", "get", "-u", "github.com/fredrikaverpil/pocket@latest")
			cmd.Dir = pocketDir
			cmd.Env = append(cmd.Env, "GOPROXY=direct")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go get -u: %w", err)
			}

			// Run go mod tidy
			if verbose {
				fmt.Println("Running go mod tidy")
			}
			cmd = pocket.Command(ctx, "go", "mod", "tidy")
			cmd.Dir = pocketDir
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go mod tidy: %w", err)
			}

			// Regenerate all files
			if verbose {
				fmt.Println("Regenerating files")
			}
			if _, err := scaffold.GenerateAll(&cfg); err != nil {
				return err
			}

			if verbose {
				fmt.Println("Done!")
			}
			return nil
		},
	}
}
