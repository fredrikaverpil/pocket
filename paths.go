package pocket

import (
	"context"
	"regexp"
	"slices"
	"strings"
)

// Paths wraps a Runnable with path filtering capabilities.
// The returned *PathFilter can be configured with builder methods.
func Paths(r Runnable) *PathFilter {
	return &PathFilter{inner: r}
}

// PathFilter wraps a Runnable with path filtering.
// It implements Runnable, so it can be used anywhere a Runnable is expected.
type PathFilter struct {
	inner     Runnable
	include   []*regexp.Regexp    // explicit include patterns
	exclude   []*regexp.Regexp    // exclusion patterns
	detect    func() []string     // detection function (nil = no detection)
	skipTasks map[string][]string // task name -> paths to skip in (empty = skip everywhere)
}

// In adds include patterns (regexp).
// Directories matching any pattern are included.
// Returns a new *PathFilter (immutable).
func (p *PathFilter) In(patterns ...string) *PathFilter {
	cp := p.clone()
	for _, pat := range patterns {
		cp.include = append(cp.include, regexp.MustCompile("^"+pat+"$"))
	}
	return cp
}

// Except adds exclude patterns (regexp).
// Directories matching any pattern are excluded from results.
// Returns a new *PathFilter (immutable).
func (p *PathFilter) Except(patterns ...string) *PathFilter {
	cp := p.clone()
	for _, pat := range patterns {
		cp.exclude = append(cp.exclude, regexp.MustCompile("^"+pat+"$"))
	}
	return cp
}

// DetectBy sets a custom detection function.
// The function should return directories relative to git root.
// Returns a new *PathFilter (immutable).
func (p *PathFilter) DetectBy(fn func() []string) *PathFilter {
	cp := p.clone()
	cp.detect = fn
	return cp
}

// SkipTask configures a task to be skipped in specific paths.
// If no paths are specified, the task is skipped everywhere within this PathFilter.
// Paths support regex patterns matched against the current execution path.
// Returns a new *PathFilter (immutable).
//
// To make skipped tasks available for manual execution, add them to ManualRun
// with a different name using WithName():
//
//	AutoRun: pocket.Paths(golang.Tasks()).
//	    DetectBy(golang.Detect()).
//	    SkipTask(golang.Test, "services/api", "services/worker"),
//	ManualRun: []pocket.Runnable{
//	    pocket.Paths(golang.Test.WithName("integration-test")).In("services/api", "services/worker"),
//	}
func (p *PathFilter) SkipTask(task *TaskDef, paths ...string) *PathFilter {
	cp := p.clone()
	if cp.skipTasks == nil {
		cp.skipTasks = make(map[string][]string)
	}
	cp.skipTasks[task.Name()] = append(cp.skipTasks[task.Name()], paths...)
	return cp
}

// Resolve returns all directories where this Runnable should run.
// It combines detection results with explicit includes, then filters by excludes.
// Results are sorted and deduplicated.
func (p *PathFilter) Resolve() []string {
	seen := make(map[string]bool)

	// Add detected directories.
	if p.detect != nil {
		for _, dir := range p.detect() {
			seen[dir] = true
		}
	}

	// Filter by includes if any are specified.
	var result []string
	for dir := range seen {
		if p.matches(dir) {
			result = append(result, dir)
		}
	}

	// If no detection but includes are specified, treat them as literal paths.
	if p.detect == nil && len(p.include) > 0 {
		for _, re := range p.include {
			pat := re.String()
			// Remove the ^...$ anchors we added.
			if len(pat) > 2 && pat[0] == '^' && pat[len(pat)-1] == '$' {
				literal := pat[1 : len(pat)-1]
				// Only add if it doesn't contain regex metacharacters.
				if !containsRegexMeta(literal) && !p.isExcluded(literal) {
					if !slices.Contains(result, literal) {
						result = append(result, literal)
					}
				}
			}
		}
	}

	slices.Sort(result)
	return result
}

// RunsIn returns true if this Runnable should run in the given directory.
// The directory should be relative to git root.
func (p *PathFilter) RunsIn(dir string) bool {
	resolved := p.Resolve()
	return slices.Contains(resolved, dir)
}

// ResolveFor returns the resolved paths filtered for the given working directory.
// If cwd is ".", returns all resolved paths.
// Otherwise, returns only paths that match cwd.
func (p *PathFilter) ResolveFor(cwd string) []string {
	resolved := p.Resolve()
	if cwd == "." {
		return resolved
	}
	var result []string
	for _, path := range resolved {
		if path == cwd {
			result = append(result, path)
		}
	}
	return result
}

// run executes the inner Runnable for each resolved path.
func (p *PathFilter) run(ctx context.Context) error {
	ec := getExecContext(ctx)
	paths := p.ResolveFor(ec.cwd)

	for _, path := range paths {
		// Create context with the current path
		pathCtx := withPath(ctx, path)

		// Merge skip rules from this PathFilter into context
		if len(p.skipTasks) > 0 {
			pathCtx = p.mergeSkipRules(pathCtx)
		}

		// Run inner runnable
		if err := p.inner.run(pathCtx); err != nil {
			return err
		}
	}
	return nil
}

// mergeSkipRules merges this PathFilter's skip rules into the context.
func (p *PathFilter) mergeSkipRules(ctx context.Context) context.Context {
	ec := getExecContext(ctx)
	newEC := *ec

	// Create or clone the skip rules map
	if newEC.skipRules == nil {
		newEC.skipRules = make(map[string][]string, len(p.skipTasks))
	} else {
		// Clone to avoid mutating the original
		newSkipRules := make(map[string][]string, len(newEC.skipRules)+len(p.skipTasks))
		for k, v := range newEC.skipRules {
			newSkipRules[k] = v
		}
		newEC.skipRules = newSkipRules
	}

	// Add this PathFilter's skip rules
	for taskName, paths := range p.skipTasks {
		newEC.skipRules[taskName] = append(newEC.skipRules[taskName], paths...)
	}

	return withExecContext(ctx, &newEC)
}

// funcs returns all functions from the inner Runnable.
func (p *PathFilter) funcs() []*TaskDef {
	return p.inner.funcs()
}

// clone creates a shallow copy of PathFilter for immutability.
func (p *PathFilter) clone() *PathFilter {
	cp := &PathFilter{
		inner:   p.inner,
		include: slices.Clone(p.include),
		exclude: slices.Clone(p.exclude),
		detect:  p.detect,
	}
	if p.skipTasks != nil {
		cp.skipTasks = make(map[string][]string, len(p.skipTasks))
		for k, v := range p.skipTasks {
			cp.skipTasks[k] = slices.Clone(v)
		}
	}
	return cp
}

// matches checks if a directory matches the include patterns.
// If no include patterns are specified, all directories match.
func (p *PathFilter) matches(dir string) bool {
	if len(p.include) == 0 {
		return !p.isExcluded(dir)
	}
	for _, re := range p.include {
		if re.MatchString(dir) && !p.isExcluded(dir) {
			return true
		}
	}
	return false
}

// isExcluded checks if a directory matches any exclude pattern.
func (p *PathFilter) isExcluded(dir string) bool {
	for _, re := range p.exclude {
		if re.MatchString(dir) {
			return true
		}
	}
	return false
}

// containsRegexMeta checks if a string contains regex metacharacters.
// Special case: "." alone is treated as a literal path, not a regex.
func containsRegexMeta(s string) bool {
	if s == "." {
		return false
	}
	return strings.ContainsAny(s, `.+*?[](){}|^$\`)
}

// collectPathMappings walks a Runnable tree and returns a map from function name to PathFilter.
// Functions not wrapped with Paths() are not included in the map.
func collectPathMappings(r Runnable) map[string]*PathFilter {
	result := make(map[string]*PathFilter)
	collectPathMappingsRecursive(r, result, nil)
	return result
}

// CollectModuleDirectories walks a Runnable tree and returns all unique directories
// where functions should run. This is used for multi-module shim generation.
func CollectModuleDirectories(r Runnable) []string {
	seen := make(map[string]bool)
	collectModuleDirectoriesRecursive(r, seen)
	// Always include root.
	seen["."] = true
	result := make([]string, 0, len(seen))
	for dir := range seen {
		result = append(result, dir)
	}
	slices.Sort(result)
	return result
}

// collectModuleDirectoriesRecursive is the recursive helper for CollectModuleDirectories.
func collectModuleDirectoriesRecursive(r Runnable, seen map[string]bool) {
	if r == nil {
		return
	}

	// Check if this is a PathFilter wrapper.
	if p, ok := r.(*PathFilter); ok {
		for _, dir := range p.Resolve() {
			seen[dir] = true
		}
		// Continue with the inner runnable.
		collectModuleDirectoriesRecursive(p.inner, seen)
		return
	}

	// Check if it's a group type with runnables.
	switch v := r.(type) {
	case *serial:
		for _, child := range v.items {
			collectModuleDirectoriesRecursive(child, seen)
		}
	case *parallel:
		for _, child := range v.items {
			collectModuleDirectoriesRecursive(child, seen)
		}
	}
}

// collectPathMappingsRecursive is the recursive helper for collectPathMappings.
func collectPathMappingsRecursive(r Runnable, result map[string]*PathFilter, currentPaths *PathFilter) {
	if r == nil {
		return
	}

	// Check if this is a PathFilter wrapper.
	if p, ok := r.(*PathFilter); ok {
		currentPaths = p
		// Continue with the inner runnable.
		collectPathMappingsRecursive(p.inner, result, currentPaths)
		return
	}

	// Check if this is a TaskDef.
	if f, ok := r.(*TaskDef); ok {
		if currentPaths != nil {
			result[f.name] = currentPaths
		}
		return
	}

	// For group types, recurse into children.
	switch v := r.(type) {
	case *serial:
		for _, child := range v.items {
			collectPathMappingsRecursive(child, result, currentPaths)
		}
	case *parallel:
		for _, child := range v.items {
			collectPathMappingsRecursive(child, result, currentPaths)
		}
	default:
		// For unknown types, just collect functions without path mapping.
		for _, fn := range r.funcs() {
			if currentPaths != nil {
				result[fn.name] = currentPaths
			}
		}
	}
}
