// Package update provides the update task for updating pocket dependencies.
package update

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
	"github.com/goyek/goyek/v3"
)

// Task returns a goyek task that updates pocket and regenerates files.
func Task(cfg pocket.Config) *goyek.DefinedTask {
	return goyek.Define(goyek.Task{
		Name:  "update",
		Usage: "update pocket dependency and regenerate files",
		Action: func(a *goyek.A) {
			pocketDir := filepath.Join(pocket.FromGitRoot(), pocket.DirName)

			// Update pocket dependency
			a.Log("Updating github.com/fredrikaverpil/pocket@latest")
			cmd := exec.CommandContext(a.Context(), "go", "get", "-u", "github.com/fredrikaverpil/pocket@latest")
			cmd.Dir = pocketDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				a.Fatalf("go get -u: %v", err)
			}

			// Run go mod tidy
			a.Log("Running go mod tidy")
			cmd = exec.CommandContext(a.Context(), "go", "mod", "tidy")
			cmd.Dir = pocketDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				a.Fatalf("go mod tidy: %v", err)
			}

			// Regenerate all files
			a.Log("Regenerating files")
			if err := scaffold.GenerateAll(&cfg); err != nil {
				a.Fatal(err)
			}

			a.Log("Done!")
		},
	})
}
