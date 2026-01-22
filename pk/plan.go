package pk

import (
	"slices"
	"sort"
)

// Plan represents the execution plan created from a Config.
// It preserves the composition tree structure while extracting metadata.
//
// The plan is created once by analyzing both the Config and filesystem,
// then reused throughout execution. This is a PUBLIC API - users can access
// the plan to inspect what will execute, build custom tooling, or implement
// their own visualization.
//
// IMPORTANT: While Plan is exported for introspection, the composition types
// (serial, parallel, pathFilter) remain internal. Users should not rely on
// type assertions against Runnable - the composition structure may change.
type Plan struct {
	// tree is the composition tree that preserves dependencies and structure.
	// Execution walks this tree, respecting Serial/Parallel composition.
	// This is exposed as a Runnable, but the concrete types are internal.
	tree Runnable

	// tasks is a flat list of all tasks for lookup, CLI dispatch, and help.
	// This is extracted from walking the Auto tree.
	tasks []*Task

	// pathMappings maps task names to their execution directories.
	// Each task may execute in one or more directories based on path filtering.
	pathMappings map[string]pathInfo

	// moduleDirectories lists directories where shims should be generated.
	// These are derived from PathMappings during plan creation.
	moduleDirectories []string

	// shimConfig holds the shim generation configuration from Config.
	shimConfig *ShimConfig
}

// ShimConfig returns the resolved shim configuration.
// If no shim config was set in Config, returns DefaultShimConfig().
func (p *Plan) ShimConfig() *ShimConfig {
	if p.shimConfig == nil {
		return DefaultShimConfig()
	}
	return p.shimConfig
}

// pathInfo describes where a task should execute.
// This is part of the public Plan API for introspection.
type pathInfo struct {
	// includePaths is the original include patterns from WithIncludePath().
	// Used for visibility filtering (which tasks are visible from which paths).
	// Empty means the task runs at root only.
	includePaths []string

	// resolvedPaths is the list of actual directories where this task should run.
	// These are resolved from include/exclude patterns against the filesystem.
	// Paths are relative to git root, normalized with forward slashes.
	// Empty means the task runs at root (".").
	resolvedPaths []string
}

// NewPlan creates an execution plan from a Config.
// It walks the composition tree to extract tasks and analyzes the filesystem.
// The filesystem is traversed ONCE during plan creation.
func NewPlan(cfg *Config) (*Plan, error) {
	gitRoot := findGitRoot()

	// Resolve skip dirs: nil uses defaults, empty slice skips nothing
	skipDirs := cfg.SkipDirs
	if skipDirs == nil {
		skipDirs = DefaultSkipDirs
	}

	allDirs, err := walkDirectories(gitRoot, skipDirs, cfg.IncludeHiddenDirs)
	if err != nil {
		return nil, err
	}
	return newPlan(cfg, gitRoot, allDirs)
}

func newPlan(cfg *Config, gitRoot string, allDirs []string) (*Plan, error) {
	if cfg == nil {
		return &Plan{
			tree:              nil,
			tasks:             []*Task{},
			pathMappings:      make(map[string]pathInfo),
			moduleDirectories: []string{},
			shimConfig:        nil,
		}, nil
	}

	if cfg.Auto == nil && len(cfg.Manual) == 0 {
		return &Plan{
			tree:              nil,
			tasks:             []*Task{},
			pathMappings:      make(map[string]pathInfo),
			moduleDirectories: []string{},
			shimConfig:        cfg.Shims,
		}, nil
	}

	collector := &taskCollector{
		tasks:        make([]*Task, 0),
		seenTasks:    make(map[*Task]bool),
		pathMappings: make(map[string]pathInfo),
		currentPath:  nil,
		gitRoot:      gitRoot,
		allDirs:      allDirs,
	}

	// Walk the Auto tree
	if cfg.Auto != nil {
		if err := collector.walk(cfg.Auto); err != nil {
			return nil, err
		}
	}

	// Walk manual tasks (they are marked as manual via Task.Manual())
	for _, r := range cfg.Manual {
		if err := collector.walk(r); err != nil {
			return nil, err
		}
	}

	// Derive ModuleDirectories from pathMappings (single source of truth)
	moduleDirectories := deriveModuleDirectories(collector.pathMappings)

	return &Plan{
		tree:              cfg.Auto, // Preserve the composition tree!
		tasks:             collector.tasks,
		pathMappings:      collector.pathMappings,
		moduleDirectories: moduleDirectories,
		shimConfig:        cfg.Shims,
	}, nil
}

// taskCollector is the internal state for walking the tree.
type taskCollector struct {
	tasks        []*Task
	seenTasks    map[*Task]bool // Track seen task pointers for deduplication.
	pathMappings map[string]pathInfo
	currentPath  *pathFilter // Current path context during tree walk.
	gitRoot      string      // Git repository root.
	allDirs      []string    // Cached directory list from filesystem walk.

	// Cumulative state
	candidates     []string         // Current allowed directories.
	activeExcludes []excludePattern // All excludes in current scope.
	activeSkips    []string         // All skipped tasks in current scope.
}

// filterPaths applies detection function or include patterns, then exclude patterns.
// If detectFunc is set, it takes precedence over includePaths.
func (pc *taskCollector) filterPaths(pf *pathFilter) []string {
	var results []string

	// Start with current candidates, but apply all GLOBAL active excludes first.
	// This ensures detection functions and inner include patterns don't see
	// directories that were excluded by an outer scope.
	candidates := pc.candidates
	for _, ex := range pc.activeExcludes {
		if len(ex.tasks) == 0 {
			candidates = excludeByPatterns(candidates, []string{ex.pattern})
		}
	}

	// If we have a detection function, it runs against the filtered candidates.
	// If no detection function, we either filter filtered candidates by includePaths
	// or keep all filtered candidates if no includePaths are specified.
	switch {
	case pf.detectFunc != nil:
		results = pf.detectFunc(candidates, pc.gitRoot)
	case len(pf.includePaths) > 0:
		for _, dir := range candidates {
			for _, pattern := range pf.includePaths {
				if matchPattern(dir, pattern) {
					results = append(results, dir)
					break
				}
			}
		}
	default:
		// No detection and no new include patterns - keep filtered candidates.
		results = candidates
	}

	return results
}

// walk recursively traverses the Runnable tree.
func (pc *taskCollector) walk(r Runnable) error {
	if r == nil {
		return nil
	}

	// Initialize candidates if this is the root call.
	if pc.candidates == nil {
		pc.candidates = pc.allDirs
	}

	// Type switch on the concrete Runnable types.
	switch v := r.(type) {
	case *Task:
		// Check if task is skipped in current scope.
		if slices.Contains(pc.activeSkips, v.name) {
			return nil
		}

		// Only collect unique task pointers for the flat tasks list.
		if !pc.seenTasks[v] {
			pc.seenTasks[v] = true
			pc.tasks = append(pc.tasks, v)
		}

		// Determine final paths for this task.
		// If task is NOT inside any pathFilter, run at root only.
		// If inside a pathFilter, use the filtered candidates with excludes applied.
		var finalPaths []string
		if pc.currentPath == nil {
			// No pathFilter - run at root only
			finalPaths = []string{"."}
		} else {
			// Inside a pathFilter - apply excludes to current candidates
			finalPaths = pc.candidates
			for _, ex := range pc.activeExcludes {
				// If ex.tasks is empty, it's a global exclude for this scope.
				// If ex.tasks is not empty, it only applies if this task is in the list.
				if len(ex.tasks) == 0 || slices.Contains(ex.tasks, v.name) {
					finalPaths = excludeByPatterns(finalPaths, []string{ex.pattern})
				}
			}
		}

		// Record path mapping.
		// Use ALL include paths from the hierarchy for visibility/shims.
		var allIncludes []string
		if pc.currentPath != nil {
			allIncludes = pc.currentPath.includePaths
		}
		if len(allIncludes) == 0 {
			allIncludes = []string{"."}
		}

		pc.pathMappings[v.name] = pathInfo{
			includePaths:  allIncludes,
			resolvedPaths: finalPaths,
		}

	case *serial:
		for _, child := range v.runnables {
			if err := pc.walk(child); err != nil {
				return err
			}
		}

	case *parallel:
		for _, child := range v.runnables {
			if err := pc.walk(child); err != nil {
				return err
			}
		}

	case *pathFilter:
		// 1. Resolve paths for this filter based on current candidates.
		v.resolvedPaths = pc.filterPaths(v)

		// 2. Save state for nesting.
		prevCandidates := pc.candidates
		prevExcludes := pc.activeExcludes
		prevSkips := pc.activeSkips
		prevPath := pc.currentPath

		// 3. Update state with new constraints.
		pc.candidates = v.resolvedPaths
		pc.activeExcludes = append(pc.activeExcludes, v.excludePaths...)
		pc.activeSkips = append(pc.activeSkips, v.skippedTasks...)
		pc.currentPath = v

		// 4. Walk inner with cumulative state.
		if err := pc.walk(v.inner); err != nil {
			return err
		}

		// 5. Restore state.
		pc.candidates = prevCandidates
		pc.activeExcludes = prevExcludes
		pc.activeSkips = prevSkips
		pc.currentPath = prevPath

	default:
		// Unknown runnable type - skip it
	}

	return nil
}

// excludeByPatterns filters out directories matching any of the patterns.
func excludeByPatterns(dirs, patterns []string) []string {
	if len(patterns) == 0 {
		return dirs
	}
	var result []string
	for _, dir := range dirs {
		excluded := false
		for _, pattern := range patterns {
			if matchPattern(dir, pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, dir)
		}
	}
	return result
}

// deriveModuleDirectories returns directories where shims should be generated.
// Shims are generated at:
//  1. Root (".") - always included if any tasks exist
//  2. Each unique include path pattern from WithIncludePath()
//
// This differs from resolved paths: if WithIncludePath("internal") is used,
// we generate a shim at "internal/", NOT at "internal/shim/", "internal/scaffold/", etc.
//
// This function derives shim directories from pathMappings, which already contains
// the includePaths for each task. This avoids tracking shim directories separately
// during the tree walk (single source of truth).
func deriveModuleDirectories(pathMappings map[string]pathInfo) []string {
	// Collect unique include paths from all tasks
	seen := make(map[string]bool)
	seen["."] = true // Always include root

	for _, info := range pathMappings {
		for _, p := range info.includePaths {
			seen[p] = true
		}
	}

	// Convert to sorted slice
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	return dirs
}

// taskRunsInPath checks if a task is visible/runnable from a specific path context.
// Used to filter help output based on TASK_SCOPE environment variable.
//
// Rules:
//   - If path is "" or "." (root), all tasks are visible
//   - Otherwise, task is visible if path matches any of the task's includePaths
//   - Tasks without includePaths (root-only tasks) are only visible from root
func (p *Plan) taskRunsInPath(taskName, path string) bool {
	// Root context sees all tasks
	if path == "" || path == "." {
		return true
	}

	info, ok := p.pathMappings[taskName]
	if !ok {
		// Task has no path mapping - it's a root-only task
		return false
	}

	return slices.Contains(info.includePaths, path)
}
