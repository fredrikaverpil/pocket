// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
	"github.com/goyek/goyek/v3"
)

const name = "markdown"

// Config defines the configuration for the Markdown task group.
type Config struct {
	// Modules maps context paths to module options.
	// The key is the path relative to the git root (use "." for root).
	Modules map[string]Options
}

// Options defines options for a Markdown module.
type Options struct {
	// Skip lists task names to skip (e.g., "format").
	Skip []string
	// Only lists task names to run (empty = run all).
	// If non-empty, only these tasks run (Skip is ignored).
	Only []string
}

// ShouldRun returns true if the given task should run based on Skip/Only options.
func (o Options) ShouldRun(task string) bool {
	if len(o.Only) > 0 {
		return slices.Contains(o.Only, task)
	}
	return !slices.Contains(o.Skip, task)
}

// New creates a Markdown task group with the given configuration.
func New(cfg Config) pocket.TaskGroup {
	return &taskGroup{config: cfg}
}

type taskGroup struct {
	config Config
}

func (tg *taskGroup) Name() string { return name }

func (tg *taskGroup) Modules() map[string]pocket.ModuleConfig {
	modules := make(map[string]pocket.ModuleConfig, len(tg.config.Modules))
	for path, opts := range tg.config.Modules {
		modules[path] = opts
	}
	return modules
}

func (tg *taskGroup) ForContext(context string) pocket.TaskGroup {
	if context == "." {
		return tg
	}
	if opts, ok := tg.config.Modules[context]; ok {
		return &taskGroup{config: Config{
			Modules: map[string]Options{context: opts},
		}}
	}
	return nil
}

func (tg *taskGroup) Tasks(cfg pocket.Config) []*goyek.DefinedTask {
	_ = cfg.WithDefaults()
	var tasks []*goyek.DefinedTask

	if modules := pocket.ModulesFor(tg, "format"); len(modules) > 0 {
		tasks = append(tasks, goyek.Define(FormatTask(modules)))
	}

	return tasks
}

// FormatTask returns a task that formats Markdown files using mdformat.
func FormatTask(modules []string) goyek.Task {
	return goyek.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(a *goyek.A) {
			for _, mod := range modules {
				if err := mdformat.Run(a.Context(), mod); err != nil {
					a.Errorf("mdformat format failed in %s: %v", mod, err)
				}
			}
		},
	}
}
