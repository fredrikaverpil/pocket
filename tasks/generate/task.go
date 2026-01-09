// Package generate provides the generate task for regenerating all generated files.
package generate

import (
	"context"
	"fmt"
	"strings"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

// Task returns a task that regenerates all generated files.
func Task(cfg pocket.Config) *pocket.Task {
	return &pocket.Task{
		Name:    "generate",
		Usage:   "regenerate all generated files (main.go, shim)",
		Builtin: true,
		Action: func(ctx context.Context, _ *pocket.RunContext) error {
			shimPaths, err := scaffold.GenerateAll(&cfg)
			if err != nil {
				return err
			}
			if pocket.IsVerbose(ctx) {
				fmt.Printf("Generated .pocket/main.go and shims:\n  %s\n", strings.Join(shimPaths, "\n  "))
			} else {
				fmt.Printf("Generated .pocket/main.go and %d shim(s)\n", len(shimPaths))
			}
			return nil
		},
	}
}
