// Package generate provides the generate task for regenerating all generated files.
package generate

import (
	"context"
	"fmt"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

// Task returns a task that regenerates all generated files.
func Task(cfg pocket.Config) *pocket.Task {
	return &pocket.Task{
		Name:    "generate",
		Usage:   "regenerate all generated files (main.go, shim)",
		Builtin: true,
		Action: func(ctx context.Context, _ map[string]string) error {
			if err := scaffold.GenerateAll(&cfg); err != nil {
				return err
			}
			if pocket.IsVerbose(ctx) {
				fmt.Println("Generated .pocket/main.go and shim")
			}
			return nil
		},
	}
}
