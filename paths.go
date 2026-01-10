// Package pocket provides core utilities for the pocket build system.
package pocket

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
)

const (
	// DirName is the name of the pocket directory.
	DirName = ".pocket"
	// ToolsDirName is the name of the tools subdirectory.
	ToolsDirName = "tools"
	// BinDirName is the name of the bin subdirectory (for symlinks).
	BinDirName = "bin"
)

var (
	gitRootOnce sync.Once
	gitRoot     string
)

// GitRoot returns the root directory of the git repository.
func GitRoot() string {
	gitRootOnce.Do(func() {
		var err error
		gitRoot, err = findGitRoot()
		if err != nil {
			panic("pocket: unable to find git root: " + err.Error())
		}
	})
	return gitRoot
}

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// FromGitRoot returns a path relative to the git root.
func FromGitRoot(elem ...string) string {
	return filepath.Join(append([]string{GitRoot()}, elem...)...)
}

// FromPocketDir returns a path relative to the .pocket directory.
func FromPocketDir(elem ...string) string {
	return FromGitRoot(append([]string{DirName}, elem...)...)
}

// FromToolsDir returns a path relative to the .pocket/tools directory.
func FromToolsDir(elem ...string) string {
	return FromPocketDir(append([]string{ToolsDirName}, elem...)...)
}

// FromBinDir returns a path relative to the .pocket/bin directory.
// If no elements are provided, returns the bin directory itself.
func FromBinDir(elem ...string) string {
	return FromPocketDir(append([]string{BinDirName}, elem...)...)
}

// BinaryName returns the binary name with the correct extension for the current OS.
// On Windows, it appends ".exe" to the name.
func BinaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// Paths wraps a Runnable with path filtering capabilities.
// The returned *PathFilter can be configured with builder methods.
func Paths(r Runnable) *PathFilter {
	return &PathFilter{inner: r}
}

// AutoDetect wraps a Runnable with automatic directory detection.
// This is a convenience function equivalent to Paths(r).Detect().
// The Runnable should implement Detectable for this to work.
func AutoDetect(r Runnable) *PathFilter {
	return Paths(r).Detect()
}

// PathFilter wraps a Runnable with path filtering.
// It implements Runnable, so it can be used anywhere a Runnable is expected.
type PathFilter struct {
	inner     Runnable
	include   []*regexp.Regexp // explicit include patterns
	exclude   []*regexp.Regexp // exclusion patterns
	detect    func() []string  // detection function (nil = no detection)
	skipRules []skipRule       // task skip rules
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

// Include is an alias for In.
func (p *PathFilter) Include(patterns ...string) *PathFilter {
	return p.In(patterns...)
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

// Detect enables detection using the inner Runnable's DefaultDetect method.
// If the inner Runnable doesn't implement Detectable, this has no effect.
// Returns a new *PathFilter (immutable).
func (p *PathFilter) Detect() *PathFilter {
	cp := p.clone()
	if d, ok := cp.inner.(Detectable); ok {
		cp.detect = d.DefaultDetect()
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

// Skip excludes a task from execution.
// With no paths: skip everywhere, task is hidden from CLI.
// With paths: skip only in those paths, task remains visible in CLI.
//
// Examples:
//
//	Skip(golang.TestTask())                    // skip everywhere
//	Skip(golang.TestTask(), "docs")            // skip only in docs
//	Skip(golang.TestTask(), "docs", "examples") // skip in docs and examples
//
// Returns a new *PathFilter (immutable).
func (p *PathFilter) Skip(task *Task, paths ...string) *PathFilter {
	if task == nil || task.Name == "" {
		return p
	}
	cp := p.clone()
	cp.skipRules = append(cp.skipRules, skipRule{
		taskName: task.Name,
		paths:    paths,
	})
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

	// Add explicitly included directories (patterns that are literal paths).
	// Note: For patterns with regex characters, we can't add them literally.
	// The include patterns are primarily for filtering, not adding.

	// Filter by includes if any are specified.
	var result []string
	for dir := range seen {
		if p.matches(dir) {
			result = append(result, dir)
		}
	}

	// If no detection but includes are specified, we need to handle literal includes.
	// For now, if detect is nil and includes are specified, treat them as literal paths.
	if p.detect == nil && len(p.include) > 0 {
		for _, re := range p.include {
			// Check if the pattern is a literal (no regex special chars).
			// For simplicity, just use the pattern string if it matches itself.
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

// Run executes the inner Runnable after setting resolved paths on tasks.
func (p *PathFilter) Run(ctx context.Context) error {
	rc := getRunContext(ctx)
	paths := p.ResolveFor(rc.cwd)

	// Set paths on all tasks in the inner Runnable.
	for _, task := range p.inner.Tasks() {
		task.SetPaths(paths)
	}

	// Set skip rules in context so tasks can check if they should be skipped.
	if len(p.skipRules) > 0 {
		ctx = withSkipRules(ctx, p.skipRules)
	}

	return p.inner.Run(ctx)
}

// Tasks returns all tasks from the inner Runnable, excluding globally skipped tasks.
// Tasks with path-specific skips are still included (they run in other paths).
func (p *PathFilter) Tasks() []*Task {
	tasks := p.inner.Tasks()
	if len(p.skipRules) == 0 {
		return tasks
	}
	// Build set of globally skipped task names (rules with no paths).
	globalSkips := make(map[string]bool)
	for _, rule := range p.skipRules {
		if len(rule.paths) == 0 {
			globalSkips[rule.taskName] = true
		}
	}
	if len(globalSkips) == 0 {
		return tasks
	}
	// Filter out globally skipped tasks.
	result := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		if !globalSkips[task.Name] {
			result = append(result, task)
		}
	}
	return result
}

// clone creates a shallow copy of PathFilter for immutability.
func (p *PathFilter) clone() *PathFilter {
	cp := &PathFilter{
		inner:     p.inner,
		include:   slices.Clone(p.include),
		exclude:   slices.Clone(p.exclude),
		detect:    p.detect,
		skipRules: slices.Clone(p.skipRules),
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
	// "." alone is the current directory, not a regex wildcard.
	if s == "." {
		return false
	}
	return strings.ContainsAny(s, `.+*?[](){}|^$\`)
}

// CollectPathMappings walks a Runnable tree and returns a map from task name to PathFilter.
// Tasks not wrapped with Paths() are not included in the map.
func CollectPathMappings(r Runnable) map[string]*PathFilter {
	result := make(map[string]*PathFilter)
	collectPathMappingsRecursive(r, result, nil)
	return result
}

// CollectModuleDirectories walks a Runnable tree and returns all unique directories
// where tasks should run. This is used for multi-module shim generation.
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

	// For other Runnables (serial, parallel), recurse into children.
	if v, ok := r.(interface{ Children() []Runnable }); ok {
		for _, child := range v.Children() {
			collectModuleDirectoriesRecursive(child, seen)
		}
	}
}

// collectPathMappingsRecursive is the recursive helper for CollectPathMappings.
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

	// Check if this is a Task.
	if t, ok := r.(*Task); ok {
		if currentPaths != nil {
			result[t.Name] = currentPaths
		}
		return
	}

	// For other Runnables (serial, parallel), recurse into children.
	// We need to check for known types.
	switch v := r.(type) {
	case interface{ Children() []Runnable }:
		for _, child := range v.Children() {
			collectPathMappingsRecursive(child, result, currentPaths)
		}
	default:
		// For unknown types, just collect tasks without path mapping.
		for _, task := range r.Tasks() {
			if currentPaths != nil {
				result[task.Name] = currentPaths
			}
		}
	}
}
