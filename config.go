package bld

// Config defines the configuration for a project using bld.
type Config struct {
	// Language configurations
	Go *GoConfig
	// Python *PythonConfig  // future
	// Lua    *LuaConfig     // future

	// CI platforms
	GitHub *GitHubConfig
}

// GoConfig defines Go project configuration.
type GoConfig struct {
	// Modules maps folder paths to their options.
	// Use "." for the root module.
	Modules map[string]GoModuleOptions
}

// GoModuleOptions defines options for a Go module.
type GoModuleOptions struct {
	SkipFormat    bool
	SkipTest      bool
	SkipLint      bool
	SkipVulncheck bool
}

// GitHubConfig defines GitHub Actions workflow configuration.
type GitHubConfig struct {
	// ExtraGoVersions specifies additional Go versions to test against,
	// beyond the versions extracted from go.mod files.
	// Uses setup-go syntax: "stable", "oldstable", or specific versions like "1.22".
	ExtraGoVersions []string

	// OSVersions specifies runner OS versions.
	// Default: ["ubuntu-latest"]
	OSVersions []string

	// Skip flags for generic workflows
	SkipPR      bool
	SkipStale   bool
	SkipRelease bool
	SkipSync    bool
}

// WithDefaults returns a copy of the config with default values applied.
func (c Config) WithDefaults() Config {
	if c.GitHub != nil {
		gh := *c.GitHub
		if len(gh.OSVersions) == 0 {
			gh.OSVersions = []string{"ubuntu-latest"}
		}
		c.GitHub = &gh
	}
	return c
}

// HasGo returns true if the project has Go modules configured.
func (c Config) HasGo() bool {
	return c.Go != nil && len(c.Go.Modules) > 0
}

// GoModulesForFormat returns module paths where format is not skipped.
func (c Config) GoModulesForFormat() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipFormat {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForTest returns module paths where test is not skipped.
func (c Config) GoModulesForTest() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipTest {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForLint returns module paths where lint is not skipped.
func (c Config) GoModulesForLint() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipLint {
			paths = append(paths, path)
		}
	}
	return paths
}

// GoModulesForVulncheck returns module paths where vulncheck is not skipped.
func (c Config) GoModulesForVulncheck() []string {
	if c.Go == nil {
		return nil
	}
	var paths []string
	for path, opts := range c.Go.Modules {
		if !opts.SkipVulncheck {
			paths = append(paths, path)
		}
	}
	return paths
}
