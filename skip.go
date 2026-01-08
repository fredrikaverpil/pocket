package pocket

import "regexp"

// SkipOption configures skip behavior for Auto mode.
type SkipOption func(*skipConfig)

// skipConfig holds the skip configuration for a TaskGroupDef.
type skipConfig struct {
	pathPatterns []*regexp.Regexp            // patterns to skip entirely
	taskPatterns map[string][]*regexp.Regexp // task -> path patterns to skip
	showAll      bool                        // show orchestrator task in help
}

func newSkipConfig(opts ...SkipOption) *skipConfig {
	cfg := &skipConfig{
		taskPatterns: make(map[string][]*regexp.Regexp),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// shouldSkipPath returns true if all tasks should be skipped for the given path.
func (c *skipConfig) shouldSkipPath(path string) bool {
	for _, re := range c.pathPatterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// shouldSkipTask returns true if the given task should be skipped for the given path.
func (c *skipConfig) shouldSkipTask(taskName, path string) bool {
	patterns, ok := c.taskPatterns[taskName]
	if !ok {
		return false
	}
	for _, re := range patterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// SkipPath skips all tasks for paths matching the regex pattern.
// Example: pocket.SkipPath(`\.pocket`) skips all tasks in .pocket/.
func SkipPath(pattern string) SkipOption {
	return func(cfg *skipConfig) {
		re := regexp.MustCompile(pattern)
		cfg.pathPatterns = append(cfg.pathPatterns, re)
	}
}

// SkipTask skips a specific task for paths matching the regex pattern.
// Example: pocket.SkipTask("go-vulncheck", `.*`) skips vulncheck everywhere.
func SkipTask(taskName, pathPattern string) SkipOption {
	return func(cfg *skipConfig) {
		re := regexp.MustCompile(pathPattern)
		cfg.taskPatterns[taskName] = append(cfg.taskPatterns[taskName], re)
	}
}

// ShowAll makes the orchestrator task (e.g., go-all) visible in help.
// By default, orchestrator tasks are hidden.
func ShowAll() SkipOption {
	return func(cfg *skipConfig) {
		cfg.showAll = true
	}
}
