package pocket

import (
	"sort"

	"github.com/goyek/goyek/v3"
)

// Config defines the configuration for a project using pocket.
type Config struct {
	// Shim controls shim script generation.
	// By default, only Posix (./pok) is generated with name "pok".
	Shim *ShimConfig

	// SkipGitDiff disables the git diff check at the end of the "all" task.
	// By default, "all" fails if there are uncommitted changes after running all tasks.
	// Set to true to disable this check.
	SkipGitDiff bool

	// Language configurations
	Go  *GoConfig
	Lua *LuaConfig

	// Documentation
	Markdown *MarkdownConfig

	// Custom maps folder paths to custom goyek tasks.
	// Use "." for the root context.
	// Tasks are included in the "all" task and shown in help output.
	//
	// Example:
	//
	//	Custom: map[string][]goyek.Task{
	//	    ".": {{Name: "deploy", Usage: "deploy the app", Action: deployAction}},
	//	},
	Custom map[string][]goyek.Task
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

// GoConfig defines Go project configuration.
type GoConfig struct {
	// Modules maps folder paths to their options.
	// Use "." for the root module.
	Modules map[string]GoModuleOptions
}

// GoModuleOptions defines options for a Go module.
type GoModuleOptions struct {
	SkipFormat    bool
	SkipTest      bool
	SkipLint      bool
	SkipVulncheck bool
}

// MarkdownConfig defines Markdown project configuration.
type MarkdownConfig struct {
	// Modules maps folder paths to their options.
	// Use "." for the root module.
	Modules map[string]MarkdownModuleOptions
}

// MarkdownModuleOptions defines options for a Markdown module.
type MarkdownModuleOptions struct {
	SkipFormat bool
}

// LuaConfig defines Lua project configuration.
type LuaConfig struct {
	// Modules maps folder paths to their options.
	// Use "." for the root module.
	Modules map[string]LuaModuleOptions
}

// LuaModuleOptions defines options for a Lua module.
type LuaModuleOptions struct {
	SkipFormat bool
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

// HasGo returns true if the project has Go modules configured.
func (c Config) HasGo() bool {
	return c.Go != nil && len(c.Go.Modules) > 0
}

// HasMarkdown returns true if markdown formatting is configured.
func (c Config) HasMarkdown() bool {
	return c.Markdown != nil && len(c.Markdown.Modules) > 0
}

// HasLua returns true if lua formatting is configured.
func (c Config) HasLua() bool {
	return c.Lua != nil && len(c.Lua.Modules) > 0
}

// CustomTasks returns all custom tasks from the config.
// For a filtered config (via ForContext), this returns only the tasks for that context.
func (c Config) CustomTasks() []goyek.Task {
	var tasks []goyek.Task
	for _, contextTasks := range c.Custom {
		tasks = append(tasks, contextTasks...)
	}
	return tasks
}

// MarkdownModulesForFormat returns module paths where format is not skipped.
func (c Config) MarkdownModulesForFormat() []string {
	if c.Markdown == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Markdown.Modules {
		if !opts.SkipFormat {
			paths = append(paths, path)
		}
	}
	return paths
}

// LuaModulesForFormat returns module paths where format is not skipped.
func (c Config) LuaModulesForFormat() []string {
	if c.Lua == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Lua.Modules {
		if !opts.SkipFormat {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForFormat returns module paths where format is not skipped.
func (c Config) GoModulesForFormat() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipFormat {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForTest returns module paths where test is not skipped.
func (c Config) GoModulesForTest() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipTest {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForLint returns module paths where lint is not skipped.
func (c Config) GoModulesForLint() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipLint {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForVulncheck returns module paths where vulncheck is not skipped.
func (c Config) GoModulesForVulncheck() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipVulncheck {
			paths = append(paths, path)
		}
	}
	return paths
}

// UniqueModulePaths returns all unique directory paths across all language configs.
// The paths are sorted and always include "." for the root context.
func (c Config) UniqueModulePaths() []string {
	seen := make(map[string]bool)
	seen["."] = true // Always include root.

	if c.Go != nil {
		for path := range c.Go.Modules {
			seen[path] = true
		}
	}
	if c.Lua != nil {
		for path := range c.Lua.Modules {
			seen[path] = true
		}
	}
	if c.Markdown != nil {
		for path := range c.Markdown.Modules {
			seen[path] = true
		}
	}
	for path := range c.Custom {
		seen[path] = true
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// ForContext returns a filtered config containing only modules for the given path.
// For the root context ("."), returns the full config unchanged.
func (c Config) ForContext(context string) Config {
	if context == "." {
		return c
	}

	filtered := Config{
		Shim:        c.Shim,        // Preserve shim config.
		SkipGitDiff: c.SkipGitDiff, // Preserve git diff setting.
	}

	// Filter Go modules.
	if c.Go != nil {
		if opts, ok := c.Go.Modules[context]; ok {
			filtered.Go = &GoConfig{
				Modules: map[string]GoModuleOptions{
					context: opts,
				},
			}
		}
	}

	// Filter Lua modules.
	if c.Lua != nil {
		if opts, ok := c.Lua.Modules[context]; ok {
			filtered.Lua = &LuaConfig{
				Modules: map[string]LuaModuleOptions{
					context: opts,
				},
			}
		}
	}

	// Filter Markdown modules.
	if c.Markdown != nil {
		if opts, ok := c.Markdown.Modules[context]; ok {
			filtered.Markdown = &MarkdownConfig{
				Modules: map[string]MarkdownModuleOptions{
					context: opts,
				},
			}
		}
	}

	// Filter custom tasks.
	if tasks, ok := c.Custom[context]; ok {
		filtered.Custom = map[string][]goyek.Task{
			context: tasks,
		}
	}

	return filtered
}
