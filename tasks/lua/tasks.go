// Package lua provides Lua-related build tasks.
package lua

import (
	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tools/stylua"
	"github.com/goyek/goyek/v3"
)

// Tasks holds the goyek tasks for Lua operations.
type Tasks struct {
	config bld.Config

	// All runs all Lua tasks.
	All *goyek.DefinedTask

	// Format formats Lua files using stylua.
	Format *goyek.DefinedTask
}

// NewTasks creates Lua tasks for the given config.
func NewTasks(cfg bld.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{config: cfg}

	t.Format = goyek.Define(goyek.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(a *goyek.A) {
			if err := stylua.Run(a.Context(), "."); err != nil {
				a.Fatal(err)
			}
		},
	})

	t.All = goyek.Define(goyek.Task{
		Name:  "lua-all",
		Usage: "run all Lua tasks (format)",
		Deps:  goyek.Deps{t.Format},
	})

	return t
}
