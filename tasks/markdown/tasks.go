// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tools/mdformat"
	"github.com/goyek/goyek/v3"
)

// Tasks holds the goyek tasks for Markdown operations.
type Tasks struct {
	config bld.Config

	// All runs all Markdown tasks.
	All *goyek.DefinedTask

	// Format formats Markdown files using mdformat.
	Format *goyek.DefinedTask
}

// NewTasks creates Markdown tasks for the given config.
func NewTasks(cfg bld.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{config: cfg}

	t.Format = goyek.Define(goyek.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(a *goyek.A) {
			modules := cfg.MarkdownModulesForFormat()
			if len(modules) == 0 {
				a.Log("no modules configured for format")
				return
			}
			for _, mod := range modules {
				if err := mdformat.Run(a.Context(), mod); err != nil {
					a.Errorf("mdformat format failed in %s: %v", mod, err)
				}
			}
		},
	})

	t.All = goyek.Define(goyek.Task{
		Name:  "md-all",
		Usage: "run all Markdown tasks (format)",
		Deps:  goyek.Deps{t.Format},
	})

	return t
}
