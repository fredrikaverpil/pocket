package pk

import (
	"sort"
)

// plan represents the execution plan created from a Config.
// It preserves the composition tree structure while extracting metadata.
//
// The plan is created once by analyzing both the Config and filesystem,
// then reused throughout execution. This is a PUBLIC API - users can access
// the plan to inspect what will execute, build custom tooling, or implement
// their own visualization.
//
// IMPORTANT: While plan is exported for introspection, the composition types
// (serial, parallel, pathFilter) remain internal. Users should not rely on
// type assertions against Runnable - the composition structure may change.
type plan struct {
	// root is the composition tree that preserves dependencies and structure.
	// Execution walks this tree, respecting Serial/Parallel composition.
	// This is exposed as a Runnable, but the concrete types are internal.
	root Runnable

	// tasks is a flat list of all tasks for lookup, CLI dispatch, and help.
	// This is extracted from walking the Root tree.
	tasks []*Task

	// pathMappings maps task names to their execution directories.
	// Each task may execute in one or more directories based on path filtering.
	pathMappings map[string]pathInfo

	// moduleDirectories lists directories where shims should be generated.
	// These are derived from PathMappings during plan creation.
	moduleDirectories []string
}

// pathInfo describes where a task should execute.
// This is part of the public Plan API for introspection.
type pathInfo struct {
	// resolvedPaths is the list of actual directories where this task should run.
	// These are resolved from include/exclude patterns against the filesystem.
	// Paths are relative to git root, normalized with forward slashes.
	// Empty means the task runs at root (".").
	resolvedPaths []string
}

// newPlan creates an execution plan from a Config root.
// It walks the composition tree to extract tasks and analyzes the filesystem.
// The filesystem is traversed ONCE during plan creation.
func newPlan(root Runnable) (*plan, error) {
	if root == nil {
		return &plan{
			root:              nil,
			tasks:             []*Task{},
			pathMappings:      make(map[string]pathInfo),
			moduleDirectories: []string{},
		}, nil
	}

	// Find git root once for the entire plan
	gitRoot := findGitRoot()

	// Walk filesystem once for the entire plan
	allDirs, err := walkDirectories(gitRoot)
	if err != nil {
		return nil, err
	}

	collector := &taskCollector{
		tasks:        make([]*Task, 0),
		seenTasks:    make(map[*Task]bool),
		pathMappings: make(map[string]pathInfo),
		currentPath:  nil,
		gitRoot:      gitRoot,
		allDirs:      allDirs,
	}

	if err := collector.walk(root); err != nil {
		return nil, err
	}

	// Derive ModuleDirectories from PathMappings
	moduleDirectories := deriveModuleDirectories(collector.pathMappings)

	return &plan{
		root:              root, // Preserve the composition tree!
		tasks:             collector.tasks,
		pathMappings:      collector.pathMappings,
		moduleDirectories: moduleDirectories,
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
}

// filterPaths applies include/exclude patterns to the cached directory list.
func (pc *taskCollector) filterPaths(includePaths, excludePaths []string) []string {
	// If no include patterns, default to root only
	var candidates []string
	if len(includePaths) == 0 {
		candidates = []string{"."}
	} else {
		// Filter by include patterns
		for _, dir := range pc.allDirs {
			for _, pattern := range includePaths {
				if matchPattern(dir, pattern) {
					candidates = append(candidates, dir)
					break
				}
			}
		}
	}

	// Apply exclude patterns
	var result []string
	for _, dir := range candidates {
		excluded := false
		for _, pattern := range excludePaths {
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

// walk recursively traverses the Runnable tree.
func (pc *taskCollector) walk(r Runnable) error {
	if r == nil {
		return nil
	}

	// Type switch on the concrete Runnable types.
	switch v := r.(type) {
	case *Task:
		// Only collect unique task pointers.
		// The same task can appear multiple times in the tree, but we only
		// add it once to the tasks list. Deduplication during execution is
		// handled by the executionTracker.
		if !pc.seenTasks[v] {
			pc.seenTasks[v] = true
			pc.tasks = append(pc.tasks, v)
		}

		// Record path mapping if we're inside a pathFilter.
		// (uses already-resolved paths from the pathFilter)
		if pc.currentPath != nil {
			pc.pathMappings[v.name] = pathInfo{
				resolvedPaths: pc.currentPath.resolvedPaths,
			}
		}

	case *serial:
		// Composition node - walk children sequentially
		for _, child := range v.runnables {
			if err := pc.walk(child); err != nil {
				return err
			}
		}

	case *parallel:
		// Composition node - walk children (order doesn't matter for collection)
		for _, child := range v.runnables {
			if err := pc.walk(child); err != nil {
				return err
			}
		}

	case *pathFilter:
		// Path filter wrapper - resolve paths using cached dirs, then walk inner
		v.resolvedPaths = pc.filterPaths(v.includePaths, v.excludePaths)

		// Set context and walk inner
		previousPath := pc.currentPath
		pc.currentPath = v
		if err := pc.walk(v.inner); err != nil {
			return err
		}
		pc.currentPath = previousPath

	default:
		// Unknown runnable type - skip it
		// This allows new types to be added without breaking plan building
	}

	return nil
}

// deriveModuleDirectories extracts unique directories from path mappings.
// These directories are where shims should be generated.
func deriveModuleDirectories(pathMappings map[string]pathInfo) []string {
	seen := make(map[string]bool)
	var dirs []string

	for _, info := range pathMappings {
		for _, path := range info.resolvedPaths {
			if !seen[path] {
				seen[path] = true
				dirs = append(dirs, path)
			}
		}
	}

	// Sort for deterministic output
	sort.Strings(dirs)

	return dirs
}
