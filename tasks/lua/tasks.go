// Package lua provides Lua-related build tasks.
package lua

import (
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
	"github.com/goyek/goyek/v3"
)

const name = "lua"

// Options defines options for a Lua module within a task group.
type Options struct {
	// Skip lists task names to skip (e.g., "format").
	Skip []string
	// Only lists task names to run (empty = run all).
	// If non-empty, only these tasks run (Skip is ignored).
	Only []string

	// Task-specific options
	Format FormatOptions
}

// ShouldRun returns true if the given task should run based on Skip/Only options.
func (o Options) ShouldRun(task string) bool {
	if len(o.Only) > 0 {
		return slices.Contains(o.Only, task)
	}
	return !slices.Contains(o.Skip, task)
}

// FormatOptions defines options for the format task.
type FormatOptions struct {
	// ConfigFile overrides the default stylua config file.
	ConfigFile string
}

// New creates a Lua task group with the given module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return &taskGroup{modules: modules}
}

type taskGroup struct {
	modules map[string]Options
}

func (tg *taskGroup) Name() string { return name }

func (tg *taskGroup) Modules() map[string]pocket.ModuleConfig {
	modules := make(map[string]pocket.ModuleConfig, len(tg.modules))
	for path, opts := range tg.modules {
		modules[path] = opts
	}
	return modules
}

func (tg *taskGroup) ForContext(context string) pocket.TaskGroup {
	if context == "." {
		return tg
	}
	if opts, ok := tg.modules[context]; ok {
		return &taskGroup{modules: map[string]Options{context: opts}}
	}
	return nil
}

func (tg *taskGroup) Tasks(cfg pocket.Config) []*goyek.DefinedTask {
	_ = cfg.WithDefaults()
	var tasks []*goyek.DefinedTask

	if mods := tg.modulesFor("format"); len(mods) > 0 {
		tasks = append(tasks, goyek.Define(FormatTask(mods)))
	}

	return tasks
}

// modulesFor returns modules with their task-specific options for a given task.
func (tg *taskGroup) modulesFor(task string) map[string]Options {
	result := make(map[string]Options)
	for path, opts := range tg.modules {
		if opts.ShouldRun(task) {
			result[path] = opts
		}
	}
	return result
}

// FormatTask returns a task that formats Lua files using stylua.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) goyek.Task {
	return goyek.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(a *goyek.A) {
			for mod, opts := range modules {
				configPath := opts.Format.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = stylua.ConfigPath()
					if err != nil {
						a.Fatalf("get stylua config: %v", err)
					}
				}
				if err := stylua.Run(a.Context(), "-f", configPath, mod); err != nil {
					a.Errorf("stylua format failed in %s: %v", mod, err)
				}
			}
		},
	}
}
