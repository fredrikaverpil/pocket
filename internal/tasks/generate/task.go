// Package generate provides the generate task for regenerating all generated files.
package generate

import (
	"fmt"
	"strings"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

// Task returns a task that regenerates all generated files.
func Task(cfg pocket.Config) *pocket.Task {
	return pocket.NewTask("generate", "regenerate all generated files (main.go, shim)", generateAction(&cfg)).
		AsBuiltin()
}

func generateAction(cfg *pocket.Config) pocket.TaskAction {
	return func(rc *pocket.RunContext) error {
		shimPaths, err := scaffold.GenerateAll(cfg)
		if err != nil {
			return err
		}
		if rc.Verbose {
			fmt.Printf("Generated .pocket/main.go and shims:\n  %s\n", strings.Join(shimPaths, "\n  "))
		} else {
			fmt.Printf("Generated .pocket/main.go and %d shim(s)\n", len(shimPaths))
		}
		return nil
	}
}
