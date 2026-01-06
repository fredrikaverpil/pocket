// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"strconv"

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
		Name:  "md-fmt",
		Usage: "format Markdown files",
		Deps:  goyek.Deps{mdformat.Prepare},
		Action: func(a *goyek.A) {
			args := buildArgs(cfg.Markdown)
			args = append(args, ".")
			if err := mdformat.Run(a.Context(), args...); err != nil {
				a.Fatal(err)
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

func buildArgs(cfg *bld.MarkdownConfig) []string {
	args := make([]string, 0, 4+len(cfg.Exclude)*2) //nolint:mnd

	// Wrap setting
	switch cfg.Wrap {
	case -1:
		args = append(args, "--wrap", "keep")
	case 0:
		args = append(args, "--wrap", "no")
	default:
		args = append(args, "--wrap", strconv.Itoa(cfg.Wrap))
	}

	// Number ordered lists
	if cfg.Number {
		args = append(args, "--number")
	}

	// Exclude patterns (requires mdformat 1.0.0+)
	for _, pattern := range cfg.Exclude {
		args = append(args, "--exclude", pattern)
	}

	return args
}
