package pocket

import (
	"slices"
	"sort"

	"github.com/goyek/goyek/v3"
)

// Config defines the configuration for a project using pocket.
type Config struct {
	// TaskGroups contains task groups (e.g., Go, Lua, Markdown) to enable.
	// Each task group provides related tasks and owns its configuration.
	//
	// Example:
	//
	//	TaskGroups: []pocket.TaskGroup{
	//	    golang.New(golang.Config{Modules: map[string]golang.Options{".": {}}}),
	//	    markdown.New(markdown.Config{Modules: map[string]markdown.Options{".": {}}}),
	//	},
	TaskGroups []TaskGroup

	// Tasks maps folder paths to custom goyek tasks.
	// Use "." for the root context.
	// Tasks are included in the "all" task and shown in help output.
	//
	// Example:
	//
	//	Tasks: map[string][]goyek.Task{
	//	    ".": {{Name: "deploy", Usage: "deploy the app", Action: deployAction}},
	//	},
	Tasks map[string][]goyek.Task

	// Shim controls shim script generation.
	// By default, only Posix (./pok) is generated with name "pok".
	Shim *ShimConfig

	// SkipGitDiff disables the git diff check at the end of the "all" task.
	// By default, "all" fails if there are uncommitted changes after running all tasks.
	// Set to true to disable this check.
	SkipGitDiff bool
}

// ShimConfig controls shim script generation.
type ShimConfig struct {
	// Name is the base name of the generated shim scripts (without extension).
	// Default: "pok"
	Name string

	// Posix generates a bash script (./pok).
	// This is enabled by default if ShimConfig is nil.
	Posix bool

	// Windows generates a batch file (pok.cmd).
	// The batch file requires Go to be installed and in PATH.
	Windows bool

	// PowerShell generates a PowerShell script (pok.ps1).
	// The PowerShell script can auto-download Go if not found.
	PowerShell bool
}

// BaseModuleConfig provides a default implementation of ModuleConfig.
// Task groups can embed this or define their own config types.
type BaseModuleConfig struct {
	// Skip lists task names to skip (e.g., "format", "lint", "test").
	Skip []string
	// Only lists task names to run (empty = run all).
	// If non-empty, only these tasks run (Skip is ignored).
	Only []string
}

// ShouldRun returns true if the given task should run based on Skip/Only options.
func (o BaseModuleConfig) ShouldRun(task string) bool {
	if len(o.Only) > 0 {
		return slices.Contains(o.Only, task)
	}
	return !slices.Contains(o.Skip, task)
}

// WithDefaults returns a copy of the config with default values applied.
func (c Config) WithDefaults() Config {
	// Default to Posix shim only if no Shim config is provided.
	if c.Shim == nil {
		c.Shim = &ShimConfig{Posix: true}
	}
	// Apply shim defaults.
	shim := *c.Shim
	if shim.Name == "" {
		shim.Name = "pok"
	}
	c.Shim = &shim

	return c
}

// GetTasks returns all custom tasks from the config.
// For a filtered config (via ForContext), this returns only the tasks for that context.
func (c Config) GetTasks() []goyek.Task {
	var tasks []goyek.Task
	for _, contextTasks := range c.Tasks {
		tasks = append(tasks, contextTasks...)
	}
	return tasks
}

// TaskPaths returns unique directory paths from custom tasks.
// The paths are sorted.
func (c Config) TaskPaths() []string {
	seen := make(map[string]bool)
	for path := range c.Tasks {
		seen[path] = true
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// ForContext returns a filtered config containing only tasks for the given path.
// For the root context ("."), returns the full config unchanged.
// Note: TaskGroup filtering is handled by TaskGroup.ForContext() separately.
func (c Config) ForContext(context string) Config {
	if context == "." {
		return c
	}

	filtered := Config{
		Shim:        c.Shim,        // Preserve shim config.
		SkipGitDiff: c.SkipGitDiff, // Preserve git diff setting.
	}

	// Filter task groups for context.
	filtered.TaskGroups = FilterTaskGroupsForContext(c.TaskGroups, context)

	// Filter custom tasks.
	if tasks, ok := c.Tasks[context]; ok {
		filtered.Tasks = map[string][]goyek.Task{
			context: tasks,
		}
	}

	return filtered
}
