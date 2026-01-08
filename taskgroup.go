package pocket

import "github.com/goyek/goyek/v3"

// TaskGroup defines a collection of related tasks for a technology (e.g., Go, Lua, Markdown).
// Implement this interface to create custom task groups.
type TaskGroup interface {
	// Name returns the task group identifier (e.g., "go", "lua", "markdown").
	Name() string

	// Modules returns the module configuration for this task group.
	// The map keys are context paths (e.g., ".", "tests", "services/api").
	Modules() map[string]ModuleConfig

	// Tasks returns the goyek tasks provided by this task group.
	Tasks(cfg Config) []*goyek.DefinedTask

	// ForContext returns a filtered task group containing only modules for the given path.
	// For the root context ("."), returns the task group unchanged.
	// Returns nil if the task group has no modules for the given context.
	ForContext(context string) TaskGroup
}

// ModuleConfig defines configuration for a module within a task group.
// Task groups can implement this interface with their own types to support
// task-group-specific options while maintaining compatibility with pocket's
// filtering and orchestration.
type ModuleConfig interface {
	// ShouldRun returns true if the given task should run based on the configuration.
	// The task parameter is the task name without prefix (e.g., "format", "lint", "test").
	ShouldRun(task string) bool
}

// ModulesFor returns module paths where the given task should run for a task group.
func ModulesFor(tg TaskGroup, task string) []string {
	modules := tg.Modules()
	if modules == nil {
		return nil
	}
	var paths []string
	for path, opts := range modules {
		if opts.ShouldRun(task) {
			paths = append(paths, path)
		}
	}
	return paths
}

// AllTaskGroupModulePaths returns all unique module paths across all task groups.
func AllTaskGroupModulePaths(taskGroups []TaskGroup) []string {
	seen := make(map[string]bool)
	for _, tg := range taskGroups {
		for path := range tg.Modules() {
			seen[path] = true
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	return paths
}

// AllModulePaths returns all unique module paths across task groups and config tasks.
// The paths always include "." for the root context.
func AllModulePaths(cfg Config) []string {
	seen := make(map[string]bool)
	seen["."] = true // Always include root.

	// Add task group module paths.
	for _, tg := range cfg.TaskGroups {
		for path := range tg.Modules() {
			seen[path] = true
		}
	}

	// Add task paths.
	for path := range cfg.Tasks {
		seen[path] = true
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	return paths
}

// FilterTaskGroupsForContext returns task groups filtered for the given context.
// Task groups with no modules for the context are excluded.
func FilterTaskGroupsForContext(taskGroups []TaskGroup, context string) []TaskGroup {
	if context == "." {
		return taskGroups
	}
	var filtered []TaskGroup
	for _, tg := range taskGroups {
		if ftg := tg.ForContext(context); ftg != nil {
			filtered = append(filtered, ftg)
		}
	}
	return filtered
}
