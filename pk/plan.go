package pk

import (
	"context"
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
	// TODO: Will be populated during filesystem analysis.
	PathMappings map[string]pathInfo

	// ModuleDirectories lists directories where shims should be generated.
	// TODO: Will be derived from PathMappings.
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
func NewPlan(ctx context.Context, root Runnable) (*plan, error) {
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

	collector := &planCollector{
		tasks:        make([]*Task, 0),
		pathMappings: make(map[string]pathInfo),
		currentPath:  nil,
		gitRoot:      gitRoot,
	}

	if err := collector.walk(ctx, root); err != nil {
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

// planCollector is the internal state for walking the tree
type planCollector struct {
	tasks        []*Task
	pathMappings map[string]pathInfo
	currentPath  *pathFilter // Current path context during tree walk
	gitRoot      string      // Git repository root
}

// walk recursively traverses the Runnable tree
func (pc *planCollector) walk(ctx context.Context, r Runnable) error {
	if r == nil {
		return nil
	}

	// Type switch on the concrete Runnable types
	switch v := r.(type) {
	case *Task:
		// Leaf node - collect the task
		pc.tasks = append(pc.tasks, v)

		// Record path mapping if we're inside a pathFilter
		if pc.currentPath != nil {
			// Resolve paths against filesystem
			resolvedPaths, err := resolvePathPatterns(
				pc.gitRoot,
				pc.currentPath.includePaths,
				pc.currentPath.excludePaths,
			)
			if err != nil {
				// If resolution fails, use include patterns as-is
				resolvedPaths = pc.currentPath.includePaths
			}

			pc.pathMappings[v.name] = pathInfo{
				ResolvedPaths: resolvedPaths,
			}
		}

	case *serial:
		// Composition node - walk children sequentially
		for _, child := range v.runnables {
			if err := pc.walk(ctx, child); err != nil {
				return err
			}
		}

	case *parallel:
		// Composition node - walk children (order doesn't matter for collection)
		for _, child := range v.runnables {
			if err := pc.walk(ctx, child); err != nil {
				return err
			}
		}

	case *pathFilter:
		// Path filter wrapper - set context and walk inner
		previousPath := pc.currentPath
		pc.currentPath = v
		if err := pc.walk(ctx, v.inner); err != nil {
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

	return dirs
}
