package golang

import "github.com/fredrikaverpil/pocket"

// AutoConfig defines configuration for auto-detecting Go modules.
type AutoConfig struct {
	// Options are the default options applied to all auto-detected modules.
	Options Options
	// Overrides specifies custom options for specific module paths.
	// These paths must still contain a go.mod file to be detected.
	Overrides map[string]Options
}

// Auto creates a Go task group that auto-detects modules by finding go.mod files.
func Auto(cfg ...AutoConfig) pocket.TaskGroup {
	var config AutoConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return pocket.NewAutoTaskGroup(
		name,
		func() []string { return pocket.DetectByFile("go.mod") },
		config.Options,
		config.Overrides,
		func(modules map[string]Options) pocket.TaskGroup {
			return &taskGroup{config: Config{Modules: modules}}
		},
	)
}
