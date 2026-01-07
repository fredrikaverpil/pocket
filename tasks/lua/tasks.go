// Package lua provides Lua-related build tasks.
package lua

import (
	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
	"github.com/goyek/goyek/v3"
)

// Tasks holds the goyek tasks for Lua operations.
type Tasks struct {
	config pocket.Config

	// Format formats Lua files using stylua.
	Format *goyek.DefinedTask
}

// NewTasks creates Lua tasks for the given config.
func NewTasks(cfg pocket.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{config: cfg}

	t.Format = goyek.Define(goyek.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(a *goyek.A) {
			modules := cfg.LuaModulesForFormat()
			if len(modules) == 0 {
				a.Log("no modules configured for format")
				return
			}
			configPath, err := stylua.ConfigPath()
			if err != nil {
				a.Fatalf("get stylua config: %v", err)
			}
			for _, mod := range modules {
				if err := stylua.Run(a.Context(), "-f", configPath, mod); err != nil {
					a.Errorf("stylua format failed in %s: %v", mod, err)
				}
			}
		},
	})

	return t
}
