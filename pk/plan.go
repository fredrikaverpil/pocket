package pk

import (
	"fmt"
	"sort"
)

// plan represents the execution plan created from a Config.
// It preserves the composition tree structure while extracting metadata.
// The plan is created once by analyzing both the Config and filesystem,
// then reused throughout execution.
type plan struct {
	// Root is the composition tree that preserves dependencies and structure.
	// Execution walks this tree, respecting Serial/Parallel composition.
	Root Runnable

	// Tasks is a flat list of all tasks for lookup, CLI dispatch, and help.
	// This is extracted from walking the Root tree.
	Tasks []*Task

	// PathMappings maps task names to their execution directories.
	PathMappings map[string]pathInfo

	// ModuleDirectories lists directories where shims should be generated.
	ModuleDirectories []string
}

// pathInfo describes where a task should execute.
type pathInfo struct {
	// ResolvedPaths is the list of actual directories where this task should run.
	// These are resolved from include/exclude patterns against the filesystem.
	// Paths are relative to git root, normalized with forward slashes.
	ResolvedPaths []string
}

// NewPlan creates an execution plan from a Config root.
// It walks the composition tree to extract tasks and analyzes the filesystem.
// The filesystem is traversed ONCE during plan creation.
func NewPlan(root Runnable) (*plan, error) {
	if root == nil {
		return &plan{
			Root:              nil,
			Tasks:             []*Task{},
			PathMappings:      make(map[string]pathInfo),
			ModuleDirectories: []string{},
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
		taskNames:    make(map[string]bool),
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
		Root:              root, // Preserve the composition tree!
		Tasks:             collector.tasks,
		PathMappings:      collector.pathMappings,
		ModuleDirectories: moduleDirectories,
	}, nil
}

// taskCollector is the internal state for walking the tree.
type taskCollector struct {
	tasks        []*Task
	taskNames    map[string]bool // Track seen task names for duplicate detection
	pathMappings map[string]pathInfo
	currentPath  *pathFilter // Current path context during tree walk
	gitRoot      string      // Git repository root
	allDirs      []string    // Cached directory list from filesystem walk
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

	// Type switch on the concrete Runnable types
	switch v := r.(type) {
	case *Task:
		// Check for duplicate task names
		if pc.taskNames[v.name] {
			return fmt.Errorf("duplicate task name: %q", v.name)
		}
		pc.taskNames[v.name] = true

		// Leaf node - collect the task
		pc.tasks = append(pc.tasks, v)

		// Record path mapping if we're inside a pathFilter
		// (uses already-resolved paths from the pathFilter)
		if pc.currentPath != nil {
			pc.pathMappings[v.name] = pathInfo{
				ResolvedPaths: pc.currentPath.resolvedPaths,
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
		for _, path := range info.ResolvedPaths {
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
