package pk

import (
	"context"
	"fmt"
	"slices"
	"sort"
)

// planKey is the context key for the execution plan.
type planKey struct{}

// PlanFromContext returns the Plan from the context.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) *Plan {
	if p, ok := ctx.Value(planKey{}).(*Plan); ok {
		return p
	}
	return nil
}

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

	// taskInstances is a flat list of all tasks with their effective names.
	// This is extracted from walking the Auto tree.
	// The effective name includes any suffix from WithNameSuffix (e.g., "py-test:3.9").
	taskInstances []taskInstance

	// taskIndex maps effective task names to taskInstance for fast lookup.
	// This is used during execution to retrieve pre-computed data.
	taskIndex map[string]*taskInstance

	// pathMappings maps effective task names to their execution directories.
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
	var skipDirs []string
	var includeHidden bool
	if cfg.Plan != nil {
		skipDirs = cfg.Plan.SkipDirs
		includeHidden = cfg.Plan.IncludeHiddenDirs
	}
	if skipDirs == nil {
		skipDirs = DefaultSkipDirs
	}

	allDirs, err := walkDirectories(gitRoot, skipDirs, includeHidden)
	if err != nil {
		return nil, err
	}
	return newPlan(cfg, gitRoot, allDirs)
}

func newPlan(cfg *Config, gitRoot string, allDirs []string) (*Plan, error) {
	if cfg == nil {
		return &Plan{
			tree:              nil,
			taskInstances:     []taskInstance{},
			taskIndex:         make(map[string]*taskInstance),
			pathMappings:      make(map[string]pathInfo),
			moduleDirectories: []string{},
			shimConfig:        nil,
		}, nil
	}

	// Extract plan config fields
	var shimConfig *ShimConfig
	if cfg.Plan != nil {
		shimConfig = cfg.Plan.Shims
	}

	if cfg.Auto == nil && len(cfg.Manual) == 0 {
		return &Plan{
			tree:              nil,
			taskInstances:     []taskInstance{},
			taskIndex:         make(map[string]*taskInstance),
			pathMappings:      make(map[string]pathInfo),
			moduleDirectories: []string{},
			shimConfig:        shimConfig,
		}, nil
	}

	collector := &taskCollector{
		taskInstances: make([]taskInstance, 0),
		seenTasks:     make(map[taskKey]bool),
		pathMappings:  make(map[string]pathInfo),
		currentPath:   nil,
		gitRoot:       gitRoot,
		allDirs:       allDirs,
	}

	// Walk the Auto tree
	if cfg.Auto != nil {
		if err := collector.walk(cfg.Auto); err != nil {
			return nil, err
		}
	}

	// Walk manual tasks - tasks in Config.Manual are automatically marked manual.
	collector.inManualSection = true
	for _, r := range cfg.Manual {
		if err := collector.walk(r); err != nil {
			return nil, err
		}
	}

	// Check for task name conflicts (builtins and duplicates)
	if err := checkTaskNameConflicts(collector.taskInstances); err != nil {
		return nil, err
	}

	// Derive ModuleDirectories from pathMappings (single source of truth)
	moduleDirectories := deriveModuleDirectories(collector.pathMappings)

	// Build task index for fast lookup during execution.
	taskIndex := make(map[string]*taskInstance, len(collector.taskInstances))
	for i := range collector.taskInstances {
		taskIndex[collector.taskInstances[i].name] = &collector.taskInstances[i]
	}

	return &Plan{
		tree:              cfg.Auto, // Preserve the composition tree!
		taskInstances:     collector.taskInstances,
		taskIndex:         taskIndex,
		pathMappings:      collector.pathMappings,
		moduleDirectories: moduleDirectories,
		shimConfig:        shimConfig,
	}, nil
}

// taskInstance stores a task with its effective name and pre-computed configuration.
// All computation happens during planning; execution just reads these values.
type taskInstance struct {
	task          *Task
	name          string         // Effective name (may include suffix like "py-test:3.9")
	contextValues []contextValue // Context values to apply when executing this task instance.
	flags         map[string]any // Pre-merged flag overrides for this task.
	isManual      bool           // Whether task is manual (from Config.Manual or Task.Manual()).

	// Execution context from path filter.
	resolvedPaths []string // Directories where this task executes.
}

// taskCollector is the internal state for walking the tree.
type taskCollector struct {
	taskInstances []taskInstance   // Tasks with their effective names.
	seenTasks     map[taskKey]bool // Track seen (task, suffix) pairs for deduplication.
	pathMappings  map[string]pathInfo
	currentPath   *pathFilter // Current path context during tree walk.
	gitRoot       string      // Git repository root.
	allDirs       []string    // Cached directory list from filesystem walk.

	// Cumulative state
	candidates          []string         // Current allowed directories.
	activeExcludes      []excludePattern // All excludes in current scope.
	activeSkips         []string         // All skipped tasks in current scope.
	activeNameSuffix    string           // Current name suffix from WithNameSuffix.
	activeContextValues []contextValue   // Context values in current scope.
	activeFlags         []flagOverride   // All flag overrides in current scope.
	inManualSection     bool             // True when walking Config.Manual tasks.
}

// taskKey uniquely identifies a task in a specific naming context.
type taskKey struct {
	task   *Task
	suffix string
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
		// No detection and no include patterns specified.
		// If we're nested inside another pathFilter, inherit its paths.
		// If we have exclude patterns, use all directories (excludes apply later).
		// Otherwise, default to root only.
		if pc.currentPath != nil || len(pf.excludePaths) > 0 {
			results = candidates
		} else {
			results = []string{"."}
		}
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

		// Build effective name with suffix (e.g., "py-test:3.9").
		effectiveName := v.name
		if pc.activeNameSuffix != "" {
			effectiveName = v.name + ":" + pc.activeNameSuffix
		}

		// Determine final paths for this task.
		// If task is NOT inside any pathFilter, run at root only.
		// If inside a pathFilter, use the filtered candidates with excludes applied.
		var finalPaths []string
		if pc.currentPath == nil {
			// No pathFilter - run at root only
			finalPaths = []string{"."}
		} else {
			// Inside a pathFilter - apply excludes to current candidates.
			// First apply global excludes (WithExcludePath), then task-specific (WithExcludeTask).
			finalPaths = pc.candidates

			// Apply global excludes first.
			for _, ex := range pc.activeExcludes {
				if len(ex.tasks) == 0 {
					finalPaths = excludeByPatterns(finalPaths, []string{ex.pattern})
				}
			}

			// Check for configuration error: global excludes removed all detected paths.
			// This likely indicates a misconfiguration where the user is running tasks
			// but excluding all directories that would match.
			if len(finalPaths) == 0 && len(pc.candidates) > 0 {
				return fmt.Errorf("task %q: excludes removed all %d detected path(s); "+
					"either adjust excludes or use WithSkipTask to skip this task entirely",
					effectiveName, len(pc.candidates))
			}

			// Apply task-specific excludes (WithExcludeTask).
			// If this removes all paths for this task, that's intentional - the task just won't run.
			for _, ex := range pc.activeExcludes {
				if len(ex.tasks) > 0 && slices.Contains(ex.tasks, v.name) {
					finalPaths = excludeByPatterns(finalPaths, []string{ex.pattern})
				}
			}
		}

		// Only collect unique (task, suffix) pairs for the flat tasks list.
		key := taskKey{task: v, suffix: pc.activeNameSuffix}
		if !pc.seenTasks[key] {
			pc.seenTasks[key] = true
			// Copy context values to avoid sharing the slice.
			var ctxValues []contextValue
			if len(pc.activeContextValues) > 0 {
				ctxValues = make([]contextValue, len(pc.activeContextValues))
				copy(ctxValues, pc.activeContextValues)
			}
			// Pre-merge flags for this task from all active scopes.
			var mergedFlags map[string]any
			for _, f := range pc.activeFlags {
				if f.taskName == v.name {
					if mergedFlags == nil {
						mergedFlags = make(map[string]any)
					}
					mergedFlags[f.flagName] = f.value
				}
			}
			pc.taskInstances = append(pc.taskInstances, taskInstance{
				task:          v,
				name:          effectiveName,
				contextValues: ctxValues,
				flags:         mergedFlags,
				isManual:      pc.inManualSection || v.IsManual(),
				resolvedPaths: finalPaths,
			})
		}

		// Record path mapping using effective name.
		// Used for visibility filtering, shim generation, and plan introspection.
		var allIncludes []string
		if pc.currentPath != nil {
			allIncludes = pc.currentPath.includePaths
		}
		if len(allIncludes) == 0 {
			allIncludes = []string{"."}
		}

		pc.pathMappings[effectiveName] = pathInfo{
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
		prevNameSuffix := pc.activeNameSuffix
		prevContextValues := pc.activeContextValues
		prevFlags := pc.activeFlags

		// 3. Update state with new constraints.
		pc.candidates = v.resolvedPaths
		pc.activeExcludes = append(pc.activeExcludes, v.excludePaths...)
		pc.activeSkips = append(pc.activeSkips, v.skippedTasks...)
		pc.activeFlags = append(pc.activeFlags, v.flags...)
		pc.currentPath = v

		// Apply name suffix (cumulative: "3.9" + "foo" -> "3.9:foo").
		if v.nameSuffix != "" {
			if pc.activeNameSuffix != "" {
				pc.activeNameSuffix = pc.activeNameSuffix + ":" + v.nameSuffix
			} else {
				pc.activeNameSuffix = v.nameSuffix
			}
		}

		// Accumulate context values (later values override earlier ones with same key).
		pc.activeContextValues = append(pc.activeContextValues, v.contextValues...)

		// 4. Walk inner with cumulative state.
		if err := pc.walk(v.inner); err != nil {
			return err
		}

		// 5. Restore state.
		pc.candidates = prevCandidates
		pc.activeExcludes = prevExcludes
		pc.activeSkips = prevSkips
		pc.currentPath = prevPath
		pc.activeNameSuffix = prevNameSuffix
		pc.activeContextValues = prevContextValues
		pc.activeFlags = prevFlags

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

// TaskInfo represents a task for introspection.
// This is the public type for CI/CD integration (e.g., matrix generation).
type TaskInfo struct {
	Name   string         `json:"name"`            // CLI command name
	Usage  string         `json:"usage,omitempty"` // Description/help text
	Paths  []string       `json:"paths"`           // Directories this task runs in (resolved)
	Flags  map[string]any `json:"flags,omitempty"` // Flag overrides set via pk.WithFlag()
	Hidden bool           `json:"hidden"`          // Whether task is hidden from help
	Manual bool           `json:"manual"`          // Whether task is manual-only
}

// Tasks returns task information for all tasks in the plan.
// This is the public introspection API for CI/CD integration.
func (p *Plan) Tasks() []TaskInfo {
	if p == nil {
		return nil
	}

	result := make([]TaskInfo, 0, len(p.taskInstances))
	for _, instance := range p.taskInstances {
		info := TaskInfo{
			Name:   instance.name, // Use effective name (may include suffix).
			Usage:  instance.task.usage,
			Flags:  instance.flags, // Pre-merged flag overrides from pk.WithFlag().
			Hidden: instance.task.hidden,
			Manual: instance.isManual, // Use pre-computed value (from Config.Manual or Task.Manual()).
		}

		// Use resolved paths from the instance.
		info.Paths = instance.resolvedPaths
		if len(info.Paths) == 0 {
			info.Paths = []string{"."}
		}

		result = append(result, info)
	}

	return result
}

// taskInstanceByName returns the taskInstance for the given effective name.
// Returns nil if not found.
func (p *Plan) taskInstanceByName(name string) *taskInstance {
	if p == nil || p.taskIndex == nil {
		return nil
	}
	return p.taskIndex[name]
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

// checkTaskNameConflicts returns an error if any task names conflict.
// This includes conflicts with builtins and duplicate user task names.
func checkTaskNameConflicts(instances []taskInstance) error {
	seen := make(map[string]bool)

	// Check each task
	for _, instance := range instances {
		if isBuiltinName(instance.name) {
			return fmt.Errorf("⚠️  task name %q conflicts with builtin command; choose a different name", instance.name)
		}
		if seen[instance.name] {
			return fmt.Errorf("⚠️  duplicate task name %q; task names must be unique", instance.name)
		}
		seen[instance.name] = true
	}
	return nil
}
