package python

import "github.com/fredrikaverpil/pocket"

// AutoConfig defines configuration for auto-detecting Python modules.
type AutoConfig struct {
	// Options are the default options applied to all auto-detected modules.
	Options Options
	// Overrides specifies custom options for specific module paths.
	// These paths must still contain a pyproject.toml file to be detected.
	Overrides map[string]Options
}

// Auto creates a Python task group that auto-detects modules by finding
// pyproject.toml, setup.py, or setup.cfg files.
func Auto(cfg ...AutoConfig) pocket.TaskGroup {
	var config AutoConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return pocket.NewAutoTaskGroup(
		name,
		func() []string { return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg") },
		config.Options,
		config.Overrides,
		func(modules map[string]Options) pocket.TaskGroup {
			return &taskGroup{modules: modules}
		},
	)
}
