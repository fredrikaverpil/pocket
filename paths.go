package pocket

import (
	"context"
	"regexp"
	"slices"
)

// P wraps a Runnable with path filtering capabilities.
// The returned *Paths can be configured with builder methods.
func P(r Runnable) *Paths {
	return &Paths{inner: r}
}

// Paths wraps a Runnable with path filtering.
// It implements Runnable, so it can be used anywhere a Runnable is expected.
type Paths struct {
	inner   Runnable
	include []*regexp.Regexp // explicit include patterns
	exclude []*regexp.Regexp // exclusion patterns
	detect  func() []string  // detection function (nil = no detection)
}

// In adds include patterns (regexp).
// Directories matching any pattern are included.
// Returns a new *Paths (immutable).
func (p *Paths) In(patterns ...string) *Paths {
	cp := p.clone()
	for _, pat := range patterns {
		cp.include = append(cp.include, regexp.MustCompile("^"+pat+"$"))
	}
	return cp
}

// Include is an alias for In.
func (p *Paths) Include(patterns ...string) *Paths {
	return p.In(patterns...)
}

// Except adds exclude patterns (regexp).
// Directories matching any pattern are excluded from results.
// Returns a new *Paths (immutable).
func (p *Paths) Except(patterns ...string) *Paths {
	cp := p.clone()
	for _, pat := range patterns {
		cp.exclude = append(cp.exclude, regexp.MustCompile("^"+pat+"$"))
	}
	return cp
}

// Detect enables detection using the inner Runnable's DefaultDetect method.
// If the inner Runnable doesn't implement Detectable, this has no effect.
// Returns a new *Paths (immutable).
func (p *Paths) Detect() *Paths {
	cp := p.clone()
	if d, ok := cp.inner.(Detectable); ok {
		cp.detect = d.DefaultDetect()
	}
	return cp
}

// DetectBy sets a custom detection function.
// The function should return directories relative to git root.
// Returns a new *Paths (immutable).
func (p *Paths) DetectBy(fn func() []string) *Paths {
	cp := p.clone()
	cp.detect = fn
	return cp
}

// Resolve returns all directories where this Runnable should run.
// It combines detection results with explicit includes, then filters by excludes.
// Results are sorted and deduplicated.
func (p *Paths) Resolve() []string {
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
func (p *Paths) RunsIn(dir string) bool {
	resolved := p.Resolve()
	return slices.Contains(resolved, dir)
}

// Run executes the inner Runnable.
// Path filtering happens at CLI level, not during Run.
func (p *Paths) Run(ctx context.Context) error {
	return p.inner.Run(ctx)
}

// Tasks returns all tasks from the inner Runnable.
func (p *Paths) Tasks() []*Task {
	return p.inner.Tasks()
}

// clone creates a shallow copy of Paths for immutability.
func (p *Paths) clone() *Paths {
	return &Paths{
		inner:   p.inner,
		include: slices.Clone(p.include),
		exclude: slices.Clone(p.exclude),
		detect:  p.detect,
	}
}

// matches checks if a directory matches the include patterns.
// If no include patterns are specified, all directories match.
func (p *Paths) matches(dir string) bool {
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
func (p *Paths) isExcluded(dir string) bool {
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
	for _, c := range s {
		switch c {
		case '.', '*', '+', '?', '[', ']', '(', ')', '{', '}', '|', '^', '$', '\\':
			return true
		}
	}
	return false
}

// CollectPathMappings walks a Runnable tree and returns a map from task name to Paths.
// Tasks not wrapped with Paths are not included in the map.
func CollectPathMappings(r Runnable) map[string]*Paths {
	result := make(map[string]*Paths)
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

	// Check if this is a Paths wrapper.
	if p, ok := r.(*Paths); ok {
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
func collectPathMappingsRecursive(r Runnable, result map[string]*Paths, currentPaths *Paths) {
	if r == nil {
		return
	}

	// Check if this is a Paths wrapper.
	if p, ok := r.(*Paths); ok {
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
