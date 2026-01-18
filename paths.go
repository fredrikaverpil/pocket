package pocket

import (
	"context"
	"regexp"
	"slices"
	"strings"
)

// PathOpt configures path filtering behavior for RunIn.
type PathOpt func(*PathFilter)

// RunIn wraps a Runnable with path filtering.
// Use options to control where the Runnable executes.
//
// Example:
//
//	pocket.RunIn(golang.Tasks(),
//	    pocket.Detect(golang.Detect()),
//	    pocket.Include("services/.*"),
//	    pocket.Exclude("vendor"),
//	)
func RunIn(r Runnable, opts ...PathOpt) *PathFilter {
	pf := &PathFilter{inner: r}
	for _, opt := range opts {
		opt(pf)
	}
	return pf
}

// Detect sets a detection function that returns directories
// where the Runnable should execute.
// The function should return directories relative to git root.
func Detect(fn func() []string) PathOpt {
	return func(pf *PathFilter) {
		pf.detect = fn
	}
}

// Include adds patterns (regex) for directories to include.
// Directories matching any pattern are included.
func Include(patterns ...string) PathOpt {
	return func(pf *PathFilter) {
		for _, pat := range patterns {
			pf.include = append(pf.include, regexp.MustCompile("^"+pat+"$"))
		}
	}
}

// Exclude adds patterns (regex) for directories to exclude.
// Directories matching any pattern are excluded from results.
func Exclude(patterns ...string) PathOpt {
	return func(pf *PathFilter) {
		for _, pat := range patterns {
			pf.exclude = append(pf.exclude, regexp.MustCompile("^"+pat+"$"))
		}
	}
}

// Skip configures a task to be skipped in specific paths.
// If no paths are specified, the task is skipped everywhere within this filter.
// Paths support regex patterns matched against the current execution path.
//
// To make skipped tasks available for manual execution, add them to ManualRun
// with a different name using WithName():
//
//	AutoRun: pocket.RunIn(golang.Tasks(),
//	    pocket.Detect(golang.Detect()),
//	    pocket.Skip(golang.Test, "services/api", "services/worker"),
//	),
//	ManualRun: []pocket.Runnable{
//	    pocket.RunIn(golang.Test.WithName("integration-test"),
//	        pocket.Include("services/api", "services/worker"),
//	    ),
//	}
func Skip(task *TaskDef, paths ...string) PathOpt {
	return func(pf *PathFilter) {
		if pf.skipTasks == nil {
			pf.skipTasks = make(map[string][]string)
		}
		pf.skipTasks[task.Name()] = append(pf.skipTasks[task.Name()], paths...)
	}
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

	// In collect mode, set path context and walk once (don't iterate paths)
	if ec.mode == modeCollect {
		prev := ec.plan.setPathContext(p)
		err := p.inner.run(ctx)
		ec.plan.setPathContext(prev)
		return err
	}

	// Execute mode: run for each resolved path
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
// Tasks that are skipped everywhere (Skip with no paths) are filtered out.
func (p *PathFilter) funcs() []*TaskDef {
	inner := p.inner.funcs()
	if len(p.skipTasks) == 0 {
		return inner
	}

	// Filter out tasks that are skipped everywhere (empty paths = skip everywhere)
	result := make([]*TaskDef, 0, len(inner))
	for _, f := range inner {
		paths, exists := p.skipTasks[f.Name()]
		if exists && len(paths) == 0 {
			// Skipped everywhere - exclude from funcs
			continue
		}
		result = append(result, f)
	}
	return result
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
